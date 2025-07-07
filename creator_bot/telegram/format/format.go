package format

import (
	"fmt"
	"html"
	"unicode/utf8"

	"ssuspy-creator-bot/consts"
)

func TruncateText(text string, maxLength int) (result string) {
	if maxLength <= 0 {
		return ""
	}

	if utf8.RuneCountInString(text) <= maxLength {
		return text
	}

	runes := []rune(text)

	endString := "..."
	endStringLen := utf8.RuneCountInString(endString)
	result = string(runes[:maxLength-endStringLen])
	return result + endString
}

func Name(name string, lastName string) string {
	if lastName != "" {
		name += fmt.Sprintf(" %s", lastName)
	}
	name = html.EscapeString(
		TruncateText(name, consts.MAX_NAME_LEN),
	)

	return name
}
