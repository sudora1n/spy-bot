package locales

import (
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed creator.*.json
var localeFS embed.FS

var Bundle *i18n.Bundle

func Init(defaultLang language.Tag) error {
	Bundle = i18n.NewBundle(defaultLang)
	Bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	entries, err := localeFS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("i18n: locale files could not be found: %v", err)
	}

	for _, f := range entries {
		name := f.Name()
		if f.IsDir() || !strings.HasPrefix(name, "creator.") || !strings.HasSuffix(name, ".json") {
			continue
		}

		data, err := localeFS.ReadFile(filepath.Join(".", name))
		if err != nil {
			return fmt.Errorf("i18n: failed to read file %s: %v", name, err)
		}

		if _, err := Bundle.ParseMessageFileBytes(data, name); err != nil {
			return fmt.Errorf("i18n: failed to parse %s: %v", name, err)
		}
	}

	return nil
}

func NewLocalizer(lang string) *i18n.Localizer {
	return i18n.NewLocalizer(Bundle, lang)
}
