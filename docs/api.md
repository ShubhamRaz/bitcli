# BitCLI API Reference

BitCLI exposes a local HTTP API on `127.0.0.1:11434` (configurable via `api.host` and `api.port`).

## Authentication

No authentication is required for local requests. CORS origins can be restricted via `api.allow_origins`.

---

## Ollama-Compatible Endpoints

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
{"model":"microsoft/BitNet-b1.58-2B-4T","response":"1-bit LLMs use...", "done":true}
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

List locally installed models.

**Response**
```json
{
  "models": [
    {
      "name": "microsoft/BitNet-b1.58-2B-4T",
      "model": "microsoft/BitNet-b1.58-2B-4T",
      "modified_at": "2025-01-01T00:00:00Z",
      "size": 0,
      "digest": "",
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

Remove a locally installed model.

**Response**: `204 No Content`

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

**Request** (OpenAI format)
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
  ]
}
```

**Streaming Response** (`stream: true`) — SSE format:
```
data: {"id":"chatcmpl_...","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hello"}}]}

data: [DONE]
```

---

## Error Responses

All errors return a JSON envelope:
```json
{
  "error": {
    "code": "model_not_found",
    "message": "model 'unknown/model' is not installed"
  }
}
```

| Code | Meaning |
|------|---------|
| `model_not_found` | Model is not in the local catalog or not installed |
| `backend_not_found` | BitNet checkout is missing; run `bitcli update backend bitnet` |
| `download_interrupted` | Network error during model download |
| `checksum_mismatch` | Downloaded file hash does not match expected SHA256 |
| `config_invalid` | Configuration file has an invalid value |
| `unavailable` | Backend setup script failed |
| `internal` | Unexpected internal error |
