package vars

import (
	"github.com/BurntSushi/toml"
	"github.com/bincooo/AutoAI"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"os"
	"strings"
)

var (
	Manager = AutoAI.NewBotManager()

	localizes *i18n.Localizer

	Proxy string
	I18nT string

	GlobalPadding     string
	GlobalPaddingSize int
	GlobalToken       string
	AutoPwd           string

	Bu     string
	Suffix string

	EnablePool bool
	Gen        bool
)

func init() {
	EnablePool = loadEnvBool("ENABLE_POOL", false)
}

func loadEnvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return strings.ToLower(value) == "true"
}

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
