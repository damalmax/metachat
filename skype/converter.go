package skype

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/thehadalone/metachat/metachat"
)

var (
	chatRegexp          = regexp.MustCompile(`conversations/([0-9]+:[^/]+)`)
	removeRegexp        = regexp.MustCompile(`</?(e|e_m|ss)\b.*?>`)
	legacyQuoteRegexp   = regexp.MustCompile(`(?s)<legacyquote\b.*?>.*?</legacyquote\b.*?>`)
	boldRegexp          = regexp.MustCompile(`<b\b.*?>(.*?)</b\b.*?>`)
	italicRegexp        = regexp.MustCompile(`<i\b.*?>(.*?)</i\b.*?>`)
	strikethroughRegexp = regexp.MustCompile(`<s\b.*?>(.*?)</s\b.*?>`)
	preformattedRegexp  = regexp.MustCompile(`(?s)<pre\b.*?>(.*?)</pre\b.*?>`)
	linkRegexp          = regexp.MustCompile(`<a\b.*?href="(.*?)">.*?</a>`)
	mentionRegexp       = regexp.MustCompile(`<at\b.*?id=".*?">(.*?)</at>`)
	quoteRegexp         = regexp.MustCompile(`(?s)<quote\b.*?authorname="(.*?)" timestamp=.*?>(.*?)</quote>`)
	urlRegexp           = regexp.MustCompile(`(https?://[^\s]+)`)
)

func convertToMetachat(resource resource) metachat.Message {
	chatGroups := chatRegexp.FindStringSubmatch(resource.ConversationLink)

	message := metachat.Message{
		Messenger: "Skype",
		Chat:      chatGroups[1],
		Author:    resource.Imdisplayname,
		Text:      resource.Content,
	}

	message.Text = removeRegexp.ReplaceAllString(message.Text, "")
	message.Text = legacyQuoteRegexp.ReplaceAllString(message.Text, "")
	message.Text = boldRegexp.ReplaceAllString(message.Text, metachat.Bold("${1}"))
	message.Text = italicRegexp.ReplaceAllString(message.Text, metachat.Italic("${1}"))
	message.Text = strikethroughRegexp.ReplaceAllString(message.Text, metachat.Strikethrough("${1}"))
	message.Text = preformattedRegexp.ReplaceAllString(message.Text, metachat.Preformatted("${1}"))
	message.Text = linkRegexp.ReplaceAllString(message.Text, "${1}")
	message.Text = mentionRegexp.ReplaceAllString(message.Text, metachat.Mention("${1}"))
	message.Text = quoteRegexp.ReplaceAllString(message.Text, metachat.Quote("${2}", "${1}"))
	message.Text = strings.Replace(message.Text, "&lt;", "<", -1)
	message.Text = strings.Replace(message.Text, "&gt;", ">", -1)
	message.Text = strings.Replace(message.Text, "&amp;", "&", -1)
	message.Text = strings.Replace(message.Text, "&quot;", "\"", -1)
	message.Text = strings.Replace(message.Text, "&apos;", "'", -1)

	return message
}

func convertToSkype(msg metachat.Message) message {
	content := metachat.BoldRegexp.ReplaceAllString(msg.Text, `<b raw_pre="*" raw_post="*">${1}</b>`)
	content = metachat.ItalicRegexp.ReplaceAllString(content, `<i raw_pre="_" raw_post="_">${1}</i>`)
	content = metachat.StrikethroughRegexp.ReplaceAllString(content, `<s raw_pre="~" raw_post="~">${1}</s>`)

	content = metachat.PreformattedRegexp.ReplaceAllString(content,
		`<pre raw_pre="{{code}}" raw_post="{{code}}">${1}</pre>`)

	content = metachat.MentionRegexp.ReplaceAllString(content, `@${1}`)
	content = metachat.QuoteRegexp.ReplaceAllString(content, "Quote from ${1}:\n${2}\n\n")
	content = urlRegexp.ReplaceAllString(content, `<a href="${1}">${1}</a>`)

	if msg.Author != "" {
		content = fmt.Sprintf(`<b raw_pre="*" raw_post="*">[%s]</b> %s`, msg.Author, content)
	}

	return message{
		ContentType: "text",
		MessageType: "RichText",
		Content:     content,
	}
}
