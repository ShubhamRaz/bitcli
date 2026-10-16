# BitCLI Configuration Reference

The BitCLI configuration file is a YAML document stored at `~/.bitcli/config.yaml` on all platforms.

Run `bitcli config path` to print the active path. Run `bitcli config init` to create a default file.

All keys can also be overridden with environment variables using the `BITCLI_` prefix and `_` in place of `.` (e.g. `BITCLI_API_PORT=8080`).

---

## Top-Level Keys

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `version` | int | `1` | Config schema version (must be `1`) |
| `default_model` | string | `microsoft/BitNet-b1.58-2B-4T` | Model used when no `--model` flag is given |
| `default_backend` | string | `bitnet` | Backend used for inference |
| `theme` | string | `auto` | Reserved for future TUI theming |

---

## `api` — Local HTTP Server

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `api.host` | string | `127.0.0.1` | Address the API server binds to |
| `api.port` | int | `11434` | Port the API server listens on |
| `api.allow_origins` | list | `[]` | CORS allowed origins (empty = disabled) |

**Environment**: `BITCLI_API_HOST`, `BITCLI_API_PORT`

---

## `runtime` — Inference Options

Defaults applied to every inference call unless overridden per-request.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `runtime.temperature` | float | `0.8` | Sampling temperature (0–2) |
| `runtime.top_p` | float | `0.9` | Nucleus sampling value (0–1) |
| `runtime.top_k` | int | `40` | Top-k sampling (not supported by `run_inference.py`) |
| `runtime.threads` | int | `0` | CPU threads (0 = auto) |
| `runtime.gpu_layers` | int | `0` | GPU offload layers (not supported yet) |
| `runtime.context_length` | int | `4096` | Context window size in tokens |
| `runtime.max_tokens` | int | `128` | Maximum tokens to generate |

---

## `backend.bitnet` — Official BitNet Backend

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `backend.bitnet.install_mode` | string | `auto` | `auto` = managed clone, `manual` = use `path` |
| `backend.bitnet.path` | string | `""` | Override path to an existing `bitnet.cpp` checkout |
| `backend.bitnet.repo_url` | string | `https://github.com/microsoft/BitNet.git` | Git URL for the official BitNet repository |
| `backend.bitnet.revision` | string | `""` | Git revision to check out (empty = default branch) |
| `backend.bitnet.python` | string | `""` | Python executable (empty = `python`) |
| `backend.bitnet.quant_type` | string | `i2_s` | Quantization type passed to `setup_env.py` |

---

## `download` — Hugging Face Hub

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `download.mirror` | string | `https://huggingface.co` | Base URL for model downloads |
| `download.concurrency` | int | `4` | Maximum parallel download connections |
| `download.retries` | int | `3` | Maximum retry attempts per file |
| `download.token_env` | string | `HF_TOKEN` | Environment variable for the HF access token |

---

## `logging` — Structured Logs

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `logging.level` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `logging.file` | bool | `true` | Write JSON logs to `~/.bitcli/logs/bitcli.log` |

---

## Example Config File

```yaml
version: 1
default_model: microsoft/BitNet-b1.58-2B-4T
default_backend: bitnet

api:
  host: 127.0.0.1
  port: 11434
  allow_origins: []

runtime:
  temperature: 0.8
  top_p: 0.9
  top_k: 40
  threads: 0
  gpu_layers: 0
  context_length: 4096
  max_tokens: 128

backend:
  bitnet:
    install_mode: auto
    path: ""
    repo_url: https://github.com/microsoft/BitNet.git
    revision: ""
    python: ""
    quant_type: i2_s

download:
  mirror: https://huggingface.co
  concurrency: 4
  retries: 3
  token_env: HF_TOKEN

logging:
  level: info
  file: true
```

---

## Setting Values from the CLI

```bash
bitcli config get runtime.temperature
bitcli config set runtime.temperature 0.7
bitcli config set api.port 8080
bitcli config path
```
