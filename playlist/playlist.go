package playlist

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/iotku/mumzic/helper"
	"github.com/iotku/mumzic/search"
	"github.com/iotku/mumzic/youtubedl"
)

const Directory = "playlists/" // Directory for saving/loading playlists

// List contains a 2D slice of "Human Friendly" titles and raw paths as well as its position along the playlist
type List struct {
	Playlist [][]string
	Position int
}

func (list *List) Save(hostname string) {
	var saveList [][]string
	for i := list.Position; i < len(list.Playlist); i++ {
		saveList = append(saveList, list.Playlist[i])
	}

	if saveList == nil {
		return
	}

	output, err := json.Marshal(saveList)
	if err != nil {
		log.Fatalln("failed to marshal playlist json")
	}

	if _, err := os.Stat(Directory); os.IsNotExist(err) {
		if err = os.Mkdir(Directory, 0700); err != nil {
			log.Fatalln("failed to create"+Directory+"directory:", err)
		}
	}
	if fileInfo, _ := os.Stat(Directory); !fileInfo.IsDir() {
		log.Fatalln("Playlist path, ", Directory, "is not a directory.")
	}

	if err := os.WriteFile(Directory+hostname, output, 0600); err != nil {
		log.Fatalln("Failed to write playlist file:", err.Error())
	}
}

func (list *List) Load(hostname string) {
	if _, err := os.Open(Directory + hostname); //#nosec G304 - hostname considered rusted source
	err == nil {
		file, err := os.ReadFile(Directory + hostname) //#nosec G304 - hostname considered rusted source
		if err != nil {
			log.Fatalln(err.Error())
		}
		var pList [][]string
		err = json.Unmarshal(file, &pList)
		if err != nil {
			log.Fatalln("json Unmarshal failed", err.Error())
		}
		log.Println("Applying plist")
		list.Playlist = pList
	}
}

// GetCurrentPath gets the raw path for the current item in the playlist
func (list *List) GetCurrentPath() string {
	return list.Playlist[list.Position][0]
}

// GetCurrentHuman gets the "Human Friendly" title for the current item in the playlist
func (list *List) GetCurrentHuman() string {
	return list.Playlist[list.Position][1]
}

// GetList returns a list of items from the current to the end of the playlist
// Note: Older items aren't removed immediately however aren't guaranteed to remain forever.
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

// HasNext returns true if there is another item remaining in the playlist
func (list *List) HasNext() bool {
	return len(list.Playlist) > list.Position+1
}

// Next shifts the playlist position forward by one if there is at least one more item in the playlist remaining
func (list *List) Next() string {
	if !list.HasNext() {
		return ""
	}
	list.Position++
	return list.GetCurrentPath()
}

// Skip moves the position by amount, generally this should be called by a playback.Player
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

// Size returns an int of how many items the playlist contains
func (list *List) Size() int {
	return len(list.Playlist)
}

// IsEmpty returns whether the playlist contains any elements.
func (list *List) IsEmpty() bool {
	return len(list.Playlist) == 0
}

// AddToQueue ads either a filesystem ID or internet URL onto the Playlist queue. On success, it returns a human friendly
// title and err is nil. On failure (ID not found or not whitelisted URL) returns empty string "" and a respective error.
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
		list.queueYT(path, human)
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

func (list *List) pAdd(path, human string) {
	list.Playlist = append(list.Playlist, []string{path, human})
}

func (list *List) QueueID(trackID int) (human string) {
	if trackID > search.MaxID || trackID < 1 {
		return ""
	}
	path, human := search.GetTrackById(trackID)
	if path == "" {
		return ""
	}
	list.pAdd(path, human)

	return human
}

func (list *List) queueYT(url, human string) string {
	// TODO Check with API if video is valid for youtube links
	list.pAdd(url, human)
	return human
}

// Count is the amount of songs enqueued on the playlist
func (list *List) Count() int {
	return list.Size() - list.Position
}
