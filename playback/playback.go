package playback

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/iotku/mumzic/config"
	"github.com/iotku/mumzic/helper"
	"github.com/iotku/mumzic/messages"
	"github.com/iotku/mumzic/playlist"
	"github.com/iotku/mumzic/search"
	"github.com/iotku/mumzic/youtubedl"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumble/MumbleProto"
	"layeh.com/gumble/gumbleffmpeg"
	_ "layeh.com/gumble/opus"
)

type Player struct {
	stream   *gumbleffmpeg.Stream
	Client   *gumble.Client
	targets  []*gumble.User
	Playlist playlist.List
	Volume   float32
	DoNext   string
	Config   *config.Config
}

func (player *Player) AddTarget(username string) {
	user := player.Client.Users.Find(username)
	if user == nil {
		return // user not found
	}
	for _, v := range player.targets {
		if v.UserID == user.UserID {
			player.RemoveTarget(v.Name)
			println("Retargeted: ", username)
			break
		}
	}
	player.targets = append(player.targets, user)
	player.TargetUsers()
}

func (player *Player) RemoveTarget(username string) {
	user := player.Client.Users.Find(username)
	for i, v := range player.targets {
		if v.UserID == user.UserID {
			player.targets = append(player.targets[:i], player.targets[i+1:]...)
			break
		}
	}
	player.TargetUsers()
}

func (player *Player) TargetUsers() {
	if len(player.targets) == 0 {
		player.Client.VoiceTarget = nil
		return
	}
	println("Target usrs and chan")
	player.Client.VoiceTarget = &gumble.VoiceTarget{ID: uint32(2)}
	ownChannel := player.Client.Self.Channel
	player.Client.VoiceTarget.AddChannel(ownChannel, false, false, "radio")
	packet := MumbleProto.VoiceTarget{
		Id:      &player.Client.VoiceTarget.ID,
		Targets: make([]*MumbleProto.VoiceTarget_Target, 0, len(player.targets)+1),
	}
	for _, v := range player.targets {
		player.Client.VoiceTarget.AddUser(v)
		packet.Targets = append(packet.Targets, &MumbleProto.VoiceTarget_Target{
			Session: []uint32{v.Session},
		})
	}

	packet.Targets = append(packet.Targets, &MumbleProto.VoiceTarget_Target{
		ChannelId: &ownChannel.ID,
	})

	err := player.Client.Conn.WriteProto(&packet)
	if err != nil {
		log.Println(err)
	}
}

func NewPlayer(client *gumble.Client, config *config.Config) *Player {
	return &Player{
		stream:  nil,
		Client:  client,
		targets: make([]*gumble.User, 0),
		Playlist: playlist.List{
			Playlist: make([][]string, 0),
			Position: 0,
		},
		Volume: config.Volume,
		DoNext: "stop", // stop, next
		Config: config,
	}
}

func (player *Player) IsStopped() bool {
	if player.stream == nil || player.stream.State() == gumbleffmpeg.StateStopped {
		return true
	}
	return false
}

func (player *Player) PlayCurrent() {
	if !player.Playlist.IsEmpty() {
		player.Play(player.Playlist.GetCurrentPath())
	}
}

// WaitForStop waits for the playback stream to end and performs the upcoming action
func (player *Player) WaitForStop() {
	if player.stream == nil {
		return
	}
	player.stream.Wait()
	switch player.DoNext {
	case "stop":
		player.Client.Self.SetComment("Not Playing.")
	case "next":
		if player.Playlist.HasNext() {
			player.Playlist.Next()
			player.PlayCurrent()
		} else {
			player.DoNext = "stop"
		}
	case "radio":
		ids := search.GetRandomTrackIDs(1)
		if player.Playlist.AddNext(strconv.Itoa(ids[0])) {
			player.Playlist.Position++
			player.PlayCurrent()
		}
	default:
	}
}

func (player *Player) Play(path string) {
	player.Stop()
	path = helper.StripHTMLTags(path)
	var err error
	if strings.HasPrefix(path, "http") {
		player.PlayYT(path)
	} else {
		err = player.PlayFile(path)
	}

	if err != nil {
		helper.ChanMsg(player.Client, "<b style=\"color:red\">Error: </b>"+err.Error())
		return
	}
	if player.DoNext != "radio" {
		player.DoNext = "next"
	}

	artPath := messages.FindCoverArtPath(player.Playlist.GetCurrentPath())
	var artImg string
	if artPath != "" {
		artImg = messages.GenerateCoverArtImg(artPath)
	}
	output := " <h2><u>Now Playing</u></h2><table><tr><td>" +
		artImg + "</td><td>" +
		"<table><tr><td>" +
		player.Playlist.GetCurrentHuman() +
		"</td></tr>" +
		"<tr><td><b>" + strconv.Itoa(player.Playlist.Count()) +
		"</b> songs remain</td></tr></table>" +
		"</td></tr></table>"
	helper.ChanMsg(player.Client, output)
	player.Client.Self.SetComment(output)
	go player.WaitForStop()
}

// IsPlaying returns true if the Stream exists and claims to be playing
func (player *Player) IsPlaying() bool {
	return player.stream != nil && player.stream.State() == gumbleffmpeg.StatePlaying
}

func (player *Player) Stop() {
	if player.IsPlaying() {
		player.stream.Stop() //#nosec G104 -- Only error this will respond with is stream not playing.
	}
}

func (player *Player) PlayFile(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		log.Println("[Error] file not found:", path)
		return errors.New("not found")
	}
	player.stream = gumbleffmpeg.New(player.Client, gumbleffmpeg.SourceFile(path))
	player.stream.Volume = player.Volume

	if err := player.stream.Play(); err != nil {
		helper.DebugPrintln(err)
	} else {
		helper.DebugPrintln("Playing:", path)
	}
	return nil
}

func (player *Player) Skip(amount int) {
	if player.Playlist.HasNext() && player.DoNext != "radio" {
		player.DoNext = "stop"
		player.Playlist.Skip(amount)
		player.PlayCurrent()
		player.DoNext = "next"
	} else if player.DoNext == "radio" {
		player.Stop()
	} else {
		player.DoNext = "stop"
		player.Stop()
	}
}

// PlayYT streams a URL through ytdl
func (player *Player) PlayYT(url string) {
	url = helper.StripHTMLTags(url)
	if !youtubedl.IsWhiteListedURL(url) {
		log.Printf("PlayYT Failed: URL %s Doesn't meet whitelist", url)
		return
	}

	player.stream = gumbleffmpeg.New(player.Client, youtubedl.GetYtdlSource(url))
	player.stream.Volume = player.Volume

	if err := player.stream.Play(); err != nil {
		helper.DebugPrintln(err)
	} else {
		helper.DebugPrintln("Playing:", url)
	}
}

func (player *Player) SetVolume(value float32) {
	player.Volume = value
	player.Config.Volume = value
	if player.stream != nil {
		player.stream.Volume = value
	}
}
