package youtubedl

import (
	"bytes"
	"log"
	"os/exec"
	"strings"

	"layeh.com/gumble/gumbleffmpeg"
)

// IsWhiteListedURL returns true if URL begins with an acceptable URL for ytdl
func IsWhiteListedURL(url string) bool {
	// ! Don't forget to end url with / !
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
	// TODO: Make special handling for playlists?
	return gumbleffmpeg.SourceExec("yt-dlp", "--no-playlist", "-f", "bestaudio", "--rm-cache-dir", "-q", "-o", "-", url)
}
