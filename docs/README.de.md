# TurboKey

[![Release](https://img.shields.io/github/v/release/houko/turbokey?sort=semver&display_name=tag)](https://github.com/houko/turbokey/releases)
[![Build](https://github.com/houko/turbokey/actions/workflows/ci.yml/badge.svg)](https://github.com/houko/turbokey/actions/workflows/ci.yml)
[![Downloads](https://img.shields.io/github/downloads/houko/turbokey/total)](https://github.com/houko/turbokey/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/houko/turbokey)](https://github.com/houko/turbokey/blob/main/go.mod)
[![License: MIT](https://img.shields.io/github/license/houko/turbokey)](https://github.com/houko/turbokey/blob/main/LICENSE)
![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)

[English](../README.md) · [简体中文](README.zh.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · **Deutsch** · [Русский](README.ru.md) · [Português](README.pt.md) · [Italiano](README.it.md) · [العربية](README.ar.md) · [עברית](README.he.md)

Ein schlankes Windows-Werkzeug für automatisches Tasten-Dauerfeuer (rapid-fire / turbo) mit nativer GUI. Binde eine Taste so, dass sie sich selbst — oder eine andere Taste — automatisch wiederholt, entweder solange sie gehalten wird oder als Ein/Aus-Umschalter. Jede Regel hat ihr eigenes Intervall.

Funktioniert mit normalen Anwendungen und mit Spielen: Tasten werden als Hardware-**Scancodes** über `SendInput` eingespeist (damit DirectInput-Spiele sie akzeptieren), und jeder Anschlag wird kurz gehalten, damit abfragebasierte Spiele ihn zuverlässig erfassen.

> Nur Windows. Erstellt mit Go + [lxn/walk](https://github.com/lxn/walk) (native Win32-Steuerelemente), eine einzige eigenständige ausführbare Datei, keine Runtime zu installieren.

<p align="center"><img src="screenshot.png" alt="TurboKey" width="380"></p>

## Download

Hol dir die neueste `turbokey.exe` von der [Releases](../../releases)-Seite. Jeder Push auf `main` baut und veröffentlicht automatisch ein neues versioniertes Release über GitHub Actions.

## Funktionen

- Pro Regel: **Auslösetaste**, **Ausgabetaste** (standardmäßig die Auslösetaste), **Modus**, **Intervall** und aktivieren/deaktivieren.
- Tastaturtasten **oder Maustasten** (links / rechts / Mitte / X1 / X2) als Auslöser und Ausgabe.
- Zwei Modi pro Regel:
  - **Halten** — wiederholt, solange die Auslösetaste physisch gedrückt gehalten wird.
  - **Umschalten** — ein Druck startet die Wiederholung, der nächste stoppt sie.
- Globaler Hauptschalter mit Hotkey **F8**.
- Beschränke das Dauerfeuer optional auf bestimmte Apps (per Prozessname); leer = funktioniert überall.
- **Infobereich**: Das Schließen des Fensters minimiert es in den Infobereich (das Werkzeug läuft weiter). Linksklick auf das Symbol stellt es wieder her; Rechtsklick öffnet ein Menü zum Anzeigen des Fensters, Umschalten des Hauptschalters oder Beenden.
- Optionales **Mit Windows starten**, als geplante Aufgabe registriert, damit es bei der Anmeldung mit erhöhten Rechten ohne UAC-Abfrage startet.
- Regeln werden in `config.json` neben der ausführbaren Datei gespeichert und beim Start neu geladen.
- Lokalisierte UI in 13 Sprachen (inkl. Arabisch und Hebräisch von rechts nach links mit vollständig gespiegeltem Layout), automatisch vom Betriebssystem erkannt, mit Auswahl in der App.
- Das Werkzeug filtert seine eigene synthetische Eingabe heraus, sodass es sich nie selbst erneut auslöst.

## Funktionsweise

- Ein `WH_KEYBOARD_LL` Low-Level-Tastatur-Hook erkennt deine Tastenanschläge und schluckt die konfigurierte Auslösetaste, sodass das Spiel nur die sauberen wiederholten Impulse sieht.
- Jeder Impuls ist `Taste runter → ~30ms halten → Taste hoch`, gesendet mit `KEYEVENTF_SCANCODE`. Das Halten ist erforderlich: Ein abfragebasiertes Spiel erfasst den Tastenzustand pro Frame und würde ein Runter/Hoch-Paar verpassen, das zwischen zwei Abtastungen fällt.
- Synthetische Ereignisse werden über `dwExtraInfo` markiert, sodass der Hook seine eigenen eingespeisten Tasten erkennt und durchlässt, statt auf sie zu reagieren.

## Voraussetzungen

- Windows 10/11 (x64).
- **Administratorrechte.** Die ausführbare Datei fordert die Erhöhung automatisch an (UAC-Abfrage beim Start). Das ist nötig, weil Windows UIPI verhindert, dass eingespeiste Eingaben eines nicht erhöhten Prozesses erhöhte Fenster erreichen — und viele Spiele laufen erhöht.

## Bauen

Kein CGO erforderlich, daher Cross-Kompilierung nach Windows von Linux/WSL oder natives Bauen unter Windows.

```sh
# Linux / WSL (Cross-Kompilierung) oder Windows (Git Bash):
./build.sh
# -> turbokey.exe
```

Das Skript installiert `rsrc`, um das Anwendungsmanifest (Common Controls v6, DPI-Erkennung, requireAdministrator) einzubetten, und führt dann `go build` aus. Manuell bauen:

```sh
go install github.com/akavel/rsrc@latest
rsrc -manifest cmd/turbokey/app.manifest -ico cmd/turbokey/icon.ico -arch amd64 -o cmd/turbokey/rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-H windowsgui -s -w" -o turbokey.exe ./cmd/turbokey
```

## Projektaufbau

```
cmd/turbokey/      Einstiegspunkt (main) + Windows-App-Manifest + Symbol
cmd/gen-icon/      Symbolgenerator zur Bauzeit (rendert icon.ico / icon.png)
internal/keys/     Tasten- & Maustastentabelle; Mapping Name <-> virtueller Tastencode
internal/i18n/     UI-Lokalisierung; eingebettete JSON-Nachrichtenkataloge (locales/)
internal/config/   Regelmodell und Laden/Speichern von config.json
internal/winput/   Win32-Wrapper: Tastatur- & Maus-Hooks, SendInput, Vordergrund
internal/engine/   Dauerfeuer-Engine: Hook-Dispatch, Hauptschalter, Worker
internal/ui/       native Win32-GUI (lxn/walk)
internal/autostart/ Anmelde-Aufgabe (Mit Windows starten)
internal/buildinfo/ Build-Version, über -ldflags injiziert
```

## Verwendung

1. Starte `turbokey.exe` und bestätige die UAC-Abfrage.
2. Wähle im Editor eine **Auslösetaste**, eine **Ausgabetaste** (Standard = wie Auslöser), einen **Modus** und ein **Intervall (ms)**, dann klicke auf **Hinzufügen**.
3. Um eine Regel zu ändern, klicke sie an (ihre Werte werden in den Editor geladen), bearbeite und klicke auf **Aktualisieren**. Doppelklicke eine Zeile zum Aktivieren/Deaktivieren; **Löschen** entfernt sie.
4. Drücke **F8** (oder hake den Hauptschalter an) zum Aktivieren, dann drücke deine Auslösetaste zum Feuern. Drücke **F8** erneut, wenn du fertig bist.

Hinweise:
- Das Intervall ist der Abstand *zwischen* den Anschlägen; jeder Anschlag hält die Taste außerdem ~30ms, sodass die praktische Obergrenze bei rund 25-30 Anschlägen/Sekunde liegt. Das reicht für jede Spielfähigkeit — schneller drücken hilft nicht, weil das Spiel pro Frame abtastet.
- Solange der Hauptschalter an ist, werden konfigurierte Auslösetasten in **allen** Anwendungen abgefangen, nicht nur in Spielen. Schalte ihn (F8) aus, wenn du ihn nicht nutzt.
- F8 ist als Haupt-Hotkey reserviert, während das Werkzeug läuft.

## Konfiguration

`config.json` (neben der ausführbaren Datei erstellt) ist menschenlesbar:

```json
{
  "rules": [
    { "name": "attack", "trigger": "J", "output": "", "mode": "hold",   "interval": 10, "enabled": true },
    { "name": "skill",  "trigger": "K", "output": "", "mode": "toggle", "interval": 50, "enabled": false }
  ]
}
```

- `output: ""` bedeutet „wie die Auslösetaste“.
- `mode`: `"hold"` oder `"toggle"`.
- Tastennamen: `A`–`Z`, `0`–`9`, `F1`–`F12`, `Space`, `Enter`, `Esc`, `Tab`, `↑ ↓ ← →`, `Ctrl`, `Alt`, `Shift` sowie Maustasten `Mouse Left`, `Mouse Right`, `Mouse Mid`, `Mouse X1`, `Mouse X2`. (Das sind sprachneutrale Bezeichner, die nicht übersetzt werden, sodass die Konfiguration über UI-Sprachen hinweg gültig bleibt.)

## Auf bestimmte Apps beschränken

Standardmäßig ist das Dauerfeuer in jeder Anwendung aktiv. Um es einzuschränken, fülle das Feld „Aktive Apps“ mit einem oder mehreren Prozessnamen (z. B. `DNFGame.exe`, durch Kommas getrennt). Das Dauerfeuer greift dann nur, während eine dieser Apps im Vordergrund ist; anderswo verhalten sich deine Tasten normal.

Prozessname unbekannt? Klicke auf **App wählen…** und wähle sie aus der Liste der laufenden Programme (angezeigt nach Fenstertitel und ausführbarer Datei). Der F8-Haupt-Hotkey funktioniert immer, unabhängig von der aktiven App.

## Sprache

Wähle die Sprache im Dropdown oben rechts im Fenster. Die Wahl wird in `config.json` gespeichert und die App startet zur Anwendung neu; „Auto“ folgt der Windows-UI-Sprache.

Mitgelieferte Sprachen: English, 简体中文, 繁體中文, 日本語, 한국어, Español, Français, Deutsch, Русский, Português, Italiano, العربية, עברית.

Rechts-nach-links-Sprachen (Arabisch, Hebräisch) spiegeln das gesamte Fensterlayout über `WS_EX_LAYOUTRTL` (walks `RightToLeftLayout`).

Auflösungsreihenfolge: die Umgebungsvariable `TURBOKEY_LANG` (`zh`/`en`), dann die gespeicherte Wahl, dann die Betriebssystemsprache.

Nachrichtenkataloge sind einfaches JSON unter `internal/i18n/locales/`, eingebettet mit `go:embed`. Um eine Sprache hinzuzufügen, lege `internal/i18n/locales/<code>.json` ab (kopiere `en.json` und übersetze die Werte) und baue neu.

## Vorbehalte

- Ein globaler Tastatur-Hook plus Eingabe-Injektion können Fehlalarme von Antivirenprogrammen auslösen.
- Manche Online-Spiele verbieten Makros/Automatisierung in ihren Nutzungsbedingungen, und Anti-Cheat-Systeme können synthetische Eingaben erkennen oder blockieren. Verantwortungsvoll und auf eigenes Risiko verwenden.

## Lizenz

[MIT](../LICENSE)
