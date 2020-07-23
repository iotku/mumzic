package helper

import (
	"github.com/iotku/mumzic/config"
	"layeh.com/gumble/gumble"
	"log"
	"regexp"
	"strings"
	"time"
)

// Message rate limiting
var msgBurstCount uint
var msgLastSentTime time.Time

func init() {
	msgBurstCount = 0
	msgLastSentTime = time.Now()
	config.VolumeLevel = 0.25 // Default volume level
}

func StripHTMLTags(str string) string {
	removeHTMLTags := regexp.MustCompile("<[^>]*>")
	str = removeHTMLTags.ReplaceAllString(str, "")
	return str
}

// Sends Message to current mumble channel
func ChanMsg(client *gumble.Client, msg string) {
	// Mumble servers often have rate limiting which should be accounted for
	// Messages sent too fast will be dropped!
	// Murmur default: Burst = 5, MessageLimit (after burst) 1/sec

	// Buffering logic to avoid messages being dropped by sending too fast
	// This is probably something appropriate to use channels for.

	currentTime := time.Now()
	if msgLastSentTime.Add(5 * time.Second).Before(currentTime) {
		msgBurstCount = 1
	} else {
		msgBurstCount++
	}

	if msgBurstCount >= 5 {
		time.Sleep(1 * time.Second)
	}
	client.Self.Channel.Send(msg, false)
	msgLastSentTime = currentTime
}

// Remove prefix from command for single argument (I.E. "!play 22" -> "22")
func LazyRemovePrefix(message, prefix string) string {
	char := config.CmdPrefix
	return strings.TrimSpace(message[len(char+prefix):])
}

func DebugPrintln(inter ...interface{}) {
	if inter != nil {
		log.Println(inter...)
	}
}
