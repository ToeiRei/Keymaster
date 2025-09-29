// Copyright (c) 2025 ToeiRei
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
	"sort"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
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
	// Initialize the bundle with English as the default fallback language.
	bundle = i18n.NewBundle(language.English) // English is the fallback

	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	availableLocales = make(map[string]string)

	// Discover and load all locale files from the embedded filesystem.
	files, err := localeFS.ReadDir("locales")
	if err != nil {
		// This should not happen with a valid embed.
		panic(fmt.Sprintf("failed to read embedded locales directory: %v", err))
	}

	// Process en-ang last to ensure it can overwrite 'en' messages when selected.
	sort.SliceStable(files, func(i, j int) bool {
		return !strings.Contains(files[i].Name(), "en-ang")
	})

	for _, file := range files {
		fileName := file.Name()
		if strings.HasPrefix(fileName, "active.") && (strings.HasSuffix(fileName, ".yaml")) {
			// Extract language code, e.g., "en" from "active.en.yaml"
			langCode := strings.TrimPrefix(fileName, "active.")
			langCode = strings.TrimSuffix(langCode, ".yaml")

			// Special-case for Old English ('en-ang'). The i18n library panics on the 'ang' subtag.
			// To work around this, we manually load the messages and add them to the main 'en' bundle.
			// This overwrites the standard English messages when en-ang is the active language.
			//
			// Why this approach?
			// 1. The i18n library panics if it sees a language subtag without pluralization rules (e.g., 'ang').
			//    This is a known limitation in the go-i18n/golang.org/x/text ecosystem, which relies on
			//    plural rules from the Unicode CLDR. If a rule is missing, loading fails.
			//    This affects other valid but less common languages like 'oc' (Occitan).
			// 2. Simply ignoring the error doesn't work, as the message file fails to load entirely.
			// 3. Using a private-use tag like 'en-x-ang' also fails, as the library's fallback logic
			//    aggressively prefers the base 'en' translations over the 'en-x-ang' ones.
			// By loading 'en-ang' messages directly into the 'en' bundle (and ensuring it's loaded last),
			// we effectively hijack the English localizer when 'en-ang' is selected. When the language
			// is switched back, Init() is re-run, and the standard 'en' file overwrites the 'en-ang' messages.
			if langCode == "en-ang" {
				filePath := path.Join("locales", fileName)
				data, err := localeFS.ReadFile(filePath)
				if err != nil {
					panic(fmt.Sprintf("failed to read embedded locale file %s: %v", fileName, err))
				}
				var raw map[string]string
				if err := yaml.Unmarshal(data, &raw); err != nil {
					panic(fmt.Sprintf("failed to parse locale file %s: %v", fileName, err))
				}
				msgs := make([]*i18n.Message, 0, len(raw))
				for id, val := range raw {
					msgs = append(msgs, &i18n.Message{ID: id, Other: val})
				}
				if err := bundle.AddMessages(language.English, msgs...); err != nil {
					panic(fmt.Sprintf("failed to add 'en-ang' messages to 'en' tag: %v", err))
				}
			} else {
				filePath := path.Join("locales", fileName)
				if _, err := bundle.LoadMessageFileFS(localeFS, filePath); err != nil {
					panic(fmt.Sprintf("failed to load standard locale file %s: %v", fileName, err))
				}
			}

			var displayName string
			switch langCode {
			case "en":
				displayName = "English"
			case "de":
				displayName = "Deutsch"
			case "en-ang":
				displayName = "Ænglisc (Olde English)"
			default:
				// For any other valid language tag, try to parse it to get a display name.
				tag, err := language.Parse(langCode)
				if err == nil {
					displayName = tag.String() // This will produce codes like 'en', 'de'
				} else {
					displayName = langCode // Fallback to the code itself if parsing fails for an unknown reason.
				}
			}
			availableLocales[langCode] = displayName
		}
	}

	SetLang(defaultLang)
}

// SetLang changes the current language for the application.
func SetLang(lang string) {
	currentLang = lang
	localizerTag := lang
	// When the user selects 'en-ang', we use the 'en' localizer because we've
	// overwritten the 'en' messages with the 'en-ang' content.
	if lang == "en-ang" {
		localizerTag = "en"
	}
	localizer = i18n.NewLocalizer(bundle, localizerTag)
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
