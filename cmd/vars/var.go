package vars

import (
	"github.com/BurntSushi/toml"
	AutoAI "github.com/bincooo/MiaoX"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	Manager = AutoAI.NewBotManager()

	localizes *i18n.Localizer

	Proxy string
	I18nT string

	GlobalPile     string
	GlobalPileSize int
	GlobalToken    string

	Bu     string
	Suffix string
)

func InitI18n() {
	i18nKit := i18n.NewBundle(language.Und)
	i18nKit.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	i18nKit.MustLoadMessageFile("lang.toml")
	localizes = i18n.NewLocalizer(i18nKit, I18nT)
}

func I18n(key string) string {
	return localizes.MustLocalize(&i18n.LocalizeConfig{
		MessageID: key + "." + I18nT,
	})
}
