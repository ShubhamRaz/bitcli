# Clone the official Microsoft BitNet repository into the managed BitCLI backend path.
$ErrorActionPreference = "Stop"
$homePath = if ($env:BITCLI_HOME) { $env:BITCLI_HOME } else { Join-Path $HOME ".bitcli" }
$target = Join-Path $homePath "backends/bitnet/current"
New-Item -ItemType Directory -Force -Path (Split-Path $target) | Out-Null
if (-not (Test-Path (Join-Path $target ".git"))) {
    git clone --recursive https://github.com/microsoft/BitNet.git $target
} else {
    git -C $target pull --ff-only
    git -C $target submodule update --init --recursive
}
Write-Host "BitNet backend ready at $target"

