package playlist

import (
	"strconv"
	"strings"

	"github.com/iotku/mumzic/helper"
	"github.com/iotku/mumzic/search"
	"github.com/iotku/mumzic/youtubedl"
	"layeh.com/gumble/gumble"
)

// Raw path // Human path
var Playlist [][]string

// Position In Playlist
var Position = 0

func pAdd(path, human string) {
	Playlist = append(Playlist, []string{path, human})
}

func GetCurrentPath() string {
	return Playlist[Position][0]
}

func GetCurrentHuman() string {
	return Playlist[Position][1]
}

func GetList(max int) []string {
	var trackList []string
	for i := Position; i < Position+max; i++ {
		if Position+max > Size() {
			return trackList
		}
		trackList = append(trackList, Playlist[i][1])
	}

	return trackList
}

func HasNext() bool {
	return len(Playlist) > Position+1
}

func Next() string {
	if !HasNext() {
		return ""
	}
	Position++
	return GetCurrentPath()
}

func Skip(amount int) string {
	if Size()+amount < 0 || !HasNext() {
		return ""
	}

	if Position+amount > Size() {
		amount = 1 // only skip one track
	}
	Position = Position + amount
	return GetCurrentPath()
}

func Size() int {
	return len(Playlist)
}

func QueueID(trackID int) (human string) {
	if trackID > search.MaxDBID || trackID < 1 {
		return ""
	}
	path, human := search.GetTrackById(trackID)
	if path == "" {
		return ""
	}
	pAdd(path, human)

	return human
}

func QueueYT(url, human string) string {
	// TODO Check with API if video is valid for youtube links
	pAdd(url, human)
	return human
}

func AddToQueue(isPrivate bool, sender string, path string, client *gumble.Client) bool {
	// For YTDL URLS
	path = helper.StripHTMLTags(path)
	if strings.HasPrefix(path, "http") && youtubedl.IsWhiteListedURL(path) == true {
		// add to Playlist
		// Get "Human" from web page title (I hope this doesn't trigger anti-spam...)
		var human string
		title := youtubedl.GetYtdlTitle(path)
		helper.DebugPrintln("Title:", title)
		if title != "" {
			human = title
		} else {
			human = path
		}
		QueueYT(path, human)
		helper.MsgDispatch(isPrivate, sender, client, "Added: "+human)
		return true
	} else if strings.HasPrefix(path, "http") == true {
		helper.MsgDispatch(isPrivate, sender, client, "URL Doesn't meet whitelist, sorry.")
		return false
	}

	// FOR IDs
	idn, _ := strconv.Atoi(path)
	human := QueueID(idn)
	if human != "" {
		helper.MsgDispatch(isPrivate, sender, client, "Added: "+human)
		return true
	}

	helper.MsgDispatch(isPrivate, sender, client, "Nothing added. (Invalid ID?)")
	return false
}
