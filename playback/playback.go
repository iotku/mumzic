package playback

import (
	"layeh.com/gopus"
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

var Players []*Player

func ChannelPlayer() *Player {
	return Players[0]
}

type Player struct {
	Stream   *gumbleffmpeg.Stream
	client   *gumble.Client
	Playlist playlist.List
	volume   float32
	DoNext   string
	SkipBy   int
}

const ID = 4

// encoder

type Encoder struct {
	*gopus.Encoder
}

func (*Encoder) ID() int {
	return ID
}
func (e *Encoder) Reset() {
	e.Encoder.ResetState()
}

func NewEncoder() gumble.AudioEncoder {
	e, _ := gopus.NewEncoder(gumble.AudioSampleRate, gumble.AudioChannels, gopus.Voip)
	e.SetBitrate(gopus.BitrateMaximum)
	return &Encoder{
		e,
	}
}

func NewPlayer(client *gumble.Client, user string) *Player {
	if user != "" {
		client = &gumble.Client{
			Self:           ChannelPlayer().client.Self,
			Config:         ChannelPlayer().client.Config,
			Conn:           ChannelPlayer().client.Conn,
			Users:          ChannelPlayer().client.Users,
			Channels:       ChannelPlayer().client.Channels,
			ContextActions: ChannelPlayer().client.ContextActions,
			AudioEncoder:   NewEncoder(),
			VoiceTarget:    nil,
		}
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

		println(client.Conn.WriteProto(&packet))
	}
	return &Player{
		Stream: nil,
		client: client,
		Playlist: playlist.List{
			Playlist: make([][]string, 1),
			Position: 0,
		},
		volume: config.VolumeLevel,
		DoNext: "stop", // stop, next, skip [int]
		SkipBy: 1,
	}
}

// Wait for playback stream to stop to perform next action
func (player *Player) WaitForStop() {
	player.Stream.Wait()
	switch player.DoNext {
	case "stop":
	//	client.Self.SetComment("Not Playing.")
	// Do nothing
	case "next":
		if player.Playlist.HasNext() {
			player.Play(player.Playlist.Next())
		} else {
			player.DoNext = "stop"
		}
	case "skip":
		if player.Playlist.HasNext() {
			player.Play(player.Playlist.Skip(player.SkipBy))
			player.DoNext = "next"
		} else {
			player.DoNext = "stop"
		}
	default:
		player.SkipBy = 1
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

	// TODO: Make this modular for multiple streams
	helper.ChanMsg(ChannelPlayer().client, "Now Playing: "+player.Playlist.GetCurrentHuman())
	ChannelPlayer().client.Self.SetComment("Now Playing: " + player.Playlist.GetCurrentHuman())
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
	//var vt = &gumble.VoiceTarget{}
	//vt.AddUser(client.Users.Find("iotku"))
	//
	//var newClient gumble.Client
	//newClient = gumble.Client{
	//	Self:           client.Self,
	//	Config:         client.Config,
	//	Conn:           client.Conn,
	//	Users:          client.Users,
	//	Channels:       client.Channels,
	//	ContextActions: client.ContextActions,
	//	AudioEncoder:   newClient.AudioEncoder,
	//	VoiceTarget:    vt,
	//}
	//
	//newClient.VoiceTarget = &gumble.VoiceTarget{ID: 2}
	//userTarget := client.Users.Find("iotku")
	//packet := MumbleProto.VoiceTarget{
	//	Id:      &newClient.VoiceTarget.ID,
	//	Targets: make([]*MumbleProto.VoiceTarget_Target, 0, 1),
	//}
	//packet.Targets = append(packet.Targets, &MumbleProto.VoiceTarget_Target{
	//	Session: []uint32{userTarget.Session},
	//})

	//println(client.Conn.WriteProto(&packet))
	player.Stream = gumbleffmpeg.New(player.client, gumbleffmpeg.SourceFile(path))
	player.Stream.Volume = config.VolumeLevel

	if err := player.Stream.Play(); err != nil {
		helper.DebugPrintln(err)
	} else {
		helper.DebugPrintln("Playing:", path)
	}

	//if err := Stream2.Play(); err != nil {
	//	helper.DebugPrintln(err)
	//} else {
	//	helper.DebugPrintln("Playing2:", path)
	//}
}

// Play youtube video
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
