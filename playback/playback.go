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
	stream      *gumbleffmpeg.Stream
	Client      *gumble.Client
	targets     []*gumble.User
	Playlist    playlist.List
	Volume      float32
	IsRadio     bool
	wantsToStop bool
	Config      *config.Config
}

func (player *Player) AddTarget(username string) {
	user := player.Client.Users.Find(username)
	if user == nil {
		return // user not found
	}
	for _, v := range player.targets {
		if v.UserID == user.UserID {
			player.RemoveTarget(v.Name)
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
		Volume:      config.Volume,
		wantsToStop: true,
		IsRadio:     false,
		Config:      config,
	}
}

// IsStopped returns true if the Stream exists and claims to be stopped
func (player *Player) IsStopped() bool {
	return player.stream == nil || player.stream.State() == gumbleffmpeg.StateStopped
}

// IsPlaying returns true if the Stream exists and claims to be playing
func (player *Player) IsPlaying() bool {
	return player.stream != nil && player.stream.State() == gumbleffmpeg.StatePlaying
}

func (player *Player) PlayCurrent() {
	if !player.Playlist.IsEmpty() {
		player.Play(player.Playlist.GetCurrentPath())
	}
}

// WaitForStop waits for the playback stream to end and performs the upcoming action
func (player *Player) WaitForStop() {
	if player.IsStopped() {
		return
	}
	player.stream.Wait()

	if player.IsRadio {
		player.Playlist.AddNext(strconv.Itoa(search.GetRandomTrackIDs(1)[0]))
	}

	if player.wantsToStop {
		player.Stop(true) // May Double Stop but this is fine?
		return
	}

	if player.Playlist.HasNext() {
		player.Playlist.Next()
		player.PlayCurrent()
	} else {
		player.Stop(true)
	}
}

func (player *Player) Play(path string) {
	if player.IsPlaying() {
		player.Stop(false)
	}

	path = helper.StripHTMLTags(path)
	var err error
	if strings.HasPrefix(path, "http") {
		err = player.PlayYT(path)
	} else {
		err = player.PlayFile(path)
	}

	if err != nil {
		helper.ChanMsg(player.Client, "<b style=\"color:red\">Error: </b>"+err.Error())
		return
	}

	player.wantsToStop = false
	nowPlaying := player.NowPlaying()
	helper.ChanMsg(player.Client, nowPlaying)
	helper.SetComment(player.Client, nowPlaying)
	player.WaitForStop()
}

func (player *Player) NowPlaying() string {
	artPath := messages.FindCoverArtPath(player.Playlist.GetCurrentPath())
	var artImg, radioMode string
	if artPath != "" {
		artImg = messages.GenerateCoverArtImg(artPath)
	}

	if player.IsRadio {
		radioMode = "<tr><td>Radio Mode: <b>Enabled</b></td></tr><tr>" +
			"<td>Use <b>radio</b> to go back to normal mode</td><tr>"
	}
	output := " <h2><u>Now Playing</u></h2><table><tr><td>" +
		artImg + "</td><td>" +
		"<table><tr><td>" +
		player.Playlist.GetCurrentHuman() +
		"</td></tr>" +
		"<tr><td><b>" + strconv.Itoa(player.Playlist.Count()) +
		"</b> songs queued</td></tr>" +
		radioMode +
		"</table>" +
		"</td></tr></table>"

	return output
}

func (player *Player) Stop(wantsToStop bool) {
	player.wantsToStop = wantsToStop
	if player.IsPlaying() {
		player.stream.Stop() //#nosec G104 -- Only error this will respond with is stream not playing.
		helper.SetComment(player.Client, "Not Playing.")
		player.stream.Wait() // This may help alleviate issues as descried below
		if player.IsPlaying() && player.wantsToStop {
			// There have been some occasions where another stream begins and turns into a garbled
			// mess. Hopefully at some point we'll catch it and determine if it's our fault.
			log.Println("WARNING: Racey stop condition, should have stopped but didn't?.")
		}
	}
}

func (player *Player) PlayFile(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		log.Println("[Error] file not found:", path)
		return errors.New("not found")
	}

	player.stream = gumbleffmpeg.New(player.Client, gumbleffmpeg.SourceFile(path))
	player.stream.Volume = player.Volume
	err := player.stream.Play()
	return err
}

func (player *Player) Skip(amount int) {
	if player.Playlist.HasNext() && !player.IsRadio {
		player.Stop(false)
		player.Playlist.Skip(amount)
		player.PlayCurrent()
	} else if player.IsRadio {
		player.Stop(false)
	} else {
		player.Stop(true)
	}
}

// PlayYT streams a URL through ytdl
func (player *Player) PlayYT(url string) error {
	url = helper.StripHTMLTags(url)
	if !youtubedl.IsWhiteListedURL(url) {
		return errors.New("URL Doesn't Meet whitelist")
	}

	player.stream = gumbleffmpeg.New(player.Client, youtubedl.GetYtdlSource(url))
	player.stream.Volume = player.Volume
	err := player.stream.Play()
	return err
}

func (player *Player) SetVolume(value float32) {
	player.Volume = value
	player.Config.Volume = value
	if player.stream != nil {
		player.stream.Volume = value
	}
}
