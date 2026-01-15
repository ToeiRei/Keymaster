// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package i18n handles internationalization for Keymaster.
// It uses go-i18n to load and manage translation files, and provides
// a simple interface for the rest of the application to get translated strings.
package i18n // import "github.com/toeirei/keymaster/internal/i18n"

import (
	"embed"
	"fmt"
	"path" // Use the 'path' package for consistent forward slashes
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	"gopkg.in/yaml.v3"
)

//go:embed locales/*.yaml
var localeFS embed.FS

var (
	bundle           *i18n.Bundle
	localizer        *i18n.Localizer
	currentLang      string
	availableLocales map[string]string
)

// Init initializes the i18n bundle, discovers available locales, and sets the default language.
func Init(defaultLang string) {
	bundle = i18n.NewBundle(language.English) // English is the fallback

	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	availableLocales = make(map[string]string)

	// Discover and load all locale files from the embedded filesystem.
	files, err := localeFS.ReadDir("locales")
	if err != nil {
		// This should not happen with a valid embed.
		panic(fmt.Sprintf("failed to read embedded locales directory: %v", err))
	}

	for _, file := range files {
		fileName := file.Name()
		if strings.HasPrefix(fileName, "active.") && (strings.HasSuffix(fileName, ".yaml")) {
			// Extract language code, e.g., "en" from "active.en.yaml"
			langCode := strings.TrimPrefix(fileName, "active.")
			langCode = strings.TrimSuffix(langCode, ".yaml")

			var displayName string
			// Special case for Old English, which has a custom display name.
			if langCode == "art-x-ang" {
				displayName = "Ænglisc (Olde English)"
			} else {
				// For all other languages, try to get the native display name.
				tag, err := language.Parse(langCode)
				if err == nil {
					displayName = display.Self.Name(tag)
				} else {
					displayName = langCode // Fallback to the code itself if parsing fails.
				}
			}
			availableLocales[langCode] = displayName

			// Load the file into the bundle
			filePath := path.Join("locales", fileName)
			_, err := bundle.LoadMessageFileFS(localeFS, filePath)
			if err != nil {
				panic(fmt.Sprintf("failed to load locale file %s: %v", fileName, err))
			}
		}
	}

	SetLang(defaultLang)
}

// SetLang changes the current language for the application.
func SetLang(lang string) {
	currentLang = lang
	// Using 'art-x-ang' treats it as a standalone language, so no special
	// fallback logic is needed. The library will just use the messages as-is.
	localizer = i18n.NewLocalizer(bundle, lang)
}

// GetLang returns the currently active language code.
func GetLang() string {
	return currentLang
}

// GetAvailableLocales returns a map of language codes to their display names.
func GetAvailableLocales() map[string]string {
	return availableLocales
}

// T is the main translation function. It retrieves a translated string by its ID.
// It supports pluralization and template variables.
func T(messageID string, templateData ...interface{}) string {
	// Only use pluralization when a template data map provides a Count value.
	var data map[string]interface{}
	var pluralCount interface{}
	if len(templateData) > 0 {
		if m, ok := templateData[0].(map[string]interface{}); ok {
			data = m
			if c, ok := m["Count"]; ok {
				pluralCount = c
			}
		}
	}

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
		PluralCount:  pluralCount,
	})
	if err != nil {
		// Fallback for missing translations
		msg = messageID
	}

	// Support for legacy `printf` style formatting when args are provided and not a map.
	if len(templateData) > 0 {
		if _, isMap := templateData[0].(map[string]interface{}); !isMap {
			return fmt.Sprintf(msg, templateData...)
		}
	}

	return msg
}
