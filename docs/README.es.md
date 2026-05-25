# TurboKey

[![Release](https://img.shields.io/github/v/release/houko/turbokey?sort=semver&display_name=tag)](https://github.com/houko/turbokey/releases)
[![Build](https://github.com/houko/turbokey/actions/workflows/ci.yml/badge.svg)](https://github.com/houko/turbokey/actions/workflows/ci.yml)
[![Downloads](https://img.shields.io/github/downloads/houko/turbokey/total)](https://github.com/houko/turbokey/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/houko/turbokey)](https://github.com/houko/turbokey/blob/main/go.mod)
[![License: MIT](https://img.shields.io/github/license/houko/turbokey)](https://github.com/houko/turbokey/blob/main/LICENSE)
![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)

[English](../README.md) · [简体中文](README.zh.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · **Español** · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md) · [Português](README.pt.md) · [Italiano](README.it.md) · [العربية](README.ar.md) · [עברית](README.he.md)

Una herramienta ligera de pulsación automática de teclas (rapid-fire / turbo) para Windows con GUI nativa. Vincula una tecla para que se repita sola — o repita otra tecla — automáticamente, ya sea mientras se mantiene pulsada o como un interruptor on/off. Cada regla tiene su propio intervalo.

Funciona con aplicaciones normales y con juegos: las teclas se inyectan como **scan codes** de hardware mediante `SendInput` (para que los juegos DirectInput las acepten), y cada pulsación se mantiene brevemente para que los juegos basados en sondeo la muestreen de forma fiable.

> Solo Windows. Hecho con Go + [lxn/walk](https://github.com/lxn/walk) (controles Win32 nativos), un único ejecutable autónomo, sin runtime que instalar.

<p align="center"><img src="screenshot.png" alt="TurboKey" width="380"></p>

## Descarga

Consigue el último `turbokey.exe` en la página de [Releases](../../releases). Cada push a `main` compila y publica automáticamente una nueva versión mediante GitHub Actions.

## Características

- Por regla: **tecla de activación**, **tecla de salida** (por defecto, la de activación), **modo**, **intervalo** y activar/desactivar.
- Teclas del teclado **o botones del ratón** (izquierdo / derecho / central / X1 / X2) como activación y salida.
- Dos modos por regla:
  - **Mantener** — se repite mientras la tecla de activación está físicamente pulsada.
  - **Alternar** — una pulsación inicia la repetición, la siguiente la detiene.
- Interruptor maestro global con tecla rápida **F8**.
- Opcionalmente limita la pulsación rápida a apps concretas (por nombre de proceso); vacío significa que funciona en todas partes.
- **Bandeja del sistema**: cerrar la ventana la minimiza a la bandeja (la herramienta sigue funcionando). Clic izquierdo en el icono para restaurarla; clic derecho para un menú que muestra la ventana, alterna el interruptor maestro o cierra.
- **Iniciar con Windows** opcional, registrado como tarea programada para que arranque con privilegios elevados al iniciar sesión sin pedir UAC.
- Las reglas se guardan en `config.json` junto al ejecutable y se recargan al arrancar.
- UI localizada en 13 idiomas (incluidos el árabe y el hebreo de derecha a izquierda con un diseño totalmente reflejado), detectada automáticamente del SO, con un selector en la app.
- La herramienta filtra su propia entrada sintética, así que nunca se reactiva a sí misma.

## Cómo funciona

- Un hook de teclado de bajo nivel `WH_KEYBOARD_LL` detecta tus pulsaciones y se traga la tecla de activación configurada, de modo que el juego solo ve los pulsos repetidos limpios.
- Cada pulso es `tecla abajo → mantener ~30ms → tecla arriba`, enviado con `KEYEVENTF_SCANCODE`. El mantenimiento es necesario: un juego basado en sondeo muestrea el estado de la tecla por fotograma y se perdería un par abajo/arriba que cayera entre dos muestras.
- Los eventos sintéticos se etiquetan mediante `dwExtraInfo`, así que el hook reconoce sus propias teclas inyectadas y las deja pasar en vez de actuar sobre ellas.

## Requisitos

- Windows 10/11 (x64).
- **Privilegios de administrador.** El ejecutable solicita la elevación automáticamente (UAC al iniciar). Es necesario porque UIPI de Windows impide que la entrada inyectada por un proceso sin elevar llegue a ventanas elevadas — y muchos juegos se ejecutan elevados.

## Compilación

No requiere CGO, así que compila de forma cruzada a Windows desde Linux/WSL o se compila nativamente en Windows.

```sh
# Linux / WSL (compilación cruzada) o Windows (Git Bash):
./build.sh
# -> turbokey.exe
```

El script instala `rsrc` para incrustar el manifiesto de la aplicación (Common Controls v6, reconocimiento de DPI, requireAdministrator) y luego ejecuta `go build`. Para compilar a mano:

```sh
go install github.com/akavel/rsrc@latest
rsrc -manifest cmd/turbokey/app.manifest -ico cmd/turbokey/icon.ico -arch amd64 -o cmd/turbokey/rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-H windowsgui -s -w" -o turbokey.exe ./cmd/turbokey
```

## Estructura del proyecto

```
cmd/turbokey/      punto de entrada (main) + manifiesto de la app Windows + icono
cmd/gen-icon/      generador de iconos en tiempo de compilación (icon.ico / icon.png)
internal/keys/     tabla de teclas y botones de ratón; mapeo nombre <-> código de tecla virtual
internal/i18n/     localización de la UI; catálogos JSON incrustados (locales/)
internal/config/   modelo de reglas y carga/guardado de config.json
internal/winput/   wrappers Win32: hooks de teclado y ratón, SendInput, primer plano
internal/engine/   motor de pulsación rápida: despacho de hooks, interruptor maestro, workers
internal/ui/       GUI Win32 nativa (lxn/walk)
internal/autostart/ tarea programada al iniciar sesión (Iniciar con Windows)
internal/buildinfo/ versión de compilación, inyectada vía -ldflags
```

## Uso

1. Inicia `turbokey.exe` y acepta el aviso de UAC.
2. En el editor, elige una **tecla de activación**, una **tecla de salida** (por defecto = igual que la activación), un **modo** y un **intervalo (ms)**, luego pulsa **Añadir**.
3. Para cambiar una regla, haz clic en ella (sus valores se cargan en el editor), edita y pulsa **Actualizar**. Doble clic en una fila para activarla/desactivarla; **Eliminar** la borra.
4. Pulsa **F8** (o marca el interruptor maestro) para activar, luego pulsa tu tecla de activación para disparar. Pulsa **F8** de nuevo al terminar.

Notas:
- El intervalo es el hueco *entre* pulsaciones; cada pulsación también mantiene la tecla ~30ms, así que el techo práctico es de unas 25-30 pulsaciones/segundo. Es de sobra para cualquier habilidad de juego — pulsar más rápido no ayuda, porque el juego muestrea por fotograma.
- Mientras el interruptor maestro está activado, las teclas de activación configuradas se interceptan en **todas** las aplicaciones, no solo en juegos. Desactívalo (F8) cuando no lo uses.
- F8 queda reservada como tecla rápida maestra mientras la herramienta se ejecuta.

## Configuración

`config.json` (creado junto al ejecutable) es legible:

```json
{
  "rules": [
    { "name": "attack", "trigger": "J", "output": "", "mode": "hold",   "interval": 10, "enabled": true },
    { "name": "skill",  "trigger": "K", "output": "", "mode": "toggle", "interval": 50, "enabled": false }
  ]
}
```

- `output: ""` significa "igual que la tecla de activación".
- `mode`: `"hold"` o `"toggle"`.
- Nombres de teclas: `A`–`Z`, `0`–`9`, `F1`–`F12`, `Space`, `Enter`, `Esc`, `Tab`, `↑ ↓ ← →`, `Ctrl`, `Alt`, `Shift` y botones del ratón `Mouse Left`, `Mouse Right`, `Mouse Mid`, `Mouse X1`, `Mouse X2`. (Son identificadores neutros respecto al idioma y no se traducen, así que la configuración sigue siendo válida entre idiomas de la UI.)

## Limitar a apps concretas

Por defecto la pulsación rápida está activa en todas las aplicaciones. Para restringirla, rellena el campo "Apps activas" con uno o más nombres de proceso (p. ej. `DNFGame.exe`, separados por comas). La pulsación rápida solo se activará entonces mientras una de esas apps esté en primer plano; en otros sitios tus teclas se comportan con normalidad.

¿No sabes el nombre del proceso? Pulsa **Elegir app…** y selecciónala de la lista de programas en ejecución (mostrados por título de ventana y ejecutable). La tecla rápida maestra F8 siempre funciona, independientemente de la app activa.

## Idioma

Elige el idioma en el desplegable de la esquina superior derecha de la ventana. La elección se guarda en `config.json` y la app se reinicia para aplicarla; "Auto" sigue el idioma de la UI de Windows.

Idiomas incluidos: English, 简体中文, 繁體中文, 日本語, 한국어, Español, Français, Deutsch, Русский, Português, Italiano, العربية, עברית.

Los idiomas de derecha a izquierda (árabe, hebreo) reflejan todo el diseño de la ventana mediante `WS_EX_LAYOUTRTL` (el `RightToLeftLayout` de walk).

Orden de resolución: la variable de entorno `TURBOKEY_LANG` (`zh`/`en`), luego la elección guardada, luego el idioma del SO.

Los catálogos de mensajes son JSON plano bajo `internal/i18n/locales/`, incrustados con `go:embed`. Para añadir un idioma, coloca `internal/i18n/locales/<code>.json` (copia `en.json` y traduce los valores) y recompila.

## Advertencias

- Un hook de teclado global más la inyección de entrada pueden provocar falsos positivos del antivirus.
- Algunos juegos en línea prohíben las macros/automatización en sus Términos de Servicio, y los sistemas anti-cheat pueden detectar o bloquear la entrada sintética. Úsalo de forma responsable y bajo tu propia responsabilidad.

## Licencia

[MIT](../LICENSE)
