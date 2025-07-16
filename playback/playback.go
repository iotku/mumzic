package playback

import (
	"errors"
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
	stream      *gumbleffmpeg.Stream
	Client      *gumble.Client
	targets     []*gumble.User
	Playlist    playlist.List
	Volume      float32
	IsRadio     bool
	wantsToStop bool
	Config      *config.Config

	// Synchronization fields
	playbackMu      sync.Mutex
	isTransitioning bool
	waitingForStop  bool // Prevents multiple WaitForStop goroutines
	skipRequested   bool // Indicates a skip operation is in progress
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

// PlayCurrent plays the playlist at the current position should the player not already be playing.
func (player *Player) PlayCurrent() {
	if !player.Playlist.IsEmpty() && !player.IsPlaying() {
		player.Play(player.Playlist.GetCurrentPath())
	}
}

// WaitForStop waits for the playback stream to end and performs the upcoming action
func (player *Player) WaitForStop() {
	// Prevent multiple WaitForStop goroutines from running concurrently
	player.playbackMu.Lock()
	if player.waitingForStop {
		log.Printf("WaitForStop: Already waiting for stop, skipping duplicate call")
		player.playbackMu.Unlock()
		return
	}
	player.waitingForStop = true
	player.playbackMu.Unlock()
	
	// Ensure we clear the flags when done
	defer func() {
		player.playbackMu.Lock()
		player.waitingForStop = false
		// Always clear skip request flag when WaitForStop completes
		if player.skipRequested {
			player.skipRequested = false
			log.Printf("WaitForStop: Clearing skip request flag on exit")
		}
		player.playbackMu.Unlock()
	}()

	// Check if a skip was requested before we even start waiting
	player.playbackMu.Lock()
	if player.skipRequested {
		log.Printf("WaitForStop: Skip was requested, skipping wait entirely")
		player.playbackMu.Unlock()
		return
	}
	player.playbackMu.Unlock()

	if player.IsStopped() {
		return
	}
	
	log.Printf("WaitForStop: Starting to wait for stream completion")
	player.stream.Wait()
	log.Printf("WaitForStop: Stream completed, processing next action")

	// Check if a skip was requested while we were waiting
	player.playbackMu.Lock()
	skipWasRequested := player.skipRequested
	if skipWasRequested {
		log.Printf("WaitForStop: Skip was requested during wait, skipping auto-advance")
		player.playbackMu.Unlock()
		return
	}
	player.playbackMu.Unlock()

	if player.wantsToStop {
		log.Printf("WaitForStop: Player wants to stop, stopping playback")
		player.Stop(true) // May Double Stop but this is fine?
		return
	}

	if player.IsRadio {
		log.Printf("WaitForStop: Radio mode, adding next random track")
		_, err := player.Playlist.AddNext(strconv.Itoa(search.GetRandomTrackIDs(1)[0]))
		if err != nil {
			helper.ChanMsg(player.Client, "<b style=\"color:red\">Error Adding Radio Track: </b>"+err.Error())
			log.Println("Radio failed to Playlist.AddNext a random track ID, stale database?: ", err)
		}
	}

	if player.Playlist.HasNext() {
		log.Printf("WaitForStop: Playing next track in playlist")
		player.Playlist.Next()
		player.PlayCurrent()
	} else {
		log.Printf("WaitForStop: No more tracks, stopping playback")
		player.Stop(true)
	}
}

func (player *Player) Play(path string) {
	// Use withPlaybackLock to ensure proper synchronization when stopping current stream before playing new one
	err := player.withPlaybackLock(func() error {
		// Use safePlay internally for proper synchronization and race condition protection
		return player.safePlay(path)
	})

	if err != nil {
		helper.ChanMsg(player.Client, "<b style=\"color:red\">Error: </b>"+err.Error())
		return
	}

	// Maintain existing functionality: display now playing information and start waiting for completion
	nowPlaying := player.NowPlaying()
	helper.ChanMsg(player.Client, nowPlaying)
	helper.SetComment(player.Client, nowPlaying)
	player.WaitForStop()
}

func (player *Player) NowPlaying() string {
	artPath := messages.FindCoverArtPath(player.Playlist.GetCurrentPath())
	var artImg, output string
	if artPath != "" {
		artImg = messages.GenerateCoverArtImg(artPath)
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

func (player *Player) Stop(wantsToStop bool) {
	// Use withPlaybackLock to ensure proper synchronization and state management
	err := player.withPlaybackLock(func() error {
		// Use safeStop internally for proper synchronization and timeout handling
		return player.safeStop(wantsToStop)
	})
	
	if err != nil {
		// Log error but maintain backward compatibility by not returning error
		// This preserves the original method signature and behavior
		log.Printf("Stop: Error during synchronized stop operation: %v", err)
		
		// Provide user feedback for timeout or other critical errors
		if strings.Contains(err.Error(), "timeout") {
			helper.ChanMsg(player.Client, "<b style=\"color:orange\">Warning: </b>Stream stop operation timed out")
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
	// Immediate command acknowledgment for user feedback
	helper.ChanMsg(player.Client, "<b>Skip command received, processing...</b>")
	
	var shouldWaitForStop bool
	
	// Use withPlaybackLock to ensure atomic stop-then-play operations without race conditions
	err := player.withPlaybackLock(func() error {
		// Signal that a skip operation is in progress to prevent WaitForStop auto-advance
		player.skipRequested = true
		if player.Playlist.HasNext() && !player.IsRadio {
			// For normal playlist skipping: stop current track, skip to new position, then play
			log.Printf("Skip: Processing playlist skip by %d positions", amount)
			
			// Safely stop the current stream - use false to not set wantsToStop
			// since we're going to play another track immediately
			err := player.safeStop(false)
			if err != nil {
				log.Printf("Skip: Error stopping current stream: %v", err)
				return err
			}
			
			// Skip to the new position in the playlist
			player.Playlist.Skip(amount)
			
			// Safely play the new current track
			if !player.Playlist.IsEmpty() {
				currentPath := player.Playlist.GetCurrentPath()
				log.Printf("Skip: Starting playback of new track: %s", currentPath)
				
				err = player.safePlay(currentPath)
				if err != nil {
					log.Printf("Skip: Error starting new track: %v", err)
					return err
				}
				
				// Update now playing information and comment
				nowPlaying := player.NowPlaying()
				helper.ChanMsg(player.Client, nowPlaying)
				helper.SetComment(player.Client, nowPlaying)
				
				// Signal that we should wait for stop after releasing the lock
				shouldWaitForStop = true
			}
			
		} else if player.IsRadio {
			// For radio mode: stop current stream and immediately add next random track
			log.Printf("Skip: Processing radio skip")
			
			err := player.safeStop(false)
			if err != nil {
				log.Printf("Skip: Error stopping radio stream: %v", err)
				return err
			}
			
			// Add next random track to playlist
			log.Printf("Skip: Adding next random track for radio mode")
			_, err = player.Playlist.AddNext(strconv.Itoa(search.GetRandomTrackIDs(1)[0]))
			if err != nil {
				log.Printf("Skip: Error adding radio track: %v", err)
				helper.ChanMsg(player.Client, "<b style=\"color:red\">Error Adding Radio Track: </b>"+err.Error())
				return err
			}
			
			// Play the next track if available
			if player.Playlist.HasNext() {
				player.Playlist.Next()
				currentPath := player.Playlist.GetCurrentPath()
				log.Printf("Skip: Starting playback of radio track: %s", currentPath)
				
				err = player.safePlay(currentPath)
				if err != nil {
					log.Printf("Skip: Error starting radio track: %v", err)
					return err
				}
				
				// Update now playing information and comment
				nowPlaying := player.NowPlaying()
				helper.ChanMsg(player.Client, nowPlaying)
				helper.SetComment(player.Client, nowPlaying)
				
				// Signal that we should wait for stop after releasing the lock
				shouldWaitForStop = true
			} else {
				log.Printf("Skip: No radio track available, stopping playback")
				err := player.safeStop(true)
				if err != nil {
					return err
				}
			}
			
		} else {
			// No next track available, just stop playback
			log.Printf("Skip: No next track available, stopping playback")
			
			err := player.safeStop(true)
			if err != nil {
				log.Printf("Skip: Error stopping playback: %v", err)
				return err
			}
		}
		
		return nil
	})
	
	if err != nil {
		// Provide error feedback to user
		helper.ChanMsg(player.Client, "<b style=\"color:red\">Skip Error: </b>"+err.Error())
		log.Printf("Skip: Operation failed: %v", err)
		return
	}
	
	// Start waiting for the new track to finish outside the lock
	if shouldWaitForStop {
		go player.WaitForStop()
	}
	
	log.Printf("Skip: Operation completed successfully")
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

// withPlaybackLock wraps operations with mutex protection to prevent race conditions
// It ensures atomic playback operations and proper transitional state management
func (player *Player) withPlaybackLock(operation func() error) error {
	player.playbackMu.Lock()
	defer player.playbackMu.Unlock()

	// Check if player is in transitional state and log warning
	if player.isTransitioning {
		log.Printf("Warning: Operation requested while player is in transitional state, waiting for completion")
	}

	// Execute the operation with proper error handling
	err := operation()
	if err != nil {
		log.Printf("Synchronized playback operation failed: %v", err)
		// Ensure transitional state is cleared on error to prevent deadlock
		if player.isTransitioning {
			log.Printf("Clearing transitional state due to operation failure")
			player.isTransitioning = false
		}
		return err
	}

	return nil
}

// safeStop safely terminates streams with timeout protection and force termination
// It implements a 5-second timeout mechanism and includes error logging for stuck streams
func (player *Player) safeStop(wantsToStop bool) error {
	// Set the wantsToStop flag first
	player.wantsToStop = wantsToStop
	
	// If no stream is playing, nothing to stop
	if !player.IsPlaying() {
		return nil
	}

	// Mark player as transitioning
	player.isTransitioning = true
	defer func() {
		player.isTransitioning = false
	}()

	// Stop the stream
	err := player.stream.Stop()
	if err != nil {
		log.Printf("safeStop: Error calling stream.Stop(): %v", err)
		// Continue with termination attempt even if Stop() failed
	}

	// Update comment immediately to provide user feedback
	helper.SetComment(player.Client, "Not Playing.")

	// Create a channel to signal when stream.Wait() completes
	waitDone := make(chan bool, 1)
	
	// Start a goroutine to wait for the stream to finish
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("safeStop: Panic during stream.Wait(): %v", r)
				waitDone <- false
			}
		}()
		
		if player.stream != nil {
			player.stream.Wait()
		}
		waitDone <- true
	}()

	// Wait for either the stream to finish or timeout (5 seconds)
	select {
	case success := <-waitDone:
		if !success {
			log.Printf("safeStop: Stream termination completed with panic recovery")
		}
	case <-time.After(5 * time.Second):
		log.Printf("safeStop: WARNING - Stream failed to terminate within 5 second timeout, attempting force termination")
		
		// Force termination by setting stream to nil and logging the issue
		if player.stream != nil {
			log.Printf("safeStop: Force terminating stuck stream (state: %v)", player.stream.State())
			// We can't force kill the underlying process, but we can abandon the reference
			// This prevents further operations on the stuck stream
			player.stream = nil
		}
		
		return errors.New("stream termination timeout - force terminated")
	}

	// Final verification that the stream has actually stopped
	if player.IsPlaying() && player.wantsToStop {
		log.Printf("safeStop: WARNING - Race condition detected: stream should have stopped but is still playing")
		// This matches the existing warning in the original Stop method
		return errors.New("race condition: stream failed to stop properly")
	}

	return nil
}

// safePlay safely starts streams with proper synchronization and transitional state management
// It ensures no new streams start while another is stopping and provides atomic stream creation
func (player *Player) safePlay(path string) error {
	// Sanitize the input path
	path = helper.StripHTMLTags(path)
	
	// Mark player as transitioning during stream creation
	player.isTransitioning = true
	defer func() {
		player.isTransitioning = false
	}()

	// If currently playing, safely stop the current stream first
	if player.IsPlaying() {
		log.Printf("safePlay: Stopping current stream before starting new one")
		err := player.safeStop(false)
		if err != nil {
			log.Printf("safePlay: Error stopping current stream: %v", err)
			return err
		}
	}

	// Determine stream type and create appropriate stream
	var err error
	if strings.HasPrefix(path, "http") {
		err = player.createYTStream(path)
	} else {
		err = player.createFileStream(path)
	}

	if err != nil {
		log.Printf("safePlay: Error creating stream for path '%s': %v", path, err)
		return err
	}

	// Start the stream
	if player.stream != nil {
		err = player.stream.Play()
		if err != nil {
			log.Printf("safePlay: Error starting stream playback: %v", err)
			// Clean up the failed stream
			player.stream = nil
			return err
		}
	}

	// Set player state to indicate successful playback start
	player.wantsToStop = false
	
	log.Printf("safePlay: Successfully started playback for: %s", path)
	return nil
}

// createFileStream creates a file-based stream with proper error handling
func (player *Player) createFileStream(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		log.Printf("createFileStream: File not found: %s", path)
		return errors.New("file not found")
	}

	player.stream = gumbleffmpeg.New(player.Client, gumbleffmpeg.SourceFile(path))
	if player.stream == nil {
		return errors.New("failed to create file stream")
	}
	
	player.stream.Volume = player.Volume
	return nil
}

// createYTStream creates a YouTube/URL-based stream with proper validation and error handling
func (player *Player) createYTStream(url string) error {
	if !youtubedl.IsWhiteListedURL(url) {
		return errors.New("URL doesn't meet whitelist requirements")
	}

	player.stream = gumbleffmpeg.New(player.Client, youtubedl.GetYtDLSource(url))
	if player.stream == nil {
		return errors.New("failed to create YouTube stream")
	}
	
	player.stream.Volume = player.Volume
	return nil
}

func (player *Player) SetVolume(value float32) {
	player.Volume = value
	player.Config.Volume = value
	if player.stream != nil {
		player.stream.Volume = value
	}
}
