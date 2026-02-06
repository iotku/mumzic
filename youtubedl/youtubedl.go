package youtubedl

import (
	"bufio"
	"bytes"
	_ "embed"
	"errors"
	"image"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"layeh.com/gumble/gumbleffmpeg"
)

// ! Don't forget to end url prefix with / !
var AllowedURLPrefixes []string
var whitelistFile = "whitelist.txt"

func init() {
	err := LoadAllowedURLPrefixesFromFile()
	if err != nil {
		log.Printf("failed to load allowed URLs: %v", err)
	}

	log.Printf("Allowed URL prefixes: %v", AllowedURLPrefixes)
}

// LoadAllowedURLPrefixesFromFile loads prefixes from each line of a provided txt file
func LoadAllowedURLPrefixesFromFile() error {
	f, err := os.Open(whitelistFile) // #nosec G304 - Internal Helper method
	if err != nil {
		return err
	}
	defer f.Close()

	return scanURLPrefixes(bufio.NewScanner(f))
}

func scanURLPrefixes(scanner *bufio.Scanner) error {
	AllowedURLPrefixes = AllowedURLPrefixes[:0] // reset
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") { // Ignore # comments
			continue
		}

		if !strings.HasPrefix(line, "http://") && !strings.HasPrefix(line, "https://") {
			log.Println("Invalid entry (" + line + ") must start with http:// or https:// ")
			continue
		}

		if !strings.HasSuffix(line, "/") { // We're not enforcing this, but this is a good idea.
			log.Println("[WARN] " + whitelistFile + ": " + line + " does not end with trailing / This may allow yt-dl URL bypasses.")
		}

		AllowedURLPrefixes = append(AllowedURLPrefixes, line)

	}

	return scanner.Err()
}

// IsWhiteListedURL returns true if URL begins with an acceptable URL for ytdl
func IsWhiteListedURL(url string) bool {
	if len(AllowedURLPrefixes) == 0 {
		log.Println("IsWhiteListedURL was called, but there are no allowedURLPrefixes")
	}

	for _, prefix := range AllowedURLPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

// SearchYouTube searches for a video on YouTube using yt-dlp and returns the first result URL
func SearchYouTube(query string) (string, error) {
	if !IsWhiteListedURL("https://www.youtube.com/") {
		return "", errors.New("YouTube is not whitelisted")
	}
	ytDL := exec.Command("yt-dlp", "--no-playlist", "--get-id", "--default-search", "ytsearch1", query)
	var output bytes.Buffer
	ytDL.Stdout = &output
	err := ytDL.Run()
	if err != nil {
		return "", errors.New("YouTube search failed for: " + query)
	}

	videoID := strings.TrimSpace(output.String())
	if videoID == "" {
		return "", errors.New("YouTube search found no results")
	}

	// Convert video ID to full YouTube URL
	return "https://www.youtube.com/watch?v=" + videoID, nil
}

func GetYtDLTitle(url string) (string, error) {
	ytDL := exec.Command("yt-dlp", "--no-playlist", "-e", url)
	var output bytes.Buffer
	ytDL.Stdout = &output
	err := ytDL.Run()
	if err != nil {
		return url, errors.New("YDL failed to get title for: " + url)
	}
	return output.String(), nil
}

func GetYtDLSource(url string) gumbleffmpeg.Source {
	return gumbleffmpeg.SourceExec("yt-dlp", "--no-playlist", "-f", "bestaudio", "--rm-cache-dir", "-q", "-o", "-", url)
}

// GetYtDLThumbnail fetches the thumbnail for a YouTube video and returns it as base64-encoded data
func GetYtDLThumbnail(url string) (image.Image, error) {
	// get the thumbnail URL using yt-dlp
	ytDL := exec.Command("yt-dlp", "--no-playlist", "--get-thumbnail", url)
	var output bytes.Buffer
	ytDL.Stdout = &output
	err := ytDL.Run()
	if err != nil {
		return nil, errors.New("Youtube-DL failed to get thumbnail URL for " + url + ": " + err.Error())
	}

	thumbnailURL := strings.TrimSpace(output.String())
	if thumbnailURL == "" {
		return nil, errors.New("No thumbnail URL found for " + url)
	}

	// If the URL is WebP, try to get a JPEG version instead
	if strings.Contains(thumbnailURL, ".webp") {
		// Try to get a different thumbnail format by modifying the URL
		// YouTube thumbnails often have multiple formats available
		jpegURL := strings.Replace(thumbnailURL, ".webp", ".jpg", 1)
		jpegURL = strings.Replace(jpegURL, "_webp", "", 1)

		// Test if the JPEG URL exists
		// #nosec G107 -- thumbnailURL is from yt-dlp for a whitelisted video, considered safe
		resp, err := http.Head(jpegURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			thumbnailURL = jpegURL
		}
	}

	// Download the thumbnail
	// #nosec G107
	resp, err := http.Get(thumbnailURL)
	if err != nil {
		return nil, errors.New("Failed to download thumbnail from " + thumbnailURL + ": " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Failed to download thumbnail, status: " + strconv.Itoa(resp.StatusCode))
	}

	// Decode the image
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, errors.New("Failed to decode thumbnail image: " + err.Error())
	}

	return img, nil
}
