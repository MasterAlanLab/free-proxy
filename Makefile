BINARY  := free-proxy
VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFLAGS := -trimpath

.PHONY: all build build-go frontend gen test vet cross clean tidy

all: build

## frontend: build the React app into internal/web/dist
frontend:
	cd frontend && bun install && bun run build

## gen: regenerate sqlc code (requires sqlc in PATH)
gen:
	cd internal/store && sqlc generate

## build: build the single static binary (frontend + go)
build: frontend build-go

## build-go: build the Go binary only (assumes frontend already built)
build-go:
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/$(BINARY) ./cmd/free-proxy

## test / vet
test:
	go test ./...
vet:
	go vet ./...

## cross: static Linux binaries for amd64 and arm64
cross: frontend
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 ./cmd/free-proxy
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64 ./cmd/free-proxy

clean:
	rm -rf dist internal/web/dist/assets

tidy:
	go mod tidy
