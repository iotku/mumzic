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
	"github.com/iotku/mumzic/search"
	"layeh.com/gumble/gumble"
)

func playOnly(client *gumble.Client) {
	// Skip Current track for frequent cases where you've just queued a new track and want to start
	if !playback.ChannelPlayer().IsPlaying() && playback.ChannelPlayer().Playlist.Size() == playback.ChannelPlayer().Playlist.Position+2 {
		playback.ChannelPlayer().Play(playback.ChannelPlayer().Playlist.Next())
		playback.ChannelPlayer().DoNext = "next"
	} else if playback.ChannelPlayer().Playlist.Size() > 0 && !playback.ChannelPlayer().IsPlaying() {
		// if Stream and Songlist exists
		playback.ChannelPlayer().Play(playback.ChannelPlayer().Playlist.GetCurrentPath())
		playback.ChannelPlayer().DoNext = "next"
	} else if playback.ChannelPlayer().Stream == nil {
		// Do nothing if nothing is queued
	}
}

func PlaybackControls(client *gumble.Client, message string, isPrivate bool, sender string) bool {
	helper.DebugPrintln("IsPlaying:", playback.ChannelPlayer().IsPlaying(), "DoNext:", playback.ChannelPlayer().DoNext)

	if isCommand(message, "play ") {
		id := helper.LazyRemovePrefix(message, "play ")
		if id != "" && playback.ChannelPlayer().Playlist.Size() == 0 {
			// Add to queue then start playing queue
			queued := playback.ChannelPlayer().Playlist.AddToQueue(isPrivate, sender, id)
			if queued == true {
				playback.ChannelPlayer().Play(playback.ChannelPlayer().Playlist.GetCurrentPath())
				playback.ChannelPlayer().DoNext = "next"
			}
		} else if id == "" {
			playOnly(client)
		} else {
			playback.ChannelPlayer().Playlist.AddToQueue(isPrivate, sender, id)
			playback.ChannelPlayer().DoNext = "next"
			playOnly(client)
		}
		return true
	}

	if isCommand(message, "play") {
		playOnly(client)
		return true
	}

	if isCommand(message, "list") {
		current := playback.ChannelPlayer().Playlist.Position
		amount := playback.ChannelPlayer().Playlist.Size() - current

		// TODO: Send to more buffer
		if amount > config.MaxLines {
			amount = config.MaxLines
		}

		for i, line := range playback.ChannelPlayer().Playlist.GetList(amount) {
			helper.MsgDispatch(isPrivate, sender, client, fmt.Sprintf("# %d: %s\n", i, line))
		}

		helper.MsgDispatch(isPrivate, sender, client, fmt.Sprintf("%d Track(s) Queued.\n", playback.ChannelPlayer().Playlist.Size()-current))
		return true
	}

	if isCommand(message, "target") {
		if client.VoiceTarget == nil {
			println("target nil")
		} else {
			println(client.VoiceTarget.ID)
		}
		//client.VoiceTarget = &gumble.VoiceTarget{ID: 2}
		//userTarget := client.Users.Find(sender)
		//packet := MumbleProto.VoiceTarget{
		//	Id:      &client.VoiceTarget.ID,
		//	Targets: make([]*MumbleProto.VoiceTarget_Target, 0, 1),
		//}
		//packet.Targets = append(packet.Targets, &MumbleProto.VoiceTarget_Target{
		//	Session: []uint32{userTarget.Session},
		//})
		//
		//println(client.Conn.WriteProto(&packet))
		player := playback.NewPlayer(nil, sender)
		player.PlayFile(playback.ChannelPlayer().Playlist.GetCurrentPath())
	}

	if isCommand(message, "untarget") {
		if client.VoiceTarget != nil {
			client.VoiceTarget.Clear()
			client.VoiceTarget = nil
		}
	}

	// If Stream object doesn't exist yet, don't do anything to avoid dereference
	if playback.ChannelPlayer().Stream == nil {
		return false
	}

	// Stop Playback
	if isCommand(message, "stop") {
		playback.ChannelPlayer().DoNext = "stop"
		err := playback.ChannelPlayer().Stream.Stop()
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
			playback.ChannelPlayer().Stream.Volume = float32(value)
		}
		return true
	}

	// Send current volume to channel
	if isCommand(message, "vol") {
		helper.MsgDispatch(isPrivate, sender, client, "Current Volume: "+fmt.Sprintf("%f", playback.ChannelPlayer().Stream.Volume))
		return true
	}

	// Skip to next track in playlist
	if isCommand(message, "skip") {
		howMany := helper.LazyRemovePrefix(message, "skip")
		value, err := strconv.Atoi(howMany)
		if err != nil {
			// If this isn't a proper value Atoi returns and error, probably harmless but maybe not smart.
			//log.Println(err)
			playback.ChannelPlayer().SkipBy = 1
		} else {
			playback.ChannelPlayer().SkipBy = value
		}
		playback.ChannelPlayer().DoNext = "skip"
		err = playback.ChannelPlayer().Stream.Stop()
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
		//#nosec G404 -- Cryptographic randomness is not required
		randsrc := rand.New(seed)

		if value > config.MaxLines {
			value = config.MaxLines
		}
		plistOrigSize := playback.ChannelPlayer().Playlist.Size()
		hadNext := playback.ChannelPlayer().Playlist.HasNext()
		for i := 0; i < value; i++ {
			id := randsrc.Intn(search.MaxDBID)
			playback.ChannelPlayer().Playlist.AddToQueue(isPrivate, sender, strconv.Itoa(id))
		}
		if !playback.ChannelPlayer().IsPlaying() && plistOrigSize == 0 {
			playback.ChannelPlayer().Play(playback.ChannelPlayer().Playlist.GetCurrentPath())
		} else if !playback.ChannelPlayer().IsPlaying() && !hadNext {
			playback.ChannelPlayer().Playlist.Skip(1)
			playback.ChannelPlayer().Play(playback.ChannelPlayer().Playlist.GetCurrentPath())
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
		return true
	}

	return false
}

func isCommand(message, command string) bool {
	return strings.HasPrefix(strings.ToLower(message), config.CmdPrefix+command) ||
		strings.HasPrefix(strings.ToLower(command), helper.BotUsername+" "+command)
}
