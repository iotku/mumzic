package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/iotku/mumzic/commands"
	"github.com/iotku/mumzic/config"
	"github.com/iotku/mumzic/helper"
	"github.com/iotku/mumzic/search"
	_ "github.com/mattn/go-sqlite3"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleutil"
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

			var isPrivate bool // Was message sent privately?

			if len(e.TextMessage.Channels) == 0 {
				// Direct Message
				log.Printf("DMSG (%s): %s", e.Sender.Name, e.Message)
				if e.Message == "Ping" {
					helper.UserMsg(e.Client, e.Sender.Name, "Pong")
				}
				isPrivate = true
			} else {
				// Channel Message
				log.Printf("CMSG (%s) %s: %s", e.Sender.Channel.Name, e.Sender.Name, e.Message)
				isPrivate = false
			}

			if strings.HasPrefix(e.Message, e.Client.Self.Name) || strings.HasPrefix(e.Message, config.CmdPrefix) {
				match := commands.PlaybackControls(e.Client, e.Message, isPrivate, e.Sender.Name)

				// probably a pointless optimization, but avoid checking for search command if playback Control was a match
				if match == true {
					return
				}

				commands.SearchCommands(e.Client, e.Message, isPrivate, e.Sender.Name)
			}
		},
	})
}
