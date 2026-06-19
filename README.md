# Blackbeard Media Stack

Docker-based home media server stack optimized for Orange Pi 3B (RK3566), with Jellyfin hardware acceleration via V4L2M2M.

This guide is terminal-only and uses Git + Docker/Docker Compose commands.

## Services

| Service | Port(s) | Purpose |
|---|---:|---|
| qBittorrent | 5080, 6881/tcp+udp | Torrent client |
| Radarr | 7878 | Movie management |
| Sonarr | 8989 | TV management |
| Prowlarr | 9696 | Indexer manager |
| Bazarr | 6767 | Subtitle management |
| Jellyfin | 8096, 7359/udp, 8920 | Media server |
| Seerr | 5055 | Request management |
| Profilarr | 6868 | Quality profile manager |
| FlareSolverr | 8191 | Cloudflare bypass proxy |
| Nginx | 80, 443 | Reverse proxy |
| Watchtower | - | Automatic image updates |
| Netdata | 19999 (host network) | Host monitoring |

## Requirements

- Linux host with Docker Engine 20.10+
- Docker Compose v2
- Git
- 4 GB RAM or more (recommended)

Quick check:

```bash
docker --version
docker compose version
git --version
```

## First-Time Setup

> Run commands on the machine that will host the containers (for example, your OrangePi over SSH).

1) Clone the repository:

```bash
git clone https://github.com/anibalnet/blackbeard.git
cd blackbeard
```

2) Create the external Docker network required by compose:

```bash
docker network create jollyroger || true
```

3) Create your environment file:

```bash
cp .env.example .env
```

4) Review key values in `.env`:

```bash
# Current user/group IDs
id

# GPU groups (commonly 44 and 105 on OrangePi)
getent group video
getent group render
```

Important variables to verify:

- `PUID` and `PGID`
- `TZ`
- `DOWNLOADS_PATH`
- `CONFIG_BASE_PATH`
- `GPU_VIDEO_GROUP` and `GPU_RENDER_GROUP`

5) Create expected directories:

```bash
mkdir -p \
  config/qbittorrent \
  config/radarr \
  config/sonarr \
  config/prowlarr \
  config/bazarr \
  config/jellyfin \
  config/jellyseerr \
  config/profilarr \
  config/nginx/logs \
  config/nginx/www \
  config/netdata/config \
  config/netdata/lib \
  config/netdata/cache \
  backups
```

6) Start the stack:

```bash
docker compose pull
docker compose up -d
docker compose ps
```

7) Follow startup logs (first boot may take a few minutes):

```bash
docker compose logs -f --tail=100
```

## Daily Operations

Check service status:

```bash
docker compose ps
```

Follow logs for all services:

```bash
docker compose logs -f --tail=100
```

Follow logs for one service:

```bash
docker compose logs -f --tail=100 radarr
```

Stop and start:

```bash
docker compose stop
docker compose start
```

Restart all or one service:

```bash
docker compose restart
docker compose restart jellyfin
```

Validate compose configuration:

```bash
docker compose config
```

## Updating the Stack

Manual update flow:

```bash
docker compose pull
docker compose up -d --force-recreate
docker compose ps
```

Note: Watchtower is also enabled to auto-update labeled containers.

## Backup and Restore (Essential)

List volumes marked for backup:

```bash
docker volume ls --filter label=backup.enable=true
```

Example backup for `radarr-config` volume:

```bash
mkdir -p backups
docker run --rm \
  -v radarr-config:/data:ro \
  -v "$(pwd)/backups:/backup" \
  alpine \
  sh -c 'tar czf /backup/radarr-config-$(date +%Y%m%d-%H%M%S).tar.gz -C /data .'
```

Example restore for `radarr-config`:

```bash
docker compose stop radarr

docker run --rm \
  -v radarr-config:/data \
  -v "$(pwd)/backups:/backup:ro" \
  alpine \
  sh -c 'rm -rf /data/* && tar xzf /backup/radarr-config-YYYYMMDD-HHMMSS.tar.gz -C /data'

docker compose start radarr
```

## Essential Troubleshooting

Containers not starting:

```bash
docker compose ps
docker compose logs --tail=200
```

Healthcheck still pending during first boot:

```bash
docker compose ps
docker inspect bazarr --format '{{json .State.Health}}'
```

Permission issues on config paths:

```bash
id
sudo chown -R $(id -u):$(id -g) config
```

Jellyfin hardware acceleration issues:

```bash
ls -la /dev/dri
ls -la /dev/video*
getent group video
getent group render
docker compose logs --tail=200 jellyfin
```

External network error:

```bash
docker network ls | grep jollyroger
docker network create jollyroger || true
docker compose up -d
```

Port already in use:

```bash
sudo lsof -i :80
sudo lsof -i :443
```

## Nginx Routes

Nginx config file: `config/nginx/nginx.conf`.

Available routes:

- `/jellyfin/`
- `/seerr`
- `/radarr`
- `/sonarr`
- `/bazarr/`
- `/prowlarr`
- `/qbittorrent/`

After first login, configure base URLs in apps that run behind subpaths (Radarr/Sonarr/Prowlarr/Bazarr).

## Basic Security Checklist

- Change qBittorrent default password immediately.
- Restrict exposed ports to trusted networks.
- Configure HTTPS in Nginx for external access.
- Keep periodic backups of `config/`, `.env`, and `backups/`.
- Review Watchtower auto-update behavior regularly.
