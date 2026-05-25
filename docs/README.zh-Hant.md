# TurboKey

[![Release](https://img.shields.io/github/v/release/houko/turbokey?sort=semver&display_name=tag)](https://github.com/houko/turbokey/releases)
[![Build](https://github.com/houko/turbokey/actions/workflows/release.yml/badge.svg)](https://github.com/houko/turbokey/actions/workflows/release.yml)
[![Downloads](https://img.shields.io/github/downloads/houko/turbokey/total)](https://github.com/houko/turbokey/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/houko/turbokey)](https://github.com/houko/turbokey/blob/main/go.mod)
[![License: MIT](https://img.shields.io/github/license/houko/turbokey)](https://github.com/houko/turbokey/blob/main/LICENSE)
![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)

[English](../README.md) · [简体中文](README.zh.md) · **繁體中文** · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md) · [Português](README.pt.md) · [Italiano](README.it.md) · [العربية](README.ar.md) · [עברית](README.he.md)

一款輕量的 Windows 按鍵連發（rapid-fire / turbo）工具，原生 GUI。把一個按鍵設定為自動重複（重複自身或另一個鍵），可選「按住模式」或「開關模式」，每條規則各自設定間隔。

相容一般程式與遊戲:透過 `SendInput` 以硬體**掃描碼**注入按鍵（DirectInput 遊戲也認），每次按下保持一小段時間,讓基於輪詢的遊戲穩定取樣到。

> 僅支援 Windows。基於 Go + [lxn/walk](https://github.com/lxn/walk)（原生 Win32 控制項）,單檔可執行程式,無需執行時環境。

<p align="center"><img src="screenshot.png" alt="TurboKey" width="380"></p>

## 下載

到 [Releases](../../releases) 頁下載最新的 `turbokey.exe`。每次推送到 `main` 都會透過 GitHub Actions 自動建置並發佈帶版本號的新 Release。

## 特性

- 每條規則可獨立設定**觸發鍵**、**輸出鍵**（預設與觸發鍵相同）、**模式**、**間隔**和啟用狀態。
- 觸發鍵和輸出鍵可以是鍵盤按鍵,也可以是**滑鼠按鈕**（左鍵、右鍵、中鍵、X1、X2）。
- 兩種模式:
  - **按住模式** — 物理按住觸發鍵期間持續重複。
  - **開關模式** — 按一下開始重複,再按一下停止。
- 全域總開關,預設熱鍵 **F8**。
- 可選擇只在特定程式中生效（依行程名比對）；留空 = 全域生效。
- **系統匣**:關閉視窗即最小化到系統匣（程式繼續執行）。左鍵單擊圖示還原視窗,右鍵叫出選單可顯示視窗、切換總開關或結束。
- 可選**開機自啟**,透過排程工作以管理員權限隨登入啟動,無 UAC 提示。
- 規則儲存到可執行檔旁的 `config.json`,啟動時自動載入。
- 13 種語言的本地化介面(含從右到左排版的阿拉伯語、希伯來語,整個視窗完全鏡像),根據 OS 自動選擇,也可在程式內切換。
- 工具會過濾掉自己產生的合成輸入,絕不會自我循環觸發。

## 工作原理

- `WH_KEYBOARD_LL` 低階鍵盤掛鉤攔截你的按鍵,吞掉設定過的觸發鍵,讓遊戲只看到乾淨的重複脈衝。
- 每次脈衝是 `按下 → 保持約 30ms → 放開`,使用 `KEYEVENTF_SCANCODE` 送出。這個保持時長是必要的:輪詢型遊戲按影格取樣按鍵狀態,如果按下/放開恰好落在兩次取樣之間會被漏掉。
- 合成事件透過 `dwExtraInfo` 加上標記,掛鉤辨識後直接放行,不會作用在自己身上。

## 系統需求

- Windows 10/11(x64)。
- **系統管理員權限。** 程式會自動請求提權(啟動時彈 UAC)。原因:Windows UIPI 阻止非提權行程向提權視窗注入輸入,而大量遊戲以管理員身份執行。

## 建置

不依賴 CGO,可在 Linux/WSL 中交叉編譯到 Windows,也可在 Windows 上原生建置。

```sh
# Linux / WSL(交叉編譯)或 Windows(Git Bash):
./build.sh
# -> turbokey.exe
```

腳本會安裝 `rsrc` 來嵌入應用程式資訊清單(Common Controls v6、DPI 感知、requireAdministrator),然後執行 `go build`。手動建置:

```sh
go install github.com/akavel/rsrc@latest
rsrc -manifest cmd/turbokey/app.manifest -ico cmd/turbokey/icon.ico -arch amd64 -o cmd/turbokey/rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-H windowsgui -s -w" -o turbokey.exe ./cmd/turbokey
```

## 專案結構

```
cmd/turbokey/      進入點(main)+ Windows 應用程式資訊清單 + 圖示
cmd/gen-icon/      建置期圖示產生器(生成 icon.ico / icon.png)
internal/keys/     按鍵 & 滑鼠鍵表,名稱 <-> 虛擬鍵碼映射
internal/i18n/     UI 本地化;內嵌 JSON 翻譯目錄(locales/)
internal/config/   規則模型與 config.json 讀寫
internal/winput/   Win32 封裝:鍵鼠掛鉤、SendInput、前景行程
internal/engine/   連發引擎:掛鉤排程、總開關、工作協程
internal/ui/       原生 Win32 GUI(lxn/walk)
internal/autostart/ 登入排程工作(開機自啟)
internal/buildinfo/ 建置版本號,由 -ldflags 注入
```

## 使用

1. 啟動 `turbokey.exe`,接受 UAC 提示。
2. 在編輯區選擇**觸發鍵**、**輸出鍵**(預設 = 與觸發鍵相同)、**模式**、**間隔 (ms)**,按一下**新增**。
3. 修改規則:單擊規則列(資料會載入到編輯區),改完按**更新**。雙擊列可啟用/停用;**刪除選取**移除規則。
4. 按 **F8**(或勾選總開關)啟用連發,然後按觸發鍵即可。用完再按一次 **F8** 關閉。

注意:
- 間隔指的是相鄰兩次按下**之間**的間隔;每次按下還會保持約 30ms,所以實際上限大約是每秒 25-30 次。這對任何遊戲技能都足夠 —— 按得更快並不會更快,因為遊戲按影格取樣。
- 總開關打開時,設定過的觸發鍵在**所有**程式裡都會被攔截,不只是遊戲。不用的時候記得按 F8 關掉。
- F8 在工具執行期間被獨佔作為總開關熱鍵。

## 設定檔

`config.json`(程式旁自動建立)人類可讀:

```json
{
  "rules": [
    { "name": "attack", "trigger": "J", "output": "", "mode": "hold",   "interval": 10, "enabled": true },
    { "name": "skill",  "trigger": "K", "output": "", "mode": "toggle", "interval": 50, "enabled": false }
  ]
}
```

- `output: ""` 表示「與觸發鍵相同」。
- `mode`:`"hold"` 或 `"toggle"`。
- 按鍵名:`A`–`Z`、`0`–`9`、`F1`–`F12`、`Space`、`Enter`、`Esc`、`Tab`、`↑ ↓ ← →`、`Ctrl`、`Alt`、`Shift`,以及滑鼠按鈕 `Mouse Left`、`Mouse Right`、`Mouse Mid`、`Mouse X1`、`Mouse X2`。(這些是語言無關識別碼,不會被翻譯,切換 UI 語言後設定依然有效。)

## 限定特定程式

預設情況下連發在所有程式裡都生效。如果想限制範圍,在「生效程式」框裡填一個或多個行程名(如 `DNFGame.exe`,逗號分隔)。這樣只有這些程式處於前景時連發才會啟動;其他程式裡按鍵照常。

不知道行程名?按**選擇程式…** 從目前執行的程式清單裡挑(依視窗標題和可執行檔顯示)。F8 總開關熱鍵不受程式範圍影響,永遠生效。

## 語言

在視窗右上角的下拉選單裡選語言。選擇會寫入 `config.json` 並自動重啟程式套用;「自動」跟隨 Windows 系統語言。

內建語言:English、简体中文、繁體中文、日本語、한국어、Español、Français、Deutsch、Русский、Português、Italiano、العربية、עברית。

從右到左的語言(阿拉伯語、希伯來語)透過 `WS_EX_LAYOUTRTL`(walk 的 `RightToLeftLayout`)鏡像整個視窗版面。

解析順序:環境變數 `TURBOKEY_LANG`(`zh`/`en`)→ 儲存的選擇 → OS 語言。

翻譯目錄是 `internal/i18n/locales/` 下的純 JSON,透過 `go:embed` 內嵌。要新增語言,把 `internal/i18n/locales/<code>.json` 放進去(複製 `en.json` 並翻譯值)然後重新建置。

## 注意事項

- 全域鍵盤掛鉤加輸入注入可能觸發防毒軟體誤判。
- 部分網遊的服務條款禁止巨集/自動化,反作弊系統可能偵測或攔截合成輸入。請遵守規則,自行承擔風險。

## 授權

[MIT](../LICENSE)
