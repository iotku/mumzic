package youtubedl

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/nfnt/resize"
	"layeh.com/gumble/gumbleffmpeg"
)

// IsWhiteListedURL returns true if URL begins with an acceptable URL for ytdl
func IsWhiteListedURL(url string) bool {
	whiteListedURLS := []string{"https://www.youtube.com/", "https://music.youtube.com/", "https://youtu.be/", "https://soundcloud.com/"}
	for i := range whiteListedURLS {
		if strings.HasPrefix(url, whiteListedURLS[i]) {
			return true
		}
	}
	return false
}

// SearchYouTube searches for a video on YouTube using yt-dlp and returns the first result URL
func SearchYouTube(query string) string {
	ytDL := exec.Command("yt-dlp", "--no-playlist", "--get-id", "--default-search", "ytsearch1", query)
	var output bytes.Buffer
	ytDL.Stdout = &output
	err := ytDL.Run()
	if err != nil {
		log.Println("Youtube-DL failed to search for:", query)
		return ""
	}
	
	videoID := strings.TrimSpace(output.String())
	if videoID == "" {
		return ""
	}
	
	// Convert video ID to full YouTube URL
	return "https://www.youtube.com/watch?v=" + videoID
}

func GetYtDLTitle(url string) string {
	ytDL := exec.Command("yt-dlp", "--no-playlist", "-e", url)
	var output bytes.Buffer
	ytDL.Stdout = &output
	err := ytDL.Run()
	if err != nil {
		log.Println("Youtube-DL failed to get title for", url)
		return ""
	}
	return output.String()
}

func GetYtDLSource(url string) gumbleffmpeg.Source {
	return gumbleffmpeg.SourceExec("yt-dlp", "--no-playlist", "-f", "bestaudio", "--rm-cache-dir", "-q", "-o", "-", url)
}

// GetYtDLThumbnail fetches the thumbnail for a YouTube video and returns it as base64-encoded data
func GetYtDLThumbnail(url string) string {
	
	// get the thumbnail URL using yt-dlp
	ytDL := exec.Command("yt-dlp", "--no-playlist", "--get-thumbnail", url)
	var output bytes.Buffer
	ytDL.Stdout = &output
	err := ytDL.Run()
	if err != nil {
		log.Println("Youtube-DL failed to get thumbnail URL for", url, ":", err)
		return ""
	}
	
	thumbnailURL := strings.TrimSpace(output.String())
	if thumbnailURL == "" {
		log.Println("No thumbnail URL found for", url)
		return ""
	}
	
	// If the URL is WebP, try to get a JPEG version instead
	if strings.Contains(thumbnailURL, ".webp") {
		// Try to get a different thumbnail format by modifying the URL
		// YouTube thumbnails often have multiple formats available
		jpegURL := strings.Replace(thumbnailURL, ".webp", ".jpg", 1)
		jpegURL = strings.Replace(jpegURL, "_webp", "", 1)
		
		// Test if the JPEG URL exists
		resp, err := http.Head(jpegURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			thumbnailURL = jpegURL
		}
	}
	
	// Download the thumbnail
	resp, err := http.Get(thumbnailURL)
	if err != nil {
		log.Println("Failed to download thumbnail from", thumbnailURL, ":", err)
		return ""
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Println("Failed to download thumbnail, status:", resp.StatusCode)
		return ""
	}
	
	// Decode the image
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		log.Println("Failed to decode thumbnail image:", err)
		return ""
	}
	
	// Resize the image to 100x100 like local cover art for consistency, possible change it later
	resizedImg := resize.Resize(100, 100, img, resize.Lanczos3)
	
	// Compress
	jpegQuality := 60
	maxSize := 4000
	var buf bytes.Buffer
	var encodedStr string
	
	for maxSize >= 4000 && jpegQuality > 0 {
		buf.Reset()
		options := jpeg.Options{Quality: jpegQuality}
		if err := jpeg.Encode(&buf, resizedImg, &options); err != nil {
			log.Println("Error encoding jpg for base64:", err)
			return ""
		}
		encodedStr = "<img src=\"data:image/jpeg;base64, " + base64.StdEncoding.EncodeToString(buf.Bytes()) + "\" />"
		maxSize = len(encodedStr)
		jpegQuality -= 10
	}
	
	// Check if the image is too large
	if len(encodedStr) > 4850 { // MaxMessageLengthWithoutImage-150 = 5000-150 = 4850
		return "" // Don't return thumbnail it's too big
	}
	
	return encodedStr
}
