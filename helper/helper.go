package helper

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/iotku/mumzic/config"
	"layeh.com/gumble/gumble"
)

type msgBundle struct {
	client   *gumble.Client
	username string // If username not "", consider as private message
	msg      string
}

// Message rate limiting
var msgBurstCount uint
var msgLastSentTime time.Time
var msgChan chan (msgBundle)

// Useful information
var BotUsername string
var ServerHostname string

func init() {
	msgBurstCount = 0
	msgLastSentTime = time.Now()
	msgChan = make(chan msgBundle)
	go watchMsgChan()
}

// watchMsgChan waits on the msgChan for incoming messages and rate limits the output in an attempt to avoid rate limits
func watchMsgChan() {
	for {
		bundle := <-msgChan
		// Mumble servers generally have rate limiting which should be accounted for
		// Messages sent too fast will be dropped!
		// Murmur default: Burst = 5, MessageLimit (after burst) 1/sec

		currentTime := time.Now()

		// Buffering logic to avoid messages being dropped by sending too fast
		// TODO: Should the cooldown for private messages remain the same as for main channel messages?
		if msgLastSentTime.Add(5 * time.Second).Before(currentTime) {
			msgBurstCount = 1
		} else {
			msgBurstCount++
		}

		if msgBurstCount >= 5 {
			time.Sleep(1 * time.Second)
		}

		// Actually send message to mumble server
		if bundle.username == "" {
			bundle.client.Self.Channel.Send(bundle.msg, false)
		} else {
			gumbleUser := bundle.client.Users.Find(bundle.username)
			if gumbleUser == nil { // User not found.
				return
			}
			gumbleUser.Send(bundle.msg)
		}

		msgLastSentTime = currentTime
	}
}

// StripHTMLTags removes all html tags from string in order to be parsed easier
func StripHTMLTags(str string) string {
	removeHTMLTags := regexp.MustCompile("<[^>]*>")
	return removeHTMLTags.ReplaceAllString(str, "")
}

// ChanMsg sends supplied to the current mumble channel the bot is occupying
func ChanMsg(client *gumble.Client, msg string) {
	msgChan <- msgBundle{client, "", msg}
}

// UserMsg sends msg directly to user by username supplied
func UserMsg(client *gumble.Client, username string, msg string) {
	msgChan <- msgBundle{client, username, msg}
}

// MsgDispatch will send to either UserMsg or ChanMsg depending on if message is private or not.
// For messages that will reply in PM (such as !list) if sent directly to bot
func MsgDispatch(client *gumble.Client, isPrivate bool, username string, msg string) {
	if isPrivate {
		UserMsg(client, username, msg)
	} else {
		ChanMsg(client, msg)
	}
}

// LazyRemovePrefix gets the remaining text from a command invocation excluding the command prefix and command name (I.E. "!play 22" -> "22")
func LazyRemovePrefix(message, prefix string) string {
	if strings.HasPrefix(message, config.CmdPrefix) {
		return strings.TrimSpace(message[len(config.CmdPrefix+prefix):])
	} else {
		return strings.TrimSpace(message[len(BotUsername+prefix)+1:])
	}
}

func DebugPrintln(inter ...interface{}) {
	if inter != nil {
		log.Println(inter...)
	}
}
