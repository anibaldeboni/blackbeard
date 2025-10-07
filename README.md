# Blackbeard - Home Media Server Stack

Blackbeard is a comprehensive Docker-based home media server solution that provides automated media acquisition, management, and streaming capabilities. This collection of services creates a complete self-hosted entertainment ecosystem.

## Overview

This stack combines popular media management applications into a unified platform accessible through a single web interface. All services are containerized using Docker Compose and proxied through Nginx for seamless integration.

## Services Included

### Media Streaming

- **Jellyfin** (Port 8096) - Open-source media server for streaming movies, TV shows, and music
- **Jellyseerr** (Port 5055) - Request management system for media discovery and automated downloads

### Download Management

- **qBittorrent** (Port 5080) - BitTorrent client with web interface for torrent downloads
- **FlareSolverr** (Port 8191) - Cloudflare bypass proxy for accessing protected torrent sites

### Media Automation

- **Radarr** (Port 7878) - Movie collection manager with automated searching and downloading
- **Sonarr** (Port 8989) - TV series collection manager with episode tracking and automation
- **Prowlarr** (Port 9696) - Indexer manager that integrates with Radarr and Sonarr
- **Jackett** (Port 9117) - API support for torrent trackers

### Infrastructure

- **Nginx** (Ports 80/443) - Reverse proxy providing unified access to all services
- **External Network** - Uses `jollyroger` Docker network for service communication

## Quick Start

1. **Prerequisites**

   - Docker and Docker Compose installed
   - External Docker network created: `docker network create jollyroger`
   - Storage directory available at `/media/STORAGE/torrents`

2. **Launch the stack**

   ```bash
   docker-compose up -d
   ```

3. **Access the services**
   - Main interface: http://localhost (redirects to Jellyfin)
   - Individual services available at: http://localhost/[service-name]

## Configuration

### Default Credentials

- **qBittorrent**: Username `admin`, Password `adminadmin`
  - Change password after first login via Tools → Options → WebUI

### Storage Configuration

- Downloads: `/media/STORAGE/torrents`
- Config files: `/home/orangepi/.config/[service-name]`
- All configurations persist across container restarts

### Network Setup

- Timezone: `America/Sao_Paulo`
- User/Group IDs: 1000:1000 (configurable)
- External network: `jollyroger` (must be created separately)

## Hardware Requirements

### GPU Support

- Video device mapping for hardware transcoding: `/dev/video0`, `/dev/video1`, `/dev/video2`
- Enables efficient video transcoding in Jellyfin

### USB Auto-mount

- Includes udev rule (`99-usb-automount.rules`) for automatic USB device mounting
- Devices mount to `/media/[UUID]` with universal access permissions

## Service Access

All services are accessible through the Nginx reverse proxy:

| Service     | URL                           | Purpose            |
| ----------- | ----------------------------- | ------------------ |
| Jellyfin    | http://localhost/jellyfin/    | Media streaming    |
| Jellyseerr  | http://localhost/jellyseerr/  | Media requests     |
| Radarr      | http://localhost/radarr       | Movie management   |
| Sonarr      | http://localhost/sonarr       | TV show management |
| Prowlarr    | http://localhost/prowlarr     | Indexer management |
| qBittorrent | http://localhost/qbittorrent/ | Torrent client     |
| Jackett     | http://localhost/jackett/     | Torrent trackers   |

## Maintenance

### Updates

- Service versions are pinned in docker-compose.yml
- Update by modifying version tags and running `docker-compose pull && docker-compose up -d`

### Backups

- Configuration data stored in `/home/orangepi/.config/`
- Media files stored in `/media/STORAGE/torrents`
- Regular backups of config directories recommended

### Logs

- Nginx logs: Accessible through nginx-logs volume
- Container logs: `docker-compose logs [service-name]`

## VPN Support

The configuration includes commented network settings for VPN integration:

- Uncomment network lines in docker-compose.yml to enable VPN routing
- Useful for routing torrent traffic through VPN while keeping other services direct

## Security Notes

- Change default qBittorrent password immediately
- Consider implementing SSL/TLS certificates for HTTPS access
- Review and adjust file permissions as needed
- Monitor access logs for unauthorized usage

## Troubleshooting

### Common Issues

1. **Services not accessible**: Verify `jollyroger` network exists
2. **Permission errors**: Check PUID/PGID values match host user
3. **Storage issues**: Confirm mount paths exist and are writable
4. **GPU transcoding**: Verify video devices exist and are accessible

### Useful Commands

```bash
# Check service status
docker-compose ps

# View logs
docker-compose logs -f [service-name]

# Restart specific service
docker-compose restart [service-name]

# Update all services
docker-compose pull && docker-compose up -d
```

## Architecture

This setup follows modern containerization best practices:

- Persistent data storage using bind mounts
- Service isolation with controlled inter-service communication
- Centralized reverse proxy for unified access
- Hardware passthrough for optimal performance
- Automated USB storage management

The name "Blackbeard" reflects the swashbuckling spirit of self-hosted media management, providing freedom and control over your entertainment library.
