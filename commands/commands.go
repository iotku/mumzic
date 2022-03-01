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
)

func AddSongToQueue(id string, sender string, isPrivate bool, player *playback.Player) bool {
	queued, err := player.Playlist.AddToQueue(id)
	if err != nil {
		helper.MsgDispatch(player.GetClient(), isPrivate, sender, "Error: "+err.Error())
		return false
	}

	helper.MsgDispatch(player.GetClient(), isPrivate, sender, "Queued: "+queued)
	return true
}

func play(id string, sender string, isPrivate bool, player *playback.Player) {
	// This is still really scuffed
	if id != "" && player.IsStopped() { // Has argument
		if player.Playlist.IsEmpty() && AddSongToQueue(id, sender, isPrivate, player) {
			player.Play(player.Playlist.GetCurrentPath())
		} else if !player.Playlist.HasNext() && AddSongToQueue(id, sender, isPrivate, player) {
			player.Skip(1)
		} else if player.Playlist.HasNext() && AddSongToQueue(id, sender, isPrivate, player) {
			player.Play(player.Playlist.GetCurrentPath())
		}
	} else if id != "" && !player.IsStopped() && AddSongToQueue(id, sender, isPrivate, player) {
	} // Just add to queue if playing

	if !player.Playlist.IsEmpty() && player.IsStopped() { // Recover from stopped player.
		player.Play(player.Playlist.GetCurrentPath())
	}

	if player.IsPlaying() {
		player.DoNext = "next"
	}
}

func PlaybackControls(player *playback.Player, message string, isPrivate bool, sender string) bool {
	helper.DebugPrintln("IsPlaying:", player.IsPlaying(), "DoNext:", player.DoNext)

	if isCommand(message, "play") {
		id := helper.LazyRemovePrefix(message, "play")
		play(id, sender, isPrivate, player)
		return true
	}

	if isCommand(message, "list") {
		current := player.Playlist.Position
		amount := player.Playlist.Size() - current

		// TODO: Send to more buffer
		if amount > config.MaxLines {
			amount = config.MaxLines
		}

		for i, line := range player.Playlist.GetList(amount) {
			helper.MsgDispatch(player.GetClient(), isPrivate, sender, fmt.Sprintf("# %d: %s\n", i, line))
		}

		helper.MsgDispatch(player.GetClient(), isPrivate, sender, fmt.Sprintf("%d Track(s) Queued.\n", player.Playlist.Size()-current))
		return true
	}

	// If Stream object doesn't exist yet, don't do anything to avoid dereference
	if player.Stream == nil {
		return false
	}

	// Stop Playback
	if isCommand(message, "stop") {
		player.DoNext = "stop"
		err := player.Stream.Stop()
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
			player.Stream.Volume = float32(value)
		}
		return true
	}

	// Send current volume to channel
	if isCommand(message, "vol") {
		helper.MsgDispatch(player.GetClient(), isPrivate, sender, "Current Volume: "+fmt.Sprintf("%f", player.Stream.Volume))
		return true
	}

	// Skip to next track in playlist
	if isCommand(message, "skip") {
		howMany := helper.LazyRemovePrefix(message, "skip")
		value, err := strconv.Atoi(howMany)
		if err != nil {
			player.Skip(1)
		} else {
			player.Skip(value)
		}
		return true
	}

	return false
}

func SearchCommands(player *playback.Player, message string, isPrivate bool, sender string) bool {
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
		plistOrigSize := player.Playlist.Size()
		hadNext := player.Playlist.HasNext()
		for i := 0; i < value; i++ {
			id := randsrc.Intn(search.MaxDBID)
			trackName, err := player.Playlist.AddToQueue(strconv.Itoa(id))
			if err == nil {
				helper.MsgDispatch(player.GetClient(), isPrivate, sender, "Added: "+trackName)
			} else {
				helper.MsgDispatch(player.GetClient(), isPrivate, sender, "Error: "+err.Error())
			}
		}
		if !player.IsPlaying() && plistOrigSize == 0 {
			player.Play(player.Playlist.GetCurrentPath())
		} else if !player.IsPlaying() && !hadNext {
			player.Skip(1)
			player.Play(player.Playlist.GetCurrentPath())
		}

		return true
	}

	if isCommand(message, "search ") {
		results := search.SearchALL(helper.LazyRemovePrefix(message, "search "))
		for i, v := range results {
			helper.MsgDispatch(player.GetClient(), isPrivate, sender, v)
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
