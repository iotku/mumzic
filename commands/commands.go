package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/iotku/mumzic/config"
	"github.com/iotku/mumzic/helper"
	"github.com/iotku/mumzic/messages"
	"github.com/iotku/mumzic/playback"
	"github.com/iotku/mumzic/search"
)

func IsCommand(message string, isPrivate bool, config *config.Config) bool {
	return strings.HasPrefix(message, config.Prefix) || strings.HasPrefix(message, config.Username) || isPrivate
}

func CommandDispatch(player *playback.Player, msg string, isPrivate bool, sender string) {
	helper.DebugPrintln("IsPlaying:", player.IsPlaying(), "DoNext:", player.DoNext, "Len:", len(player.Playlist.Playlist), "Count", player.Playlist.Count(), "PlPos:", player.Playlist.Position, "DoNext:", player.DoNext, "HasNext:", player.Playlist.HasNext())
	command, arg := getCommandAndArg(msg, player.Client.Self.Name, isPrivate, player.Config)

	switch command {
	case "play", "add":
		play(arg, sender, isPrivate, player)
	case "playnow":
		if player.Playlist.AddNext(arg) {
			player.Skip(1)
		} else {
			helper.MsgDispatch(player.Client, isPrivate, sender, "Not Added: invalid ID or URL") // TODO Standardize error messages
		}
	case "playnext", "addnext":
		if player.Playlist.AddNext(arg) {
			helper.MsgDispatch(player.Client, isPrivate, sender, "Added: "+player.Playlist.GetNextHuman())
		} else {
			helper.MsgDispatch(player.Client, isPrivate, sender, "Not Added: invalid ID or URL") // TODO Standardize error messages
		}
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
		helper.MsgDispatch(
			player.Client,
			isPrivate,
			sender,
			"https://github.com/iotku/mumzic/blob/master/USAGE.md",
		)
	case "rand", "random":
		rand(player, sender, isPrivate, arg)
	case "radio":
		if player.DoNext != "radio" {
			helper.MsgDispatch(player.Client, isPrivate, sender, "Enabled Radio Mode, Shuffling forever.")
			player.Playlist.AddNext(strconv.Itoa(search.GetRandomTrackIDs(1)[0]))
			player.DoNext = "radio"
			player.Skip(1)
		} else {
			player.DoNext = "next"
			helper.MsgDispatch(player.Client, isPrivate, sender, "Disabled Radio Mode.")
		}
	case "search", "find":
		find(player, sender, isPrivate, arg)
	case "saveconf":
		player.Config.Save()
	case "more":
		helper.MsgDispatch(player.Client, isPrivate, sender, messages.GetMoreTable(sender))
	case "less":
		helper.MsgDispatch(player.Client, isPrivate, sender, messages.GetLessTable(sender))
	case "summon":
		joinUserChannel(player, sender)
	case "uinfo":
		info := player.Client.Self.Hash
		println(info)
		helper.MsgDispatch(player.Client, isPrivate, sender, info)
	}
}

func joinUserChannel(player *playback.Player, sender string) {
	client := player.Client
	user := client.Users.Find(sender)
	if user == nil {
		return // user not found
	}
	channel := client.Channels.Find(user.Channel.Name)

	client.Self.Move(channel)
	if client.Self.Channel.Name == channel.Name {
		player.TargetUsers()
	}
}

func addSongToQueue(id, sender string, isPrivate bool, player *playback.Player) bool {
	queued, err := player.Playlist.AddToQueue(id)
	if err != nil {
		helper.MsgDispatch(player.Client, isPrivate, sender, "Error: "+err.Error())
		return false
	}

	helper.MsgDispatch(player.Client, isPrivate, sender, "Queued: "+queued)
	return true
}

func getCommandAndArg(msg, name string, isPrivate bool, conf *config.Config) (command, arg string) {
	if strings.HasPrefix(msg, conf.Prefix) {
		msg = msg[len(conf.Prefix):]
	} else if strings.HasPrefix(msg, name) {
		msg = strings.TrimSpace(msg[len(name):])
	}

	split := strings.Split(msg, " ")
	for i := 1; i < len(split); i++ {
		arg += split[i] + " "
	}
	if strings.HasPrefix(msg, name) && len(split) == 1 {
		return "", ""
	}
	return strings.ToLower(split[0]), strings.TrimSpace(arg)
}

func play(id string, sender string, isPrivate bool, player *playback.Player) {
	if id != "" && player.IsStopped() { // Has argument
		if player.Playlist.IsEmpty() && addSongToQueue(id, sender, isPrivate, player) {
			player.PlayCurrent()
		} else if !player.Playlist.HasNext() && addSongToQueue(id, sender, isPrivate, player) {
			player.Skip(1)
		} else if player.Playlist.HasNext() && addSongToQueue(id, sender, isPrivate, player) {
			player.PlayCurrent()
		}
	} else if id != "" && !player.IsStopped() && addSongToQueue(id, sender, isPrivate, player) {
	} // Just add to queue if playing

	if !player.Playlist.IsEmpty() && player.IsStopped() { // Recover from stopped player.
		player.PlayCurrent()
	}
}

func vol(player *playback.Player, sender string, isPrivate bool, arg string) {
	if arg != "" {
		argInt, err := strconv.Atoi(arg)
		if err != nil || argInt <= 0 || argInt > 100 {
			helper.MsgDispatch(player.Client, isPrivate, sender, "Invalid Volume: Valid range <b>[1-100]</b>")
			return
		}
		player.SetVolume(0.01 * float32(argInt))
	}
	helper.MsgDispatch(player.Client, isPrivate, sender, "Current Volume: "+fmt.Sprintf("%f", player.Volume))
}

func list(player *playback.Player, sender string, isPrivate bool) {
	playlist := player.Playlist.GetList(player.Playlist.Count())

	output := messages.MakeTable("Playlist", "# Track Name")
	messages.SaveMoreRows(sender, playlist, output)
	output.AddRow("---")
	output.AddRow(strconv.Itoa(player.Playlist.Count()) + " Track(s) queued.")

	helper.MsgDispatch(player.Client, isPrivate, sender, output.String())
}

func find(player *playback.Player, sender string, isPrivate bool, arg string) {
	results := search.SearchALL(arg)

	output := messages.MakeTable("Search Results")
	messages.SaveMoreRows(sender, results, output)

	helper.MsgDispatch(player.Client, isPrivate, sender, output.String())
}

func rand(player *playback.Player, sender string, isPrivate bool, arg string) {
	value, err := strconv.Atoi(arg)
	if err != nil || value < 1 {
		value = 1
	} else if value > config.MaxLines {
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
	helper.MsgDispatch(player.Client, isPrivate, sender, output.String())

	if !player.IsPlaying() && plistOrigSize == 0 {
		player.PlayCurrent()
	} else if !player.IsPlaying() && !hadNext {
		player.Skip(1)
	}
}
