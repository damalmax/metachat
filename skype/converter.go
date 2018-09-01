package skype

import (
	"regexp"
	"strings"

	"github.com/thehadalone/metachat/metachat"
)

var (
	chatRegexp          = regexp.MustCompile(`conversations/([0-9]+:[^/]+)`)
	removeRegexp        = regexp.MustCompile(`</?(e|ss|quote|legacyquote)\b.*?>`)
	boldRegexp          = regexp.MustCompile(`</?b\b.*?>`)
	italicRegexp        = regexp.MustCompile(`</?i\b.*?>`)
	strikethroughRegexp = regexp.MustCompile(`</?s\b.*?>`)
	monospaceRegexp     = regexp.MustCompile(`</?pre\b.*?>`)
	linkRegexp          = regexp.MustCompile(`<a\b.*?href="(.*?)">.*?</a>`)
	mentionRegexp       = regexp.MustCompile(`<at\b.*?id=".*?">(.*?)</at>`)
)

func convert(resource resource) metachat.Message {
	chatGroups := chatRegexp.FindStringSubmatch(resource.ConversationLink)

	message := metachat.Message{
		Messenger: "Skype",
		Chat:      chatGroups[1],
		Author:    resource.Imdisplayname,
		Text:      resource.Content,
	}

	message.Text = removeRegexp.ReplaceAllString(message.Text, "")
	message.Text = boldRegexp.ReplaceAllString(message.Text, "*")
	message.Text = italicRegexp.ReplaceAllString(message.Text, "_")
	message.Text = strikethroughRegexp.ReplaceAllString(message.Text, "~")
	message.Text = monospaceRegexp.ReplaceAllString(message.Text, "{code}")
	message.Text = linkRegexp.ReplaceAllString(message.Text, "$1")
	message.Text = mentionRegexp.ReplaceAllString(message.Text, "@$1")
	message.Text = strings.Replace(message.Text, "&lt;", "<", -1)
	message.Text = strings.Replace(message.Text, "&gt;", ">", -1)
	message.Text = strings.Replace(message.Text, "&amp;", "&", -1)
	message.Text = strings.Replace(message.Text, "&quot;", "\"", -1)
	message.Text = strings.Replace(message.Text, "&apos;", "'", -1)

	return message
}
