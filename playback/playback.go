package playback

import (
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

var Stream *gumbleffmpeg.Stream

// This can probably be replaced by with good control flow and/or channels, might be subject to race conditions
var DoNext = "stop" // stop, next, skip [int]
var SkipBy = 1

// Wait for playback stream to stop to perform next action
func WaitForStop(client *gumble.Client) {
	Stream.Wait()
	switch DoNext {
	case "stop":
		client.Self.SetComment("Not Playing.")
		// Do nothing
	case "next":
		if playlist.HasNext() {
			Play(playlist.Next(), client)
		} else {
			DoNext = "stop"
		}
	case "skip":
		if playlist.HasNext() {
			Play(playlist.Skip(SkipBy), client)
			DoNext = "next"
		} else {
			DoNext = "stop"
		}
	default:
		SkipBy = 1
	}
}

func Play(path string, client *gumble.Client) {
	// Stop if currently playing
	Stop()
	path = helper.StripHTMLTags(path)
	if strings.HasPrefix(path, "http") {
		PlayYT(path, client)
	} else {
		PlayFile(path, client)
	}

	helper.ChanMsg(client, "Now Playing: "+playlist.GetCurrentHuman())
	client.Self.SetComment("Now Playing: " + playlist.GetCurrentHuman())
	go WaitForStop(client)
}

func IsPlaying() bool {
	return Stream != nil && Stream.State() == gumbleffmpeg.StatePlaying
}

func Stop() {
	if IsPlaying() {
		Stream.Stop() //#nosec G104 -- Only error this will respond with is stream not playing.
	}
}

func PlayFile(path string, client *gumble.Client) {
	Stream = gumbleffmpeg.New(client, gumbleffmpeg.SourceFile(path))
	Stream.Volume = config.VolumeLevel

	if err := Stream.Play(); err != nil {
		helper.DebugPrintln(err)
	} else {
		helper.DebugPrintln("Playing:", path)
	}
}

// Play youtube video
func PlayYT(url string, client *gumble.Client) {
	url = helper.StripHTMLTags(url)
	if youtubedl.IsWhiteListedURL(url) == false {
		log.Printf("PlayYT Failed: URL %s Doesn't meet whitelist", url)
		return
	}

	Stream = gumbleffmpeg.New(client, youtubedl.GetYtdlSource(url))
	Stream.Volume = config.VolumeLevel

	if err := Stream.Play(); err != nil {
		helper.DebugPrintln(err)
	} else {
		helper.DebugPrintln("Playing:", url)
	}
}
