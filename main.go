package main

import (
	"flag"
	"fmt"
	"github.com/iotku/mumzic/commands"
	"github.com/iotku/mumzic/search"
	_ "github.com/mattn/go-sqlite3"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleutil"
	"os"
	"path/filepath"
)

func main() {
	files := make(map[string]string)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: [flags]\n", os.Args[0])
		flag.PrintDefaults()
	}
	// Start main gumble loop
	gumbleutil.Main(gumbleutil.AutoBitrate, gumbleutil.Listener{
		Connect: func(e *gumble.ConnectEvent) {
			for _, file := range flag.Args() {
				key := filepath.Base(file)
				files[key] = file
			}
			fmt.Printf("audio player loaded! (%d files)\n", search.MaxDBID)
		},
		TextMessage: func(e *gumble.TextMessageEvent) {
			if e.Sender == nil {
				fmt.Println("Sender Was Null")
				return
			}

			fmt.Println(e.Message)
			match := commands.PlaybackControls(e.Client, e.Message)

			// probably a pointless optimization, but avoid checking for search command if playback Control was a match
			if match == true {
				return
			}

			commands.SearchCommands(e.Client, e.Message)
		},
	})
}
