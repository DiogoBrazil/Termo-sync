#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
(cd frontend && npm ci && npm test && npm run build)
go test ./...
go vet ./...
mkdir -p build
GOOS=windows GOARCH=amd64 go build -tags production -trimpath -ldflags='-s -w -H windowsgui' -o build/DepoimentoStudio.exe .
chmod +x build/DepoimentoStudio.exe
if [ -d build/tools ]; then chmod +x build/tools/*.exe build/tools/*.dll 2>/dev/null || true; fi
sha256sum build/DepoimentoStudio.exe > build/DepoimentoStudio.exe.sha256
printf '%s\n' 'Executável gerado. FFmpeg não é baixado ou incluído automaticamente; consulte docs/install.md.'
