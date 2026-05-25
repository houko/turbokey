# TurboKey

[![Release](https://img.shields.io/github/v/release/houko/turbokey?sort=semver&display_name=tag)](https://github.com/houko/turbokey/releases)
[![Build](https://github.com/houko/turbokey/actions/workflows/ci.yml/badge.svg)](https://github.com/houko/turbokey/actions/workflows/ci.yml)
[![Downloads](https://img.shields.io/github/downloads/houko/turbokey/total)](https://github.com/houko/turbokey/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/houko/turbokey)](https://github.com/houko/turbokey/blob/main/go.mod)
[![License: MIT](https://img.shields.io/github/license/houko/turbokey)](https://github.com/houko/turbokey/blob/main/LICENSE)
![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)

[English](../README.md) · [简体中文](README.zh.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · **한국어** · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Русский](README.ru.md) · [Português](README.pt.md) · [Italiano](README.it.md) · [العربية](README.ar.md) · [עברית](README.he.md)

네이티브 GUI를 갖춘 가벼운 Windows 키 연타(rapid-fire / turbo) 도구입니다. 키 하나를 자동 반복으로 바인딩할 수 있습니다(같은 키 또는 다른 키). "누르고 있는 동안" 또는 "토글" 두 가지 모드를 지원하며, 규칙마다 간격을 따로 설정합니다.

일반 애플리케이션과 게임 모두에서 동작합니다: 키는 `SendInput`을 통해 하드웨어 **스캔 코드**로 주입되므로(DirectInput 게임도 인식), 각 입력은 잠시 유지되어 폴링 기반 게임에서도 안정적으로 감지됩니다.

> Windows 전용. Go + [lxn/walk](https://github.com/lxn/walk)(네이티브 Win32 컨트롤)로 제작, 단일 실행 파일, 런타임 설치 불필요.

<p align="center"><img src="screenshot.png" alt="TurboKey" width="380"></p>

## 다운로드

[Releases](../../releases) 페이지에서 최신 `turbokey.exe`를 받으세요. `main`에 푸시할 때마다 GitHub Actions가 버전이 매겨진 새 릴리스를 자동 빌드하여 배포합니다.

## 기능

- 규칙마다 **트리거 키**, **출력 키**(기본값은 트리거와 동일), **모드**, **간격**, 사용/해제를 설정.
- 트리거와 출력에 키보드 키 **또는 마우스 버튼**(왼쪽 / 오른쪽 / 가운데 / X1 / X2) 사용 가능.
- 두 가지 모드:
  - **누름** — 트리거 키를 물리적으로 누르고 있는 동안 반복.
  - **토글** — 한 번 누르면 시작, 다시 누르면 정지.
- 전역 마스터 스위치, 기본 단축키 **F8**.
- 특정 앱(프로세스 이름으로 매칭)에서만 연타를 활성화할 수 있습니다. 비우면 어디서나 동작합니다.
- **시스템 트레이**: 창을 닫으면 트레이로 최소화(도구는 계속 실행). 트레이 아이콘 왼쪽 클릭으로 복원, 오른쪽 클릭으로 메뉴(창 표시, 마스터 전환, 종료).
- 선택적으로 **Windows 시작 시 실행**. 예약 작업으로 등록되어 UAC 프롬프트 없이 로그온 시 관리자 권한으로 실행됩니다.
- 규칙은 실행 파일 옆의 `config.json`에 저장되고 시작 시 다시 로드됩니다.
- 13개 언어 현지화 UI(오른쪽에서 왼쪽으로 쓰는 아랍어, 히브리어 완전 미러 레이아웃 포함). OS에서 자동 감지, 앱 내 선택기 제공.
- 도구는 자신이 만든 합성 입력을 걸러내므로 자기 자신을 재트리거하지 않습니다.

## 작동 방식

- `WH_KEYBOARD_LL` 저수준 키보드 후크가 키 입력을 감지하고 설정된 트리거 키를 삼켜서, 게임에는 깨끗한 반복 펄스만 전달됩니다.
- 각 펄스는 `키 다운 → 약 30ms 유지 → 키 업`으로, `KEYEVENTF_SCANCODE`와 함께 전송됩니다. 이 유지 시간은 필수입니다: 폴링 기반 게임은 프레임마다 키 상태를 샘플링하므로, 두 샘플 사이에 끼인 다운/업 쌍은 놓칠 수 있습니다.
- 합성 이벤트는 `dwExtraInfo`로 태그되어, 후크가 자신이 주입한 키를 인식하고 그대로 통과시킵니다.

## 요구 사항

- Windows 10/11(x64).
- **관리자 권한.** 실행 파일은 시작 시 자동으로 권한 상승을 요청합니다(UAC 프롬프트). Windows UIPI가 비권한 프로세스의 주입 입력이 권한 상승된 창에 도달하는 것을 차단하기 때문에 필요합니다 — 많은 게임이 관리자로 실행됩니다.

## 빌드

CGO가 필요 없으므로 Linux/WSL에서 Windows로 크로스 컴파일하거나 Windows에서 네이티브로 빌드할 수 있습니다.

```sh
# Linux / WSL(크로스 컴파일) 또는 Windows(Git Bash):
./build.sh
# -> turbokey.exe
```

이 스크립트는 애플리케이션 매니페스트(Common Controls v6, DPI 인식, requireAdministrator)를 임베드하기 위해 `rsrc`를 설치한 다음 `go build`를 실행합니다. 수동 빌드:

```sh
go install github.com/akavel/rsrc@latest
rsrc -manifest cmd/turbokey/app.manifest -ico cmd/turbokey/icon.ico -arch amd64 -o cmd/turbokey/rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-H windowsgui -s -w" -o turbokey.exe ./cmd/turbokey
```

## 프로젝트 구조

```
cmd/turbokey/      진입점(main) + Windows 애플리케이션 매니페스트 + 아이콘
cmd/gen-icon/      빌드 시 아이콘 생성기(icon.ico / icon.png 렌더링)
internal/keys/     키 & 마우스 버튼 테이블, 이름 <-> 가상 키 코드 매핑
internal/i18n/     UI 현지화; 임베드된 JSON 메시지 카탈로그(locales/)
internal/config/   규칙 모델과 config.json 로드/저장
internal/winput/   Win32 래퍼: 키보드 & 마우스 후크, SendInput, 포그라운드
internal/engine/   연타 엔진: 후크 디스패치, 마스터 스위치, 워커
internal/ui/       네이티브 Win32 GUI(lxn/walk)
internal/autostart/ 로그온 예약 작업(Windows 시작 시 실행)
internal/buildinfo/ 빌드 버전, -ldflags로 주입
```

## 사용법

1. `turbokey.exe`를 실행하고 UAC 프롬프트를 수락합니다.
2. 편집기에서 **트리거 키**, **출력 키**(기본값 = 트리거와 동일), **모드**, **간격(ms)**을 선택하고 **추가**를 클릭합니다.
3. 규칙을 변경하려면 규칙을 클릭하고(값이 편집기에 로드됨) 수정한 후 **수정**을 클릭합니다. 행을 더블 클릭하면 사용/해제, **삭제**로 제거합니다.
4. **F8**(또는 마스터 스위치 체크)로 활성화한 다음 트리거 키를 눌러 발동합니다. 끝나면 **F8**을 다시 누릅니다.

참고:
- 간격은 입력 **사이**의 간격입니다. 각 입력도 약 30ms 유지되므로 실용적인 상한은 초당 약 25-30회입니다. 어떤 게임 스킬에도 충분합니다 — 더 빨리 눌러도 소용없는데, 게임은 프레임 단위로 샘플링하기 때문입니다.
- 마스터 스위치가 켜져 있는 동안 설정된 트리거 키는 게임뿐 아니라 **모든** 애플리케이션에서 가로채집니다. 사용하지 않을 때는 F8로 끄세요.
- 도구가 실행되는 동안 F8은 마스터 단축키로 예약됩니다.

## 설정

`config.json`(실행 파일 옆에 생성)은 사람이 읽을 수 있습니다:

```json
{
  "rules": [
    { "name": "attack", "trigger": "J", "output": "", "mode": "hold",   "interval": 10, "enabled": true },
    { "name": "skill",  "trigger": "K", "output": "", "mode": "toggle", "interval": 50, "enabled": false }
  ]
}
```

- `output: ""`는 "트리거 키와 동일"을 의미합니다.
- `mode`: `"hold"` 또는 `"toggle"`.
- 키 이름: `A`–`Z`, `0`–`9`, `F1`–`F12`, `Space`, `Enter`, `Esc`, `Tab`, `↑ ↓ ← →`, `Ctrl`, `Alt`, `Shift`, 그리고 마우스 버튼 `Mouse Left`, `Mouse Right`, `Mouse Mid`, `Mouse X1`, `Mouse X2`. (이들은 언어 중립 식별자이며 번역되지 않으므로, UI 언어를 바꿔도 설정이 유효합니다.)

## 특정 앱으로 제한

기본적으로 연타는 모든 애플리케이션에서 활성화됩니다. 제한하려면 "적용 앱" 필드에 프로세스 이름을 하나 이상 입력하세요(예: `DNFGame.exe`, 쉼표로 구분). 그러면 해당 앱이 포그라운드에 있을 때만 연타가 작동하고, 다른 곳에서는 키가 정상 동작합니다.

프로세스 이름을 모르나요? **앱 선택…**을 클릭하여 현재 실행 중인 프로그램 목록(창 제목과 실행 파일로 표시)에서 선택하세요. F8 마스터 단축키는 적용 앱과 무관하게 항상 작동합니다.

## 언어

창 오른쪽 상단의 드롭다운에서 언어를 선택합니다. 선택은 `config.json`에 저장되고 적용을 위해 앱이 재시작됩니다. "자동"은 Windows UI 언어를 따릅니다.

내장 언어: English, 简体中文, 繁體中文, 日本語, 한국어, Español, Français, Deutsch, Русский, Português, Italiano, العربية, עברית.

오른쪽에서 왼쪽으로 쓰는 언어(아랍어, 히브리어)는 `WS_EX_LAYOUTRTL`(walk의 `RightToLeftLayout`)로 전체 창 레이아웃을 미러링합니다.

해석 순서: 환경 변수 `TURBOKEY_LANG`(`zh`/`en`) → 저장된 선택 → OS 언어.

메시지 카탈로그는 `internal/i18n/locales/` 아래의 일반 JSON이며 `go:embed`로 임베드됩니다. 언어를 추가하려면 `internal/i18n/locales/<code>.json`을 넣고(`en.json`을 복사하여 값을 번역) 다시 빌드하세요.

## 주의 사항

- 전역 키보드 후크와 입력 주입은 백신 오탐을 유발할 수 있습니다.
- 일부 온라인 게임은 이용 약관에서 매크로/자동화를 금지하며, 안티치트가 합성 입력을 감지하거나 차단할 수 있습니다. 책임감 있게 사용하시고 위험은 본인 부담입니다.

## 라이선스

[MIT](../LICENSE)
