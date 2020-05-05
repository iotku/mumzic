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
	"log"
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

// This can probably be replaced by with good control flow and/or channels, might be subject to race conditions
var doNext = "stop" // stop, next, skip [int]
var isWaiting bool
var isPlaying bool
var skipBy = 1

// Eventually allow these to be grabbed from configuration file
var volumeLevel float32
var cmdPrefix = "!"
var maxLines = 10 // Max amount of lines you wish commands to output (before hopefully, going into an unimplemented more buffer)

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
		debugPrintln(err)
	} else {
		debugPrintln("Playing:", path)
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
		debugPrintln(err)
	} else {
		debugPrintln("Playing:", url)
	}

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
	isPlaying = true
	chanMsg(client, "Now Playing: "+metalist[currentsong])
	if strings.HasPrefix(path, "http") {
		playYT(path, client)
	} else {
		playFile(path, client)
	}

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
			isPlaying = false
		}
	case "skip":
		if currentsong+skipBy < 0 {
			break
		}
		if len(songlist) > (currentsong + skipBy) {
			currentsong = currentsong + skipBy
			play(songlist[currentsong], client)
			doNext = "next"
			skipBy = 1
		} else if len(songlist) > (currentsong + 1) {
			currentsong = currentsong + 1
			play(songlist[currentsong], client)
			doNext = "next"
			skipBy = 1
		}
	default:
		isWaiting = false
	}
	isWaiting = false
	return

}

func playOnly(client *gumble.Client) {
	// Skip Current track for frequent cases where you've just queued a new track and want to start
	if isPlaying == false && len(songlist) == currentsong+2 {
		currentsong = currentsong + 1
		play(songlist[currentsong], client)
		doNext = "next"
	} else if len(songlist) > 0 && isPlaying == false {
		// if stream and songlist exists
		play(songlist[currentsong], client)
		doNext = "next"
	} else if stream == nil {
		// Do nothing if nothing is queued
	}
}

func debugPrintln(inter ...interface{}) {
	log.Println(inter)
}

func playbackControls(client *gumble.Client, message string, songdb string, maxDBID int) {
	debugPrintln("isPlaying:", isPlaying, "doNext:", doNext)
	if isCommand(message, cmdPrefix+"play ") {
		id := lazyRemovePrefix(message, "play ")
		if id != "" && len(songlist) == 0 {
			// Add to queue then start playing queue
			queued := addToQueue(id, client)
			if queued == true {
				play(songlist[currentsong], client)
				doNext = "next"
			}
		} else if id == "" {
			playOnly(client)
		} else {
			addToQueue(id, client)
			doNext = "next"
			playOnly(client)
		}
		return
	}

	if isCommand(message, cmdPrefix+"play") {
		playOnly(client)
		return
	}

	if isCommand(message, cmdPrefix+"list") {
		current := currentsong
		amount := len(songlist) - current

		// Try poorly to avoid messages being dropped by mumble server for sending too fast
		var throttle bool
		if amount > maxLines {
			amount = maxLines
			throttle = true
		}

		for i := 0; i < amount; i++ {
			chanMsg(client, fmt.Sprintf("# %d: %s\n", i, metalist[current+i]))
			if throttle && i > 4 {
				time.Sleep(150 * time.Millisecond)
			}

		}

		chanMsg(client, fmt.Sprintf("%d Track(s) Queued.\n", len(songlist)-current))
		return
	}

	if isCommand(message, cmdPrefix+"rand") {
		howMany := lazyRemovePrefix(message, "rand")
		value, err := strconv.Atoi(howMany)
		if err != nil {
			return
		}
		seed := rand.NewSource(time.Now().UnixNano())
		randsrc := rand.New(seed)

		if value > maxLines {
			value = maxLines
		}
		for i := 0; i < value; i++ {
			id := randsrc.Intn(maxDBID)
			addToQueue(strconv.Itoa(id), client)
		}

		return
	}

	if isCommand(message, cmdPrefix+"search ") {
		SearchALL(songdb, lazyRemovePrefix(message, "search "), client)
		return
	}

	// If stream object doesn't exist yet, don't do anything to avoid dereference
	if stream == nil {
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
		howMany := lazyRemovePrefix(message, "skip")
		value, err := strconv.Atoi(howMany)
		if err != nil {
			log.Println(err)
			skipBy = 1
		} else {
			skipBy = value
		}
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
	message = strings.ToLower(message)
	isCommand := strings.HasPrefix(message, command)
	if isCommand {
		debugPrintln("Command: ", command, "Message:", message)
	}
	return isCommand
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

// Query SQLite database to count maximum amount of rows, as to not point to non existent ID
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
	rows, err := db.Query(query, args...)
	checkErr(err)

	// Don't forget to close in function where called.
	return rows
}
