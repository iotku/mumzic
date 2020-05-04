package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleffmpeg"
	"layeh.com/gumble/gumbleutil"
	_ "layeh.com/gumble/opus"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var stream *gumbleffmpeg.Stream
var songlist = make([]string, 0)
var metalist = make([]string, 0)
var currentsong = 0
var doNext = "stop" // stop, next, skip [int]
var isWaiting bool
var isPlaying bool

// Eventually allow these to be grabbed from configuration file
var volumeLevel float32
var cmdPrefix = "!"

// Database generated from gendb
var songdb = "./media.db"
var maxDBID = getMaxID(songdb)

func main() {
	files := make(map[string]string)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: [flags] [audio files...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	volumeLevel = 0.25 // Default volume level

	gumbleutil.Main(gumbleutil.AutoBitrate, gumbleutil.Listener{
		Connect: func(e *gumble.ConnectEvent) {
			for _, file := range flag.Args() {
				key := filepath.Base(file)
				files[key] = file
			}

			fmt.Printf("audio player loaded! (%d files)\n", maxDBID)
		},
		TextMessage: func(e *gumble.TextMessageEvent) {
			if e.Sender == nil {
				fmt.Println("Sender Was Null")
				return
			}

			fmt.Println(e.Message)

			playbackControls(e.Client, e.Message, songdb, maxDBID)
		},
	})
}

func playFile(path string, client *gumble.Client) {
	if stream != nil {
		stream.Stop()
	}

	stream = gumbleffmpeg.New(client, gumbleffmpeg.SourceFile(path))
	stream.Volume = volumeLevel
	if err := stream.Play(); err != nil {
		fmt.Printf("%s\n", err)
	} else {
		fmt.Printf("Playing %s\n", path)
	}
}

func isWhiteListedUrl(url string) bool {
	whiteListedURLS := []string{"https://www.youtube.com/", "https://youtu.be/", "https://soundcloud.com/"}
	for i := range whiteListedURLS {
		if strings.HasPrefix(url, whiteListedURLS[i]) {
			return true
		}
	}

	return false
}

// Play youtube video
func playYT(url string, client *gumble.Client) {
	removeHtmlTags := regexp.MustCompile("<[^>]*>")
	url = removeHtmlTags.ReplaceAllString(url, "")
	if isWhiteListedUrl(url) == false {
		chanMsg(client, "URL Doesn't meet whitelist, sorry.")
		return
	}

	if stream != nil {
		stream.Stop()
	}

	stream = gumbleffmpeg.New(client, gumbleffmpeg.SourceExec("youtube-dl", "-f", "bestaudio", "--rm-cache-dir", "-q", "-o", "-", url))
	stream.Volume = volumeLevel

	if err := stream.Play(); err != nil {
		fmt.Printf("%s\n", err)
	} else {
		fmt.Printf("Playing %s\n", url)
	}

}
func playID(id int, client *gumble.Client) string {
	path, human := GetTrackById(songdb, id)
	chanMsg(client, "Now Playing: "+human)
	playFile(path, client)
	return human
}

// Sends Message to current mumble channel
func chanMsg(client *gumble.Client, msg string) { client.Self.Channel.Send(msg, false) }

func queueYT(url, human string) string {
	// TODO Check with API if video is valid for youtube links
	songlist = append(songlist, url)
	metalist = append(metalist, human)
	return human
}

func addToQueue(path string, client *gumble.Client) bool {
	// For YTDL URLS
	removeHtmlTags := regexp.MustCompile("<[^>]*>")
	path = removeHtmlTags.ReplaceAllString(path, "")
	if strings.HasPrefix(path, "http") && isWhiteListedUrl(path) == true {
		// add to playlist
		queueYT(path, path)
		chanMsg(client, "Added: "+path)
		return true
	} else if strings.HasPrefix(path, "http") == true {
		chanMsg(client, "URL Doesn't meet whitelist, sorry.")
		// Don't do anything
		return false
	}

	// FOR IDs
	idn, _ := strconv.Atoi(path)
	human := queueID(idn)
	if human != "" {
		chanMsg(client, "Added: "+human)
		return true
	}

	chanMsg(client, "Nothing added.")
	return false
}

func queueID(trackID int) (human string) {
	if trackID > maxDBID || trackID < 1 {
		return ""
	}
	path, human := GetTrackById(songdb, trackID)
	if path == "" {
		return ""
	}
	songlist = append(songlist, path)
	metalist = append(metalist, human)

	return human
}

func play(path string, client *gumble.Client) {
	if stream != nil {
		if stream.State() == gumbleffmpeg.StatePlaying {
			stream.Stop()
		}
	}

	removeHtmlTags := regexp.MustCompile("<[^>]*>")
	path = removeHtmlTags.ReplaceAllString(path, "")
	chanMsg(client, "Now Playing: "+metalist[currentsong])
	if strings.HasPrefix(path, "http") {
		playYT(path, client)
	} else {
		playFile(path, client)
	}
	isPlaying = true

	go waitForStop(client)

}

// Probably horrific logic
func waitForStop(client *gumble.Client) {
	// wait for playback to stop
	if isWaiting == true {
		return
	}
	isWaiting = true
	stream.Wait()
	switch doNext {
	case "stop":
		isPlaying = false
		// Do nothing
	case "next":
		if len(songlist) > currentsong+1 {
			currentsong++
			play(songlist[currentsong], client)
		} else {
			doNext = "stop"
		}
	case "skip":
		if len(songlist) > (currentsong + 1) {
			currentsong = currentsong + 1
			play(songlist[currentsong], client)
			doNext = "next"
		}
	default:
		isWaiting = false
	}

	isWaiting = false
	return

}

func playbackControls(client *gumble.Client, message string, songdb string, maxDBID int) {
	if isCommand(message, cmdPrefix+"play") {
		id := lazyRemovePrefix(message, "play")
		if stream != nil && len(songlist) > 0 && id == "" {
			// if stream and songlist exists
			stream.Play()
			doNext = "next"
		} else if id == "" && stream == nil {
			// Do nothing if nothing is queued
		} else if id != "" && len(songlist) == 0 {
			// Add to queue then start playing queue
			queued := addToQueue(id, client)
			if queued == true {
				play(songlist[currentsong], client)
				doNext = "next"
			}
		} else {
			addToQueue(id, client)
			doNext = "next"
		}
		return
	}

	if isCommand(message, cmdPrefix+"list") {
		current := currentsong
		amount := len(songlist) - current

		var output string
		for i := 0; i < amount; i++ {
			output += fmt.Sprintf("# %d: %s\n", i, metalist[current+i])
		}

		trkqueued := fmt.Sprintf("%d Track(s) Queued.\n", len(songlist)-current)

		chanMsg(client, trkqueued+output)
	}

	if isCommand(message, cmdPrefix+"rand") {
		seed := rand.NewSource(time.Now().UnixNano())
		randsrc := rand.New(seed)
		id := randsrc.Intn(maxDBID)
		addToQueue(strconv.Itoa(id), client)
		if isPlaying == false {
			play(songlist[currentsong], client)
		}
		return
	}

	if isCommand(message, cmdPrefix+"search ") {
		SearchALL(songdb, lazyRemovePrefix(message, "search "), client)
		return
	}

	// If stream object doesn't exist yet, don't do anything to avoid dereference
	if stream == nil {
		fmt.Println("Stream is null")
		return
	}

	// Stop Playback
	if isCommand(message, cmdPrefix+"stop") {
		doNext = "stop"
		err := stream.Stop()
		if err != nil {
			fmt.Println(err.Error())
		}
		return
	}

	// Pause playback, maybe resumed with .Play()
	if isCommand(message, cmdPrefix+"pause") {
		stream.Pause()
		return
	}

	// Set volume
	// At some point consider switching to percentage based system
	if isCommand(message, cmdPrefix+"vol ") {
		message = "." + lazyRemovePrefix(message, "vol")
		value, err := strconv.ParseFloat(message, 32)

		if err == nil {
			fmt.Printf("%f", value)
			volumeLevel = float32(value)
			stream.Volume = float32(value)
		}
	}

	// Send current volume to channel
	if isCommand(message, cmdPrefix+"vol") {
		chanMsg(client, "Current Volume: "+fmt.Sprintf("%f", stream.Volume))
		return
	}

	// Skip to next track in playlist
	if isCommand(message, cmdPrefix+"skip") {
		doNext = "skip"
		stream.Stop()
		return
	}
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func isCommand(message, command string) bool {
	return strings.HasPrefix(message, command)
}

// Remove prefix from command for single argument (I.E. "!play 22" -> "22")
func lazyRemovePrefix(message, prefix string) string {
	char := cmdPrefix
	return strings.TrimSpace(message[len(char+prefix):len(message)])
}

// Query SQLite database to get filepath related to ID
func GetTrackById(songdb string, trackID int) (filepath, humanout string) {
	if trackID > maxDBID {
		return "", ""
	}
	db, err := sql.Open("sqlite3", songdb)
	checkErr(err)
	defer db.Close()
	var path, artist, title, album string
	err = db.QueryRow("select path,artist,title,album from MUSIC where id = ?", trackID).Scan(&path, &artist, &title, &album)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", ""
		}
	}
	checkErr(err)

	humanout = artist + " - " + title
	return path, humanout
}

// Query SQLite database to count maximum amount of rows, as to not point to non existant ID
// TODO: Perhaps catch error instead?
func getMaxID(database string) int {
	db, err := sql.Open("sqlite3", database)
	defer db.Close()
	checkErr(err)
	var count int
	err = db.QueryRow("SELECT id FROM music WHERE ID = (SELECT MAX(ID) FROM music);").Scan(&count)
	checkErr(err)
	return count
}

func SearchALL(songdb, Query string, client *gumble.Client) {
	Query = fmt.Sprintf("%%%s%%", Query)
	rows := makeDbQuery(songdb, "SELECT * FROM music where (artist || \" \" || title)  LIKE ? LIMIT 25", Query)
	defer rows.Close()

	var id int
	var artist, album, title, path string

	for rows.Next() {
		err := rows.Scan(&id, &artist, &album, &title, &path)
		checkErr(err)
		chanMsg(client, fmt.Sprintf("#%d | %s - %s (%s)\n", id, artist, title, album))
	}

	return
}

// Helper Functions
func makeDbQuery(songdb, query string, args ...interface{}) *sql.Rows {
	db, err := sql.Open("sqlite3", songdb)
	checkErr(err)
	defer db.Close()
	fmt.Println(query)
	fmt.Println(args)
	rows, err := db.Query(query, args...)
	checkErr(err)

	// Don't forget to close in function where called.
	return rows
}
