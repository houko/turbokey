//go:build windows

// Package i18n provides minimal UI string localization. Message catalogs live as
// JSON files under locales/ and are embedded into the binary. The active language
// is chosen from the TURBOKEY_LANG environment variable, falling back to the
// Windows UI language (Chinese -> zh, everything else -> en).
package i18n

import (
	"embed"
	"encoding/json"
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

type Lang string

const (
	EN Lang = "en"
	ZH Lang = "zh"
)

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
// Windows UI language. An empty/unknown value falls through to the next source.
// Call once at startup before building the UI.
func Init(pref string) {
	for _, code := range []string{os.Getenv("TURBOKEY_LANG"), pref} {
		switch code {
		case "zh":
			current = ZH
			return
		case "en":
			current = EN
			return
		}
	}
	current = detectOS()
}

// Current returns the active language.
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

var (
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procGetUserDefaultUILanguage = kernel32.NewProc("GetUserDefaultUILanguage")
)

// detectOS maps the Windows UI language to a supported Lang.
func detectOS() Lang {
	r, _, _ := procGetUserDefaultUILanguage.Call()
	const langChinese = 0x04 // primary language id, covers zh-CN / zh-TW / ...
	if uint16(r)&0x3ff == langChinese {
		return ZH
	}
	return EN
}
