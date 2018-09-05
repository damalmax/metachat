package slack

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nlopes/slack/slackevents"
	"github.com/thehadalone/metachat/metachat"
)

var (
	boldRegexp          = regexp.MustCompile(`\*(.*?)\*`)
	italicRegexp        = regexp.MustCompile(`_(.*?)_`)
	strikethroughRegexp = regexp.MustCompile(`~(.*?)~`)
	preformattedRegexp  = regexp.MustCompile("(?s)```(.*?)```")
	mentionRegexp       = regexp.MustCompile(`<@(.*?)>`)
	urlRegexp           = regexp.MustCompile(`<(https?://.*?)>`)
)

func convertToSlack(msg metachat.Message) string {
	content := metachat.BoldRegexp.ReplaceAllString(msg.Text, "*${1}*")
	content = metachat.ItalicRegexp.ReplaceAllString(content, "_${1}_")
	content = metachat.StrikethroughRegexp.ReplaceAllString(content, "~${1}~")
	content = metachat.PreformattedRegexp.ReplaceAllString(content, "```${1}```")
	content = metachat.MentionRegexp.ReplaceAllString(content, "@${1}")
	content = metachat.QuoteRegexp.ReplaceAllString(content, "Quote from ${1}:\n${2}\n\n")
	content = metachat.EditRegexp.ReplaceAllString(content, "Edit: ${1}")

	if msg.Author != "" {
		content = fmt.Sprintf("*[%s]* %s", msg.Author, content)
	}

	return content
}

func (c *Client) convertToMetachat(event *slackevents.MessageEvent, chat string, edit bool) (metachat.Message, error) {
	author, _ := c.usersByID.get(event.User)

	content := boldRegexp.ReplaceAllString(event.Text, metachat.Bold("${1}"))
	content = italicRegexp.ReplaceAllString(content, metachat.Italic("${1}"))
	content = strikethroughRegexp.ReplaceAllString(content, metachat.Strikethrough("${1}"))
	content = preformattedRegexp.ReplaceAllString(content, metachat.Preformatted("${1}"))
	content = urlRegexp.ReplaceAllString(content, "${1}")
	content = mentionRegexp.ReplaceAllStringFunc(content, func(match string) string {
		id := mentionRegexp.FindStringSubmatch(match)[1]
		name, _ := c.usersByID.get(id)

		return "@" + name
	})

	content = strings.Replace(content, "&lt;", "<", -1)
	content = strings.Replace(content, "&gt;", ">", -1)
	content = strings.Replace(content, "&amp;", "&", -1)

	if edit {
		content = metachat.Edit(content)
	}

	return metachat.Message{
		Messenger: "Slack",
		Chat:      chat,
		Author:    author,
		Text:      content,
	}, nil
}
