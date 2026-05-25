# TurboKey

[![Release](https://img.shields.io/github/v/release/houko/turbokey?sort=semver&display_name=tag)](https://github.com/houko/turbokey/releases)
[![Build](https://github.com/houko/turbokey/actions/workflows/ci.yml/badge.svg)](https://github.com/houko/turbokey/actions/workflows/ci.yml)
[![Downloads](https://img.shields.io/github/downloads/houko/turbokey/total)](https://github.com/houko/turbokey/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/houko/turbokey)](https://github.com/houko/turbokey/blob/main/go.mod)
[![License: MIT](https://img.shields.io/github/license/houko/turbokey)](https://github.com/houko/turbokey/blob/main/LICENSE)
![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)

[English](../README.md) · [简体中文](README.zh.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md) · **Português** · [Italiano](README.it.md) · [العربية](README.ar.md) · [עברית](README.he.md)

Uma ferramenta leve de disparo automático de teclas (rapid-fire / turbo) para Windows com GUI nativa. Vincule uma tecla para repetir a si mesma — ou outra tecla — automaticamente, seja enquanto pressionada ou como um interruptor liga/desliga. Cada regra tem o seu próprio intervalo.

Funciona com aplicativos comuns e com jogos: as teclas são injetadas como **scan codes** de hardware via `SendInput` (para que jogos DirectInput as aceitem), e cada pressão é mantida brevemente para que jogos baseados em polling a amostrem de forma confiável.

> Apenas Windows. Feito com Go + [lxn/walk](https://github.com/lxn/walk) (controles Win32 nativos), um único executável autônomo, sem runtime para instalar.

<p align="center"><img src="screenshot.png" alt="TurboKey" width="380"></p>

## Download

Pegue o `turbokey.exe` mais recente na página de [Releases](../../releases). Cada push para `main` compila e publica automaticamente uma nova versão via GitHub Actions.

## Recursos

- Por regra: **tecla de disparo**, **tecla de saída** (padrão: a de disparo), **modo**, **intervalo** e ativar/desativar.
- Teclas do teclado **ou botões do mouse** (esquerdo / direito / meio / X1 / X2) como disparo e saída.
- Dois modos por regra:
  - **Segurar** — repete enquanto a tecla de disparo está fisicamente pressionada.
  - **Alternar** — uma pressão inicia a repetição, a próxima para.
- Interruptor mestre global com tecla de atalho **F8**.
- Opcionalmente limite o disparo rápido a apps específicos (por nome de processo); vazio = funciona em todo lugar.
- **Bandeja do sistema**: fechar a janela a minimiza para a bandeja (a ferramenta continua rodando). Clique esquerdo no ícone para restaurá-la; clique direito para um menu que mostra a janela, alterna o interruptor mestre ou sai.
- **Iniciar com o Windows** opcional, registrado como tarefa agendada para iniciar com privilégios elevados no logon sem pedir UAC.
- As regras são salvas em `config.json` ao lado do executável e recarregadas ao iniciar.
- UI localizada em 13 idiomas (incluindo árabe e hebraico da direita para a esquerda com layout totalmente espelhado), detectada automaticamente do SO, com um seletor no app.
- A ferramenta filtra sua própria entrada sintética, então nunca se redispara.

## Como funciona

- Um hook de teclado de baixo nível `WH_KEYBOARD_LL` detecta suas teclas e engole a tecla de disparo configurada, de modo que o jogo só vê os pulsos repetidos limpos.
- Cada pulso é `tecla pressionada → segurar ~30ms → tecla solta`, enviado com `KEYEVENTF_SCANCODE`. A retenção é necessária: um jogo baseado em polling amostra o estado da tecla por quadro e perderia um par pressionar/soltar que cair entre duas amostras.
- Os eventos sintéticos são marcados via `dwExtraInfo`, então o hook reconhece suas próprias teclas injetadas e as deixa passar em vez de agir sobre elas.

## Requisitos

- Windows 10/11 (x64).
- **Privilégios de administrador.** O executável solicita a elevação automaticamente (prompt UAC ao iniciar). Isso é necessário porque o UIPI do Windows impede que a entrada injetada por um processo não elevado alcance janelas elevadas — e muitos jogos rodam elevados.

## Compilação

Não requer CGO, então compila de forma cruzada para Windows a partir de Linux/WSL ou compila nativamente no Windows.

```sh
# Linux / WSL (compilação cruzada) ou Windows (Git Bash):
./build.sh
# -> turbokey.exe
```

O script instala o `rsrc` para embutir o manifesto da aplicação (Common Controls v6, reconhecimento de DPI, requireAdministrator) e então executa `go build`. Para compilar manualmente:

```sh
go install github.com/akavel/rsrc@latest
rsrc -manifest cmd/turbokey/app.manifest -ico cmd/turbokey/icon.ico -arch amd64 -o cmd/turbokey/rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-H windowsgui -s -w" -o turbokey.exe ./cmd/turbokey
```

## Estrutura do projeto

```
cmd/turbokey/      ponto de entrada (main) + manifesto da app Windows + ícone
cmd/gen-icon/      gerador de ícones em tempo de compilação (renderiza icon.ico / icon.png)
internal/keys/     tabela de teclas e botões do mouse; mapeamento nome <-> código de tecla virtual
internal/i18n/     localização da UI; catálogos JSON embutidos (locales/)
internal/config/   modelo de regras e carregamento/salvamento de config.json
internal/winput/   wrappers Win32: hooks de teclado e mouse, SendInput, primeiro plano
internal/engine/   motor de disparo rápido: despacho de hooks, interruptor mestre, workers
internal/ui/       GUI Win32 nativa (lxn/walk)
internal/autostart/ tarefa agendada no logon (Iniciar com o Windows)
internal/buildinfo/ versão de build, injetada via -ldflags
```

## Uso

1. Inicie o `turbokey.exe` e aceite o prompt UAC.
2. No editor, escolha uma **tecla de disparo**, uma **tecla de saída** (padrão = igual ao disparo), um **modo** e um **intervalo (ms)**, depois clique em **Adicionar**.
3. Para alterar uma regra, clique nela (seus valores carregam no editor), edite e clique em **Atualizar**. Duplo clique numa linha para ativá-la/desativá-la; **Excluir** a remove.
4. Pressione **F8** (ou marque o interruptor mestre) para ativar, depois pressione sua tecla de disparo para acionar. Pressione **F8** de novo quando terminar.

Notas:
- O intervalo é o espaço *entre* as pressões; cada pressão também segura a tecla ~30ms, então o teto prático é cerca de 25-30 pressões/segundo. É de sobra para qualquer habilidade de jogo — pressionar mais rápido não ajuda, porque o jogo amostra por quadro.
- Enquanto o interruptor mestre está ligado, as teclas de disparo configuradas são interceptadas em **todos** os aplicativos, não só em jogos. Desligue-o (F8) quando não estiver usando.
- F8 fica reservada como tecla de atalho mestre enquanto a ferramenta roda.

## Configuração

`config.json` (criado ao lado do executável) é legível:

```json
{
  "rules": [
    { "name": "attack", "trigger": "J", "output": "", "mode": "hold",   "interval": 10, "enabled": true },
    { "name": "skill",  "trigger": "K", "output": "", "mode": "toggle", "interval": 50, "enabled": false }
  ]
}
```

- `output: ""` significa "igual à tecla de disparo".
- `mode`: `"hold"` ou `"toggle"`.
- Nomes de teclas: `A`–`Z`, `0`–`9`, `F1`–`F12`, `Space`, `Enter`, `Esc`, `Tab`, `↑ ↓ ← →`, `Ctrl`, `Alt`, `Shift`, e botões do mouse `Mouse Left`, `Mouse Right`, `Mouse Mid`, `Mouse X1`, `Mouse X2`. (São identificadores neutros quanto ao idioma e não são traduzidos, então a configuração permanece válida entre idiomas da UI.)

## Limitar a apps específicos

Por padrão, o disparo rápido fica ativo em todos os aplicativos. Para restringi-lo, preencha o campo "Apps ativos" com um ou mais nomes de processo (ex.: `DNFGame.exe`, separados por vírgula). O disparo rápido então só engata enquanto um desses apps estiver em primeiro plano; em outros lugares suas teclas se comportam normalmente.

Não sabe o nome do processo? Clique em **Escolher app…** e selecione-o da lista de programas em execução (mostrados por título da janela e executável). A tecla de atalho mestre F8 sempre funciona, independentemente do app ativo.

## Idioma

Escolha o idioma no menu suspenso no canto superior direito da janela. A escolha é salva em `config.json` e o app reinicia para aplicá-la; "Auto" segue o idioma da UI do Windows.

Idiomas incluídos: English, 简体中文, 繁體中文, 日本語, 한국어, Español, Français, Deutsch, Русский, Português, Italiano, العربية, עברית.

Idiomas da direita para a esquerda (árabe, hebraico) espelham todo o layout da janela via `WS_EX_LAYOUTRTL` (o `RightToLeftLayout` do walk).

Ordem de resolução: a variável de ambiente `TURBOKEY_LANG` (`zh`/`en`), depois a escolha salva, depois o idioma do SO.

Os catálogos de mensagens são JSON simples sob `internal/i18n/locales/`, embutidos com `go:embed`. Para adicionar um idioma, coloque `internal/i18n/locales/<code>.json` (copie `en.json` e traduza os valores) e recompile.

## Ressalvas

- Um hook de teclado global mais a injeção de entrada podem disparar falsos positivos de antivírus.
- Alguns jogos online proíbem macros/automação em seus Termos de Serviço, e sistemas anti-cheat podem detectar ou bloquear entrada sintética. Use com responsabilidade e por sua conta e risco.

## Licença

[MIT](../LICENSE)
