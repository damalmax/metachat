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
	quoteRegexp         = regexp.MustCompile(`(?s)<quote\b.*?authorname="(.*?)".*?>(.*?)</quote>`)
	urlRegexp           = regexp.MustCompile(`(https?://[^\s]+)`)
	editRegexp          = regexp.MustCompile(`</?e_m\b.*?>`)
)

func convertToMetachat(resource resource) metachat.Message {
	chatGroups := chatRegexp.FindStringSubmatch(resource.ConversationLink)
	content := resource.Content

	edit := editRegexp.MatchString(content)

	content = removeRegexp.ReplaceAllString(content, "")
	content = legacyQuoteRegexp.ReplaceAllString(content, "")
	content = boldRegexp.ReplaceAllString(content, metachat.Bold("${1}"))
	content = italicRegexp.ReplaceAllString(content, metachat.Italic("${1}"))
	content = strikethroughRegexp.ReplaceAllString(content, metachat.Strikethrough("${1}"))
	content = preformattedRegexp.ReplaceAllString(content, metachat.Preformatted("${1}"))
	content = linkRegexp.ReplaceAllString(content, "${1}")
	content = mentionRegexp.ReplaceAllString(content, metachat.Mention("${1}"))
	content = quoteRegexp.ReplaceAllString(content, metachat.Quote("${2}", "${1}"))
	content = strings.Replace(content, "&lt;", "<", -1)
	content = strings.Replace(content, "&gt;", ">", -1)
	content = strings.Replace(content, "&amp;", "&", -1)
	content = strings.Replace(content, "&quot;", "\"", -1)
	content = strings.Replace(content, "&apos;", "'", -1)

	if edit {
		content = metachat.Edit(content)
	}

	return metachat.Message{
		Messenger: "Skype",
		Chat:      chatGroups[1],
		Author:    resource.Imdisplayname,
		Text:      content,
	}
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
	content = metachat.EditRegexp.ReplaceAllString(content, `Edit: ${1}`)

	if msg.Author != "" {
		content = fmt.Sprintf(`<b raw_pre="*" raw_post="*">[%s]</b> %s`, msg.Author, content)
	}

	return message{
		ContentType: "text",
		MessageType: "RichText",
		Content:     content,
	}
}
