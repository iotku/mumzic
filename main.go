package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iotku/mumzic/database"

	"github.com/iotku/mumzic/commands"
	"github.com/iotku/mumzic/config"
	"github.com/iotku/mumzic/playback"
	_ "github.com/mattn/go-sqlite3"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleutil"
)

func main() {
	var channelPlayer *playback.Player
	var bConfig *config.Config
	var hostname, username string

	cleanUp := func() {
		if bConfig != nil {
			bConfig.Channel = channelPlayer.Client.Self.Channel.Name
			bConfig.Save()
		}

		if channelPlayer != nil {
			channelPlayer.Playlist.Save(bConfig.Hostname)
		}
		config.CloseDatabase()
		database.Close()
	}

	// Capture shutdown signal (ctrl+c)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Received shutdown signal")
		cleanUp()
		os.Exit(0)
	}()

	gumbleutil.Main(gumbleutil.AutoBitrate, gumbleutil.Listener{
		Connect: func(e *gumble.ConnectEvent) {
			hostname = getValueFromFlag(flag.Lookup("server"))
			username = getValueFromFlag(flag.Lookup("username"))
			bConfig = config.NewConfig(hostname)
			if e.Client.Channels.Find(bConfig.Channel) != nil {
				e.Client.Self.Move(e.Client.Channels.Find(bConfig.Channel))
			}

			channelPlayer = playback.NewPlayer(e.Client, bConfig)
			channelPlayer.Playlist.Load(bConfig.Hostname)
			log.Printf("audio player loaded! (%d files)\n", database.GetMaxID())
		},
		TextMessage: func(e *gumble.TextMessageEvent) {
			if e.Sender == nil {
				return
			}

			isPrivate := len(e.TextMessage.Channels) == 0 // If no channels, is private message
			logMessage(e, isPrivate)

			if commands.IsCommand(e.Message, isPrivate, username, bConfig) {
				go commands.CommandDispatch(channelPlayer, e.Message, isPrivate, e.Sender.Name)
			}
		},
		ChannelChange: func(e *gumble.ChannelChangeEvent) {
			if bConfig != nil && !e.Channel.IsRoot() {
				bConfig.Channel = e.Channel.Name
			}
			if channelPlayer != nil {
				channelPlayer.TargetUsers()
			}
		},
		Disconnect: func(e *gumble.DisconnectEvent) {
			log.Println("Disconnecting: ", e.Type)
			cleanUp()
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

func getValueFromFlag(lookup *flag.Flag) string {
	if lookup == nil {
		panic("getValueFromFlag: flagNotFound: nil")
	}

	return lookup.Value.String()
}
