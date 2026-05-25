# TurboKey

A lightweight Windows key auto-fire (rapid-fire / turbo) tool with a native GUI.
Bind a key to repeat itself — or another key — automatically, either while held
or as an on/off toggle. Each binding has its own interval.

It works with normal applications and with games: keys are injected as hardware
**scan codes** via `SendInput` (so DirectInput games accept them), and each press
is held briefly so poll-based games reliably sample it.

> Windows only. Built with Go + [lxn/walk](https://github.com/lxn/walk) (native
> Win32 controls), single self-contained executable, no runtime to install.

## Features

- Per-rule **trigger key**, **output key** (defaults to the trigger), **mode**,
  **interval**, and enable/disable.
- Two modes per rule:
  - **Hold** — repeats while the trigger key is physically held down.
  - **Toggle** — one press starts repeating, the next press stops.
- Global master switch with an **F8** hotkey.
- Optionally limit rapid-fire to specific apps (matched by process name); empty
  means it works everywhere.
- **System tray**: closing the window minimizes to the tray (the tool keeps
  running). Left-click the tray icon to restore it; right-click for a menu to
  show the window, toggle the master switch, or quit.
- Rules are saved to `config.json` next to the executable and reloaded on start.
- Localized UI in 13 languages (incl. right-to-left Arabic & Hebrew with a fully
  mirrored layout), auto-detected from the OS, with an in-app picker.
- The tool filters out its own synthetic input, so it never re-triggers itself.

## How it works

- A `WH_KEYBOARD_LL` low-level keyboard hook detects your key presses and swallows
  the configured trigger key, so the game only sees the clean repeated pulses.
- Each pulse is `key-down → hold ~30ms → key-up`, sent with `KEYEVENTF_SCANCODE`.
  The hold is required: a poll-based game samples key state per frame and would
  miss a down/up pair that lands between two samples.
- Synthetic events are tagged via `dwExtraInfo`, so the hook recognizes and passes
  its own injected keys instead of acting on them.

## Requirements

- Windows 10/11 (x64).
- **Administrator privileges.** The executable requests elevation automatically
  (UAC prompt on launch). This is required because Windows UIPI blocks injected
  input from a non-elevated process from reaching elevated windows — and many
  games run elevated.

## Build

No CGO is required, so it cross-compiles to Windows from Linux/WSL or builds
natively on Windows.

```sh
# Linux / WSL (cross-compile) or Windows (Git Bash):
./build.sh
# -> turbokey.exe
```

The script installs `rsrc` to embed the application manifest (Common Controls v6,
DPI awareness, requireAdministrator), then runs `go build`. To build by hand:

```sh
go install github.com/akavel/rsrc@latest
rsrc -manifest cmd/turbokey/app.manifest -arch amd64 -o cmd/turbokey/rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-H windowsgui -s -w" -o turbokey.exe ./cmd/turbokey
```

## Project layout

```
cmd/turbokey/      entry point (main) + Windows application manifest
internal/keys/     key table; display-name <-> virtual-key-code mapping
internal/i18n/     UI localization; embedded JSON message catalogs (locales/)
internal/config/   rule model and config.json load/save
internal/winput/   Win32 wrappers: low-level keyboard hook + SendInput
internal/engine/   rapid-fire engine: hook callback, master switch, workers
internal/ui/       native Win32 GUI (lxn/walk)
```

## Usage

1. Launch `turbokey.exe` and accept the UAC prompt.
2. In the bottom row, pick **trigger key**, **output key** (`(同触发键)` = same as
   trigger), **mode**, and **interval (ms)**, then click **添加规则** (Add rule).
3. Press **F8** (or tick the master switch) to enable. Double-click a row to
   enable/disable it.
4. Press your trigger key to fire. Press **F8** again when you are done.

Notes:
- The interval is the gap *between* presses; each press also holds the key ~30ms,
  so the practical ceiling is roughly 25–30 presses/second. That is plenty for any
  game skill — pressing faster does not help, because the game samples per frame.
- While the master switch is on, configured trigger keys are intercepted in **every**
  application, not just games. Turn it off (F8) when you are not using it.
- F8 is reserved as the master hotkey while the tool runs.

## Configuration

`config.json` (created next to the executable) is human-readable:

```json
{
  "rules": [
    { "name": "attack", "trigger": "J", "output": "", "mode": "hold",   "interval": 10, "enabled": true },
    { "name": "skill",  "trigger": "K", "output": "", "mode": "toggle", "interval": 50, "enabled": false }
  ]
}
```

- `output: ""` means "same as the trigger key".
- `mode`: `"hold"` or `"toggle"`.
- Key names: `A`–`Z`, `0`–`9`, `F1`–`F12`, `Space`, `Enter`, `Esc`, `Tab`,
  `↑ ↓ ← →`, `Ctrl`, `Alt`, `Shift`. (These are language-neutral identifiers and
  are not translated, so the config stays valid across UI languages.)

## Limiting to specific apps

By default rapid-fire is active in every application. To restrict it, fill the
"Active apps" field with one or more process names (e.g. `DNFGame.exe`, comma-
separated). Rapid-fire then only engages while one of those apps is in the
foreground; elsewhere your keys behave normally.

Don't know the process name? Click **Choose app…** and pick it from the list of
currently running programs (shown by window title and executable). The master F8
hotkey always works regardless of the active app.

## Language

Pick the language from the dropdown in the top-right of the window. The choice is
saved to `config.json` and the app relaunches to apply it; "Auto" follows the
Windows UI language.

Bundled languages: English, 简体中文, 繁體中文, 日本語, 한국어, Español, Français,
Deutsch, Русский, Português, Italiano, العربية, עברית.

Right-to-left languages (Arabic, Hebrew) mirror the entire window layout via
`WS_EX_LAYOUTRTL` (walk's `RightToLeftLayout`).

Resolution order: the `TURBOKEY_LANG` environment variable (`zh`/`en`), then the
saved choice, then the OS language.

Message catalogs are plain JSON under `internal/i18n/locales/`, embedded with
`go:embed`. To add a language, drop in `internal/i18n/locales/<code>.json` (copy
`en.json` and translate the values) and rebuild.

## Caveats

- A global keyboard hook plus input injection may trigger antivirus false positives.
- Some online games forbid macros/automation in their Terms of Service, and
  anti-cheat systems may detect or block synthetic input. Use responsibly and at
  your own risk.

## License

[MIT](LICENSE)
