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
	stream   *gumbleffmpeg.Stream
	client   *gumble.Client
	targets  []*gumble.User
	Playlist playlist.List
	Volume   float32
	DoNext   string
}

func (player *Player) AddTarget(username string) {
	user := player.client.Users.Find(username)
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
	player.targetUsers()
	println("Added: ", username)
}

func (player *Player) RemoveTarget(username string) {
	user := player.client.Users.Find(username)
	for i, v := range player.targets {
		if v.UserID == user.UserID {
			player.targets = append(player.targets[:i], player.targets[i+1:]...)
			println("Removed: ", username)
			player.targetUsers()
			return
		}
	}
}

func (player *Player) targetUsers() {
	player.client.VoiceTarget = &gumble.VoiceTarget{ID: uint32(2)}

	player.client.VoiceTarget.AddChannel(player.client.Self.Channel, false, false, "radio")
	ownChannel := player.client.Self.Channel
	packet := MumbleProto.VoiceTarget{
		Id:      &player.client.VoiceTarget.ID,
		Targets: make([]*MumbleProto.VoiceTarget_Target, 0, len(player.targets)+1),
	}
	for _, v := range player.targets {
		player.client.VoiceTarget.AddUser(v)
		packet.Targets = append(packet.Targets, &MumbleProto.VoiceTarget_Target{
			Session: []uint32{v.Session},
		})
	}

	packet.Targets = append(packet.Targets, &MumbleProto.VoiceTarget_Target{
		ChannelId: &ownChannel.ID,
	})

	err := player.client.Conn.WriteProto(&packet)
	if err != nil {
		log.Println(err)
	}
}

func NewPlayer(client *gumble.Client) *Player {
	return &Player{
		stream:  nil,
		client:  client,
		targets: make([]*gumble.User, 0),
		Playlist: playlist.List{
			Playlist: make([][]string, 0),
			Position: 0,
		},
		Volume: config.VolumeLevel,
		DoNext: "stop", // stop, next
	}
}

func (player *Player) GetClient() *gumble.Client {
	return player.client
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
	player.stream.Wait()
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
	player.DoNext = "next"

	helper.ChanMsg(player.client, "Now Playing: "+player.Playlist.GetCurrentHuman())
	player.client.Self.SetComment("Now Playing: " + player.Playlist.GetCurrentHuman())
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

func (player *Player) PlayFile(path string) {
	player.stream = gumbleffmpeg.New(player.client, gumbleffmpeg.SourceFile(path))
	player.stream.Volume = player.Volume

	if err := player.stream.Play(); err != nil {
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

	player.stream = gumbleffmpeg.New(player.client, youtubedl.GetYtdlSource(url))
	player.stream.Volume = player.Volume

	if err := player.stream.Play(); err != nil {
		helper.DebugPrintln(err)
	} else {
		helper.DebugPrintln("Playing:", url)
	}
}

func (player *Player) SetVolume(value float32) {
	player.Volume = value
	config.VolumeLevel = value // TODO: Keep track of independent volume levels
	if player.stream != nil {
		player.stream.Volume = value
	}
}
