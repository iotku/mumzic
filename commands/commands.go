package commands

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/iotku/mumzic/config"
	"github.com/iotku/mumzic/helper"
	"github.com/iotku/mumzic/playback"
	"github.com/iotku/mumzic/playlist"
	"github.com/iotku/mumzic/search"
	"layeh.com/gumble/gumble"
)

func playOnly(client *gumble.Client) {
	// Skip Current track for frequent cases where you've just queued a new track and want to start
	if playback.IsPlaying == false && playlist.Size() == playlist.Position+2 {
		playback.Play(playlist.Next(), client)
		playback.DoNext = "next"
	} else if playlist.Size() > 0 && playback.IsPlaying == false {
		// if Stream and Songlist exists
		playback.Play(playlist.GetCurrentPath(), client)
		playback.DoNext = "next"
	} else if playback.Stream == nil {
		// Do nothing if nothing is queued
	}
}

func PlaybackControls(client *gumble.Client, message string, isPrivate bool, sender string) bool {
	helper.DebugPrintln("IsPlaying:", playback.IsPlaying, "IsWaiting:", playback.IsWaiting, "DoNext:", playback.DoNext)
	
    if isCommand(message, "play ") {
		id := helper.LazyRemovePrefix(message, "play ")
		if id != "" && playlist.Size() == 0 {
			// Add to queue then start playing queue
			queued := playlist.AddToQueue(isPrivate, sender, id, client)
			if queued == true {
				playback.Play(playlist.GetCurrentPath(), client)
				playback.DoNext = "next"
			}
		} else if id == "" {
			playOnly(client)
		} else {
			playlist.AddToQueue(isPrivate, sender, id, client)
			playback.DoNext = "next"
			playOnly(client)
		}
		return true
	}

	if isCommand(message, "play") {
		playOnly(client)
		return true
	}

	if isCommand(message, "list") {
		current := playlist.Position
		amount := playlist.Size() - current

		// Try poorly to avoid messages being dropped by mumble server for sending too fast
		if amount > config.MaxLines {
			amount = config.MaxLines
		}

		for i, line := range playlist.GetList(amount) {
			helper.MsgDispatch(isPrivate, sender, client, fmt.Sprintf("# %d: %s\n", i, line))
		}

		helper.MsgDispatch(isPrivate, sender, client, fmt.Sprintf("%d Track(s) Queued.\n", playlist.Size()-current))
		return true
	}

	// If Stream object doesn't exist yet, don't do anything to avoid dereference
	if playback.Stream == nil {
		return false
	}

	// Stop Playback
	if isCommand(message, "stop") {
		playback.DoNext = "stop"
		err := playback.Stream.Stop()
		if err != nil {
			fmt.Println(err.Error())
		}
		return true
	}

	// Set volume
	// TODO: At some point consider switching to percentage based system
	if isCommand(message, "vol ") {
		message = "." + helper.LazyRemovePrefix(message, "vol")
		value, err := strconv.ParseFloat(message, 32)

		if err == nil {
			fmt.Println("Current Volume: ", value)
			config.VolumeLevel = float32(value)
			playback.Stream.Volume = float32(value)
		}
	}

	// Send current volume to channel
	if isCommand(message, "vol") {
		helper.MsgDispatch(isPrivate, sender, client, "Current Volume: "+fmt.Sprintf("%f", playback.Stream.Volume))
		return true
	}

	// Skip to next track in playlist
	if isCommand(message, "skip") {
		howMany := helper.LazyRemovePrefix(message, "skip")
		value, err := strconv.Atoi(howMany)
		if err != nil {
			// If this isn't a proper value Atoi returns and error, probably harmless but maybe not smart.
			//log.Println(err)
			playback.SkipBy = 1
		} else {
			playback.SkipBy = value
		}
		playback.DoNext = "skip"
		err = playback.Stream.Stop()
		helper.DebugPrintln(err)
		return true
	}

	return false
}

func SearchCommands(client *gumble.Client, message string, isPrivate bool, sender string) bool {
	if search.MaxDBID == 0 {
		return true
	} // Don't perform any database related commands if the database doesn't exist (or contains no rows)
	if isCommand(message, "rand") {
		howMany := helper.LazyRemovePrefix(message, "rand")
		value, err := strconv.Atoi(howMany)
		if err != nil {
			return true
		}
		seed := rand.NewSource(time.Now().UnixNano())
        //#nosec G404 G102 -- Cryptographic randomness is not required
		randsrc := rand.New(seed)

		if value > config.MaxLines {
			value = config.MaxLines
		}
		for i := 0; i < value; i++ {
			id := randsrc.Intn(search.MaxDBID)
			playlist.AddToQueue(isPrivate, sender, strconv.Itoa(id), client)
		}

		return true
	}

	if isCommand(message, "search ") {
		results := search.SearchALL(helper.LazyRemovePrefix(message, "search "))
		for i, v := range results {
			helper.MsgDispatch(isPrivate, sender, client, v)
			if i == config.MaxLines { // TODO, Send extra results into 'more' buffer
				break
			}
		}
		return true
	}

	if isCommand(message, "saveconf") {
		config.SaveConfig()
	}

	return false
}

func isCommand(message, command string) bool {
	message = strings.ToLower(message)
	command = strings.ToLower(command)
	if strings.HasPrefix(message, config.CmdPrefix+command) || strings.HasPrefix(message, helper.BotUsername+" "+command) {
		helper.DebugPrintln("Command: ", command, "Message:", message)
		return true
	} else {
		//helper.DebugPrintln("Not Command:", command, "Is actually", message, "Where cmd prefix is", config.CmdPrefix)
		return false
	}
}
