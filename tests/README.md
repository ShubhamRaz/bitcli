# BitCLI Tests

Most tests live beside their packages under `internal/`.

Recommended checks once Go is installed:

```bash
go test ./...
go test -race ./...
go vet ./...
```

Networked integration tests should use a fake Hugging Face server and fake BitNet process unless explicitly testing the official backend on a prepared machine.

