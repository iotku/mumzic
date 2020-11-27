package playlist

import (
	"strconv"
	"strings"

	"github.com/iotku/mumzic/helper"
	"github.com/iotku/mumzic/search"
	"github.com/iotku/mumzic/youtubedl"
	"layeh.com/gumble/gumble"
)

// Playlist elements
var Songlist = make([]string, 0) // Contains Raw paths (filepath or url)
var Metalist = make([]string, 0) // Contains "Human" readable Titles corresponding to Songlist entry

// Position In playlist
var Currentsong = 0

func QueueID(trackID int) (human string) {
	if trackID > search.MaxDBID || trackID < 1 {
		return ""
	}
	path, human := search.GetTrackById(trackID)
	if path == "" {
		return ""
	}
	Songlist = append(Songlist, path)
	Metalist = append(Metalist, human)

	return human
}

func QueueYT(url, human string) string {
	// TODO Check with API if video is valid for youtube links
	Songlist = append(Songlist, url)
	Metalist = append(Metalist, human)
	return human
}

func AddToQueue(isPrivate bool, sender string, path string, client *gumble.Client) bool {
	// For YTDL URLS
	path = helper.StripHTMLTags(path)
	if strings.HasPrefix(path, "http") && youtubedl.IsWhiteListedURL(path) == true {
		// add to playlist
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
