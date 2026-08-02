# BitCLI Installer for Windows
# Usage (one-liner from PowerShell):
#   irm https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.ps1 | iex
#
# Everything is installed into %USERPROFILE%\.bitcli\
# To uninstall: Remove-Item -Recurse -Force "$HOME\.bitcli"

# NOTE: This script intentionally avoids `exit` because when run via `irm | iex`
# `exit` closes the entire PowerShell window. We use `throw` instead, which shows
# a red error but keeps the shell open.

$ProgressPreference = "SilentlyContinue"

# ── Config ───────────────────────────────────────────────────────────────────
$BITCLI_HOME = if ($env:BITCLI_HOME) { $env:BITCLI_HOME } else { Join-Path $HOME ".bitcli" }
$BITCLI_BIN  = Join-Path $BITCLI_HOME "bin"
$BINARY      = Join-Path $BITCLI_BIN "bitcli.exe"
$GITHUB_REPO = "ShubhamRaz/bitcli"

function Write-Step([string]$msg) {
    Write-Host ""
    Write-Host "  ==> $msg" -ForegroundColor Cyan
}
function Write-Ok([string]$msg) {
    Write-Host "  [OK] $msg" -ForegroundColor Green
}
function Write-Warn([string]$msg) {
    Write-Host "  [!]  $msg" -ForegroundColor Yellow
}
function Abort([string]$msg) {
    Write-Host ""
    Write-Host "  ERROR: $msg" -ForegroundColor Red
    Write-Host ""
    # Use throw instead of exit so the PowerShell window stays open.
    throw "BitCLI install failed: $msg"
}

# ── Run git safely (Start-Process avoids NativeCommandError on git stderr) ────
function Invoke-Git([string[]]$Arguments, [string]$WorkDir) {
    if (-not $WorkDir) { $WorkDir = $PWD.Path }
    $errFile = [System.IO.Path]::GetTempFileName()
    try {
        $proc = Start-Process git `
            -ArgumentList $Arguments `
            -WorkingDirectory $WorkDir `
            -Wait -PassThru -NoNewWindow `
            -RedirectStandardError $errFile
        if ($proc.ExitCode -ne 0) {
            $msg = (Get-Content $errFile -Raw -ErrorAction SilentlyContinue) -replace "`r`n", " "
            Abort "git $($Arguments[0]) failed: $msg"
        }
    } finally {
        Remove-Item $errFile -ErrorAction SilentlyContinue
    }
}

# ── Banner ────────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "  BitCLI Installer" -ForegroundColor White
Write-Host "  -------------------------------------------------------" -ForegroundColor DarkGray
Write-Host "  Install directory : $BITCLI_HOME" -ForegroundColor DarkGray
Write-Host "  To uninstall      : Remove-Item -Recurse -Force `"$BITCLI_HOME`"" -ForegroundColor DarkGray
Write-Host ""

# ── Step 1: Create directory layout ──────────────────────────────────────────
Write-Step "Creating BitCLI home"
New-Item -ItemType Directory -Force -Path $BITCLI_BIN                        | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $BITCLI_HOME "models")  | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $BITCLI_HOME "tools")   | Out-Null
Write-Ok "Created $BITCLI_HOME"

# ── Step 2: Check git ─────────────────────────────────────────────────────────
Write-Step "Checking for git"
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Abort "git is not installed.`n  Install from: https://git-scm.com/download/win`n  Then re-run this installer."
}
Write-Ok "git found"

# ── Step 3: Get bitcli.exe ────────────────────────────────────────────────────
Write-Step "Installing BitCLI binary"

$arch      = if ([System.Environment]::Is64BitOperatingSystem) { "x86_64" } else { "x86" }
$assetName = "bitcli-windows-$arch.exe"
$gotBinary = $false

# Try GitHub Releases.
try {
    $release = Invoke-RestMethod `
        -Uri "https://api.github.com/repos/$GITHUB_REPO/releases/latest" `
        -Headers @{ "User-Agent" = "bitcli-installer/1.0" } `
        -ErrorAction Stop
    $asset = $release.assets | Where-Object { $_.name -eq $assetName } | Select-Object -First 1
    if ($asset) {
        Write-Host "  Downloading $($release.tag_name) ..."
        Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $BINARY -UseBasicParsing -ErrorAction Stop
        Write-Ok "Downloaded bitcli.exe"
        $gotBinary = $true
    } else {
        Write-Warn "Release $($release.tag_name) exists but no Windows binary attached yet."
    }
} catch {
    Write-Warn "No GitHub release found. Will build from source."
}

# Build from source fallback.
if (-not $gotBinary) {
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Abort "Go is not installed and no pre-built binary is available.`n  Install Go from https://go.dev/dl/ then re-run."
    }

    $tmpSrc = Join-Path $env:TEMP "bitcli-src-$(Get-Random)"
    Write-Host "  Cloning repository ..."
    Invoke-Git -Arguments @("clone", "--depth=1", "https://github.com/$GITHUB_REPO.git", $tmpSrc)

    Write-Host "  Building binary (~30 seconds) ..."
    $errFile = [System.IO.Path]::GetTempFileName()
    try {
        $proc = Start-Process go `
            -ArgumentList @("build", "-buildvcs=false", "-o", $BINARY, "./cmd/bitcli") `
            -WorkingDirectory $tmpSrc `
            -Wait -PassThru -NoNewWindow `
            -RedirectStandardError $errFile
        if ($proc.ExitCode -ne 0) {
            $msg = (Get-Content $errFile -Raw -ErrorAction SilentlyContinue)
            Abort "go build failed:`n$msg"
        }
    } finally {
        Remove-Item $errFile -ErrorAction SilentlyContinue
        Remove-Item -Recurse -Force $tmpSrc -ErrorAction SilentlyContinue
    }
    Write-Ok "Built bitcli.exe from source"
}

# ── Step 4: Activate PATH ─────────────────────────────────────────────────────
Write-Step "Updating PATH for this session"
$env:PATH = "$BITCLI_BIN;$env:PATH"
Write-Ok "PATH updated (current session only)"

# ── Step 5: Run bitcli setup ──────────────────────────────────────────────────
Write-Step "Running bitcli setup"
Write-Host "  Installs cmake, clang, uv, and clones the BitNet backend." -ForegroundColor DarkGray
Write-Host "  This may take several minutes on first run." -ForegroundColor DarkGray
Write-Host ""
& $BINARY setup
if ($LASTEXITCODE -ne 0) {
    Write-Warn "bitcli setup exited $LASTEXITCODE. Run  bitcli setup  to retry."
}

# ── Done ──────────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "  -------------------------------------------------------" -ForegroundColor DarkGray
Write-Host "  BitCLI installed!" -ForegroundColor Green
Write-Host ""
Write-Host "  Quick start:" -ForegroundColor White
Write-Host "    bitcli doctor" -ForegroundColor Yellow
Write-Host "    bitcli pull microsoft/BitNet-b1.58-2B-4T" -ForegroundColor Yellow
Write-Host "    bitcli run --prompt `"Hello!`"" -ForegroundColor Yellow
Write-Host ""
Write-Host "  Permanent PATH — add to PowerShell profile (notepad `$PROFILE):" -ForegroundColor White
Write-Host "    . `"$BITCLI_HOME\env.ps1`"" -ForegroundColor Yellow
Write-Host ""
Write-Host "  Uninstall: Remove-Item -Recurse -Force `"$BITCLI_HOME`"" -ForegroundColor DarkGray
Write-Host ""
