package playback

import (
	"layeh.com/gumble/gumble/MumbleProto"
	"log"
	"strings"

	"github.com/iotku/mumzic/config"
	"github.com/iotku/mumzic/helper"
	"github.com/iotku/mumzic/playlist"
	"github.com/iotku/mumzic/youtubedl"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleffmpeg"
	_ "layeh.com/gumble/opus"
)

type Player struct {
	Stream   *gumbleffmpeg.Stream
	client   *gumble.Client
	Playlist playlist.List
	volume   float32
	DoNext   string
}

func TargetUser(client *gumble.Client, user string) {
	client.VoiceTarget = &gumble.VoiceTarget{ID: uint32(2)}
	client.VoiceTarget.AddUser(client.Users.Find(user))
	userTarget := client.Users.Find(user)
	packet := MumbleProto.VoiceTarget{
		Id:      &client.VoiceTarget.ID,
		Targets: make([]*MumbleProto.VoiceTarget_Target, 0, 1),
	}
	packet.Targets = append(packet.Targets, &MumbleProto.VoiceTarget_Target{
		Session: []uint32{userTarget.Session},
	})

	err := client.Conn.WriteProto(&packet)
	if err != nil {
		log.Println(err)
	}
}

func NewPlayer(client *gumble.Client) *Player {
	return &Player{
		Stream: nil,
		client: client,
		Playlist: playlist.List{
			Playlist: make([][]string, 0),
			Position: 0,
		},
		volume: config.VolumeLevel,
		DoNext: "stop", // stop, next
	}
}

func (player *Player) GetClient() *gumble.Client {
	return player.client
}

func (player *Player) IsStopped() bool {
	if player.Stream == nil || player.Stream.State() == gumbleffmpeg.StateStopped {
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
	player.Stream.Wait()
	switch player.DoNext {
	case "stop":
	//	client.Self.SetComment("Not Playing.")
	// Do nothing
	case "next":
		if player.Playlist.HasNext() {
			player.Playlist.Next()
			player.PlayCurrent()
		} else {
			player.DoNext = "stop"
		}
	default:
	}
}

func (player *Player) Play(path string) {
	// Stop if currently playing
	player.Stop()
	path = helper.StripHTMLTags(path)
	if strings.HasPrefix(path, "http") {
		player.PlayYT(path)
	} else {
		player.PlayFile(path)
	}

	helper.ChanMsg(player.client, "Now Playing: "+player.Playlist.GetCurrentHuman())
	player.client.Self.SetComment("Now Playing: " + player.Playlist.GetCurrentHuman())
	go player.WaitForStop()
}

func (player *Player) IsPlaying() bool {
	return player.Stream != nil && player.Stream.State() == gumbleffmpeg.StatePlaying
}

func (player *Player) Stop() {
	if player.IsPlaying() {
		player.Stream.Stop() //#nosec G104 -- Only error this will respond with is stream not playing.
	}
}

func (player *Player) PlayFile(path string) {
	player.Stream = gumbleffmpeg.New(player.client, gumbleffmpeg.SourceFile(path))
	player.Stream.Volume = config.VolumeLevel

	if err := player.Stream.Play(); err != nil {
		helper.DebugPrintln(err)
	} else {
		helper.DebugPrintln("Playing:", path)
	}
}

func (player *Player) Skip(amount int) {
	if player.Playlist.HasNext() {
		player.DoNext = "stop"
		player.Playlist.Skip(amount)
		player.PlayCurrent()
		player.DoNext = "next"
	} else {
		player.DoNext = "stop"
		player.Stop()
	}
}

// PlayYT streams a URL through ytdl
func (player *Player) PlayYT(url string) {
	url = helper.StripHTMLTags(url)
	if youtubedl.IsWhiteListedURL(url) == false {
		log.Printf("PlayYT Failed: URL %s Doesn't meet whitelist", url)
		return
	}

	player.Stream = gumbleffmpeg.New(player.client, youtubedl.GetYtdlSource(url))
	player.Stream.Volume = config.VolumeLevel

	if err := player.Stream.Play(); err != nil {
		helper.DebugPrintln(err)
	} else {
		helper.DebugPrintln("Playing:", url)
	}
}
