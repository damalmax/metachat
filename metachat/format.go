package metachat

import (
	"fmt"
	"regexp"
)

// TODO docs
var (
	BoldRegexp          = regexp.MustCompile("#{bold}(.*?){bold}#")
	ItalicRegexp        = regexp.MustCompile("#{italic}(.*?){italic}#")
	StrikethroughRegexp = regexp.MustCompile("#{strikethrough}(.*?){strikethrough}#")
	PreformattedRegexp  = regexp.MustCompile("(?s)#{preformatted}(.*?){preformatted}#")
	MentionRegexp       = regexp.MustCompile("#{mention}(.*?){mention}#")
	QuoteRegexp         = regexp.MustCompile("#{quote author=(.*?)}(.*?){quote}#")
)

func Bold(text string) string {
	return fmt.Sprintf("#{bold}%s{bold}#", text)
}

func Italic(text string) string {
	return fmt.Sprintf("#{italic}%s{italic}#", text)
}

func Strikethrough(text string) string {
	return fmt.Sprintf("#{strikethrough}%s{strikethrough}#", text)
}

func Preformatted(text string) string {
	return fmt.Sprintf("#{preformatted}%s{preformatted}#", text)
}

func Mention(text string) string {
	return fmt.Sprintf("#{mention}%s{mention}#", text)
}

func Quote(text, author string) string {
	return fmt.Sprintf("#{quote author=%s}%s{quote}#", author, text)
}
