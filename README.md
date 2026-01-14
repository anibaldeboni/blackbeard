# Blackbeard Media Stack

A comprehensive Docker-based home media server solution providing automated media acquisition, management, and streaming. Version 2.1 brings a unified CLI tool with hardware monitoring and simplified installation.

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
# Install the stack (creates directories, .env, network)
./stack.sh stack install

# Start all services
./stack.sh stack start

# Check health status
./stack.sh stack health
```

The `install` command automatically:
- Checks Docker and Docker Compose installation
- Creates the `jollyroger` Docker network
- Copies `.env.example` to `.env`
- Detects and configures your UID/GID
- Creates all config directories
- Detects GPU devices and configures groups

## Stack Management CLI

The unified `stack.sh` script provides all management commands organized by groups:

```bash
./stack.sh <group> <command> [options]
```

### Groups Overview

| Group | Description |
|-------|-------------|
| `stack` | Stack lifecycle management (install, start, stop, logs) |
| `backup` | Volume backup and restore operations |
| `docker` | Docker cleanup operations |
| `hw` | Hardware monitoring (temperature, GPU/VPU) |

### Stack Commands

```bash
# Setup
./stack.sh stack install            # First-time installation
./stack.sh stack check              # Verify installation status
./stack.sh stack uninstall          # Remove containers and network

# Runtime
./stack.sh stack start              # Start all services
./stack.sh stack stop               # Stop all services
./stack.sh stack restart            # Restart all services
./stack.sh stack status             # View container status
./stack.sh stack health             # Check health status
./stack.sh stack logs [service]     # View logs (all or specific service)
./stack.sh stack restart-svc <svc>  # Restart specific service
./stack.sh stack update             # Pull new images and recreate
./stack.sh stack resources          # View CPU/memory usage
./stack.sh stack validate           # Validate docker-compose config
```

### Backup Commands

Backups use the `backup.enable=true` label to identify important volumes.

```bash
./stack.sh backup volumes           # List volumes marked for backup
./stack.sh backup all               # Backup all marked volumes
./stack.sh backup volume <name>     # Backup specific volume
./stack.sh backup restore <file>    # Restore from backup file
./stack.sh backup list              # List available backups
./stack.sh backup cleanup [days]    # Remove old backups (default: 7 days)

# Custom backup location
BACKUP_DIR=/mnt/nas ./stack.sh backup all
```

### Docker Cleanup Commands

```bash
./stack.sh docker disk              # Show disk usage
./stack.sh docker list              # List unused images
./stack.sh docker dangling          # Remove dangling images (safe)
./stack.sh docker prune             # Remove all unused images
./stack.sh docker prune-old [days]  # Remove images older than N days
./stack.sh docker clean             # Full cleanup (containers, networks, images)
./stack.sh docker protected         # Show images in use
```

### Hardware Monitoring Commands (OrangePi/RK3566)

```bash
./stack.sh hw temp                  # Show CPU and GPU temperature
./stack.sh hw temp cpu              # Show CPU temperature only
./stack.sh hw temp gpu              # Show GPU temperature only
./stack.sh hw temp-monitor [sec]    # Monitor temperature continuously
./stack.sh hw gpu                   # Show GPU/VPU status
./stack.sh hw gpu-monitor [sec]     # Monitor GPU/VPU continuously
./stack.sh hw status                # Full hardware status
```

Temperature color coding:
- **Green** - Cool: < 45°C
- **Yellow** - Normal: 45-59°C
- **Orange** - Warm: 60-74°C
- **Red** - Hot: 75-84°C
- **Bold Red** - Critical: >= 85°C

## Environment Configuration

The `.env` file is created automatically during installation. Key variables:

```bash
# User/Group IDs (auto-detected by install)
PUID=1000
PGID=1000

# Timezone
TZ=America/Sao_Paulo

# Storage paths
DOWNLOADS_PATH=/media/STORAGE/downloads
CONFIG_BASE_PATH=./config

# Resource limits (CPU cores / memory)
JELLYFIN_CPU_LIMIT=4.0
JELLYFIN_MEM_LIMIT=4g
QBITTORRENT_CPU_LIMIT=2.0
QBITTORRENT_MEM_LIMIT=2g

# GPU groups (auto-detected by install)
GPU_VIDEO_GROUP=44
GPU_RENDER_GROUP=105
```

Run `id` to verify your user and group IDs.

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

# Monitor GPU usage
./stack.sh hw gpu
./stack.sh hw gpu-monitor
```

> **Note:** For non-Rockchip systems, a commented alternative using `lscr.io/linuxserver/jellyfin` is available in docker-compose.yml.

### Automatic Updates (Watchtower)

Watchtower is included and configured to automatically update containers daily at 4 AM. It only updates containers with the `com.centurylinklabs.watchtower.enable=true` label.

Configuration:
- `WATCHTOWER_CLEANUP=true` - Removes old images after update
- `WATCHTOWER_LABEL_ENABLE=true` - Only updates labeled containers
- `WATCHTOWER_SCHEDULE=0 0 4 * * *` - Runs daily at 4 AM
- Watchtower itself is excluded from auto-updates

### Nginx Reverse Proxy

Nginx provides a unified access point for all services. Configuration: `config/nginx/nginx.conf`

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

**Important:** Configure each application's base URL before using the reverse proxy:
- Radarr/Sonarr/Prowlarr/Bazarr: Settings > General > URL Base (e.g., `/radarr`)
- qBittorrent: Settings > WebUI > Alternative WebUI enabled

### USB Automount (Optional)

A udev rule is provided to automatically mount USB storage devices.

```bash
# Install udev rule
sudo cp udev/99-usb-automount.rules /etc/udev/rules.d/

# Reload rules
sudo udevadm control --reload-rules
sudo udevadm trigger
```

**Note:** Ensure `DOWNLOADS_PATH` in `.env` points to the mounted USB device path (e.g., `/media/STORAGE/downloads`).

## Backup and Restore

### Automated Backup

```bash
# Backup all volumes with backup label
./stack.sh backup all

# List available backups
./stack.sh backup list

# Restore a specific volume
./stack.sh backup restore backups/20260114_120000/radarr-config.tar.gz
```

Backups are stored in `backups/YYYYMMDD_HHMMSS/`

### Manual Backup

```bash
./stack.sh stack stop
tar -czf media-stack-backup-$(date +%Y%m%d).tar.gz config/ docker-compose.yml .env
./stack.sh stack start
```

## Troubleshooting

### Check Installation Status
```bash
./stack.sh stack check
```

### Containers not starting
```bash
./stack.sh stack health
./stack.sh stack logs <service>
```

### Permission issues
```bash
id  # Check PUID/PGID match .env
sudo chown -R $PUID:$PGID ./config
```

### Jellyfin not detecting GPU
```bash
./stack.sh hw gpu                    # Check GPU status
ls -la /dev/dri
sudo usermod -aG video,render $USER
```

### Service not healthy
```bash
docker inspect <container> | grep -A 20 Health
```

### Monitor System Resources
```bash
./stack.sh stack resources           # Container CPU/memory
./stack.sh hw temp                   # System temperature
./stack.sh docker disk               # Docker disk usage
```

## Security Checklist

- Change qBittorrent default password
- Configure VPN for torrents
- Use HTTPS with nginx (Let's Encrypt)
- Regular configuration backups
- Keep containers updated (Watchtower handles this)
- Configure firewall appropriately

## Directory Structure

```
blackbeard/
├── docker-compose.yml          # Service definitions
├── .env                        # Environment configuration
├── .env.example                # Configuration template
├── stack.sh                    # Unified management CLI
├── config/                     # Service configurations
│   ├── nginx/
│   │   ├── nginx.conf
│   │   └── logs/
│   ├── qbittorrent/
│   ├── radarr/
│   ├── sonarr/
│   ├── prowlarr/
│   ├── bazarr/
│   ├── jellyfin/
│   ├── jellyseerr/
│   └── profilarr/
├── backups/                    # Volume backups
├── catalogs/                   # Jellyseerr catalogs
└── udev/                       # USB automount rules
```

## Service Startup Order

```
1. qbittorrent + flaresolverr (base services)
2. prowlarr (depends on flaresolverr)
3. radarr + sonarr (depend on qbittorrent + prowlarr)
4. bazarr (depends on radarr + sonarr + flaresolverr)
5. jellyfin (independent)
6. jellyseerr + profilarr (depend on radarr + sonarr + jellyfin)
7. nginx (depends on all services)
```

## Version History

### Version 2.1.0
- Unified CLI script `stack.sh` replacing individual scripts
- Added `stack install` for automated first-time setup
- Added `stack check` for installation verification
- Hardware monitoring commands (`hw temp`, `hw gpu`)
- Improved backup/restore workflow
- Docker cleanup utilities

### Version 2.0.0
- Health checks for all services
- Dependency conditions with `service_healthy`
- Configurable CPU and memory limits
- Centralized `.env` configuration
- Rockchip GPU support for Jellyfin
- Watchtower for automatic updates

---

**Version:** 2.1.0
**Status:** Production Ready
