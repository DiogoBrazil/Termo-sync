$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')
function Check-Exit { if ($LASTEXITCODE -ne 0) { throw "Comando falhou: $LASTEXITCODE" } }
Push-Location frontend
try {
 npm ci; Check-Exit
 npm test; Check-Exit
 npm run build; Check-Exit
} finally { Pop-Location }
go test ./...; Check-Exit
go vet ./...; Check-Exit
New-Item -ItemType Directory -Force build | Out-Null
$env:GOOS='windows'; $env:GOARCH='amd64'
go build -tags production -trimpath -ldflags '-s -w -H windowsgui' -o build/DepoimentoStudio.exe .; Check-Exit
Get-FileHash build/DepoimentoStudio.exe -Algorithm SHA256 | Format-List | Out-File build/DepoimentoStudio.exe.sha256
Write-Host 'Executável gerado. Consulte docs/install.md antes de empacotar FFmpeg.'
