# BitCLI Architecture

BitCLI is a Go CLI tool and local HTTP API server that manages and runs Microsoft BitNet 1-bit LLMs. It wraps the official `bitnet.cpp` Python scripts without modifying them.

---

## Design Principles

- **No inference code** — BitCLI never implements language model math; it delegates to the compiled BitNet backend.
- **Backend abstraction** — The `backend.Backend` interface makes it possible to swap in llama.cpp, ONNX Runtime, or any future runtime without CLI changes.
- **Local-first** — All data (models, chat history, downloads) lives in `~/.bitcli/`.
- **Standards-compliant API** — The API provides clean native REST endpoints and standard OpenAI-compatible endpoints.

---

## Package Map

```
bitcli/
├── cmd/bitcli/          CLI entry point and command wiring (Cobra)
│   ├── main.go          Binary entry
│   ├── root.go          Root command and global flags
│   ├── factory.go       Dependency injection (newApp)
│   ├── pull.go          bitcli pull
│   ├── run.go           bitcli run
│   ├── chat.go          bitcli chat / history / delete
│   ├── serve.go         bitcli serve
│   ├── list.go          bitcli list
│   ├── remove.go        bitcli remove
│   ├── doctor.go        bitcli doctor
│   ├── version.go       bitcli version
│   ├── config.go        bitcli config get/set/path/init
│   └── update.go        bitcli update backend bitnet
│
├── internal/
│   ├── api/             Gin HTTP server (BitCLI REST + OpenAI endpoints)
│   ├── cache/           On-disk model cache layout and safety checks
│   ├── config/          YAML config schema, defaults, validation, Viper service
│   ├── database/        SQLite connection, schema migrations, 3 repositories
│   ├── downloader/      Hugging Face Hub client, resumable downloads, verifier
│   ├── hardware/        CPU/GPU/memory detection (platform-specific files)
│   ├── logger/          Zap-based structured logging
│   ├── model/           Catalog, resolver, service, repository interface
│   ├── process/         exec.Cmd wrapper (stream + wait), Supervisor
│   ├── runtime/         Backend-neutral generation service
│   │   └── backend/     Backend interface, registry, type definitions
│   │       └── bitnet/  Official BitNet adapter (command builder, installer)
│   └── utils/           ID generation, SHA256, error codes, platform
│
├── tests/
│   └── integration/     End-to-end tests using httptest + in-memory SQLite
│
├── docs/                User-facing documentation
├── scripts/             Build and backend-install shell/PowerShell scripts
└── assets/              Static assets (logo etc.)
```

---

## Component Interaction Diagram

```
  User
    │
    ▼
 ┌──────────────────────────────────┐
 │   cmd/bitcli  (Cobra CLI)        │
 │                                  │
 │  pull │ run │ chat │ serve │ ... │
 └──────────────────┬───────────────┘
                    │ newApp() DI factory
                    ▼
 ┌──────────────────────────────────────────────────────────┐
 │  app struct                                              │
 │  ┌──────────┐  ┌──────────┐  ┌────────────────────────┐ │
 │  │  config  │  │ database │  │  model.Service         │ │
 │  │  Viper   │  │ SQLite   │  │  (catalog + repo)      │ │
 │  └──────────┘  └──────────┘  └────────────────────────┘ │
 │  ┌─────────────────────┐  ┌───────────────────────────┐  │
 │  │  downloader.Service │  │  runtime.Service          │  │
 │  │  HFClient + mpb     │  │  ┌──────────────────────┐ │  │
 │  └─────────────────────┘  │  │  backend.Registry    │ │  │
 │  ┌──────────────────────┐ │  │  bitnet.Backend      │ │  │
 │  │  hardware.Service    │ │  │  process.Runner      │ │  │
 │  └──────────────────────┘ │  └──────────────────────┘ │  │
 │                            └───────────────────────────┘  │
 └──────────────────────────────────────────────────────────┘
                    │
          (bitcli serve)
                    ▼
 ┌──────────────────────────────────┐
 │  api.Server  (Gin)               │
 │  POST /api/generate              │
 │  POST /api/chat                  │
 │  GET  /api/models                │
 │  POST /v1/chat/completions       │
 └──────────────────────────────────┘
                    │
          (inference request)
                    ▼
 ┌──────────────────────────────────┐
 │  bitnet.Backend                  │
 │  python run_inference.py -m ...  │
 │  (official Microsoft BitNet)     │
 └──────────────────────────────────┘
```

---

## Data Flow: `bitcli pull`

1. CLI resolves model ID via `model.Resolver` → `Catalog` → `Artifact`
2. `downloader.Service.PullModel()` HEAD-requests the HF Hub to get size + ETag
3. File downloaded with resume support (Range header) and `mpb` progress bar
4. File hash verified (`utils.SHA256File`)
5. `model.Service.Save()` persists metadata to SQLite

## Data Flow: `bitcli run`

1. `app.ensureModel()` — local lookup, auto-pull if missing
2. `runtime.Service.Generate()` — looks up backend via registry
3. `bitnet.Backend.Prepare()` — runs `setup_env.py` once per model+quant (idempotent marker file)
4. `bitnet.Backend.Generate()` — spawns `python run_inference.py -m <path> -p <prompt> ...`
5. `process.Runner.RunStream()` — pipes stdout/stderr line-by-line to `backend.TokenEvent` channel
6. CLI prints tokens as they arrive

## Data Flow: `bitcli serve`

Same as `run`, but wrapped in a Gin HTTP handler with NDJSON or SSE streaming.

---

## Adding a New Backend

1. Implement `backend.Backend` interface in a new `internal/runtime/backend/<name>/` package
2. Register it in `cmd/bitcli/factory.go` via `registry.Register()`
3. No CLI changes required — the backend ID maps automatically from model metadata

---

## Local Storage Layout

```
~/.bitcli/
├── config.yaml          YAML configuration file
├── bitcli.db            SQLite database (models, downloads, chat history)
├── logs/
│   └── bitcli.log       Structured JSON log (when logging.file=true)
├── models/
│   └── microsoft/
│       └── bitnet-b1.58-2B-4T-gguf/
│           └── main/
│               └── ggml-model-i2_s.gguf
├── downloads/           Partial download files (.partial)
├── backends/
│   └── bitnet/
│       └── current/     Managed official Microsoft BitNet git checkout
└── chats/               (Reserved for future use)
```
