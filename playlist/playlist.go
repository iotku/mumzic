package playlist

import (
	"errors"
	"strconv"
	"strings"

	"github.com/iotku/mumzic/helper"
	"github.com/iotku/mumzic/search"
	"github.com/iotku/mumzic/youtubedl"
)

type List struct {
	Playlist [][]string
	Position int
}

func (list *List) pAdd(path, human string) {
	list.Playlist = append(list.Playlist, []string{path, human})
}

func (list *List) GetCurrentPath() string {
	return list.Playlist[list.Position][0]
}

func (list *List) GetCurrentHuman() string {
	return list.Playlist[list.Position][1]
}

func (list *List) GetList(max int) []string {
	var trackList []string
	for i := list.Position; i < list.Position+max; i++ {
		if list.Position+max > list.Size() {
			return trackList
		}
		trackList = append(trackList, list.Playlist[i][1])
	}

	return trackList
}

func (list *List) HasNext() bool {
	return len(list.Playlist) > list.Position+1
}

func (list *List) Next() string {
	if !list.HasNext() {
		return ""
	}
	list.Position++
	return list.GetCurrentPath()
}

func (list *List) Skip(amount int) string {
	println("Skipping", amount)
	if list.Size()+amount < 0 || !list.HasNext() {
		return ""
	}

	if list.Position+amount >= list.Size() {
		amount = 1 // only skip one track
	}
	list.Position += amount
	return list.GetCurrentPath()
}

func (list *List) Size() int {
	return len(list.Playlist)
}

func (list *List) IsEmpty() bool {
	return len(list.Playlist) == 0
}

func (list *List) QueueID(trackID int) (human string) {
	if trackID > search.MaxDBID || trackID < 1 {
		return ""
	}
	path, human := search.GetTrackById(trackID)
	if path == "" {
		return ""
	}
	list.pAdd(path, human)

	return human
}

func (list *List) QueueYT(url, human string) string {
	// TODO Check with API if video is valid for youtube links
	list.pAdd(url, human)
	return human
}

// Add id or url to player queue
// Return Human Name on success or "", nil on failure
func (list *List) AddToQueue(path string) (string, error) {
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
		list.QueueYT(path, human)
		return human, nil
	} else if strings.HasPrefix(path, "http") == true {
		return "", errors.New("URL Doesn't meet whitelist")
	}

	// FOR IDs
	idn, _ := strconv.Atoi(path)
	human := list.QueueID(idn)
	if human != "" {
		return human, nil
	}

	return "", errors.New("nothing added. (Invalid ID?)")
}
