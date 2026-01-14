# Blackbeard Media Stack

A comprehensive Docker-based home media server solution providing automated media acquisition, management, and streaming. Version 2.0 brings health checks, resource management, and reliability improvements.

## Services

| Service | Port(s) | Description |
|---------|---------|-------------|
| qBittorrent | 5080, 6881 | Torrent client with VueTorrent interface |
| Radarr | 7878 | Movie management with automated search |
| Sonarr | 8989 | TV series management with episode tracking |
| Prowlarr | 9696 | Unified indexer manager |
| Bazarr | 6767 | Automatic subtitle management |
| Jellyfin | 8096, 7359/udp, 8920 | Media server with Rockchip GPU acceleration |
| Jellyseerr | 5055 | Media request interface |
| Profilarr | 6868 | Quality profile manager |
| FlareSolverr | 8191 | Cloudflare bypass proxy |
| Nginx | 80, 443 | Reverse proxy |
| Watchtower | - | Automatic container updates |

## Requirements

- Docker Engine 20.10+
- Docker Compose 2.0+
- 8GB+ RAM recommended
- Rockchip SoC with Hantro VPU for hardware acceleration (optional)
- Alternative: Any GPU with `/dev/dri` support (use commented Jellyfin config)

## Quick Start

```bash
# Create Docker network
docker network create jollyroger

# Configure environment variables
cp .env.example .env
nano .env  # Set PUID, PGID, paths

# Start the stack
./manage-stack.sh start

# Check status
./manage-stack.sh health
```

### Environment Configuration

Run `id` to get your user and group IDs:

```bash
PUID=1000                              # Your user ID
PGID=1000                              # Your group ID
TZ=America/Sao_Paulo                   # Your timezone
DOWNLOADS_PATH=/media/STORAGE/downloads # Download location
```

## Stack Management

The `manage-stack.sh` script provides convenient management commands:

```bash
./manage-stack.sh start              # Start all services
./manage-stack.sh stop               # Stop all services
./manage-stack.sh restart            # Restart all services
./manage-stack.sh status             # View status
./manage-stack.sh health             # Check health status
./manage-stack.sh logs [service]     # View logs
./manage-stack.sh restart-svc <svc>  # Restart specific service
./manage-stack.sh update             # Update images
./manage-stack.sh backup             # Backup configurations
./manage-stack.sh resources          # View resource usage
./manage-stack.sh help               # Show help
```

## Service Configuration

### 1. qBittorrent (Port 5080)
- Default user: `admin`
- Password: Check `docker logs qbittorrent`
- Change password immediately in Tools > Options > WebUI
- Set download path to `/downloads`

### 2. Prowlarr (Port 9696) - Configure First
- Add your preferred indexers
- Connect Radarr and Sonarr in Settings > Apps
- Indexers sync automatically to connected apps

### 3. Radarr (Port 7878) & Sonarr (Port 8989)
- Add qBittorrent as download client:
  - Host: `qbittorrent`, Port: `5080`
  - Category: `movies` (Radarr) or `tv` (Sonarr)
- Set root folders:
  - Radarr: `/downloads/movies`
  - Sonarr: `/downloads/tv`

### 4. Bazarr (Port 6767)
- Connect to Radarr and Sonarr
- Configure subtitle providers
- Set preferred languages

### 5. Jellyfin (Port 8096)
- Complete the setup wizard
- Add media libraries:
  - Movies: `/downloads/movies`
  - TV Shows: `/downloads/tv`
- Enable hardware acceleration in Settings > Playback

### 6. Jellyseerr (Port 5055)
- Connect to Jellyfin at `http://jellyfin:8096`
- Link Radarr and Sonarr
- Configure user permissions

## Advanced Configuration

### Hardware Acceleration (Jellyfin)

Jellyfin uses the `nyanmisaka/jellyfin:latest-rockchip` image with Rockchip GPU support (Hantro VPU). The following devices are mapped:

- `/dev/dri` - DRM rendering
- `/dev/video0` - RGA (scaling/conversion)
- `/dev/video1` - Hantro VPU Decoder
- `/dev/video2` - Hantro VPU Encoder

```bash
# Verify devices
ls -la /dev/dri
ls -la /dev/video*

# Check groups
getent group video   # Usually 44
getent group render  # Usually 105
```

> **Note:** For non-Rockchip systems, a commented alternative using `lscr.io/linuxserver/jellyfin` is available in docker-compose.yml.

### Resource Limits

Adjust CPU and memory limits in `.env`:

```bash
JELLYFIN_CPU_LIMIT=2.0
JELLYFIN_MEM_LIMIT=2g
QBITTORRENT_CPU_LIMIT=4.0
QBITTORRENT_MEM_LIMIT=4g
```

### Automatic Updates (Watchtower)

Watchtower is included in the stack and configured to automatically update containers daily at 4 AM. It only updates containers with the `com.centurylinklabs.watchtower.enable=true` label.

Configuration:
- `WATCHTOWER_CLEANUP=true` - Removes old images after update
- `WATCHTOWER_LABEL_ENABLE=true` - Only updates labeled containers
- `WATCHTOWER_SCHEDULE=0 0 4 * * *` - Runs daily at 4 AM
- Watchtower itself is excluded from auto-updates

### Nginx Reverse Proxy

Nginx provides a unified access point for all services. The configuration file is located at `config/nginx/nginx.conf`.

**Server Names:** `localhost`, `blackbeard.local`

**Available Routes:**

| Path | Service | Notes |
|------|---------|-------|
| `/` | Jellyfin | Redirects to `/jellyfin/` |
| `/jellyfin/` | Jellyfin | Media server (port 8096) |
| `/jellyseerr` | Jellyseerr | Request manager (port 5055) |
| `/radarr` | Radarr | Movie manager (port 7878) |
| `/sonarr` | Sonarr | TV manager (port 8989) |
| `/bazarr` | Bazarr | Subtitle manager (port 6767) |
| `/prowlarr` | Prowlarr | Indexer manager (port 9696) |
| `/qbittorrent/` | qBittorrent | Torrent client (port 5080) |

**Proxy Features:**
- WebSocket support for real-time updates (Upgrade headers)
- Automatic retry on backend failures (3 attempts, 30s timeout)
- Proper forwarding headers (X-Real-IP, X-Forwarded-For/Proto/Host)
- URL rewriting for sub-path routing

**Important:** Before using the reverse proxy, configure each application's base URL:
- Radarr/Sonarr/Prowlarr/Bazarr: Settings > General > URL Base (e.g., `/radarr`)
- qBittorrent: Settings > WebUI > Alternative WebUI enabled

### USB Automount (Optional)

A udev rule is provided to automatically mount USB storage devices, useful for external media drives.

**What it does:**
- Automatically mounts USB block devices when connected
- Uses the device label as mount point name (e.g., `/media/STORAGE`)
- Falls back to device UUID if no label is set (e.g., `/media/1234-ABCD`)
- Uses `systemd-mount` for non-blocking mount operations

**Installation:**
```bash
# Copy the udev rule
sudo cp udev/99-usb-automount.rules /etc/udev/rules.d/

# Reload udev rules
sudo udevadm control --reload-rules
sudo udevadm trigger
```

**Note:** Ensure your `DOWNLOADS_PATH` in `.env` points to the mounted USB device path (e.g., `/media/STORAGE/downloads`).

## Backup

### Automated
```bash
./manage-stack.sh backup
```
Creates backup in `backups/YYYYMMDD_HHMMSS/`

### Manual
```bash
docker compose down
tar -czf media-stack-backup-$(date +%Y%m%d).tar.gz config/ docker-compose.yml .env
docker compose up -d
```

## Troubleshooting

### Containers not starting in order
```bash
./manage-stack.sh health
docker compose logs <service>
```

### Permission issues
```bash
id  # Check PUID/PGID
sudo chown -R $PUID:$PGID ./config
```

### Jellyfin not detecting GPU
```bash
ls -la /dev/dri
sudo usermod -aG video,render $USER
```

### Service not healthy
```bash
docker inspect <container> | grep -A 20 Health
```

## Security Checklist

- Change qBittorrent default password
- Configure VPN for torrents
- Use HTTPS with nginx (Let's Encrypt)
- Regular configuration backups
- Keep containers updated
- Configure firewall appropriately

## Version 2.0 Changes

### Critical Fixes
- Fixed `dependent_on` typo to `depends_on`
- Added health checks to all services
- Implemented dependency conditions with `service_healthy`
- Configurable CPU and memory limits

### New Features
- Centralized `.env` configuration
- Management script `manage-stack.sh`
- Backup and Watchtower labels
- Tmpfs for Jellyfin transcoding cache
- Defined hostnames for all containers
- Standardized UMASK (002)
- Rockchip GPU support for Jellyfin (Hantro VPU)
- Watchtower included for automatic updates

### Service Startup Order
```
1. qbittorrent + flaresolverr (base services)
2. prowlarr (depends on flaresolverr)
3. radarr + sonarr (depend on qbittorrent + prowlarr)
4. bazarr (depends on radarr + sonarr + flaresolverr)
5. jellyfin (independent)
6. jellyseerr + profilarr (depend on radarr + sonarr + jellyfin)
7. nginx (depends on all services)
```

## Documentation

- [QUICKSTART.md](QUICKSTART.md) - 5-minute setup guide
- [SUMMARY.md](SUMMARY.md) - Executive summary of v2.0 changes
- [REFACTORING_CHANGES.md](REFACTORING_CHANGES.md) - Detailed refactoring notes
- [.env.example](.env.example) - Available environment variables

---

**Version:** 2.0.0
**Status:** Production Ready
