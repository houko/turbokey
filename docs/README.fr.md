# TurboKey

[![Release](https://img.shields.io/github/v/release/houko/turbokey?sort=semver&display_name=tag)](https://github.com/houko/turbokey/releases)
[![Build](https://github.com/houko/turbokey/actions/workflows/release.yml/badge.svg)](https://github.com/houko/turbokey/actions/workflows/release.yml)
[![Downloads](https://img.shields.io/github/downloads/houko/turbokey/total)](https://github.com/houko/turbokey/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/houko/turbokey)](https://github.com/houko/turbokey/blob/main/go.mod)
[![License: MIT](https://img.shields.io/github/license/houko/turbokey)](https://github.com/houko/turbokey/blob/main/LICENSE)
![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)

[English](../README.md) · [简体中文](README.zh.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · **Français** · [Deutsch](README.de.md) · [Русский](README.ru.md) · [Português](README.pt.md) · [Italiano](README.it.md) · [العربية](README.ar.md) · [עברית](README.he.md)

Un outil léger de frappe automatique (rapid-fire / turbo) pour Windows, avec une interface native. Associez une touche à sa propre répétition — ou à celle d'une autre touche — automatiquement, soit pendant qu'elle est maintenue, soit en bascule on/off. Chaque règle a son propre intervalle.

Fonctionne avec les applications classiques et avec les jeux : les touches sont injectées en tant que **scan codes** matériels via `SendInput` (pour que les jeux DirectInput les acceptent), et chaque appui est maintenu brièvement pour que les jeux basés sur le polling l'échantillonnent de façon fiable.

> Windows uniquement. Réalisé avec Go + [lxn/walk](https://github.com/lxn/walk) (contrôles Win32 natifs), un seul exécutable autonome, sans runtime à installer.

<p align="center"><img src="screenshot.png" alt="TurboKey" width="380"></p>

## Téléchargement

Récupérez le dernier `turbokey.exe` sur la page [Releases](../../releases). Chaque push sur `main` compile et publie automatiquement une nouvelle version via GitHub Actions.

## Fonctionnalités

- Par règle : **touche de déclenchement**, **touche de sortie** (par défaut, celle de déclenchement), **mode**, **intervalle** et activer/désactiver.
- Touches du clavier **ou boutons de la souris** (gauche / droit / milieu / X1 / X2) en déclenchement et en sortie.
- Deux modes par règle :
  - **Maintenir** — se répète tant que la touche de déclenchement est physiquement maintenue.
  - **Bascule** — un appui lance la répétition, l'appui suivant l'arrête.
- Interrupteur principal global avec raccourci **F8**.
- Limitez éventuellement la frappe rapide à des apps précises (par nom de processus) ; vide = fonctionne partout.
- **Barre d'état système** : fermer la fenêtre la réduit dans la barre (l'outil continue de tourner). Clic gauche sur l'icône pour la restaurer ; clic droit pour un menu permettant d'afficher la fenêtre, basculer l'interrupteur principal ou quitter.
- **Démarrer avec Windows** en option, enregistré comme tâche planifiée pour se lancer avec privilèges élevés à l'ouverture de session, sans invite UAC.
- Les règles sont enregistrées dans `config.json` à côté de l'exécutable et rechargées au démarrage.
- Interface localisée en 13 langues (dont l'arabe et l'hébreu de droite à gauche avec une disposition entièrement en miroir), détectée automatiquement depuis l'OS, avec un sélecteur dans l'app.
- L'outil filtre sa propre entrée synthétique, donc il ne se redéclenche jamais lui-même.

## Comment ça marche

- Un hook clavier bas niveau `WH_KEYBOARD_LL` détecte vos appuis et avale la touche de déclenchement configurée, de sorte que le jeu ne voit que les impulsions répétées propres.
- Chaque impulsion est `touche enfoncée → maintien ~30ms → touche relâchée`, envoyée avec `KEYEVENTF_SCANCODE`. Le maintien est nécessaire : un jeu basé sur le polling échantillonne l'état des touches par image et raterait une paire enfoncé/relâché tombant entre deux échantillons.
- Les événements synthétiques sont marqués via `dwExtraInfo`, ainsi le hook reconnaît ses propres touches injectées et les laisse passer au lieu d'agir dessus.

## Prérequis

- Windows 10/11 (x64).
- **Privilèges administrateur.** L'exécutable demande l'élévation automatiquement (invite UAC au lancement). C'est nécessaire car l'UIPI de Windows empêche l'entrée injectée par un processus non élevé d'atteindre les fenêtres élevées — et de nombreux jeux s'exécutent en mode élevé.

## Compilation

Aucun CGO requis, donc compilation croisée vers Windows depuis Linux/WSL ou compilation native sous Windows.

```sh
# Linux / WSL (compilation croisée) ou Windows (Git Bash) :
./build.sh
# -> turbokey.exe
```

Le script installe `rsrc` pour incorporer le manifeste de l'application (Common Controls v6, prise en charge DPI, requireAdministrator), puis lance `go build`. Pour compiler à la main :

```sh
go install github.com/akavel/rsrc@latest
rsrc -manifest cmd/turbokey/app.manifest -ico cmd/turbokey/icon.ico -arch amd64 -o cmd/turbokey/rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-H windowsgui -s -w" -o turbokey.exe ./cmd/turbokey
```

## Structure du projet

```
cmd/turbokey/      point d'entrée (main) + manifeste de l'app Windows + icône
cmd/gen-icon/      générateur d'icônes à la compilation (rend icon.ico / icon.png)
internal/keys/     table des touches & boutons souris ; mapping nom <-> code de touche virtuelle
internal/i18n/     localisation de l'UI ; catalogues JSON intégrés (locales/)
internal/config/   modèle de règles et chargement/sauvegarde de config.json
internal/winput/   wrappers Win32 : hooks clavier & souris, SendInput, premier plan
internal/engine/   moteur de frappe rapide : dispatch des hooks, interrupteur principal, workers
internal/ui/       GUI Win32 native (lxn/walk)
internal/autostart/ tâche planifiée à l'ouverture de session (Démarrer avec Windows)
internal/buildinfo/ version de build, injectée via -ldflags
```

## Utilisation

1. Lancez `turbokey.exe` et acceptez l'invite UAC.
2. Dans l'éditeur, choisissez une **touche de déclenchement**, une **touche de sortie** (par défaut = identique au déclenchement), un **mode** et un **intervalle (ms)**, puis cliquez sur **Ajouter**.
3. Pour modifier une règle, cliquez dessus (ses valeurs se chargent dans l'éditeur), modifiez et cliquez sur **Mettre à jour**. Double-cliquez sur une ligne pour l'activer/désactiver ; **Supprimer** l'efface.
4. Appuyez sur **F8** (ou cochez l'interrupteur principal) pour activer, puis appuyez sur votre touche de déclenchement pour tirer. Réappuyez sur **F8** quand vous avez terminé.

Remarques :
- L'intervalle est l'écart *entre* les appuis ; chaque appui maintient aussi la touche ~30ms, donc le plafond pratique est d'environ 25-30 appuis/seconde. C'est largement suffisant pour n'importe quelle compétence de jeu — appuyer plus vite n'aide pas, car le jeu échantillonne par image.
- Tant que l'interrupteur principal est activé, les touches de déclenchement configurées sont interceptées dans **toutes** les applications, pas seulement les jeux. Désactivez-le (F8) quand vous ne l'utilisez pas.
- F8 est réservée comme raccourci principal pendant que l'outil tourne.

## Configuration

`config.json` (créé à côté de l'exécutable) est lisible :

```json
{
  "rules": [
    { "name": "attack", "trigger": "J", "output": "", "mode": "hold",   "interval": 10, "enabled": true },
    { "name": "skill",  "trigger": "K", "output": "", "mode": "toggle", "interval": 50, "enabled": false }
  ]
}
```

- `output: ""` signifie « identique à la touche de déclenchement ».
- `mode` : `"hold"` ou `"toggle"`.
- Noms de touches : `A`–`Z`, `0`–`9`, `F1`–`F12`, `Space`, `Enter`, `Esc`, `Tab`, `↑ ↓ ← →`, `Ctrl`, `Alt`, `Shift`, et boutons de souris `Mouse Left`, `Mouse Right`, `Mouse Mid`, `Mouse X1`, `Mouse X2`. (Ce sont des identifiants neutres et non traduits, donc la configuration reste valide d'une langue d'UI à l'autre.)

## Limiter à des apps précises

Par défaut, la frappe rapide est active dans toutes les applications. Pour la restreindre, remplissez le champ « Apps actives » avec un ou plusieurs noms de processus (p. ex. `DNFGame.exe`, séparés par des virgules). La frappe rapide ne s'enclenche alors que lorsque l'une de ces apps est au premier plan ; ailleurs, vos touches se comportent normalement.

Vous ne connaissez pas le nom du processus ? Cliquez sur **Choisir une app…** et sélectionnez-la dans la liste des programmes en cours (affichés par titre de fenêtre et exécutable). Le raccourci principal F8 fonctionne toujours, quelle que soit l'app active.

## Langue

Choisissez la langue dans le menu déroulant en haut à droite de la fenêtre. Le choix est enregistré dans `config.json` et l'app redémarre pour l'appliquer ; « Auto » suit la langue de l'UI de Windows.

Langues incluses : English, 简体中文, 繁體中文, 日本語, 한국어, Español, Français, Deutsch, Русский, Português, Italiano, العربية, עברית.

Les langues de droite à gauche (arabe, hébreu) reflètent toute la disposition de la fenêtre via `WS_EX_LAYOUTRTL` (le `RightToLeftLayout` de walk).

Ordre de résolution : la variable d'environnement `TURBOKEY_LANG` (`zh`/`en`), puis le choix enregistré, puis la langue de l'OS.

Les catalogues de messages sont du JSON simple sous `internal/i18n/locales/`, intégrés avec `go:embed`. Pour ajouter une langue, déposez `internal/i18n/locales/<code>.json` (copiez `en.json` et traduisez les valeurs) et recompilez.

## Mises en garde

- Un hook clavier global plus l'injection d'entrée peuvent déclencher des faux positifs antivirus.
- Certains jeux en ligne interdisent les macros/l'automatisation dans leurs conditions d'utilisation, et les systèmes anti-triche peuvent détecter ou bloquer l'entrée synthétique. Utilisez de façon responsable et à vos propres risques.

## Licence

[MIT](../LICENSE)
