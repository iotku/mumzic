package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/iotku/mumzic/config"
	"github.com/iotku/mumzic/helper"
	"github.com/iotku/mumzic/playback"
	"github.com/iotku/mumzic/search"
)

func CommandDispatch(player *playback.Player, message string, isPrivate bool, sender string) {
	helper.DebugPrintln("IsPlaying:", player.IsPlaying(), "DoNext:", player.DoNext)
	command, arg := getCommandAndArg(message, isPrivate)

	switch command {
	case "play":
		play(arg, sender, isPrivate, player)
	case "stop":
		player.DoNext = "stop"
		player.Stop()
	case "skip":
		value, err := strconv.Atoi(arg)
		if err != nil {
			player.Skip(1)
		} else {
			player.Skip(value)
		}
	case "vol", "volume":
		vol(player, sender, isPrivate, arg)
	case "list":
		list(player, sender, isPrivate)
	case "target":
		player.AddTarget(sender)
	case "untarget":
		player.RemoveTarget(sender)
	case "help":
		helper.MsgDispatch(player.GetClient(), isPrivate, sender, "https://github.com/iotku/mumzic/blob/master/USAGE.md")
	case "rand":
		rand(player, sender, isPrivate, arg)
	case "search", "find":
		find(player, sender, isPrivate, arg)
	case "saveconf":
		config.SaveConfig()
	}
}

func AddSongToQueue(id string, sender string, isPrivate bool, player *playback.Player) bool {
	queued, err := player.Playlist.AddToQueue(id)
	if err != nil {
		helper.MsgDispatch(player.GetClient(), isPrivate, sender, "Error: "+err.Error())
		return false
	}

	helper.MsgDispatch(player.GetClient(), isPrivate, sender, "Queued: "+queued)
	return true
}

func getCommandAndArg(message string, isPrivate bool) (command string, arg string) {
	var offset int
	if !isPrivate && strings.HasPrefix(message, config.CmdPrefix) {
		message = message[len(config.CmdPrefix):]
	} else if strings.HasPrefix(message, helper.BotUsername) {
		offset = 1
	}
	split := strings.Split(message, " ")
	for i := offset + 1; i < len(split); i++ {
		arg += split[i] + " "
	}
	arg = strings.TrimSpace(arg)

	return strings.ToLower(split[offset]), arg
}

func play(id string, sender string, isPrivate bool, player *playback.Player) {
	if id != "" && player.IsStopped() { // Has argument
		if player.Playlist.IsEmpty() && AddSongToQueue(id, sender, isPrivate, player) {
			player.PlayCurrent()
		} else if !player.Playlist.HasNext() && AddSongToQueue(id, sender, isPrivate, player) {
			player.Skip(1)
		} else if player.Playlist.HasNext() && AddSongToQueue(id, sender, isPrivate, player) {
			player.PlayCurrent()
		}
	} else if id != "" && !player.IsStopped() && AddSongToQueue(id, sender, isPrivate, player) {
	} // Just add to queue if playing

	if !player.Playlist.IsEmpty() && player.IsStopped() { // Recover from stopped player.
		player.PlayCurrent()
	}
}

func vol(player *playback.Player, sender string, isPrivate bool, arg string) {
	// TODO: At some point consider switching to percentage based system
	if arg != "" {
		value, err := strconv.ParseFloat("."+arg, 32)
		if err == nil {
			player.SetVolume(float32(value))
		}
	} else {
		helper.MsgDispatch(player.GetClient(), isPrivate, sender, "Current Volume: "+fmt.Sprintf("%f", player.Volume))
	}
}

func list(player *playback.Player, sender string, isPrivate bool) {
	current := player.Playlist.Position
	amount := player.Playlist.Size() - current

	// TODO: Send to more buffer
	if amount > config.MaxLines {
		amount = config.MaxLines
	}

	output := makeTable("Playlist", "#", "Track Name")
	for i, line := range player.Playlist.GetList(amount) {
		output.addRow(strconv.Itoa(i), line)
	}
	output.addRow("---")
	output.addRow(strconv.Itoa(player.Playlist.Size()-current), " Track(s) queued.")
	helper.MsgDispatch(player.GetClient(), isPrivate, sender, output.String())
}

func find(player *playback.Player, sender string, isPrivate bool, arg string) {
	results := search.SearchALL(arg)
	output := makeTable("Search Results")
	for i, v := range results {
		output.addRow(v)
		if i == config.MaxLines { // TODO, Send extra results into 'more' buffer
			break
		}
	}
	helper.MsgDispatch(player.GetClient(), isPrivate, sender, output.String())
}

func rand(player *playback.Player, sender string, isPrivate bool, arg string) {
	value, err := strconv.Atoi(arg)
	if err != nil || value < 1 {
		value = 1
	}

	if value > config.MaxLines {
		value = config.MaxLines
	}
	plistOrigSize := player.Playlist.Size()
	hadNext := player.Playlist.HasNext()

	output := makeTable("Randomly Added")
	idList := search.GetRandomTrackIDs(value)
	for _, v := range idList {
		human := player.Playlist.QueueID(v)
		if human != "" {
			output.addRow("Added: <b>" + human + "</b>")
		} else {
			output.addRow("Error: <b>" + "failed to add ID#" + strconv.Itoa(v) + "</b>")
		}
	}
	helper.MsgDispatch(player.GetClient(), isPrivate, sender, output.String())

	if !player.IsPlaying() && plistOrigSize == 0 {
		player.PlayCurrent()
	} else if !player.IsPlaying() && !hadNext {
		player.Skip(1)
	}
}
