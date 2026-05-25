#!/usr/bin/env bash
# Cross-compile the Windows GUI executable from Linux/WSL. No CGO required.
set -euo pipefail

export PATH="$HOME/.local/go/bin:$HOME/go/bin:$PATH"
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

cd "$(dirname "$0")"

# Module init (idempotent).
[ -f go.mod ] || go mod init turbokey

# Embed the application manifest as a Windows resource object, placed in the
# main package directory so the linker picks it up. rsrc is a host-native tool.
if [ ! -f cmd/turbokey/rsrc_windows_amd64.syso ]; then
  go install github.com/akavel/rsrc@latest
  rsrc -manifest cmd/turbokey/app.manifest -ico cmd/turbokey/icon.ico -arch amd64 -o cmd/turbokey/rsrc_windows_amd64.syso
fi

# Resolve dependencies as seen by the Windows build.
GOOS=windows GOARCH=amd64 go mod tidy

# Build the GUI exe (-H windowsgui hides the console window).
VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath \
    -ldflags "-H windowsgui -s -w -X turbokey/internal/buildinfo.Version=${VERSION}" \
    -o turbokey.exe ./cmd/turbokey

echo "--- built ---"
ls -lh turbokey.exe
file turbokey.exe 2>/dev/null || true
