# AGENTS.md

This file provides guidance to agents when working with this repository.

## Overview

Blackbeard is a Docker-based home media server stack optimized for an **OrangePi 3B (RK3566, ARM64)** host, with Jellyfin hardware acceleration via V4L2M2M (Hantro VPU).

> **IMPORTANT:** This stack runs on a **remote OrangePi host**. Never run deployment or Docker commands locally. Instead, provide step-by-step instructions for the user to execute on the remote host (via SSH) and wait for their response before proceeding.

## Architecture

- **Host:** OrangePi 3B (RK3566, ARM64, Linux)
- **Container runtime:** Docker Engine 20.10+ with Docker Compose v2
- **Source control:** Git
- **External network:** `jollyroger` (must exist before starting the stack)
- **Project name:** `media-stack`
- **Config directory:** `config/` (bind-mounted into containers)
- **Nginx config:** `config/nginx/nginx.conf`

## Services

| Service | Port(s) | Purpose |
|---|---:|---|
| qBittorrent | 5080, 6881/tcp+udp | Torrent client |
| Radarr | 7878 | Movie management |
| Sonarr | 8989 | TV management |
| Prowlarr | 9696 | Indexer manager |
| Bazarr | 6767 | Subtitle management |
| Jellyfin | 8096, 7359/udp, 8920 | Media server (V4L2M2M HW transcode) |
| Seerr | 5055 | Media request management |
| Profilarr | 6868 | Quality profile manager |
| FlareSolverr | 8191 | Cloudflare bypass proxy |
| Nginx | 80, 443 | Reverse proxy |
| Watchtower | — | Automatic container image updates |
| Netdata | 19999 (host network) | Host monitoring |

### Service Startup Order

```
qbittorrent + flaresolverr → prowlarr → radarr + sonarr → bazarr → jellyfin → seerr + profilarr → nginx
```

## Key Files

| Path | Purpose |
|---|---|
| `docker-compose.yml` | Full stack definition (13 services) |
| `.env` | Environment variables (PUID, PGID, TZ, paths, GPU groups, resource limits) |
| `.env.example` | Template for `.env` |
| `config/nginx/nginx.conf` | Nginx reverse proxy configuration |
| `config/nginx/www/` | Static files served by Nginx |

## Environment Variables

Critical variables that must be set in `.env`:

- `PUID` / `PGID` — user/group IDs on the host
- `TZ` — timezone (e.g. `America/Sao_Paulo`)
- `DOWNLOADS_PATH` — path to downloads directory on host
- `CONFIG_BASE_PATH` — base path for config bind mounts
- `GPU_VIDEO_GROUP` — GID of the `video` group (commonly `44`)
- `GPU_RENDER_GROUP` — GID of the `render` group (commonly `105`)

Per-service CPU and memory limits are also configurable (e.g. `JELLYFIN_CPU_LIMIT`, `JELLYFIN_MEM_LIMIT`).

## Common Docker Compose Commands

```bash
# Start the stack
docker compose up -d

# Check service status
docker compose ps

# Follow logs (all services or one)
docker compose logs -f --tail=100
docker compose logs -f --tail=100 jellyfin

# Restart all or one service
docker compose restart
docker compose restart jellyfin

# Stop / start
docker compose stop
docker compose start

# Validate compose config
docker compose config

# Pull latest images and recreate
docker compose pull
docker compose up -d --force-recreate
```

## Nginx Routes

All services are proxied via Nginx at these subpaths:

- `/jellyfin/`
- `/seerr`
- `/radarr`
- `/sonarr`
- `/bazarr/`
- `/prowlarr`
- `/qbittorrent/`
- `/assets/` (landing page static assets mounted from `config/nginx/www/assets/`)

When adding or changing routes, edit `config/nginx/nginx.conf`. Services that run behind a subpath (Radarr, Sonarr, Prowlarr, Bazarr) require their base URL to be configured in their own UI settings after first login.

## Backup / Restore Pattern

Volumes with label `backup.enable=true` are the backup targets. Use an Alpine container with `tar` to create or restore archives in the `backups/` directory.

```bash
# Backup a volume
docker run --rm \
  -v <volume-name>:/data:ro \
  -v "$(pwd)/backups:/backup" \
  alpine \
  sh -c 'tar czf /backup/<volume-name>-$(date +%Y%m%d-%H%M%S).tar.gz -C /data .'

# Restore a volume (stop service first)
docker compose stop <service>
docker run --rm \
  -v <volume-name>:/data \
  -v "$(pwd)/backups:/backup:ro" \
  alpine \
  sh -c 'rm -rf /data/* && tar xzf /backup/<archive>.tar.gz -C /data'
docker compose start <service>
```

## Jellyfin Hardware Acceleration

Jellyfin uses the RK3566 VPU via V4L2M2M. The following devices are passed through:

- `/dev/dri` — Mali GPU (rendering)
- `/dev/video0` — Rockchip RGA (graphics accelerator)
- `/dev/video1` — Hantro VPU decoder
- `/dev/video2` — Hantro VPU encoder
- `/dev/media0`, `/dev/media1` — media controller nodes

To troubleshoot:

```bash
ls -la /dev/dri
ls -la /dev/video*
getent group video
getent group render
docker compose logs --tail=200 jellyfin
```

## Deployment Workflow

Since the stack runs on a remote host, always guide deployments as follows:

1. Present each step clearly with the exact command to run.
2. Wait for the user to confirm success or report errors before proceeding to the next step.
3. Never assume a command succeeded without user confirmation.

Typical deploy after a change:

```bash
# On the remote OrangePi host
cd ~/blackbeard
git pull
docker compose pull          # optional, only if images changed
docker compose up -d
docker compose ps
```

## Conventions

- All container images use `restart: unless-stopped`.
- Watchtower manages automatic updates via the `com.centurylinklabs.watchtower.enable=true` label.
- All services share the `jollyroger` external network.
- Resource limits (CPU and memory) are defined per-service via `.env` variables.
+ Health checks are defined for every service; allow extra `start_period` time on first boot.

## Documentation Maintenance

Whenever significant changes are made to `docker-compose.yml` or the project structure, **both `AGENTS.md` and `README.md` must be updated** to reflect those changes. This includes:

- Adding, removing, or renaming services
- Changing exposed ports or adding new ones
- Adding or removing environment variables in `.env.example`
- Adding new volume mounts or bind-mount paths
- Changing the external network name or project name
- Adding new Nginx routes or modifying existing ones
- Changes to the service startup order or `depends_on` relationships
- Modifications to the directory structure under `config/`