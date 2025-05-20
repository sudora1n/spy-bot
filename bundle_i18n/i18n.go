package bundlei18n

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var Bundle *i18n.Bundle

func Init(localeDir string, defaultLang language.Tag) error {
	Bundle = i18n.NewBundle(defaultLang)
	Bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	files, err := filepath.Glob(filepath.Join(localeDir, "messages.*.json"))
	if err != nil {
		return fmt.Errorf("i18n: locale files could not be found: %v", err)
	}
	for _, f := range files {
		if _, err := Bundle.LoadMessageFile(f); err != nil {
			return fmt.Errorf("i18n: failed to load %s: %v", f, err)
		}
	}
	return nil
}

func NewLocalizer(lang string) *i18n.Localizer {
	return i18n.NewLocalizer(Bundle, lang)
}
