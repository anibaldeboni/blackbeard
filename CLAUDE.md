# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Blackbeard is a Docker-based home media server stack (Jellyfin, Radarr, Sonarr, Prowlarr, qBittorrent, etc.) optimized for Orange Pi 3B (RK3566, ARM64) with V4L2M2M hardware transcoding. It includes `flint`, a Go CLI tool for stack management, hardware monitoring, backups, and Docker cleanup.

## Build & Development Commands

All CLI commands run from `cli/`:

```bash
cd cli/

make build          # Build for current platform → bin/flint
make build-arm64    # Cross-compile for OrangePi (linux/arm64, CGO_ENABLED=0)
make build-all      # All platforms (darwin-arm64, linux-amd64, linux-arm64)

make test           # go test ./... -v -count=1
make test-race      # go test ./... -v -race -count=1
make lint           # golangci-lint run ./...
make fmt            # go fmt ./...
make tidy           # go mod tidy

make deploy SSH_TARGET=user@host   # Build ARM64 + SCP to OrangePi
```

Run a single test: `cd cli && go test ./internal/backup/ -v -run TestSpecificName`

## CLI Architecture (`cli/`)

- **Framework:** Kong (struct-based CLI, NOT Cobra) — `cmd/flint/main.go`
- **Go 1.24**, module at `cli/go.mod`
- **Docker integration:** Compose v2 SDK + Engine API client (no shell subprocess calls)
- **Binary:** ~55MB due to compose v2 dependency tree

### Command Groups (~29 subcommands across 4 groups)

| Group | Package | Description |
|-------|---------|-------------|
| `stack` | `internal/stack/` | Lifecycle: install, start/stop/restart, status, health, logs, update, validate |
| `backup` | `internal/backup/` | Volume backup/restore using Alpine container + tar |
| `docker` | `internal/cleanup/` | Disk usage, image cleanup, dangling/prune/protected |
| `hw` | `internal/hw/` | CPU, memory, disk, network, temperature, GPU/VPU monitoring |

### Key Patterns

- **Shared context (`Ctx` struct):** Passed to all command `Run()` methods — holds `Config`, `Clients` (Docker), `Printer` (UI), `Yes` flag, and Go `context.Context`
- **Lazy Docker init:** Docker/Compose clients only initialized when the command actually needs them (e.g., `hw *` and `stack validate` skip Docker setup)
- **Centralized UI:** All terminal output goes through `internal/ui/Printer` — supports `--no-color`, styled headers with gradient, table formatting, confirmation prompts
- **Config resolution:** `--project-dir` flag → `BLACKBEARD_DIR` env → auto-discover by walking up from cwd; `.env` loaded via godotenv

### Docker Compose Integration

- Project name: `media-stack`, network: `jollyroger` (external)
- `LogConsumer` interface requires `Err()` method (not just `Log`/`Status`/`Register`) — this is a compose v2 SDK requirement
- `compose v2@v2.32.4` is the known-good version that resolves cleanly

### Hardware Monitoring

- Uses `gopsutil/v4` for system metrics
- Direct sysfs reads for GPU temperature fallback (`/sys/class/thermal/`)
- Temperature classified by sensor key: CPU, GPU, SOC
- Gracefully returns "N/A" on non-Linux (macOS development is fine)

## Docker Compose (`docker-compose.yml`)

13 services with health checks, dependency ordering, and per-service resource limits configured via `.env`. Service startup flows: qbittorrent+flaresolverr → prowlarr → radarr+sonarr → bazarr → jellyfin → jellyseerr+profilarr → nginx.

Volumes marked with `backup.enable=true` label are picked up by the backup system.

## Cross-Compilation

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o bin/flint-linux-arm64 ./cmd/flint
```

Version info (git tag, commit, build time) injected via `-ldflags -X`.
