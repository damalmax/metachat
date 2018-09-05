package telegram

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/thehadalone/metachat/metachat"
)

func convertToMetachat(msg *tgbotapi.Message, edit bool) metachat.Message {
	content := formatText(msg)
	if msg.ReplyToMessage != nil {
		content = metachat.Quote(formatText(msg.ReplyToMessage), author(msg.ReplyToMessage)) + " " + content
	}

	if edit {
		content = metachat.Edit(content)
	}

	return metachat.Message{
		Messenger: "Telegram",
		Chat:      strconv.FormatInt(msg.Chat.ID, 10),
		Author:    author(msg),
		Text:      content,
	}
}

func convertToTelegram(message metachat.Message) tgbotapi.MessageConfig {
	content := strings.Replace(message.Text, "*", "\\*", -1)
	content = strings.Replace(content, "_", "\\_", -1)
	content = metachat.BoldRegexp.ReplaceAllString(content, "*${1}*")
	content = metachat.ItalicRegexp.ReplaceAllString(content, "_${1}_")
	content = metachat.PreformattedRegexp.ReplaceAllString(content, "```${1}```")
	content = metachat.MentionRegexp.ReplaceAllString(content, "@${1}")
	content = metachat.QuoteRegexp.ReplaceAllString(content, "Quote from ${1}:\n${2}\n\n")
	content = metachat.EditRegexp.ReplaceAllString(content, "Edit: ${1}")

	if message.Author != "" {
		content = fmt.Sprintf("*[%s]* %s", message.Author, content)
	}

	msg := tgbotapi.NewMessage(0, content)
	msg.ParseMode = tgbotapi.ModeMarkdown

	return msg
}

func formatText(msg *tgbotapi.Message) string {
	content := msg.Text
	if msg.Entities != nil {
		chunks := make([]string, 0)
		lastChunkEnd := 0
		for _, entity := range *msg.Entities {
			switch entity.Type {
			case "mention":
				start := entity.Offset
				end := start + entity.Length
				chunks = append(chunks, content[lastChunkEnd:start])
				chunks = append(chunks, metachat.Mention(content[start+1:end]))
				lastChunkEnd = end

			case "bold":
				start := entity.Offset
				end := start + entity.Length
				chunks = append(chunks, content[lastChunkEnd:start])
				chunks = append(chunks, metachat.Bold(content[start:end]))
				lastChunkEnd = end

			case "italic":
				start := entity.Offset
				end := start + entity.Length
				chunks = append(chunks, content[lastChunkEnd:start])
				chunks = append(chunks, metachat.Italic(content[start:end]))
				lastChunkEnd = end

			case "code", "pre":
				start := entity.Offset
				end := start + entity.Length
				chunks = append(chunks, content[lastChunkEnd:start])
				chunks = append(chunks, metachat.Preformatted(content[start:end]))
				lastChunkEnd = end
			}
		}

		chunks = append(chunks, content[lastChunkEnd:])
		content = strings.Join(chunks, "")
	}

	return content
}

func author(msg *tgbotapi.Message) string {
	return fmt.Sprintf("%s %s", msg.From.FirstName, msg.From.LastName)
}
