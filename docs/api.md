# BitCLI API Reference

BitCLI exposes a local HTTP API on `127.0.0.1:11434` (configurable via `api.host` and `api.port`).

The API is ollama-compatible: native endpoints match ollama's request/response
shapes, and ollama clients can use `GET /api/tags`, `POST /api/pull`,
`POST /api/delete`, `POST /api/show` directly. OpenAI-compatible endpoints are
served under `/v1/*`.

## Authentication

No authentication is required for local requests. CORS: when `api.allow_origins`
is empty, browser clients from `http://localhost` / `http://127.0.0.1` origins
are allowed (like ollama's default); set `api.allow_origins` to restrict or
extend this.

---

## Liveness

### `GET /`

Returns `200` with the body `BitCLI is running` (like ollama's root endpoint).

### `GET /health`

Returns `{"status":"ok"}`.

---

## Native REST Endpoints

### `POST /api/generate`

Generate a text completion from a prompt.

**Request**
```json
{
  "model": "microsoft/BitNet-b1.58-2B-4T",
  "prompt": "Explain 1-bit LLMs.",
  "stream": true,
  "options": {
    "temperature": 0.8,
    "num_ctx": 4096,
    "num_predict": 128
  }
}
```

**Streaming Response** (`application/x-ndjson`)

Each line is a JSON object:
```json
{"model":"microsoft/BitNet-b1.58-2B-4T","created_at":"2025-01-01T00:00:00Z","response":"1-bit","done":false}
{"model":"microsoft/BitNet-b1.58-2B-4T","created_at":"2025-01-01T00:00:01Z","response":"","done":true}
```

**Non-Streaming Response** (`stream: false`)
```json
{"model":"microsoft/BitNet-b1.58-2B-4T","created_at":"2025-01-01T00:00:01Z","response":"1-bit LLMs use...","done":true,"done_reason":"stop","total_duration":123456789}
```

---

### `POST /api/chat`

Continue a multi-turn conversation.

**Request**
```json
{
  "model": "microsoft/BitNet-b1.58-2B-4T",
  "messages": [
    {"role": "user", "content": "What is BitNet?"}
  ],
  "stream": true
}
```

**Streaming Response**
```json
{"model":"...","created_at":"...","message":{"role":"assistant","content":"BitNet is"},"done":false}
{"model":"...","created_at":"...","message":{"role":"assistant","content":""},"done":true}
```

---

### `GET /api/models`

List locally installed models. `GET /api/tags` is an alias (ollama's primary
model-list endpoint) with an identical response.

**Response**
```json
{
  "models": [
    {
      "name": "microsoft/BitNet-b1.58-2B-4T",
      "model": "microsoft/BitNet-b1.58-2B-4T",
      "modified_at": "2025-01-01T00:00:00Z",
      "size": 1190000000,
      "digest": "sha256:...",
      "details": {
        "family": "BitNet b1.58",
        "parameter_size": "2.4B",
        "quantization_level": "i2_s"
      }
    }
  ]
}
```

---

### `DELETE /api/models/{name}`

Remove a locally installed model. When another installed model shares the same
artifact file (for example the base and `-chat` variants), the shared file is
kept for the surviving model.

**Response**: `200 OK` with `{"status":"success"}`

---

### `POST /api/pull` (ollama-compatible)

Download a model. Streams JSON-line progress while downloading.

**Request**
```json
{"model": "microsoft/BitNet-b1.58-2B-4T", "stream": true}
```

**Streaming Response** (`application/x-ndjson`)
```json
{"status":"pulling manifest"}
{"status":"downloading","completed":524288,"total":1190000000}
{"status":"verifying sha256 digest"}
{"status":"success"}
```

---

### `POST /api/delete` (ollama-compatible)

Remove a model by request body: `{"model": "microsoft/BitNet-b1.58-2B-4T"}`.

**Response**: `200 OK` with `{"status":"success"}`

---

### `POST /api/show` (ollama-compatible)

Show model details for an installed model.

**Request**: `{"model": "microsoft/BitNet-b1.58-2B-4T"}`

**Response** includes `details`, `model_info`, `parameters`, `modified_at`,
`size`, `digest`, `repo`, and `file_name`.

---

### `GET /api/version`

Return the BitCLI version.

**Response**
```json
{"version": "0.1.0"}
```

---

### `GET /api/hardware`

Return hardware detection results.

**Response** — see `hardware.Report` struct in `internal/hardware/detector.go`.

---

## OpenAI-Compatible Endpoints

### `POST /v1/chat/completions`

**Request** (OpenAI format). Sampling fields accept explicit zero values
(e.g. `"temperature": 0` for greedy decoding).
```json
{
  "model": "microsoft/BitNet-b1.58-2B-4T",
  "messages": [{"role": "user", "content": "Hello"}],
  "stream": false,
  "temperature": 0.8,
  "max_tokens": 64
}
```

**Non-Streaming Response**
```json
{
  "id": "chatcmpl_...",
  "object": "chat.completion",
  "created": 1700000000,
  "model": "microsoft/BitNet-b1.58-2B-4T",
  "choices": [
    {
      "index": 0,
      "finish_reason": "stop",
      "message": {"role": "assistant", "content": "Hello! How can I help?"}
    }
  ],
  "usage": {"prompt_tokens": 1, "completion_tokens": 8, "total_tokens": 9}
}
```

**Streaming Response** (`stream: true`) — SSE format. All chunks share one
`id`; the first chunk carries `"delta":{"role":"assistant"}`:
```
data: {"id":"chatcmpl_...","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}

data: [DONE]
```

---

### `POST /v1/completions`

Legacy text completions. Accepts `prompt` as a string or an array of strings.

**Request**
```json
{"model": "microsoft/BitNet-b1.58-2B-4T", "prompt": "Once upon a time", "stream": false}
```

**Response**: OpenAI `text_completion` object with `choices[].text`.

---

### `GET /v1/models`

OpenAI-format model list.

**Response**
```json
{"object":"list","data":[{"id":"microsoft/BitNet-b1.58-2B-4T","object":"model","created":1700000000,"owned_by":"bitcli"}]}
```

---

## Error Responses

All errors return a JSON envelope (OpenAI-shaped, with a `type` field):
```json
{
  "error": {
    "code": "model_not_found",
    "message": "model 'unknown/model' is not installed",
    "type": "invalid_request_error"
  }
}
```

HTTP status codes map to error classes: `404` for `model_not_found`,
`503` for `backend_not_found` / `unavailable`, `502` for download failures,
`400` for invalid input, `500` for internal errors — matching ollama's
behavior for unknown models.

| Code | Meaning |
|------|---------|
| `model_not_found` | Model is not in the local catalog or not installed |
| `backend_not_found` | BitNet checkout is missing; run `bitcli update backend bitnet` |
| `download_interrupted` | Network error during model download |
| `checksum_mismatch` | Downloaded file hash does not match expected SHA256 |
| `config_invalid` | Configuration file has an invalid value |
| `unavailable` | Backend setup script failed |
| `internal` | Unexpected internal error |
