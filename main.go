package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: [flags]\n", os.Args[0])
		flag.PrintDefaults()
	}
	// Start main gumble loop
	gumbleutil.Main(gumbleutil.AutoBitrate, gumbleutil.Listener{
		Connect: func(e *gumble.ConnectEvent) {
			fmt.Printf("audio player loaded! (%d files)\n", search.MaxDBID)
			helper.BotUsername = e.Client.Self.Name
			helper.ServerHostname = "unknown" // TODO: How to get server hostname so we can do per server configurations
			config.LoadConfig(helper.ServerHostname)
			if config.LastChannel != "" && e.Client.Channels.Find(config.LastChannel) != nil {
				fmt.Println("Joining ", config.LastChannel)
				e.Client.Self.Move(e.Client.Channels.Find(config.LastChannel))
			} else {
				fmt.Println("Not Joining", config.LastChannel)
			}

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
				go commands.PlaybackControls(e.Client, e.Message, isPrivate, e.Sender.Name)
				go commands.SearchCommands(e.Client, e.Message, isPrivate, e.Sender.Name)
			}
		},
		ChannelChange: func(e *gumble.ChannelChangeEvent) {
			if e.Channel.Name != "Root" {
				config.LastChannel = e.Channel.Name
				fmt.Println("Last Channel Changed to", config.LastChannel)
			}
		},
		Disconnect: func(e *gumble.DisconnectEvent) {
			fmt.Println("Disconnecting: ", e.Type)
			config.SaveConfig()
		},
	})
}
