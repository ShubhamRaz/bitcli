# BitCLI

BitCLI is an Ollama-like runtime and model manager for Microsoft BitNet models. It provides a modern Go CLI, local HTTP API, model cache, downloader, SQLite metadata store, hardware diagnostics, chat history, and a backend adapter for Microsoft's official `bitnet.cpp`.

BitCLI does not implement inference and does not modify BitNet. Model execution is delegated to the official Microsoft BitNet repository.

## Install

**Windows** (PowerShell — one line, requires Git):
```powershell
irm https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.ps1 | iex
```

**Linux / macOS** (one line, requires Git):
```bash
curl -fsSL https://raw.githubusercontent.com/ShubhamRaz/bitcli/main/scripts/install.sh | sh
```

Everything is installed into `~/.bitcli/`. To uninstall completely, delete that folder.
See [docs/install.md](docs/install.md) for details including custom directories and offline setup.


## Status

This repository is a production-oriented, feature-complete implementation of BitCLI. It includes the core architecture, CLI, API server, model manager, downloader, backend abstraction, BitNet adapter, hardware detection, chat history, unit tests, integration tests, and documentation.

> **Build Requirement**: Go 1.21+ is required (Go 1.25+ preferred). Run `go mod tidy` once after cloning to generate the `go.sum` dependency lockfile.

## Requirements

- **Go 1.21+** (Go 1.25 recommended) — [Install Go](https://go.dev/dl/)
- Git
- Python, CMake, and Clang for the official BitNet backend (model inference only)
- Platform build tools required by Microsoft BitNet

## Build

```bash
# Fetch dependencies and generate go.sum
go mod tidy

# Build the binary
go build -o bin/bitcli ./cmd/bitcli

# Run all tests
go test ./...

# Or use the Makefile
make all
```

## Quick Start

```bash
go build -o bin/bitcli ./cmd/bitcli

bitcli update backend bitnet
bitcli pull microsoft/BitNet-b1.58-2B-4T
bitcli run microsoft/BitNet-b1.58-2B-4T --prompt "Explain BitNet in one paragraph."
bitcli chat
bitcli serve
```

The default model cache is `~/.bitcli/models` on Windows, Linux, and macOS.

## Commands

```bash
bitcli setup
bitcli setup --skip-backend
bitcli pull MODEL
bitcli run [MODEL] --prompt "Hello"
bitcli chat
bitcli chat history
bitcli chat delete SESSION_ID
bitcli serve
bitcli list
bitcli remove MODEL
bitcli doctor
bitcli version
bitcli config path
bitcli config get runtime.temperature
bitcli config set runtime.temperature 0.7
bitcli update backend bitnet
```

## API

BitCLI serves Ollama-compatible endpoints:

- `POST /api/chat`
- `POST /api/generate`
- `GET /api/models`
- `DELETE /api/models/{id}`
- `GET /api/version`

It also serves:

- `POST /v1/chat/completions`

The server binds to `127.0.0.1:11434` by default.

## Backend Policy

BitCLI only wraps official BitNet commands:

- `python setup_env.py -md <model-dir> -q <quant-type>`
- `python run_inference.py -m <gguf-path> -p <prompt> ...`

The backend interface is designed so future adapters can support llama.cpp, MLC, ONNX Runtime, or TensorRT-LLM without changing CLI commands.

## License

MIT

