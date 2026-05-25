# TurboKey

[![Release](https://img.shields.io/github/v/release/houko/turbokey?sort=semver&display_name=tag)](https://github.com/houko/turbokey/releases)
[![Build](https://github.com/houko/turbokey/actions/workflows/ci.yml/badge.svg)](https://github.com/houko/turbokey/actions/workflows/ci.yml)
[![Downloads](https://img.shields.io/github/downloads/houko/turbokey/total)](https://github.com/houko/turbokey/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/houko/turbokey)](https://github.com/houko/turbokey/blob/main/go.mod)
[![License: MIT](https://img.shields.io/github/license/houko/turbokey)](https://github.com/houko/turbokey/blob/main/LICENSE)
![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)

[English](../README.md) · **简体中文** · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md) · [Português](README.pt.md) · [Italiano](README.it.md) · [العربية](README.ar.md) · [עברית](README.he.md)

一款轻量的 Windows 按键连发（rapid-fire / turbo）工具，原生 GUI。把一个按键绑成自动重复（重复自身或另一个键），可选「按住模式」或「开关模式」，每条规则各自设置间隔。

兼容普通程序与游戏：通过 `SendInput` 以硬件**扫描码**注入按键（DirectInput 游戏也认），每次按下保持一小段时间，让基于轮询的游戏稳定采样到。

> 仅支持 Windows。基于 Go + [lxn/walk](https://github.com/lxn/walk)（原生 Win32 控件），单文件可执行程序，无需运行时。

<p align="center"><img src="screenshot.png" alt="TurboKey" width="380"></p>

## 下载

到 [Releases](../../releases) 页下载最新的 `turbokey.exe`。每次推送到 `main` 都会通过 GitHub Actions 自动构建并发布带版本号的新 Release。

## 特性

- 每条规则可独立设置**触发键**、**输出键**（默认与触发键相同）、**模式**、**间隔**和启用状态。
- 触发键和输出键可以是键盘按键，也可以是**鼠标按钮**（左键、右键、中键、X1、X2）。
- 两种模式：
  - **按住模式** — 物理按住触发键期间持续重复。
  - **开关模式** — 按一下开始重复，再按一下停止。
- 全局总开关，默认热键 **F8**。
- 可选择只在特定程序中生效（按进程名匹配）；留空 = 全局生效。
- **系统托盘**：关闭窗口即最小化到托盘（程序继续运行）。左键单击托盘图标恢复窗口，右键调出菜单可显示窗口、切换总开关或退出。
- 可选**开机自启**，通过计划任务以管理员权限随登录启动，无 UAC 提示。
- 规则保存到可执行文件旁的 `config.json`，启动时自动加载。
- 13 种语言的本地化界面（含从右到左排版的阿拉伯语、希伯来语，整个窗口完全镜像），根据 OS 自动选择，也可在程序内切换。
- 工具会过滤掉自己产生的合成输入，绝不会自我循环触发。

## 工作原理

- `WH_KEYBOARD_LL` 低级键盘钩子拦截你的按键，吞掉配置过的触发键，让游戏只看到干净的重复脉冲。
- 每次脉冲是 `按下 → 保持约 30ms → 抬起`，使用 `KEYEVENTF_SCANCODE` 发送。这个保持时长是必要的：轮询型游戏按帧采样按键状态，如果按下/抬起恰好落在两次采样之间会被漏掉。
- 合成事件通过 `dwExtraInfo` 打标，钩子识别后直接放行，不会作用在自己身上。

## 系统要求

- Windows 10/11（x64）。
- **管理员权限。** 程序会自动请求提权（启动时弹 UAC）。原因：Windows UIPI 阻止非提权进程向提权窗口注入输入，而大量游戏以管理员身份运行。

## 构建

不依赖 CGO，可在 Linux/WSL 中交叉编译到 Windows，也可在 Windows 上原生构建。

```sh
# Linux / WSL（交叉编译）或 Windows（Git Bash）：
./build.sh
# -> turbokey.exe
```

脚本会安装 `rsrc` 来嵌入应用清单（Common Controls v6、DPI 感知、requireAdministrator），然后执行 `go build`。手动构建：

```sh
go install github.com/akavel/rsrc@latest
rsrc -manifest cmd/turbokey/app.manifest -ico cmd/turbokey/icon.ico -arch amd64 -o cmd/turbokey/rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-H windowsgui -s -w" -o turbokey.exe ./cmd/turbokey
```

## 项目结构

```
cmd/turbokey/      入口（main）+ Windows 应用清单 + 图标
cmd/gen-icon/      构建期图标生成器（生成 icon.ico / icon.png）
internal/keys/     按键 & 鼠标键表，名称 <-> 虚拟键码映射
internal/i18n/     UI 本地化；内嵌 JSON 翻译目录（locales/）
internal/config/   规则模型与 config.json 读写
internal/winput/   Win32 封装：键鼠钩子、SendInput、前台进程
internal/engine/   连发引擎：钩子调度、总开关、工作协程
internal/ui/       原生 Win32 GUI（lxn/walk）
internal/autostart/ 登录计划任务（开机自启）
internal/buildinfo/ 构建版本号，由 -ldflags 注入
```

## 使用

1. 启动 `turbokey.exe`，接受 UAC 提示。
2. 在编辑区选择**触发键**、**输出键**（默认 = 与触发键相同）、**模式**、**间隔 (ms)**，点击**添加**。
3. 修改规则：单击规则行（数据会加载到编辑区），改完点**更新**。双击一行可启用/停用；**删除选中**移除规则。
4. 按 **F8**（或勾选总开关）启用连发，然后按触发键即可。用完再按一次 **F8** 关闭。

注意：
- 间隔指的是相邻两次按下**之间**的间隔；每次按下还会保持约 30ms，所以实际上限大约是每秒 25-30 次。这对任何游戏技能都足够 —— 按得更快并不会更快，因为游戏按帧采样。
- 总开关打开时，配置过的触发键在**所有**程序里都会被拦截，不只是游戏。不用的时候记得按 F8 关掉。
- F8 在工具运行期间被独占作为总开关热键。

## 配置文件

`config.json`（程序旁自动创建）人类可读：

```json
{
  "rules": [
    { "name": "attack", "trigger": "J", "output": "", "mode": "hold",   "interval": 10, "enabled": true },
    { "name": "skill",  "trigger": "K", "output": "", "mode": "toggle", "interval": 50, "enabled": false }
  ]
}
```

- `output: ""` 表示「与触发键相同」。
- `mode`：`"hold"` 或 `"toggle"`。
- 按键名：`A`–`Z`、`0`–`9`、`F1`–`F12`、`Space`、`Enter`、`Esc`、`Tab`、`↑ ↓ ← →`、`Ctrl`、`Alt`、`Shift`，以及鼠标按钮 `Mouse Left`、`Mouse Right`、`Mouse Mid`、`Mouse X1`、`Mouse X2`。（这些是语言无关标识符，不会被翻译，切换 UI 语言后配置依然有效。）

## 限定特定程序

默认情况下连发在所有程序里都生效。如果想限制范围，在「生效程序」框里填一个或多个进程名（如 `DNFGame.exe`，逗号分隔）。这样只有这些程序处于前台时连发才会启动；其他程序里按键照常。

不知道进程名？点**选择程序…** 从当前运行的程序列表里挑（按窗口标题和可执行文件显示）。F8 总开关热键不受程序范围影响，永远生效。

## 语言

在窗口右上角的下拉菜单里选语言。选择会写入 `config.json` 并自动重启程序应用；「自动」跟随 Windows 系统语言。

内置语言：English、简体中文、繁體中文、日本語、한국어、Español、Français、Deutsch、Русский、Português、Italiano、العربية、עברית。

从右到左的语言（阿拉伯语、希伯来语）通过 `WS_EX_LAYOUTRTL`（walk 的 `RightToLeftLayout`）镜像整个窗口布局。

解析顺序：环境变量 `TURBOKEY_LANG`（`zh`/`en`）→ 保存的选择 → OS 语言。

翻译目录是 `internal/i18n/locales/` 下的纯 JSON，通过 `go:embed` 内嵌。要新增语言，把 `internal/i18n/locales/<code>.json` 放进去（复制 `en.json` 并翻译值）然后重新构建。

## 注意事项

- 全局键盘钩子加输入注入可能触发杀毒软件误报。
- 部分网游的服务条款禁止宏/自动化，反作弊系统可能检测或拦截合成输入。请遵守规则，自行承担风险。

## 许可证

[MIT](../LICENSE)
