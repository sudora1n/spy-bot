package format

import (
	"fmt"
	"html"
	"ssuspy-bot/consts"
	"strings"
	"unicode/utf8"
)

func TruncateText(text string, maxLength int, replaceN bool) (result string) {
	return CustomTruncateText(text, maxLength, "...", replaceN)
}

func CustomTruncateText(text string, maxLength int, endString string, replaceN bool) (result string) {
	if replaceN {
		text = strings.ReplaceAll(text, "\n", " ")
	}

	if maxLength <= 0 {
		return ""
	}

	if utf8.RuneCountInString(text) <= maxLength {
		return text
	}

	runes := []rune(text)

	endStringLen := utf8.RuneCountInString(endString)
	result = string(runes[:maxLength-endStringLen])
	return result + endString
}

func Name(firstName string, lastName string) string {
	if lastName != "" {
		firstName += fmt.Sprintf(" %s", lastName)
	}
	firstName = html.EscapeString(
		TruncateText(firstName, consts.MAX_NAME_LEN, true),
	)

	return firstName
}

func Caption(text string) string {
	return html.EscapeString(
		TruncateText(text, consts.MAX_MEDIA_CAPTION_LEN, false),
	)
}

func ChooseTime(editDate, date int64) int64 {
	if editDate > 0 {
		return editDate
	}
	return date
}
