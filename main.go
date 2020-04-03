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
	"os"
	"path/filepath"
	"strconv"
	"strings"
)
var stream *gumbleffmpeg.Stream
var volumeLevel float32

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

func playbackControls (client *gumble.Client, message string, songdb string, maxDBID int) {
	if isCommand(message, "!play ") {
		if stream != nil {
			stream.Stop()
		}
		value := lazyRemovePrefix(message, "play ")
		id, _ := strconv.Atoi(value)
		path, _ := GetTrackById(songdb, id, maxDBID)
		fmt.Println(path)
		stream = gumbleffmpeg.New(client, gumbleffmpeg.SourceFile(path))
		stream.Volume = volumeLevel
		if err := stream.Play(); err != nil {
			fmt.Printf("%s\n", err)
		} else {
			fmt.Printf("Playing %s\n", path)
		}
		return
	}

	// If stream object doesn't exist yet, don't do anything to avoid dereference
	if stream == nil {
		fmt.Println("Stream is null")
		return
	}

	// Start stream, resumes .Pause()
	if isCommand(message, "!play") {
		stream.Play()
		return
	}

	// Stop Playback
	if isCommand(message, "!stop") {
		err := stream.Stop()
		if err != nil {
			fmt.Println(err.Error())
		}
		return
	}

	// Pause playback, maybe resumed with .Play()
	if isCommand(message, "!pause")  {
		stream.Pause()
		return
	}

	// Set volume
	// At some point consider switching to percentage based system
	if isCommand(message, "!volume ") {
		message = "." + lazyRemovePrefix(message, "!volume")
		value, err := strconv.ParseFloat(message, 32)

		if err == nil {
			fmt.Printf("%f", value)
			volumeLevel = float32(value)
			stream.Volume = float32(value)
		}
	}

	// Skip to next track in playlist
	if isCommand(message, "!skip") {
		return
	}
}
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func isCommand (message, command string) bool {
	return strings.HasPrefix(message, command)
}

func lazyRemovePrefix(message, prefix string) string {
	char := "!"
	return strings.TrimSpace(message[len(char+prefix):len(message)])
}

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

func getMaxID(database string) int {
	db, err := sql.Open("sqlite3", database)
	defer db.Close()
	checkErr(err)
	var count int
	err = db.QueryRow("SELECT id FROM music WHERE ID = (SELECT MAX(ID) FROM music);").Scan(&count)
	checkErr(err)
	err = db.QueryRow("SELECT COUNT(*) FROM music").Scan(&count)
	return count
}

