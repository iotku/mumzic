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
	"strconv"
	"strings"
	"time"
)

var stream *gumbleffmpeg.Stream

// Eventually allow these to be grabbed from configuration file
var volumeLevel float32
var cmdPrefix = "!"

func main() {
	files := make(map[string]string)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: [flags] [audio files...]\n", os.Args[0])
		flag.PrintDefaults()
	}

	// Database generated from gendb
	var songdb = "./media.db"
	maxDBID := getMaxID(songdb)

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

func playFile(client *gumble.Client, path string) {
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

func playID(songdb string, client *gumble.Client, id, maxDBID int) string {
	path, human := GetTrackById(songdb, id, maxDBID)
	client.Self.Channel.Send("Now Playing: "+human, false)
	playFile(client, path)
	return human
}

func playbackControls(client *gumble.Client, message string, songdb string, maxDBID int) {
	if isCommand(message, cmdPrefix+"play ") {
		id, _ := strconv.Atoi(lazyRemovePrefix(message, "play "))
		playID(songdb, client, id, maxDBID)
		return
	}

	if isCommand(message, cmdPrefix+"rand") {
		seed := rand.NewSource(time.Now().UnixNano())
		randsrc := rand.New(seed)
		id := randsrc.Intn(maxDBID)
		playID(songdb, client, id, maxDBID)
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

	// Start stream, resumes .Pause()
	if isCommand(message, cmdPrefix+"play") {
		stream.Play()
		return
	}

	// Stop Playback
	if isCommand(message, cmdPrefix+"stop") {
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
	if isCommand(message, cmdPrefix+"volume ") {
		message = "." + lazyRemovePrefix(message, "!volume") // TODO: Check volume prefix removal (Probably not ! )
		value, err := strconv.ParseFloat(message, 32)

		if err == nil {
			fmt.Printf("%f", value)
			volumeLevel = float32(value)
			stream.Volume = float32(value)
		}
	}

	// Skip to next track in playlist
	if isCommand(message, cmdPrefix+"skip") {
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
func GetTrackById(songdb string, trackID, maxDBID int) (filepath, humanout string) {
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
		client.Self.Channel.Send(fmt.Sprintf("#%d | %s - %s (%s)\n", id, artist, title, album), false)
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
