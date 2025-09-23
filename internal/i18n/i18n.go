// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package i18n provides internationalization and localization support for Keymaster.
// It uses the go-i18n library to load and manage translation files, allowing the
// user interface to be displayed in multiple languages.
package i18n

import (
	"embed"
	"io/fs"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// localeFS embeds the YAML translation files from the 'locales' directory
// into the application binary.
//
//go:embed locales/*.yaml
var localeFS embed.FS

// bundle stores all the loaded translation messages from the locale files.
var bundle *i18n.Bundle

// localizer is used to translate messages into a specific language.
var localizer *i18n.Localizer

// Init initializes the i18n bundle and sets up the localizer for a specific language.
// It parses all embedded YAML files from the 'locales' directory.
func Init(lang string) {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	files, _ := fs.ReadDir(localeFS, "locales")
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		data, _ := localeFS.ReadFile("locales/" + f.Name())
		bundle.ParseMessageFileBytes(data, f.Name())
	}

	localizer = i18n.NewLocalizer(bundle, lang)
}

// T is a convenience function to translate a message by its ID.
// If the i18n system has not been initialized, it will default to English.
// If a translation for the given ID is not found, it returns the ID itself.
func T(messageID string) string {
	if localizer == nil {
		Init("en")
	}
	msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: messageID})
	if err != nil {
		// If the message ID is not found, go-i18n returns an error.
		// In this case, we return the message ID itself as a fallback.
		return messageID
	}
	return msg
}

// SetLang changes the active language of the localizer.
func SetLang(lang string) {
	Init(lang)
}
