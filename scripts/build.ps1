# Build the BitCLI binary for Windows or the current PowerShell platform.
$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Force -Path "bin" | Out-Null
go build -ldflags "-s -w" -o "bin/bitcli.exe" ./cmd/bitcli

