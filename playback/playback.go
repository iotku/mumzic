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
		if len(playlist.Songlist) > playlist.Currentsong+1 {
			playlist.Currentsong++
			Play(playlist.Songlist[playlist.Currentsong], client)
		} else {
			DoNext = "stop"
			IsPlaying = false
		}
	case "skip":
		if playlist.Currentsong+SkipBy < 0 {
			IsPlaying = false
			break
		}
		if len(playlist.Songlist) > (playlist.Currentsong + SkipBy) {
			playlist.Currentsong = playlist.Currentsong + SkipBy
			Play(playlist.Songlist[playlist.Currentsong], client)
			DoNext = "next"
			SkipBy = 1
		} else if len(playlist.Songlist) > (playlist.Currentsong + 1) {
			playlist.Currentsong = playlist.Currentsong + 1
			Play(playlist.Songlist[playlist.Currentsong], client)
			DoNext = "next"
			SkipBy = 1
		} else {
			DoNext = "stop"
			IsPlaying = false
		}
	default:
		IsWaiting = false
	}
	IsWaiting = false
	return
}

func Play(path string, client *gumble.Client) {
	if Stream != nil {
		if Stream.State() == gumbleffmpeg.StatePlaying {
			err := Stream.Stop()
			helper.DebugPrintln(err)
		}
	}

	path = helper.StripHTMLTags(path)
	IsPlaying = true
	helper.ChanMsg(client, "Now Playing: "+playlist.Metalist[playlist.Currentsong])
	client.Self.SetComment("Now Playing: " + playlist.Metalist[playlist.Currentsong])
	if strings.HasPrefix(path, "http") {
		PlayYT(path, client)
	} else {
		PlayFile(path, client)
	}

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
