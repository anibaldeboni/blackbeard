BINARY_NAME := flint
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w \
    -X main.version=$(VERSION) \
    -X main.commit=$(COMMIT) \
    -X main.buildTime=$(BUILD_TIME)

GO      := go
GOFLAGS := -trimpath
MAIN    := ./cmd/flint

.PHONY: build build-arm64 build-all test test-race lint fmt tidy clean install deploy help

build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) $(MAIN)

build-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-arm64 $(MAIN)

build-all: build build-arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-amd64 $(MAIN)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-darwin-arm64 $(MAIN)

test:
	$(GO) test ./... -v -count=1

test-race:
	$(GO) test ./... -v -race -count=1

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin/

install:
	$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS)" $(MAIN)

deploy: build-arm64
	@if [ -z "$(SSH_TARGET)" ]; then echo "Set SSH_TARGET=user@host"; exit 1; fi
	scp bin/$(BINARY_NAME)-linux-arm64 $(SSH_TARGET):~/blackbeard/flint
	ssh $(SSH_TARGET) "chmod +x ~/blackbeard/flint"

help:
	@echo "Targets:"
	@echo "  build       - Build for current platform"
	@echo "  build-arm64 - Build for OrangePi (linux/arm64)"
	@echo "  build-all   - Build for all platforms"
	@echo "  test        - Run unit tests"
	@echo "  test-race   - Run tests with race detector"
	@echo "  lint        - Run golangci-lint"
	@echo "  fmt         - Format code"
	@echo "  tidy        - Run go mod tidy"
	@echo "  clean       - Remove build artifacts"
	@echo "  install     - Install to GOPATH/bin"
	@echo "  deploy      - Deploy to OrangePi via SSH"
