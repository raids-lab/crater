package i18n

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
)

type Language string

const (
	En   Language = "en"
	ZhCN Language = "zh-CN"
)

var currentLang Language = En

var translations = mergeCatalogs(
	catalogRoot,
	catalogAuth,
	catalogConfig,
	catalogErrors,
)

func mergeCatalogs(catalogs ...map[Language]map[string]string) map[Language]map[string]string {
	out := map[Language]map[string]string{}
	for _, c := range catalogs {
		for lang, kv := range c {
			if _, ok := out[lang]; !ok {
				out[lang] = map[string]string{}
			}
			for k, v := range kv {
				if _, exists := out[lang][k]; exists {
					panic(fmt.Sprintf("duplicate i18n key: lang=%s key=%s", lang, k))
				}
				out[lang][k] = v
			}
		}
	}
	return out
}

// SetLanguage sets the current language for translations.
func SetLanguage(lang string) {
	l := Language(lang)
	if _, ok := translations[l]; ok {
		currentLang = l
	} else {
		currentLang = En
	}
}

// GetCurrentLanguage returns the current language code.
func GetCurrentLanguage() string {
	return string(currentLang)
}

// T returns the translated string for the given key.
func T(key string, args ...interface{}) string {
	format, ok := translations[currentLang][key]
	if !ok {
		format, ok = translations[En][key]
		if !ok {
			return key
		}
	}
	return fmt.Sprintf(format, args...)
}

func GetSupportedLanguages() []string {
	return []string{string(En), string(ZhCN)}
}

func GetLanguageDisplay() map[string]string {
	return map[string]string{
		string(En):   "English",
		string(ZhCN): "简体中文",
	}
}

func DetectLanguage() string {
	lang := os.Getenv("CRATER_LANG")
	if lang != "" {
		return lang
	}
	lang = os.Getenv("LANG")
	if strings.HasPrefix(lang, "zh") {
		return string(ZhCN)
	}
	return string(En)
}

func PadRight(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
