package playback

import (
	"context"
	"errors"
	"image"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

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
	IsRadio  bool
	Config   *config.Config

	// Syncronization
	mu         sync.RWMutex
	stopCtx    context.Context
	stopCancel context.CancelFunc
	stopDone   chan struct{}
	isPlaying  bool
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
	ctx, cancel := context.WithCancel(context.Background())
	return &Player{
		stream:  nil,
		Client:  client,
		targets: make([]*gumble.User, 0),
		Playlist: playlist.List{
			Playlist: make([][]string, 0),
			Position: 0,
		},
		Volume:     config.Volume,
		IsRadio:    false,
		Config:     config,
		stopCtx:    ctx,
		stopCancel: cancel,
		stopDone:   make(chan struct{}),
		isPlaying:  false,
	}
}

// IsStopped returns true if the Stream exists and claims to be stopped
func (player *Player) IsStopped() bool {
	return player.stream == nil || player.stream.State() == gumbleffmpeg.StateStopped
}

// IsPlaying returns true if the Stream exists and claims to be playing
func (player *Player) IsPlaying() bool {
	player.mu.RLock()
	defer player.mu.RUnlock()
	return player.isPlaying && player.stream != nil && player.stream.State() == gumbleffmpeg.StatePlaying
}

// PlayCurrent plays the playlist at the current position should the player not already be playing.
func (player *Player) PlayCurrent() {
	if !player.Playlist.IsEmpty() && !player.IsPlaying() {
		player.Play(player.Playlist.GetCurrentPath())
	}
}

// WaitForStop waits for the playback stream to end and performs the upcoming action
func (player *Player) WaitForStop() {
	if player.IsStopped() {
		return
	}

	// Wait for either stream completion or stop signal
	done := make(chan struct{})
	go func() {
		defer close(done)
		player.stream.Wait()
	}()

	select {
	case <-done: // Stream finished
	case <-player.stopCtx.Done(): // Stop requested
		player.ensureStreamStopped()
		return
	}

	if !player.waitForActualStop(3 * time.Second) {
		log.Println("WARNING: Stream did not stop properly within timeout")
		return
	}

	player.mu.RLock()
	shouldContinue := player.isPlaying
	player.mu.RUnlock()

	if !shouldContinue {
		player.markStopped()
		return
	}

	if player.IsRadio {
		err := player.Playlist.AddNext(strconv.Itoa(search.GetRandomTrackIDs(1)[0]))
		if err != nil {
			helper.ChanMsg(player.Client, "<b style=\"color:red\">Error Adding Radio Track: </b>"+err.Error())
			log.Println("Radio failed to Playlist.AddNext a random track ID, stale database?: ", err)
		}
	}

	if player.Playlist.HasNext() {
		player.Playlist.Next()
		player.PlayCurrent()
	} else {
		player.requestStop()
	}
}

func (player *Player) Play(path string) {
	if player.IsPlaying() {
		player.requestStop()
		player.waitForActualStop(3 * time.Second)
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

	player.markPlaying()
	nowPlaying := player.NowPlaying()
	helper.ChanMsg(player.Client, nowPlaying)
	helper.SetComment(player.Client, nowPlaying)
	go player.WaitForStop()
}

func (player *Player) NowPlaying() string {
	currentPath := player.Playlist.GetCurrentPath()
	var artImg, output string

	var img image.Image
	var err error
	if strings.HasPrefix(currentPath, "http") { // ytdlp thumbnail
		img, err = youtubedl.GetYtDLThumbnail(currentPath)
	} else { // Local files
		coverArtPath := messages.FindCoverArtPath(currentPath)
		if coverArtPath != "" {
			img, err = messages.DecodeImage(coverArtPath)
		}
	}

	if img != nil {
		artImg = messages.GenerateCoverArtImg(img)
	}

	if err != nil {
		log.Println("Could not generate thumbnail: " + err.Error())
	}

	output = " <h2><u>Now Playing</u></h2><table><tr><td>" + artImg + "</td><td>" + "<table><tr><td>" +
		player.Playlist.GetCurrentHuman() + "</td></tr>"
	if player.IsRadio {
		output += "<tr><td><b>Radio</b> Mode: <b>Enabled</b></td></tr><tr>"
	} else {
		output += "<tr><td><b>" + strconv.Itoa(player.Playlist.Count()) + "</b> songs queued</td></tr>"
	}
	output += "</table>" + "</td></tr></table>"

	return output
}

func (player *Player) Stop(shouldStop bool) {
	if shouldStop {
		player.requestStop()
	} else {
		player.markStopped()
	}

	if player.stream != nil {
		player.ensureStreamStopped()
		helper.SetComment(player.Client, "Not Playing.")

		if !player.waitForActualStop(3 * time.Second) {
			log.Println("WARNING: Stream did not stop properly within timeout")
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
		player.Stop(true)
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

	player.stream = gumbleffmpeg.New(player.Client, youtubedl.GetYtDLSource(url))
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

// markPlaying marks the player as actively playing
func (player *Player) markPlaying() {
	player.mu.Lock()
	defer player.mu.Unlock()

	// Reset context for new playback session
	if player.stopCtx.Err() != nil {
		player.stopCtx, player.stopCancel = context.WithCancel(context.Background())
	}
	player.isPlaying = true
}

// requestStop signals that playback should stop
func (player *Player) requestStop() {
	player.mu.Lock()
	defer player.mu.Unlock()
	player.isPlaying = false
	player.stopCancel()
}

// markStopped marks the player as stopped without requesting a stop
func (player *Player) markStopped() {
	player.mu.Lock()
	defer player.mu.Unlock()
	player.isPlaying = false
}

// ensureStreamStopped aggressively stops the stream if it's running
func (player *Player) ensureStreamStopped() {
	if player.stream != nil && player.stream.State() == gumbleffmpeg.StatePlaying {
		player.stream.Stop() //#nosec G104 -- Only error this will respond with is stream not playing.
	}
}

// waitForActualStop waits for the stream to actually stop for the supplied timeout
func (player *Player) waitForActualStop(timeout time.Duration) bool {
	if player.stream == nil {
		return true
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if player.stream.State() == gumbleffmpeg.StateStopped {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}

	return player.stream.State() == gumbleffmpeg.StateStopped
}
