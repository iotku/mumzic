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
var IsWaiting bool
var IsPlaying bool
var SkipBy = 1

// Probably horrific logic
func WaitForStop(client *gumble.Client) {
	// wait for playback to stop
	if IsWaiting == true {
		return
	}
	IsWaiting = true
	Stream.Wait()
	switch DoNext {
	case "stop":
		IsPlaying = false
		client.Self.SetComment("Not Playing.")
		// Do nothing
	case "next":
		if playlist.HasNext() {
			Play(playlist.Next(), client)
		} else {
			DoNext = "stop"
			IsPlaying = false
		}
	case "skip":
		if playlist.HasNext() {
			Play(playlist.Skip(SkipBy), client)
			DoNext = "next"
		} else {
			DoNext = "stop"
			IsPlaying = false
		}
	default:
		IsWaiting = false
	}
	IsWaiting = false
	SkipBy = 1
	return
}

func Stop(client *gumble.Client) {
	if Stream != nil {
		if Stream.State() == gumbleffmpeg.StatePlaying {
			Stream.Stop() // Only error this will respond with is if not playing.
		}
	}
}

func Play(path string, client *gumble.Client) {
	// Stop if currently playing
	Stop(client)
	path = helper.StripHTMLTags(path)
	IsPlaying = true
	if strings.HasPrefix(path, "http") {
		PlayYT(path, client)
	} else {
		PlayFile(path, client)
	}

	helper.ChanMsg(client, "Now Playing: "+playlist.GetCurrentHuman())
	client.Self.SetComment("Now Playing: " + playlist.GetCurrentHuman())
	go WaitForStop(client)
}

func PlayFile(path string, client *gumble.Client) {
	if Stream != nil {
		err := Stream.Stop()
		helper.DebugPrintln(err)
	}

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

	if Stream != nil {
		err := Stream.Stop()
		helper.DebugPrintln(err)
	}

	Stream = gumbleffmpeg.New(client, youtubedl.GetYtdlSource(url))
	Stream.Volume = config.VolumeLevel

	if err := Stream.Play(); err != nil {
		helper.DebugPrintln(err)
	} else {
		helper.DebugPrintln("Playing:", url)
	}
}
