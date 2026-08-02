VERSION   ?= 0.1.0
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE      ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)
LDFLAGS   := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
BINARY    := bin/bitcli

.PHONY: all build test race vet lint clean tidy help

all: tidy build test vet

## build: Compile the bitcli binary into bin/
build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/bitcli

## test: Run all unit tests
test:
	go test ./...

## race: Run tests with the data-race detector
race:
	go test -race ./...

## vet: Run go vet static analysis
vet:
	go vet ./...

## lint: Run golangci-lint (requires golangci-lint to be installed)
lint:
	golangci-lint run ./...

## tidy: Fetch modules and update go.sum
tidy:
	go mod tidy

## clean: Remove compiled artifacts
clean:
	rm -rf bin/

## help: Show this message
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
