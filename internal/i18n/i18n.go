package i18n

import (
	"embed"
	"io/fs"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

//go:embed locales/*.yaml
var localeFS embed.FS

var bundle *i18n.Bundle
var localizer *i18n.Localizer

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

func T(messageID string) string {
	if localizer == nil {
		Init("en")
	}
	msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: messageID})
	if err != nil {
		return messageID
	}
	return msg
}

func SetLang(lang string) {
	Init(lang)
}
