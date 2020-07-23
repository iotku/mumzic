package youtubedl

import (
	"bytes"
	"layeh.com/gumble/gumbleffmpeg"
	"log"
	"os/exec"
	"strings"
)

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

func GetYtdlTitle(url string) string {
	ytdl := exec.Command("youtube-dl", "-e", url)
	var output bytes.Buffer
	ytdl.Stdout = &output
	err := ytdl.Run()
	if err != nil {
		log.Println("Youtube-DL failed to get title for", url)
		return ""
	}
	return output.String()
}

func GetYtdlSource(url string) gumbleffmpeg.Source {
	// TODO: Enforce --no-playlist
	gumbleSource := gumbleffmpeg.SourceExec("youtube-dl", "-f", "bestaudio", "--rm-cache-dir", "-q", "-o", "-", url)
	return gumbleSource
}
