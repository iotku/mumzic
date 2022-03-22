package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/iotku/mumzic/messages"

	"github.com/iotku/mumzic/config"
	"github.com/iotku/mumzic/helper"
	"github.com/iotku/mumzic/playback"
	"github.com/iotku/mumzic/search"
)

func CommandDispatch(player *playback.Player, message string, isPrivate bool, sender string) {
	helper.DebugPrintln("IsPlaying:", player.IsPlaying(), "DoNext:", player.DoNext)
	command, arg := getCommandAndArg(message, isPrivate, player.Config, player.GetClient().Self.Name)

	switch command {
	case "play", "add":
		play(arg, sender, isPrivate, player)
	case "stop":
		player.DoNext = "stop"
		player.Stop()
	case "skip", "next":
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
    case "retarget":
        player.TargetUsers()
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
		player.Config.Save()
    case "more":
        helper.MsgDispatch(player.GetClient(), isPrivate, sender, messages.GetMoreTable(sender))
    case "less":
        helper.MsgDispatch(player.GetClient(), isPrivate, sender, messages.GetLessTable(sender))
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

func getCommandAndArg(message string, isPrivate bool, config *config.Config, username string) (command string, arg string) {
	var offset int
    if !isPrivate && strings.HasPrefix(message, config.Prefix) {
		message = message[len(config.Prefix):]
	} else if strings.HasPrefix(message, username) {
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
	output := messages.MakeTable("Playlist", "# Track Name")
    playlist := player.Playlist.GetList(player.Playlist.Count())
    extraCount := messages.SaveMoreRows(sender, playlist, output)
    output.AddRow("---")
	output.AddRow(strconv.Itoa(player.Playlist.Count()) + " Track(s) queued.")
    if extraCount > 0 {
        output.AddRow("There are " + strconv.Itoa(extraCount) + " extra entries, use <b>more</b> and <b>less</b> to see them.")
    }
	helper.MsgDispatch(player.GetClient(), isPrivate, sender, output.String())
}

func find(player *playback.Player, sender string, isPrivate bool, arg string) {
	results := search.SearchALL(arg)
	output := messages.MakeTable("Search Results")
    extraCount := messages.SaveMoreRows(sender, results, output)
    if extraCount > 0 {
        output.AddRow("---")
        output.AddRow("There are " + strconv.Itoa(extraCount) + " additional results.")
        output.AddRow("Use <b>more</b> and <b>less</b> to see them.")
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

	output := messages.MakeTable("Randomly Added")
	idList := search.GetRandomTrackIDs(value)
	for _, v := range idList {
		human := player.Playlist.QueueID(v)
		if human != "" {
			output.AddRow("Added: <b>" + human + "</b>")
		} else {
			output.AddRow("Error: <b>" + "failed to add ID#" + strconv.Itoa(v) + "</b>")
		}
	}
	helper.MsgDispatch(player.GetClient(), isPrivate, sender, output.String())

	if !player.IsPlaying() && plistOrigSize == 0 {
		player.PlayCurrent()
	} else if !player.IsPlaying() && !hadNext {
		player.Skip(1)
	}
}
