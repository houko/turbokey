//go:build windows

// Package i18n provides minimal UI string localization. Message catalogs live as
// JSON files under locales/ and are embedded into the binary. The active language
// is chosen from the TURBOKEY_LANG environment variable, then a caller-supplied
// preference (the saved config), then the Windows UI language.
package i18n

import (
	"embed"
	"encoding/json"
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

type Lang string

// EN is the fallback language used when a key is missing in the active catalog.
const EN Lang = "en"

// langEntry pairs a language code with its native display name. The empty code is
// the "Auto" (follow-OS) option; its label is resolved via T("lang.auto").
type langEntry struct {
	code string
	name string
}

// langs is the ordered list of selectable languages shown in the picker.
var langs = []langEntry{
	{"", ""}, // Auto
	{"en", "English"},
	{"zh", "简体中文"},
	{"zh-Hant", "繁體中文"},
	{"ja", "日本語"},
	{"ko", "한국어"},
	{"es", "Español"},
	{"fr", "Français"},
	{"de", "Deutsch"},
	{"ru", "Русский"},
	{"pt", "Português"},
	{"it", "Italiano"},
}

//go:embed locales/*.json
var localesFS embed.FS

// dict[lang][key] -> message, loaded from the embedded JSON catalogs.
var dict = map[Lang]map[string]string{}

var current = EN

func init() {
	entries, err := localesFS.ReadDir("locales")
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name() // e.g. "en.json"
		data, err := localesFS.ReadFile("locales/" + name)
		if err != nil {
			continue
		}
		m := map[string]string{}
		if json.Unmarshal(data, &m) == nil {
			dict[Lang(strings.TrimSuffix(name, ".json"))] = m
		}
	}
}

// Init selects the active language, in priority order: the TURBOKEY_LANG
// environment variable, then pref (e.g. the saved config language), then the
// Windows UI language. An empty value falls through to the next source.
// Call once at startup before building the UI.
func Init(pref string) {
	for _, code := range []string{os.Getenv("TURBOKEY_LANG"), pref} {
		if code != "" {
			current = Lang(code)
			return
		}
	}
	current = detectOS()
}

// Current returns the active language code.
func Current() Lang { return current }

// T returns the message for key in the active language, falling back to English,
// then to the key itself.
func T(key string) string {
	if m, ok := dict[current]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if s, ok := dict[EN][key]; ok {
		return s
	}
	return key
}

// DisplayNames returns the picker labels (index 0 is the localized "Auto").
func DisplayNames() []string {
	out := make([]string, len(langs))
	for i, l := range langs {
		if l.code == "" {
			out[i] = T("lang.auto")
		} else {
			out[i] = l.name
		}
	}
	return out
}

// IndexOf returns the picker index for a language code ("" -> Auto at 0).
func IndexOf(code string) int {
	for i, l := range langs {
		if l.code == code {
			return i
		}
	}
	return 0
}

// CodeAt returns the language code at a picker index.
func CodeAt(i int) string {
	if i >= 0 && i < len(langs) {
		return langs[i].code
	}
	return ""
}

var (
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procGetUserDefaultUILanguage = kernel32.NewProc("GetUserDefaultUILanguage")
)

// detectOS maps the Windows UI language to a supported language code.
func detectOS() Lang {
	r, _, _ := procGetUserDefaultUILanguage.Call()
	id := uint16(r)
	primary := id & 0x3ff
	sub := id >> 10
	switch primary {
	case 0x04: // Chinese
		if sub == 1 || sub == 3 || sub == 5 { // TW / HK / MO -> Traditional
			return "zh-Hant"
		}
		return "zh"
	case 0x11:
		return "ja"
	case 0x12:
		return "ko"
	case 0x0a:
		return "es"
	case 0x0c:
		return "fr"
	case 0x07:
		return "de"
	case 0x19:
		return "ru"
	case 0x16:
		return "pt"
	case 0x10:
		return "it"
	default:
		return EN
	}
}
