# BitCLI Installer for Windows
# Usage (one-liner from PowerShell):
#   irm https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.ps1 | iex
#
# Everything is installed into %USERPROFILE%\.bitcli\
# To uninstall completely: Remove-Item -Recurse -Force "$HOME\.bitcli"

# NOTE: Do NOT set $ErrorActionPreference = "Stop" globally — git writes to stderr
# which PowerShell treats as errors. We check $LASTEXITCODE manually instead.
$ProgressPreference = "SilentlyContinue"  # Speeds up Invoke-WebRequest

# ── Config ──────────────────────────────────────────────────────────────────
$BITCLI_VERSION = "latest"
$BITCLI_HOME    = if ($env:BITCLI_HOME) { $env:BITCLI_HOME } else { Join-Path $HOME ".bitcli" }
$BITCLI_BIN     = Join-Path $BITCLI_HOME "bin"
$BINARY         = Join-Path $BITCLI_BIN "bitcli.exe"
$GITHUB_REPO    = "ShubhamRaz/bitcli"

function Write-Step([string]$msg) {
    Write-Host ""
    Write-Host "  ==> $msg" -ForegroundColor Cyan
}

function Write-Ok([string]$msg) {
    Write-Host "  v  $msg" -ForegroundColor Green
}

function Write-Warn([string]$msg) {
    Write-Host "  !  $msg" -ForegroundColor Yellow
}

function Abort([string]$msg) {
    Write-Host ""
    Write-Host "  ERROR: $msg" -ForegroundColor Red
    Write-Host ""
    exit 1
}

# ── Banner ───────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "  BitCLI Installer" -ForegroundColor White
Write-Host "  ─────────────────────────────────────────────────────────" -ForegroundColor DarkGray
Write-Host "  Install directory: $BITCLI_HOME" -ForegroundColor DarkGray
Write-Host "  Everything stays in this folder. To uninstall:" -ForegroundColor DarkGray
Write-Host "    Remove-Item -Recurse -Force `"$BITCLI_HOME`"" -ForegroundColor DarkGray
Write-Host ""

# ── Step 1: Create directory layout ─────────────────────────────────────────
Write-Step "Creating BitCLI home at $BITCLI_HOME"
New-Item -ItemType Directory -Force -Path $BITCLI_BIN | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $BITCLI_HOME "models") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $BITCLI_HOME "tools") | Out-Null
Write-Ok "Directory layout created"

# ── Step 2: Check git (required, not bundled) ────────────────────────────────
Write-Step "Checking for git"
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Abort "git is not installed.`n  Please install Git from https://git-scm.com/download/win`n  then re-run this installer."
}
Write-Ok "git found at $(Get-Command git | Select-Object -ExpandProperty Source)"

# ── Step 3: Download or build bitcli.exe ─────────────────────────────────────
Write-Step "Installing BitCLI binary"

$arch      = if ([System.Environment]::Is64BitOperatingSystem) { "x86_64" } else { "x86" }
$assetName = "bitcli-windows-$arch.exe"
$downloaded = $false

# Try GitHub Releases first.
if ($BITCLI_VERSION -eq "latest") {
    $apiUrl = "https://api.github.com/repos/$GITHUB_REPO/releases/latest"
    try {
        $release = Invoke-RestMethod -Uri $apiUrl -Headers @{ "User-Agent" = "bitcli-installer/1.0" } -ErrorAction Stop
        $asset   = $release.assets | Where-Object { $_.name -eq $assetName } | Select-Object -First 1
        if ($asset) {
            Write-Host "  Downloading release $($release.tag_name) ..."
            Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $BINARY -UseBasicParsing -ErrorAction Stop
            Write-Ok "bitcli.exe downloaded to $BINARY"
            $downloaded = $true
        }
    } catch {
        # No release yet — fall through to build-from-source.
    }
}

if (-not $downloaded) {
    Write-Warn "No pre-built release binary found. Building from source..."

    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Abort "Go is not installed and no pre-built binary is available.`n  Install Go from https://go.dev/dl/ then re-run this installer."
    }

    $tmpSrc = Join-Path $env:TEMP "bitcli-src-$(Get-Random)"

    Write-Host "  Cloning source from https://github.com/$GITHUB_REPO ..."
    # Redirect stderr->stdout so git progress does not trigger PowerShell errors.
    $cloneOut = & git clone --depth=1 "https://github.com/$GITHUB_REPO.git" $tmpSrc 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host $cloneOut
        Abort "git clone failed (exit $LASTEXITCODE). Check your internet connection."
    }

    Write-Host "  Building binary (this takes ~30s) ..."
    Push-Location $tmpSrc
    try {
        & go build -buildvcs=false -o $BINARY ./cmd/bitcli 2>&1 | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Abort "go build failed (exit $LASTEXITCODE)."
        }
        Write-Ok "bitcli.exe built and installed at $BINARY"
    } finally {
        Pop-Location
        Remove-Item -Recurse -Force $tmpSrc -ErrorAction SilentlyContinue
    }
}

# ── Step 4: Add ~/.bitcli/bin to PATH for this session ───────────────────────
Write-Step "Configuring PATH for this session"
$env:PATH = "$BITCLI_BIN;$env:PATH"
Write-Ok "PATH updated for current session"

# ── Step 5: Run bitcli setup ─────────────────────────────────────────────────
Write-Step "Running bitcli setup (installs cmake, clang, uv, BitNet backend)"
Write-Host "  This may take several minutes on first run." -ForegroundColor DarkGray
Write-Host ""
& $BINARY setup
if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "  bitcli setup exited with code $LASTEXITCODE." -ForegroundColor Yellow
    Write-Host "  Run  bitcli setup  manually to retry." -ForegroundColor Yellow
}

# ── Step 6: Done ─────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "  ─────────────────────────────────────────────────────────" -ForegroundColor DarkGray
Write-Host "  BitCLI is installed!" -ForegroundColor Green
Write-Host ""
Write-Host "  To make BitCLI available in all future PowerShell sessions," -ForegroundColor White
Write-Host "  add this line to your PowerShell profile:" -ForegroundColor White
Write-Host ""
Write-Host "    . `"$BITCLI_HOME\env.ps1`"" -ForegroundColor Yellow
Write-Host ""
Write-Host "  To open your profile for editing:" -ForegroundColor White
Write-Host "    notepad `$PROFILE" -ForegroundColor DarkGray
Write-Host ""
Write-Host "  To uninstall everything:" -ForegroundColor White
Write-Host "    Remove-Item -Recurse -Force `"$BITCLI_HOME`"" -ForegroundColor DarkGray
Write-Host ""
