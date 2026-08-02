# BitCLI Installer for Windows
# Usage (one-liner from PowerShell):
#   irm https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.ps1 | iex
#
# Everything is installed into %USERPROFILE%\.bitcli\
# To uninstall completely: Remove-Item -Recurse -Force "$HOME\.bitcli"

$ErrorActionPreference = "Stop"
$ProgressPreference    = "SilentlyContinue"  # Speeds up Invoke-WebRequest

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
    Write-Host "  ✓  $msg" -ForegroundColor Green
}

function Write-Warn([string]$msg) {
    Write-Host "  !  $msg" -ForegroundColor Yellow
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
foreach ($dir in @($BITCLI_BIN, (Join-Path $BITCLI_HOME "models"), (Join-Path $BITCLI_HOME "tools"))) {
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
}
Write-Ok "Directory layout created"

# ── Step 2: Check git (required, not bundled) ────────────────────────────────
Write-Step "Checking for git"
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Host ""
    Write-Host "  ERROR: git is not installed." -ForegroundColor Red
    Write-Host "  Please install Git for Windows from https://git-scm.com/download/win" -ForegroundColor Red
    Write-Host "  then re-run this installer." -ForegroundColor Red
    exit 1
}
Write-Ok "git found at $(Get-Command git | Select-Object -ExpandProperty Source)"

# ── Step 3: Download bitcli.exe binary ──────────────────────────────────────
Write-Step "Downloading BitCLI binary"

$arch = if ([System.Environment]::Is64BitOperatingSystem) { "x86_64" } else { "x86" }
$assetName = "bitcli-windows-$arch.exe"

if ($BITCLI_VERSION -eq "latest") {
    $apiUrl  = "https://api.github.com/repos/$GITHUB_REPO/releases/latest"
    try {
        $release = Invoke-RestMethod -Uri $apiUrl -Headers @{ "User-Agent" = "bitcli-installer" }
        $asset   = $release.assets | Where-Object { $_.name -eq $assetName } | Select-Object -First 1
        if ($asset) {
            $downloadUrl = $asset.browser_download_url
            Write-Ok "Found release $($release.tag_name)"
        } else {
            throw "Asset $assetName not found in release"
        }
    } catch {
        # No GitHub release yet — build from source as fallback
        Write-Warn "No pre-built release found. Building from source..."
        $downloadUrl = $null
    }
} else {
    $downloadUrl = "https://github.com/$GITHUB_REPO/releases/download/$BITCLI_VERSION/$assetName"
}

if ($downloadUrl) {
    Write-Host "  Downloading from $downloadUrl ..."
    Invoke-WebRequest -Uri $downloadUrl -OutFile $BINARY -UseBasicParsing
    Write-Ok "bitcli.exe downloaded to $BINARY"
} else {
    # Fallback: build from source if Go is available
    if (Get-Command go -ErrorAction SilentlyContinue) {
        Write-Host "  Building bitcli from source (Go detected)..."
        $tmpSrc = Join-Path $env:TEMP "bitcli-src-$(Get-Random)"
        git clone --depth=1 "https://github.com/$GITHUB_REPO.git" $tmpSrc 2>&1 | Out-Null
        Push-Location $tmpSrc
        try {
            go build -buildvcs=false -o $BINARY ./cmd/bitcli
            Write-Ok "bitcli.exe built and installed"
        } finally {
            Pop-Location
            Remove-Item -Recurse -Force $tmpSrc -ErrorAction SilentlyContinue
        }
    } else {
        Write-Host ""
        Write-Host "  ERROR: No pre-built binary found and Go is not installed." -ForegroundColor Red
        Write-Host "  Install Go from https://go.dev/dl/ then re-run this installer." -ForegroundColor Red
        exit 1
    }
}

# ── Step 4: Add ~/.bitcli/bin to PATH for this session ───────────────────────
Write-Step "Configuring PATH for this session"
$env:PATH = "$BITCLI_BIN;$env:PATH"
Write-Ok "PATH updated (current session)"

# ── Step 5: Run bitcli setup (installs cmake, clang, uv, BitNet backend) ─────
Write-Step "Running bitcli setup (this may take several minutes)"
Write-Host ""
& $BINARY setup
if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "  ERROR: bitcli setup failed (exit $LASTEXITCODE)." -ForegroundColor Red
    Write-Host "  Run  bitcli setup  manually once the issue is resolved." -ForegroundColor Red
    exit $LASTEXITCODE
}

# ── Step 6: Permanent PATH instructions ──────────────────────────────────────
Write-Host ""
Write-Host "  ─────────────────────────────────────────────────────────" -ForegroundColor DarkGray
Write-Host "  BitCLI is installed!" -ForegroundColor Green
Write-Host ""
Write-Host "  The installer added BitCLI tools to PATH for this session." -ForegroundColor White
Write-Host "  To make it permanent, add this line to your PowerShell profile:" -ForegroundColor White
Write-Host ""
Write-Host "    . `"$BITCLI_HOME\env.ps1`"" -ForegroundColor Yellow
Write-Host ""
Write-Host "  To open your profile for editing:" -ForegroundColor White
Write-Host "    notepad `$PROFILE" -ForegroundColor DarkGray
Write-Host ""
Write-Host "  To uninstall everything:" -ForegroundColor White
Write-Host "    Remove-Item -Recurse -Force `"$BITCLI_HOME`"" -ForegroundColor DarkGray
Write-Host ""
