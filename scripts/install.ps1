# BitCLI Installer for Windows
# Usage (one-liner from PowerShell):
#   irm https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.ps1 | iex
#
# Everything is installed into %USERPROFILE%\.bitcli\
# To uninstall completely: Remove-Item -Recurse -Force "$HOME\.bitcli"

$ProgressPreference = "SilentlyContinue"

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
function Write-Ok([string]$msg)   { Write-Host "  v  $msg" -ForegroundColor Green }
function Write-Warn([string]$msg) { Write-Host "  !  $msg" -ForegroundColor Yellow }
function Abort([string]$msg) {
    Write-Host ""
    Write-Host "  ERROR: $msg" -ForegroundColor Red
    Write-Host ""
    exit 1
}

# ── Invoke-Git: runs git and redirects stderr to a temp file ─────────────────
# Avoids PowerShell NativeCommandError which fires on any git stderr output.
function Invoke-Git {
    param([string[]]$Arguments, [string]$WorkDir = $PWD)
    $errFile = [System.IO.Path]::GetTempFileName()
    try {
        $proc = Start-Process -FilePath "git" `
            -ArgumentList $Arguments `
            -WorkingDirectory $WorkDir `
            -Wait -PassThru -NoNewWindow `
            -RedirectStandardError $errFile
        if ($proc.ExitCode -ne 0) {
            $errText = Get-Content $errFile -Raw -ErrorAction SilentlyContinue
            Abort "git $($Arguments[0]) failed (exit $($proc.ExitCode)): $errText"
        }
    } finally {
        Remove-Item $errFile -ErrorAction SilentlyContinue
    }
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

# ── Step 2: Check git ────────────────────────────────────────────────────────
Write-Step "Checking for git"
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Abort "git is not installed.`n  Install from https://git-scm.com/download/win then re-run."
}
Write-Ok "git found at $(Get-Command git | Select-Object -ExpandProperty Source)"

# ── Step 3: Get bitcli.exe ───────────────────────────────────────────────────
Write-Step "Installing BitCLI binary"

$arch      = if ([System.Environment]::Is64BitOperatingSystem) { "x86_64" } else { "x86" }
$assetName = "bitcli-windows-$arch.exe"
$gotBinary = $false

# Try GitHub Releases first.
try {
    $apiUrl  = "https://api.github.com/repos/$GITHUB_REPO/releases/latest"
    $release = Invoke-RestMethod -Uri $apiUrl -Headers @{ "User-Agent" = "bitcli-installer/1.0" } -ErrorAction Stop
    $asset   = $release.assets | Where-Object { $_.name -eq $assetName } | Select-Object -First 1
    if ($asset) {
        Write-Host "  Downloading release $($release.tag_name) ..."
        Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $BINARY -UseBasicParsing -ErrorAction Stop
        Write-Ok "bitcli.exe downloaded to $BINARY"
        $gotBinary = $true
    } else {
        Write-Warn "Release $($release.tag_name) found but no Windows binary yet."
    }
} catch {
    Write-Warn "No GitHub release available. Will build from source."
}

# Build from source fallback.
if (-not $gotBinary) {
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Abort "Go is not installed and no pre-built binary is available.`n  Install Go from https://go.dev/dl/ then re-run."
    }

    $tmpSrc = Join-Path $env:TEMP "bitcli-src-$(Get-Random)"
    Write-Host "  Cloning repository (shallow) ..."

    Invoke-Git -Arguments @("clone", "--depth=1", "https://github.com/$GITHUB_REPO.git", $tmpSrc)

    Write-Host "  Building binary (~30s) ..."
    $buildErrFile = [System.IO.Path]::GetTempFileName()
    try {
        $proc = Start-Process -FilePath "go" `
            -ArgumentList @("build", "-buildvcs=false", "-o", $BINARY, "./cmd/bitcli") `
            -WorkingDirectory $tmpSrc `
            -Wait -PassThru -NoNewWindow `
            -RedirectStandardError $buildErrFile
        if ($proc.ExitCode -ne 0) {
            $errText = Get-Content $buildErrFile -Raw -ErrorAction SilentlyContinue
            Abort "go build failed (exit $($proc.ExitCode)):`n$errText"
        }
    } finally {
        Remove-Item $buildErrFile -ErrorAction SilentlyContinue
        Remove-Item -Recurse -Force $tmpSrc -ErrorAction SilentlyContinue
    }
    Write-Ok "bitcli.exe built and installed at $BINARY"
}

# ── Step 4: Add to PATH for this session ─────────────────────────────────────
Write-Step "Configuring PATH for this session"
$env:PATH = "$BITCLI_BIN;$env:PATH"
Write-Ok "PATH updated for current session"

# ── Step 5: Run bitcli setup ─────────────────────────────────────────────────
Write-Step "Running bitcli setup"
Write-Host "  This installs cmake, clang, uv, and the BitNet backend." -ForegroundColor DarkGray
Write-Host "  May take several minutes on first run." -ForegroundColor DarkGray
Write-Host ""
& $BINARY setup
if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Warn "bitcli setup exited with code $LASTEXITCODE — run  bitcli setup  to retry."
}

# ── Step 6: Done ─────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "  ─────────────────────────────────────────────────────────" -ForegroundColor DarkGray
Write-Host "  BitCLI is installed!" -ForegroundColor Green
Write-Host ""
Write-Host "  For future sessions, add this to your PowerShell profile:" -ForegroundColor White
Write-Host "    . `"$BITCLI_HOME\env.ps1`"" -ForegroundColor Yellow
Write-Host ""
Write-Host "  Open your profile with:  notepad `$PROFILE" -ForegroundColor DarkGray
Write-Host "  Uninstall with:  Remove-Item -Recurse -Force `"$BITCLI_HOME`"" -ForegroundColor DarkGray
Write-Host ""
