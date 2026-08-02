# BitCLI Installation Guide

BitCLI uses a fully self-contained install model — everything lives inside a single directory (`~/.bitcli/`). To uninstall, delete that directory. Nothing is written to system-wide paths.

---

## Requirements

- **Git** — required to clone the Microsoft BitNet backend
- **Internet access** — for downloading models and tools
- **Windows 10/11** or **Linux** or **macOS**

---

## Install

### Windows (PowerShell — one line)

```powershell
irm https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.ps1 | iex
```

### Linux / macOS (one line)

```bash
curl -fsSL https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.sh | sh
```

Both installers:
1. Create `~/.bitcli/` with the full directory layout
2. Download the `bitcli` binary into `~/.bitcli/bin/`
3. Run `bitcli setup` which installs cmake, clang, and uv into `~/.bitcli/tools/`
4. Clone the official Microsoft BitNet backend into `~/.bitcli/backends/bitnet/current/`
5. Write `env.ps1` / `env.sh` activation scripts

---

## What Gets Installed Where

```
~/.bitcli/                              ← Delete this = uninstall everything
├── bin/
│   └── bitcli[.exe]                   ← The BitCLI CLI binary
├── tools/
│   ├── cmake/                         ← Portable cmake (if not found system-wide)
│   ├── llvm/bin/                      ← Portable LLVM/clang (if not found system-wide)
│   └── uv/                            ← uv Python manager (always bundled, ~5 MB)
├── backends/
│   └── bitnet/current/                ← Cloned Microsoft BitNet repo
├── models/
│   └── microsoft/
│       └── bitnet-b1.58-2B-4T-gguf/  ← Downloaded model weights
├── downloads/                         ← Partial download temp files
├── config.yaml                        ← BitCLI configuration
├── bitcli.db                          ← SQLite database (models, history)
├── env.ps1                            ← PowerShell activation script
└── env.sh                             ← POSIX shell activation script
```

---

## Post-Install: Activate Tools in Future Shells

The installer activates the bundled tools for the current session automatically. To activate them in every new terminal:

**Windows (PowerShell)**
```powershell
# Add to your PowerShell profile ($PROFILE):
. "$HOME\.bitcli\env.ps1"
```

**Linux / macOS**
```bash
# Add to ~/.bashrc or ~/.zshrc:
. "$HOME/.bitcli/env.sh"
```

> **Note**: You do not need to do this to use `bitcli` commands — the binary automatically injects its tool paths at startup.

---

## Manual Setup

If you already have `bitcli` built from source, you can run setup manually at any time:

```bash
# Full setup (install tools + clone BitNet backend)
bitcli setup

# Check setup result
bitcli doctor

# Skip cloning the BitNet backend (if you manage it yourself)
bitcli setup --skip-backend

# Force re-download of portable tools
bitcli setup --force
```

---

## Uninstall

### Windows
```powershell
# Interactive uninstaller with confirmation prompt
.\scripts\uninstall.ps1

# Or just delete the folder directly
Remove-Item -Recurse -Force "$HOME\.bitcli"
```

### Linux / macOS
```bash
# Interactive uninstaller with confirmation prompt
sh scripts/uninstall.sh

# Or just delete the folder directly
rm -rf "$HOME/.bitcli"
```

Uninstalling removes:
- The `bitcli` binary
- All downloaded model weights
- The Microsoft BitNet backend clone
- Bundled tools (cmake, clang, uv)
- SQLite database, config file, chat history

---

## Custom Install Directory

By default BitCLI installs to `~/.bitcli`. You can override this:

**Windows**
```powershell
$env:BITCLI_HOME = "D:\BitCLI"
irm https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.ps1 | iex
```

**Linux / macOS**
```bash
BITCLI_HOME=/opt/bitcli curl -fsSL https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.sh | sh
```

---

## Offline / Air-Gapped Setup

If you cannot access the internet during setup:

1. Build `bitcli` from source on a machine with internet access:
   ```bash
   go build -buildvcs=false -o bitcli ./cmd/bitcli
   ```
2. Copy the binary to `~/.bitcli/bin/bitcli` on the target machine
3. Pre-download portable tool archives and extract them into `~/.bitcli/tools/`
4. Clone Microsoft BitNet manually into `~/.bitcli/backends/bitnet/current/`
5. Copy model GGUF files into `~/.bitcli/models/`

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `bitcli setup` says git is missing | Install Git: https://git-scm.com/download |
| `bitnet.cpp: missing` after setup | Re-run `bitcli setup` or `bitcli update backend bitnet` |
| `cmake: missing` in `bitcli doctor` | `bitcli setup` will download portable cmake |
| `clang: missing` in `bitcli doctor` | `bitcli setup` will download portable LLVM |
| Build error on Windows | Run `go build -buildvcs=false ...` |
| Models folder missing | `bitcli doctor` creates it automatically |
