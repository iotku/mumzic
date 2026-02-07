package helper

import (
	"log"
	"regexp"
	"time"

	"layeh.com/gumble/gumble"
)

type msgBundle struct {
	client   *gumble.Client
	username string // If username not "", consider as private message
	msg      string
}

// Message rate limiting
var msgBurstCount uint = 0
var msgLastSentTime time.Time
var msgChan = make(chan msgBundle)

// How long to wait when throttling
const waitDuration = 1*time.Second + 120*time.Millisecond

func init() {
	msgLastSentTime = time.Now()
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
		if msgLastSentTime.Add(6 * time.Second).Before(currentTime) {
			msgBurstCount = 0
		}
		msgBurstCount++

		if msgBurstCount >= 5 {
			time.Sleep(waitDuration)
		}

		// Actually send message to mumble server
		if bundle.username == "" {
			bundle.client.Self.Channel.Send(bundle.msg, false)
		} else if bundle.username == bundle.client.Self.Name { // set comment for self
			bundle.client.Self.SetComment(bundle.msg)
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

// SetComment sets the comment for itself
func SetComment(client *gumble.Client, comment string) {
	msgChan <- msgBundle{client, client.Self.Name, comment}
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

func DebugPrintln(inter ...interface{}) {
	if inter != nil {
		log.Println(inter...)
	}
}

func LogErr(err error, why string) {
	if err != nil {
		log.Println("[error] " + why + ": " + err.Error())
	}
}
