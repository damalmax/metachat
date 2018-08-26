package skype

import (
	"regexp"

	"github.com/thehadalone/metachat/metachat"
)

var (
	chatRegexp  = regexp.MustCompile(`conversations/([0-9]+:[^/]+)`)
	emojiRegexp = regexp.MustCompile("<ss.*?>(?P<text>.*?)</ss>")
	linkRegexp  = regexp.MustCompile("<a href=.*?>(?P<link>.*?)</a>")
)

func convert(resource resource) metachat.Message {
	chatGroups := chatRegexp.FindStringSubmatch(resource.ConversationLink)

	message := metachat.Message{
		Messenger: "Skype",
		Chat:      chatGroups[1],
		Author:    resource.Imdisplayname,
		Text:      resource.Content,
	}

	message.Text = emojiRegexp.ReplaceAllString(message.Text, "${text}")
	message.Text = linkRegexp.ReplaceAllString(message.Text, "${link}")

	return message
}
