# BitCLI Installer for Windows
# Usage:  irm https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.ps1 | iex
#
# Requires: Git, Go (https://go.dev/dl/)
# Installs into: %USERPROFILE%\.bitcli\
# To uninstall:  Remove-Item -Recurse -Force "$HOME\.bitcli"

# Do NOT use $ErrorActionPreference = "Stop" — git writes to stderr which kills the script.
# Do NOT use exit — when run via irm|iex, exit closes the entire PowerShell window.

$ProgressPreference = "SilentlyContinue"

# ── Config ───────────────────────────────────────────────────────────────────
$BITCLI_HOME = if ($env:BITCLI_HOME) { $env:BITCLI_HOME } else { Join-Path $HOME ".bitcli" }
$BITCLI_BIN  = Join-Path $BITCLI_HOME "bin"
$BINARY      = Join-Path $BITCLI_BIN "bitcli.exe"
$GITHUB_REPO = "ShubhamRaz/bitcli"

# ── Helpers ──────────────────────────────────────────────────────────────────
function Write-Step([string]$msg) { Write-Host "`n  ==> $msg" -ForegroundColor Cyan }
function Write-Ok([string]$msg)   { Write-Host "  [OK] $msg" -ForegroundColor Green }
function Write-Warn([string]$msg) { Write-Host "  [!]  $msg" -ForegroundColor Yellow }

# ── Banner ───────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "  BitCLI Installer (build from source)" -ForegroundColor White
Write-Host "  -------------------------------------------------------" -ForegroundColor DarkGray
Write-Host "  Install dir : $BITCLI_HOME" -ForegroundColor DarkGray
Write-Host "  Uninstall   : Remove-Item -Recurse -Force `"$BITCLI_HOME`"" -ForegroundColor DarkGray

# ── Step 1: Check prerequisites ──────────────────────────────────────────────
Write-Step "Checking prerequisites"

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Host "  ERROR: git is not installed." -ForegroundColor Red
    Write-Host "  Install from: https://git-scm.com/download/win" -ForegroundColor Red
    throw "git is required but not found."
}
Write-Ok "git found"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "  ERROR: Go is not installed." -ForegroundColor Red
    Write-Host "  Install from: https://go.dev/dl/" -ForegroundColor Red
    throw "Go is required but not found."
}
Write-Ok "go found ($(go version))"

# ── Step 2: Create directory layout ──────────────────────────────────────────
Write-Step "Creating directory layout"
New-Item -ItemType Directory -Force -Path $BITCLI_BIN                       | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $BITCLI_HOME "models") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $BITCLI_HOME "tools")  | Out-Null
Write-Ok "Created $BITCLI_HOME"

# ── Step 3: Clone source ─────────────────────────────────────────────────────
Write-Step "Cloning source from GitHub"
$tmpSrc = Join-Path $env:TEMP "bitcli-src-$(Get-Random)"

# Use cmd /c to run git so PowerShell does not intercept stderr.
cmd /c "git clone --depth=1 `"https://github.com/$GITHUB_REPO.git`" `"$tmpSrc`" 2>&1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "  ERROR: git clone failed (exit $LASTEXITCODE)." -ForegroundColor Red
    throw "git clone failed."
}
Write-Ok "Cloned to $tmpSrc"

# ── Step 4: Build binary ─────────────────────────────────────────────────────
Write-Step "Building bitcli.exe (this takes ~30 seconds)"

# Use cmd /c so paths with spaces are handled correctly by Go.
cmd /c "cd /d `"$tmpSrc`" && go build -buildvcs=false -o `"$BINARY`" ./cmd/bitcli 2>&1"
if ($LASTEXITCODE -ne 0) {
    Remove-Item -Recurse -Force $tmpSrc -ErrorAction SilentlyContinue
    Write-Host "  ERROR: go build failed (exit $LASTEXITCODE)." -ForegroundColor Red
    throw "go build failed."
}

# Clean up source.
Remove-Item -Recurse -Force $tmpSrc -ErrorAction SilentlyContinue
Write-Ok "Built bitcli.exe at $BINARY"

# ── Step 5: Update PATH for this session ─────────────────────────────────────
Write-Step "Updating PATH for this session"
if ($env:PATH -notlike "*$BITCLI_BIN*") {
    $env:PATH = "$BITCLI_BIN;$env:PATH"
}
Write-Ok "PATH updated for current session"

# ── Step 6: Run bitcli setup ─────────────────────────────────────────────────
Write-Step "Running bitcli setup"
Write-Host "  This downloads cmake, clang, uv, and clones the BitNet backend." -ForegroundColor DarkGray
Write-Host "  May take several minutes on first run." -ForegroundColor DarkGray
Write-Host ""
& $BINARY setup
if ($LASTEXITCODE -ne 0) {
    Write-Warn "bitcli setup exited $LASTEXITCODE. Run  bitcli setup  to retry."
}

# ── Step 7: Configure permanent PATH & Environment ───────────────────────────
Write-Step "Configuring environment variables"

$currentUserPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
if ($currentUserPath -notlike "*$BITCLI_BIN*") {
    $newPath = if ([string]::IsNullOrEmpty($currentUserPath)) { $BITCLI_BIN } else { "$currentUserPath;$BITCLI_BIN" }
    [Environment]::SetEnvironmentVariable("Path", $newPath, [EnvironmentVariableTarget]::User)
    Write-Ok "Added $BITCLI_BIN to User PATH"
} else {
    Write-Ok "BitCLI is already in User PATH"
}

# Add env.ps1 to PowerShell profile for toolchain PATH (cmake, ninja, clang, uv)
$profilePath = $PROFILE
$envLine = ". `"$BITCLI_HOME\env.ps1`""
try {
    if (Test-Path $profilePath) {
        $profileContent = Get-Content $profilePath -Raw -ErrorAction SilentlyContinue
        if ($profileContent -notlike "*env.ps1*") {
            Add-Content -Path $profilePath -Value "`n$envLine"
            Write-Ok "Added toolchain environment to PowerShell profile"
        }
    } else {
        $profileDir = Split-Path $profilePath
        if (-not (Test-Path $profileDir)) {
            New-Item -ItemType Directory -Path $profileDir -Force | Out-Null
        }
        Set-Content -Path $profilePath -Value $envLine
        Write-Ok "Created PowerShell profile with BitCLI environment"
    }
} catch {
    Write-Warn "Could not update PowerShell profile: $_"
}

# ── Quick Start ──────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "  Quick start:" -ForegroundColor White
Write-Host "    bitcli doctor                                 # check system" -ForegroundColor Yellow
Write-Host "    bitcli pull microsoft/BitNet-b1.58-2B-4T      # download model" -ForegroundColor Yellow
Write-Host "    bitcli run --prompt `"Hello!`"                   # run inference" -ForegroundColor Yellow
Write-Host "    bitcli chat                                   # interactive chat" -ForegroundColor Yellow
Write-Host "    bitcli serve                                  # start API server" -ForegroundColor Yellow
Write-Host ""
Write-Host "  Uninstall: Remove-Item -Recurse -Force `"$BITCLI_HOME`"" -ForegroundColor DarkGray
Write-Host ""
