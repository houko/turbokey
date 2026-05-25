# TurboKey

[![Release](https://img.shields.io/github/v/release/houko/turbokey?sort=semver&display_name=tag)](https://github.com/houko/turbokey/releases)
[![Build](https://github.com/houko/turbokey/actions/workflows/release.yml/badge.svg)](https://github.com/houko/turbokey/actions/workflows/release.yml)
[![Downloads](https://img.shields.io/github/downloads/houko/turbokey/total)](https://github.com/houko/turbokey/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/houko/turbokey)](https://github.com/houko/turbokey/blob/main/go.mod)
[![License: MIT](https://img.shields.io/github/license/houko/turbokey)](https://github.com/houko/turbokey/blob/main/LICENSE)
![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)

[English](../README.md) · [简体中文](README.zh.md) · [繁體中文](README.zh-Hant.md) · **日本語** · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md) · [Português](README.pt.md) · [Italiano](README.it.md) · [العربية](README.ar.md) · [עברית](README.he.md)

ネイティブ GUI を備えた、軽量な Windows 用キー連打（rapid-fire / turbo）ツールです。1 つのキーを自動連打にバインドできます（同じキーでも別のキーでも可）。「押している間だけ」と「トグル」の 2 モード、ルールごとに間隔を個別設定できます。

通常のアプリでもゲームでも動作します:キーは `SendInput` 経由でハードウェア**スキャンコード**として送信されるため、DirectInput ゲームでも認識され、各押下は短時間保持されるので、ポーリング型ゲームでも確実にサンプリングされます。

> Windows 専用。Go + [lxn/walk](https://github.com/lxn/walk)（ネイティブ Win32 コントロール）で実装、単一実行ファイル、ランタイム不要。

<p align="center"><img src="screenshot.png" alt="TurboKey" width="380"></p>

## ダウンロード

[Releases](../../releases) ページから最新の `turbokey.exe` を入手してください。`main` への push ごとに GitHub Actions がバージョン付き Release を自動ビルドして公開します。

## 機能

- ルールごとに**トリガーキー**、**出力キー**（デフォルトはトリガーと同じ）、**モード**、**間隔**、有効/無効を設定可能。
- キーボードのキーでも**マウスボタン**（左 / 右 / 中 / X1 / X2）でも、トリガーと出力に使用可能。
- 2 つのモード:
  - **押し続け** — トリガーキーを物理的に押している間、繰り返します。
  - **切替** — 1 回押すと開始、もう一度押すと停止。
- グローバルなマスタースイッチ、デフォルトホットキー **F8**。
- 特定のアプリ（プロセス名で照合）にのみ有効化できます。空 = どこでも有効。
- **システムトレイ**:ウィンドウを閉じるとトレイに最小化（プロセスは継続）。アイコン左クリックで復元、右クリックでメニュー(表示・マスター切替・終了)。
- 任意で**Windows 起動時に実行**できます。タスクスケジューラに登録され、UAC プロンプトなしで管理者権限でログオン時に起動します。
- ルールは実行ファイルの隣の `config.json` に保存され、起動時に読み込まれます。
- 13 言語のローカライズ UI（右から左に書く言語の完全ミラーレイアウトも対応:アラビア語、ヘブライ語）。OS から自動検出、アプリ内で切替も可能。
- 自身が注入した合成イベントはフィルタされるため、自己再帰トリガーは発生しません。

## 仕組み

- `WH_KEYBOARD_LL` 低レベルキーボードフックがあなたのキー入力を検出し、設定されたトリガーキーを飲み込みます。ゲームには綺麗な連打パルスだけが見えます。
- 各パルスは `キーダウン → 約 30ms 保持 → キーアップ` で、`KEYEVENTF_SCANCODE` 付きで送信されます。この保持は必須:ポーリング型ゲームはフレーム単位でキー状態をサンプリングするため、サンプリング間に挟まれた down/up ペアは見逃される可能性があります。
- 合成イベントには `dwExtraInfo` でタグを付けるので、フックは自分が注入したキーを認識して素通しします。

## 動作要件

- Windows 10/11（x64）。
- **管理者権限。** 実行ファイルは起動時に自動で昇格を要求します(UAC ダイアログ)。Windows UIPI により非昇格プロセスから昇格ウィンドウへの入力注入はブロックされるため、必須です。多くのゲームが管理者として実行されます。

## ビルド

CGO 不要なので、Linux/WSL からの Windows 向けクロスコンパイルも、Windows 上のネイティブビルドも可能です。

```sh
# Linux / WSL（クロスコンパイル）または Windows（Git Bash）:
./build.sh
# -> turbokey.exe
```

このスクリプトはアプリケーションマニフェスト（Common Controls v6、DPI 認識、requireAdministrator）を埋め込むために `rsrc` をインストールしてから `go build` を実行します。手動でビルドする場合:

```sh
go install github.com/akavel/rsrc@latest
rsrc -manifest cmd/turbokey/app.manifest -ico cmd/turbokey/icon.ico -arch amd64 -o cmd/turbokey/rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-H windowsgui -s -w" -o turbokey.exe ./cmd/turbokey
```

## プロジェクト構成

```
cmd/turbokey/      エントリポイント(main)+ Windows アプリマニフェスト + アイコン
cmd/gen-icon/      ビルド時アイコン生成ツール(icon.ico / icon.png を生成)
internal/keys/     キー & マウスボタンテーブル、名前 <-> 仮想キーコードマッピング
internal/i18n/     UI ローカリゼーション;埋め込み JSON メッセージカタログ(locales/)
internal/config/   ルールモデルと config.json の読み書き
internal/winput/   Win32 ラッパー:キーボード & マウスフック、SendInput、フォアグラウンド
internal/engine/   連打エンジン:フックディスパッチ、マスタースイッチ、ワーカー
internal/ui/       ネイティブ Win32 GUI(lxn/walk)
internal/autostart/ ログオン時のスケジュールタスク(Windows 起動時に実行)
internal/buildinfo/ ビルドバージョン、-ldflags で注入
```

## 使い方

1. `turbokey.exe` を起動し、UAC プロンプトを承諾します。
2. エディタで**トリガーキー**、**出力キー**(デフォルト = トリガーと同じ)、**モード**、**間隔(ms)**を選び、**追加**をクリック。
3. ルールを変更するには、行をクリックして(エディタに値がロードされます)、編集後**更新**。行をダブルクリックで有効/無効切替、**削除**で削除。
4. **F8**(またはマスタースイッチのチェック)で有効化、トリガーキーを押すと連打開始。終わったら **F8** をもう一度押して停止。

注意:
- 「間隔」は連続するキー押下の**間**の間隔です。各押下も約 30ms 保持されるため、実用上の上限は秒間 25-30 回ほど。どんなゲームスキルにも十分です—それ以上速く押しても無意味で、ゲームはフレーム単位でしかサンプリングしません。
- マスタースイッチが ON の間、設定されたトリガーキーは**すべて**のアプリで横取りされます(ゲームだけではありません)。使わないときは F8 で OFF にしてください。
- ツール実行中は F8 がマスターホットキーとして予約されます。

## 設定

`config.json`(実行ファイルの隣に自動生成)は人間が読める形式:

```json
{
  "rules": [
    { "name": "attack", "trigger": "J", "output": "", "mode": "hold",   "interval": 10, "enabled": true },
    { "name": "skill",  "trigger": "K", "output": "", "mode": "toggle", "interval": 50, "enabled": false }
  ]
}
```

- `output: ""` は「トリガーキーと同じ」の意味。
- `mode`:`"hold"` または `"toggle"`。
- キー名:`A`–`Z`、`0`–`9`、`F1`–`F12`、`Space`、`Enter`、`Esc`、`Tab`、`↑ ↓ ← →`、`Ctrl`、`Alt`、`Shift`、およびマウスボタン `Mouse Left`、`Mouse Right`、`Mouse Mid`、`Mouse X1`、`Mouse X2`。(これらは言語非依存の識別子なので翻訳されません。UI 言語を切り替えても設定は有効のままです。)

## 特定のアプリに限定する

デフォルトでは連打はすべてのアプリで有効です。制限するには「対象アプリ」フィールドにプロセス名を 1 つ以上入力します(例:`DNFGame.exe`、カンマ区切り)。連打はこれらのアプリがフォアグラウンドのときだけ発動し、それ以外ではキーは通常通り動作します。

プロセス名がわからない?**アプリを選択…** をクリックして、現在実行中のプログラムのリスト(ウィンドウタイトルと実行ファイル名で表示)から選びます。F8 マスターホットキーはアプリの絞り込みに関係なく常に動作します。

## 言語

ウィンドウ右上のドロップダウンで言語を選択します。選択は `config.json` に保存され、適用のためにアプリが再起動します。「自動」は Windows の UI 言語に従います。

同梱言語:English、简体中文、繁體中文、日本語、한국어、Español、Français、Deutsch、Русский、Português、Italiano、العربية、עברית。

右から左の言語(アラビア語、ヘブライ語)は `WS_EX_LAYOUTRTL`(walk の `RightToLeftLayout`)でウィンドウレイアウト全体をミラーします。

解決順序:環境変数 `TURBOKEY_LANG`(`zh`/`en`)→ 保存された選択 → OS の言語。

メッセージカタログは `internal/i18n/locales/` 配下のプレーン JSON で、`go:embed` で埋め込まれています。言語を追加するには `internal/i18n/locales/<code>.json` を作成して(`en.json` をコピーして値を翻訳)、再ビルドします。

## 注意事項

- グローバルキーボードフックと入力注入は、ウイルス対策ソフトに誤検知される可能性があります。
- 一部のオンラインゲームは利用規約でマクロ/自動化を禁止しており、アンチチートが合成入力を検出/ブロックする場合があります。責任を持ってご利用ください。

## ライセンス

[MIT](../LICENSE)
