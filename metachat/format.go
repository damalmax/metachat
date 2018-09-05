package metachat

import (
	"fmt"
	"regexp"
)

// Set of pre-compiled regular expressions for all Metachat tags.
var (
	BoldRegexp          = regexp.MustCompile("#{bold}(.*?){bold}#")
	ItalicRegexp        = regexp.MustCompile("#{italic}(.*?){italic}#")
	StrikethroughRegexp = regexp.MustCompile("#{strikethrough}(.*?){strikethrough}#")
	PreformattedRegexp  = regexp.MustCompile("(?s)#{preformatted}(.*?){preformatted}#")
	MentionRegexp       = regexp.MustCompile("#{mention}(.*?){mention}#")
	QuoteRegexp         = regexp.MustCompile("#{quote author=(.*?)}(.*?){quote}#")
	EditRegexp          = regexp.MustCompile("#{edit}(.*?){edit}#")
)

// Bold marks text as bold using Metachat tag.
func Bold(text string) string {
	return fmt.Sprintf("#{bold}%s{bold}#", text)
}

// Italic marks text as italic using Metachat tag.
func Italic(text string) string {
	return fmt.Sprintf("#{italic}%s{italic}#", text)
}

// Strikethrough marks text as strikethrough using Metachat tag.
func Strikethrough(text string) string {
	return fmt.Sprintf("#{strikethrough}%s{strikethrough}#", text)
}

// Preformatted marks text as preformatted using Metachat tag.
func Preformatted(text string) string {
	return fmt.Sprintf("#{preformatted}%s{preformatted}#", text)
}

// Mention marks text as mention using Metachat tag.
func Mention(text string) string {
	return fmt.Sprintf("#{mention}%s{mention}#", text)
}

// Quote marks text as quote using Metachat tag.
func Quote(text, author string) string {
	return fmt.Sprintf("#{quote author=%s}%s{quote}#", author, text)
}

// Edit marks text as edited using Metachat tag.
func Edit(text string) string {
	return fmt.Sprintf("#{edit}%s{edit}#", text)
}
