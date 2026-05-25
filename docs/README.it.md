# TurboKey

[English](../README.md) · [简体中文](README.zh.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md) · [Português](README.pt.md) · **Italiano** · [العربية](README.ar.md) · [עברית](README.he.md)

Uno strumento leggero per la pressione automatica dei tasti (rapid-fire / turbo) per Windows con GUI nativa. Associa un tasto affinché si ripeta da solo — o ripeta un altro tasto — automaticamente, sia mentre è tenuto premuto sia come interruttore on/off. Ogni regola ha il proprio intervallo.

Funziona con le applicazioni comuni e con i giochi: i tasti vengono iniettati come **scan code** hardware tramite `SendInput` (così i giochi DirectInput li accettano), e ogni pressione viene mantenuta brevemente affinché i giochi basati sul polling la campionino in modo affidabile.

> Solo Windows. Realizzato con Go + [lxn/walk](https://github.com/lxn/walk) (controlli Win32 nativi), un singolo eseguibile autonomo, senza runtime da installare.

<p align="center"><img src="screenshot.png" alt="TurboKey" width="380"></p>

## Download

Scarica l'ultimo `turbokey.exe` dalla pagina [Releases](../../releases). Ogni push su `main` compila e pubblica automaticamente una nuova versione tramite GitHub Actions.

## Caratteristiche

- Per regola: **tasto di attivazione**, **tasto di uscita** (predefinito: quello di attivazione), **modalità**, **intervallo** e abilita/disabilita.
- Tasti della tastiera **o pulsanti del mouse** (sinistro / destro / centrale / X1 / X2) come attivazione e uscita.
- Due modalità per regola:
  - **Tieni** — si ripete finché il tasto di attivazione è fisicamente tenuto premuto.
  - **Alterna** — una pressione avvia la ripetizione, la successiva la ferma.
- Interruttore principale globale con tasto di scelta rapida **F8**.
- Facoltativamente limita la pressione rapida ad app specifiche (per nome del processo); vuoto = funziona ovunque.
- **Area di notifica**: chiudere la finestra la riduce a icona nell'area di notifica (lo strumento continua a funzionare). Clic sinistro sull'icona per ripristinarla; clic destro per un menu che mostra la finestra, alterna l'interruttore principale o esce.
- **Avvia con Windows** opzionale, registrato come attività pianificata per avviarsi con privilegi elevati all'accesso senza richiesta UAC.
- Le regole vengono salvate in `config.json` accanto all'eseguibile e ricaricate all'avvio.
- UI localizzata in 13 lingue (incluse arabo ed ebraico da destra a sinistra con layout completamente speculare), rilevata automaticamente dal sistema operativo, con un selettore nell'app.
- Lo strumento filtra il proprio input sintetico, quindi non si riattiva mai da solo.

## Come funziona

- Un hook della tastiera di basso livello `WH_KEYBOARD_LL` rileva le tue pressioni e inghiotte il tasto di attivazione configurato, così il gioco vede solo gli impulsi ripetuti puliti.
- Ogni impulso è `tasto giù → mantieni ~30ms → tasto su`, inviato con `KEYEVENTF_SCANCODE`. Il mantenimento è necessario: un gioco basato sul polling campiona lo stato dei tasti per fotogramma e perderebbe una coppia giù/su che cade tra due campionamenti.
- Gli eventi sintetici sono etichettati tramite `dwExtraInfo`, così l'hook riconosce i propri tasti iniettati e li lascia passare invece di agire su di essi.

## Requisiti

- Windows 10/11 (x64).
- **Privilegi di amministratore.** L'eseguibile richiede l'elevazione automaticamente (richiesta UAC all'avvio). È necessario perché l'UIPI di Windows impedisce all'input iniettato da un processo non elevato di raggiungere le finestre elevate — e molti giochi vengono eseguiti con privilegi elevati.

## Compilazione

Non richiede CGO, quindi si può compilare in cross-compilazione verso Windows da Linux/WSL o nativamente su Windows.

```sh
# Linux / WSL (cross-compilazione) o Windows (Git Bash):
./build.sh
# -> turbokey.exe
```

Lo script installa `rsrc` per incorporare il manifesto dell'applicazione (Common Controls v6, riconoscimento DPI, requireAdministrator), poi esegue `go build`. Per compilare a mano:

```sh
go install github.com/akavel/rsrc@latest
rsrc -manifest cmd/turbokey/app.manifest -ico cmd/turbokey/icon.ico -arch amd64 -o cmd/turbokey/rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-H windowsgui -s -w" -o turbokey.exe ./cmd/turbokey
```

## Struttura del progetto

```
cmd/turbokey/      punto di ingresso (main) + manifesto dell'app Windows + icona
cmd/gen-icon/      generatore di icone in fase di compilazione (renderizza icon.ico / icon.png)
internal/keys/     tabella tasti e pulsanti del mouse; mappatura nome <-> codice tasto virtuale
internal/i18n/     localizzazione UI; cataloghi di messaggi JSON incorporati (locales/)
internal/config/   modello delle regole e caricamento/salvataggio di config.json
internal/winput/   wrapper Win32: hook tastiera e mouse, SendInput, primo piano
internal/engine/   motore di pressione rapida: dispatch degli hook, interruttore principale, worker
internal/ui/       GUI Win32 nativa (lxn/walk)
internal/autostart/ attività pianificata all'accesso (Avvia con Windows)
internal/buildinfo/ versione di build, iniettata tramite -ldflags
```

## Utilizzo

1. Avvia `turbokey.exe` e accetta la richiesta UAC.
2. Nell'editor, scegli un **tasto di attivazione**, un **tasto di uscita** (predefinito = uguale all'attivazione), una **modalità** e un **intervallo (ms)**, poi clicca su **Aggiungi**.
3. Per modificare una regola, cliccala (i suoi valori si caricano nell'editor), modifica e clicca su **Aggiorna**. Doppio clic su una riga per abilitarla/disabilitarla; **Elimina** la rimuove.
4. Premi **F8** (o spunta l'interruttore principale) per attivare, poi premi il tuo tasto di attivazione per sparare. Premi di nuovo **F8** quando hai finito.

Note:
- L'intervallo è lo spazio *tra* le pressioni; ogni pressione mantiene anche il tasto ~30ms, quindi il tetto pratico è di circa 25-30 pressioni/secondo. È più che sufficiente per qualsiasi abilità di gioco — premere più velocemente non aiuta, perché il gioco campiona per fotogramma.
- Mentre l'interruttore principale è acceso, i tasti di attivazione configurati vengono intercettati in **tutte** le applicazioni, non solo nei giochi. Spegnilo (F8) quando non lo usi.
- F8 è riservato come tasto di scelta rapida principale mentre lo strumento è in esecuzione.

## Configurazione

`config.json` (creato accanto all'eseguibile) è leggibile:

```json
{
  "rules": [
    { "name": "attack", "trigger": "J", "output": "", "mode": "hold",   "interval": 10, "enabled": true },
    { "name": "skill",  "trigger": "K", "output": "", "mode": "toggle", "interval": 50, "enabled": false }
  ]
}
```

- `output: ""` significa "uguale al tasto di attivazione".
- `mode`: `"hold"` o `"toggle"`.
- Nomi dei tasti: `A`–`Z`, `0`–`9`, `F1`–`F12`, `Space`, `Enter`, `Esc`, `Tab`, `↑ ↓ ← →`, `Ctrl`, `Alt`, `Shift`, e pulsanti del mouse `Mouse Left`, `Mouse Right`, `Mouse Mid`, `Mouse X1`, `Mouse X2`. (Sono identificatori neutri rispetto alla lingua e non vengono tradotti, quindi la configurazione resta valida tra le lingue dell'UI.)

## Limitare ad app specifiche

Per impostazione predefinita la pressione rapida è attiva in ogni applicazione. Per limitarla, compila il campo "App attive" con uno o più nomi di processo (es. `DNFGame.exe`, separati da virgole). La pressione rapida si attiva quindi solo mentre una di queste app è in primo piano; altrove i tuoi tasti si comportano normalmente.

Non conosci il nome del processo? Clicca su **Scegli app…** e selezionala dall'elenco dei programmi in esecuzione (mostrati per titolo della finestra ed eseguibile). Il tasto di scelta rapida principale F8 funziona sempre, indipendentemente dall'app attiva.

## Lingua

Scegli la lingua dal menu a discesa in alto a destra della finestra. La scelta viene salvata in `config.json` e l'app si riavvia per applicarla; "Auto" segue la lingua dell'UI di Windows.

Lingue incluse: English, 简体中文, 繁體中文, 日本語, 한국어, Español, Français, Deutsch, Русский, Português, Italiano, العربية, עברית.

Le lingue da destra a sinistra (arabo, ebraico) rispecchiano l'intero layout della finestra tramite `WS_EX_LAYOUTRTL` (il `RightToLeftLayout` di walk).

Ordine di risoluzione: la variabile d'ambiente `TURBOKEY_LANG` (`zh`/`en`), poi la scelta salvata, poi la lingua del sistema operativo.

I cataloghi dei messaggi sono semplici JSON in `internal/i18n/locales/`, incorporati con `go:embed`. Per aggiungere una lingua, inserisci `internal/i18n/locales/<code>.json` (copia `en.json` e traduci i valori) e ricompila.

## Avvertenze

- Un hook globale della tastiera più l'iniezione di input possono attivare falsi positivi degli antivirus.
- Alcuni giochi online vietano le macro/l'automazione nei loro Termini di Servizio, e i sistemi anti-cheat possono rilevare o bloccare l'input sintetico. Usa in modo responsabile e a tuo rischio.

## Licenza

[MIT](../LICENSE)
