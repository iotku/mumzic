package main

import (
	"flag"
	"fmt"
	"github.com/iotku/mumzic/playback"
	"log"
	"os"
	"strings"

	"github.com/iotku/mumzic/commands"
	"github.com/iotku/mumzic/config"
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

	var channelPlayer *playback.Player
    var bConfig *config.Config
    // Start main gumble loop
	gumbleutil.Main(gumbleutil.AutoBitrate, gumbleutil.Listener{
		Connect: func(e *gumble.ConnectEvent) {
            if hostName := flag.Lookup("server").Value; hostName == nil {
                bConfig = config.NewConfig("unknown")
            } else {
                bConfig = config.NewConfig(hostName.String())
            }

			if bConfig.Channel != "" && e.Client.Channels.Find(bConfig.Channel) != nil {
				fmt.Println("Joining ", bConfig.Channel)
				e.Client.Self.Move(e.Client.Channels.Find(bConfig.Channel))
			} else {
				fmt.Println("Not Joining", bConfig.Channel)
			}

			channelPlayer = playback.NewPlayer(e.Client, bConfig)
			fmt.Printf("audio player loaded! (%d files)\n", search.MaxDBID)
		},
		TextMessage: func(e *gumble.TextMessageEvent) {
			if e.Sender == nil {
				fmt.Println("Sender Was Null")
				return
			}

			isPrivate := len(e.TextMessage.Channels) == 0 // If no channels, is private message
			logMessage(e, isPrivate)

			if isCommand(e, isPrivate, bConfig) {
				go commands.CommandDispatch(channelPlayer, e.Message, isPrivate, e.Sender.Name)
			}
		},
		ChannelChange: func(e *gumble.ChannelChangeEvent) {
			if e.Channel.Name != "Root" {
				bConfig.Channel = e.Channel.Name
				fmt.Println("Last Channel Changed to", bConfig.Channel)
			}
            if channelPlayer != nil {
                channelPlayer.TargetUsers()
            }
		},
		Disconnect: func(e *gumble.DisconnectEvent) {
			fmt.Println("Disconnecting: ", e.Type)
			bConfig.Save()
            config.CloseDatabase()
		},
	})
}

func logMessage(e *gumble.TextMessageEvent, isPrivate bool) {
	if isPrivate {
		log.Printf("DMSG (%s): %s", e.Sender.Name, e.Message)
	} else {
		log.Printf("CMSG (%s) %s: %s", e.Sender.Channel.Name, e.Sender.Name, e.Message)
	}
}

func isCommand(e *gumble.TextMessageEvent, isPrivate bool, config *config.Config) bool {
	return strings.HasPrefix(e.Message, config.Prefix) || strings.HasPrefix(e.Message, e.Client.Self.Name) || isPrivate
}
