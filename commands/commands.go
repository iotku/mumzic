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

type messageTable struct {
    table *strings.Builder
    cols int
}

func makeTable(header string, columns ...string) messageTable {
    var b strings.Builder
    fmt.Fprintf(&b, "<h2 style=\"margin-top:16px; margin-bottom:2px; margin-left:0px; margin-right:0px; -qt-block-indent:0; text-indent:0px;\"><b><u><span style=\"font-size:x-large\">%s</span></u></b></h2>", header)
    fmt.Fprintf(&b, "<table align=\"left\" border=\"0\" style=\"margin-top:0px; margin-bottom:0px; margin-left:0px; margin-right:0px;\" cellspacing=\"2\" cellpadding=\"0\"><thead>")
    if len(columns) != 0 {
        fmt.Fprintf(&b, "<tr>")
    }
    for _, v := range columns {
        fmt.Fprintf(&b, "<th align=\"left\">%s</th>", v)
    }
    if len(columns) != 0 {
        fmt.Fprintf(&b, "</tr>")
    }
    fmt.Fprintf(&b, "</thead><tbody>")
    return messageTable{&b, len(columns)}
}

func (msgTbl messageTable) addRow(cells ...string) {
    fmt.Fprintf(msgTbl.table, "<tr>")
    for _,v := range cells {
        fmt.Fprintf(msgTbl.table, "<td><p>%s</p></td>", v)
    }
    fmt.Fprintf(msgTbl.table, "</tr>")
}

func (msgTbl messageTable) String() string{
    fmt.Fprintf(msgTbl.table, "</tbody></table>")
    println("Got here")
    return msgTbl.table.String()
}

func PlaybackControls(player *playback.Player, message string, isPrivate bool, sender string) bool {
	helper.DebugPrintln("IsPlaying:", player.IsPlaying(), "DoNext:", player.DoNext)

    if isCommand(message, "target") {
        player.AddTarget(sender)
    }

    if isCommand(message, "untarget") {
        player.RemoveTarget(sender)
    }

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

        output := makeTable("Playlist", "#", "Track Name")
		for i, line := range player.Playlist.GetList(amount) {
			output.addRow(strconv.Itoa(i), line)
		}
        output.addRow("---")
        output.addRow(strconv.Itoa(player.Playlist.Size()-current), " Track(s) queued.")
        helper.MsgDispatch(player.GetClient(), isPrivate, sender, output.String())

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
		randsrc := rand.New(seed) //#nosec G404 -- Cryptographic randomness is not required

		if value > config.MaxLines {
			value = config.MaxLines
		}
		plistOrigSize := player.Playlist.Size()
		hadNext := player.Playlist.HasNext()

        output := makeTable("Randomly Added")
		for i := 0; i < value; i++ {
			id := randsrc.Intn(search.MaxDBID)
			trackName, err := player.Playlist.AddToQueue(strconv.Itoa(id))
			if err == nil {
                output.addRow("Added: <b>" + trackName + "</b>")
			} else {
                output.addRow("Error: <b>" + err.Error() + "</b>")
			}
		}
		helper.MsgDispatch(player.GetClient(), isPrivate, sender, output.String())

		if !player.IsPlaying() && plistOrigSize == 0 {
			player.PlayCurrent()
		} else if !player.IsPlaying() && !hadNext {
			player.Skip(1)
		}

		return true
	}

	if isCommand(message, "search ") {
		results := search.SearchALL(helper.LazyRemovePrefix(message, "search "))
        output := makeTable("Search Results")
		for i, v := range results {
            output.addRow(v)
			if i == config.MaxLines { // TODO, Send extra results into 'more' buffer
				break
			}
		}
		helper.MsgDispatch(player.GetClient(), isPrivate, sender, output.String())
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
