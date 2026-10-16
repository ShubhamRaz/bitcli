# BitCLI Uninstaller for Windows
# Removes ~/.bitcli/ entirely — deletes the binary, all models, tools, backend,
# database, config, and chat history. Nothing else on the system is modified.
#
# Usage:
#   .\scripts\uninstall.ps1

$ErrorActionPreference = "Stop"

$BITCLI_HOME = if ($env:BITCLI_HOME) { $env:BITCLI_HOME } else { Join-Path $HOME ".bitcli" }

Write-Host ""
Write-Host "  BitCLI Uninstaller" -ForegroundColor White
Write-Host "  ─────────────────────────────────────────────────────────" -ForegroundColor DarkGray
Write-Host "  This will permanently delete:" -ForegroundColor White
Write-Host "    $BITCLI_HOME" -ForegroundColor Yellow
Write-Host ""
Write-Host "  This includes the bitcli binary, all downloaded models," -ForegroundColor DarkGray
Write-Host "  tools (cmake, clang, uv), the BitNet backend clone," -ForegroundColor DarkGray
Write-Host "  database, config, and chat history." -ForegroundColor DarkGray
Write-Host ""

$confirm = Read-Host "  Type YES to confirm uninstall"
if ($confirm -ne "YES") {
    Write-Host "  Uninstall cancelled." -ForegroundColor Yellow
    exit 0
}

Write-Host ""
Write-Host "  Removing $BITCLI_HOME ..." -ForegroundColor Cyan

if (Test-Path $BITCLI_HOME) {
    Remove-Item -Recurse -Force $BITCLI_HOME
    Write-Host "  ✓ Removed $BITCLI_HOME" -ForegroundColor Green
} else {
    Write-Host "  ! $BITCLI_HOME does not exist — nothing to remove." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "  BitCLI has been completely removed." -ForegroundColor Green
Write-Host "  If you added env.ps1 to your PowerShell profile, remove that line manually." -ForegroundColor DarkGray
Write-Host ""
