#!/bin/bash

# ============================================================================
# Stack Management Script - Unified CLI
# ============================================================================
#
# Usage: ./stack.sh <group> <command> [options]
#
# Groups:
#   stack   - Stack management (start, stop, restart, logs, etc.)
#   backup  - Volume backup/restore operations
#   docker  - Docker cleanup operations
#   hw      - Hardware monitoring (temperature, GPU/VPU)
# ============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
ORANGE='\033[0;91m'
BOLD_RED='\033[1;31m'
NC='\033[0m'

# Hardware paths (OrangePi/RK3566)
CPU_TEMP_PATH="/sys/devices/virtual/thermal/thermal_zone0/temp"
GPU_TEMP_PATH="/sys/devices/virtual/thermal/thermal_zone1/temp"
GPU_FREQ_PATH="/sys/class/devfreq/fde60000.gpu"

# Configuration
COMPOSE_FILE="docker-compose.yml"
ENV_FILE=".env"
ENV_EXAMPLE=".env.example"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# ============================================================================
# Utility Functions
# ============================================================================

print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# ============================================================================
# Stack Management Functions
# ============================================================================

check_env_file() {
    if [ ! -f "$ENV_FILE" ]; then
        print_warning ".env file not found!"
        if [ -f "$ENV_EXAMPLE" ]; then
            echo -e "${YELLOW}Creating .env from .env.example...${NC}"
            cp "$ENV_EXAMPLE" "$ENV_FILE"
            print_success "Created .env file"
            print_warning "Please edit .env file with your configurations before starting the stack"
            exit 1
        else
            print_error ".env.example not found!"
            exit 1
        fi
    fi
}

check_network() {
    if ! docker network inspect jollyroger &>/dev/null; then
        print_warning "Network 'jollyroger' not found. Creating..."
        docker network create jollyroger
        print_success "Network 'jollyroger' created"
    fi
}

stack_validate() {
    print_header "Validating Configuration"
    if docker compose config --quiet; then
        print_success "Docker Compose configuration is valid"
    else
        print_error "Docker Compose configuration has errors"
        exit 1
    fi
}

stack_status() {
    print_header "Stack Status"
    docker compose ps
}

stack_health() {
    print_header "Health Status"
    docker compose ps --format "table {{.Name}}\t{{.Status}}\t{{.State}}"
}

stack_logs() {
    SERVICE=${1:-}
    if [ -z "$SERVICE" ]; then
        docker compose logs -f --tail=100
    else
        docker compose logs -f --tail=100 "$SERVICE"
    fi
}

stack_start() {
    check_env_file
    check_network
    stack_validate

    print_header "Starting Media Stack"
    docker compose up -d
    print_success "Stack started successfully"

    echo ""
    print_warning "Waiting for services to be healthy (this may take a few minutes)..."
    sleep 10
    stack_health
}

stack_stop() {
    print_header "Stopping Media Stack"
    docker compose down
    print_success "Stack stopped successfully"
}

stack_restart() {
    stack_stop
    echo ""
    stack_start
}

stack_restart_service() {
    SERVICE=$1
    if [ -z "$SERVICE" ]; then
        print_error "Please specify a service name"
        exit 1
    fi

    print_header "Restarting $SERVICE"
    docker compose restart "$SERVICE"
    print_success "$SERVICE restarted successfully"
}

stack_update() {
    print_header "Updating Stack Images"
    docker compose pull
    print_success "Images updated successfully"

    echo ""
    print_warning "Recreating containers with new images..."
    docker compose up -d --force-recreate
    print_success "Stack updated successfully"
}

stack_resources() {
    print_header "Resource Usage"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"
}

# ============================================================================
# Stack Install Functions
# ============================================================================

stack_install() {
    print_header "Blackbeard Media Stack - Installation"

    local SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local ERRORS=0

    echo ""
    print_info "Installation directory: $SCRIPT_DIR"
    echo ""

    # Step 1: Check Docker
    print_info "[1/8] Checking Docker..."
    if command -v docker &>/dev/null; then
        local DOCKER_VERSION=$(docker --version | cut -d' ' -f3 | tr -d ',')
        print_success "Docker installed (v$DOCKER_VERSION)"
    else
        print_error "Docker is not installed!"
        print_info "Install Docker: https://docs.docker.com/engine/install/"
        ERRORS=$((ERRORS + 1))
    fi

    # Step 2: Check Docker Compose
    print_info "[2/8] Checking Docker Compose..."
    if docker compose version &>/dev/null; then
        local COMPOSE_VERSION=$(docker compose version --short)
        print_success "Docker Compose installed (v$COMPOSE_VERSION)"
    else
        print_error "Docker Compose is not installed!"
        ERRORS=$((ERRORS + 1))
    fi

    # Step 3: Create Docker network
    print_info "[3/8] Creating Docker network..."
    if docker network inspect jollyroger &>/dev/null; then
        print_success "Network 'jollyroger' already exists"
    else
        if docker network create jollyroger &>/dev/null; then
            print_success "Network 'jollyroger' created"
        else
            print_error "Failed to create network 'jollyroger'"
            ERRORS=$((ERRORS + 1))
        fi
    fi

    # Step 4: Create .env file
    print_info "[4/8] Setting up environment file..."
    if [ -f "$SCRIPT_DIR/.env" ]; then
        print_success ".env file already exists"
    elif [ -f "$SCRIPT_DIR/.env.example" ]; then
        cp "$SCRIPT_DIR/.env.example" "$SCRIPT_DIR/.env"
        print_success "Created .env from .env.example"
        print_warning "Remember to edit .env with your settings!"
    else
        print_error ".env.example not found!"
        ERRORS=$((ERRORS + 1))
    fi

    # Step 5: Detect user/group IDs
    print_info "[5/8] Detecting user configuration..."
    local CURRENT_UID=$(id -u)
    local CURRENT_GID=$(id -g)
    print_info "  Your UID: $CURRENT_UID"
    print_info "  Your GID: $CURRENT_GID"

    if [ -f "$SCRIPT_DIR/.env" ]; then
        # Update PUID/PGID in .env if they differ
        local ENV_PUID=$(grep "^PUID=" "$SCRIPT_DIR/.env" | cut -d'=' -f2)
        local ENV_PGID=$(grep "^PGID=" "$SCRIPT_DIR/.env" | cut -d'=' -f2)

        if [ "$ENV_PUID" != "$CURRENT_UID" ] || [ "$ENV_PGID" != "$CURRENT_GID" ]; then
            print_warning "PUID/PGID in .env differ from current user"
            read -p "Update .env with current UID/GID? (Y/n) " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Nn]$ ]]; then
                sed -i.bak "s/^PUID=.*/PUID=$CURRENT_UID/" "$SCRIPT_DIR/.env"
                sed -i.bak "s/^PGID=.*/PGID=$CURRENT_GID/" "$SCRIPT_DIR/.env"
                rm -f "$SCRIPT_DIR/.env.bak"
                print_success "Updated PUID=$CURRENT_UID, PGID=$CURRENT_GID"
            fi
        else
            print_success "PUID/PGID already configured correctly"
        fi
    fi

    # Step 6: Create config directories
    print_info "[6/8] Creating config directories..."
    local CONFIG_DIRS=(
        "config/qbittorrent"
        "config/radarr"
        "config/sonarr"
        "config/prowlarr"
        "config/bazarr"
        "config/jellyfin"
        "config/jellyfin-gpu"
        "config/jellyseerr"
        "config/profilarr"
        "config/nginx/logs"
        "backups"
    )

    for dir in "${CONFIG_DIRS[@]}"; do
        if [ ! -d "$SCRIPT_DIR/$dir" ]; then
            mkdir -p "$SCRIPT_DIR/$dir"
            print_success "Created $dir"
        else
            print_info "  $dir (exists)"
        fi
    done

    # Step 7: Create/check downloads directory
    print_info "[7/8] Setting up downloads directory..."
    local DOWNLOADS_PATH="/media/STORAGE/downloads"
    if [ -f "$SCRIPT_DIR/.env" ]; then
        DOWNLOADS_PATH=$(grep "^DOWNLOADS_PATH=" "$SCRIPT_DIR/.env" | cut -d'=' -f2)
    fi

    if [ -d "$DOWNLOADS_PATH" ]; then
        print_success "Downloads directory exists: $DOWNLOADS_PATH"
    else
        print_warning "Downloads directory not found: $DOWNLOADS_PATH"
        read -p "Create downloads directory? (Y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Nn]$ ]]; then
            if mkdir -p "$DOWNLOADS_PATH" 2>/dev/null; then
                print_success "Created $DOWNLOADS_PATH"
            else
                print_warning "Could not create directory (may need sudo)"
                print_info "Run: sudo mkdir -p $DOWNLOADS_PATH && sudo chown $CURRENT_UID:$CURRENT_GID $DOWNLOADS_PATH"
            fi
        fi
    fi

    # Step 8: Check GPU devices (optional)
    print_info "[8/8] Checking GPU devices (optional)..."
    local GPU_AVAILABLE=0
    if [ -d "/dev/dri" ]; then
        print_success "/dev/dri available (GPU rendering)"
        GPU_AVAILABLE=1
    fi
    if [ -e "/dev/video0" ]; then
        print_success "/dev/video0 available (VPU)"
    fi
    if [ -e "/dev/video1" ]; then
        print_success "/dev/video1 available (VPU decoder)"
    fi
    if [ -e "/dev/video2" ]; then
        print_success "/dev/video2 available (VPU encoder)"
    fi
    if [ $GPU_AVAILABLE -eq 0 ]; then
        print_warning "No GPU devices found (hardware transcoding disabled)"
    fi

    # Detect GPU groups
    if [ $GPU_AVAILABLE -eq 1 ] && [ -f "$SCRIPT_DIR/.env" ]; then
        local VIDEO_GID=$(getent group video 2>/dev/null | cut -d: -f3)
        local RENDER_GID=$(getent group render 2>/dev/null | cut -d: -f3)

        if [ -n "$VIDEO_GID" ]; then
            local ENV_VIDEO=$(grep "^GPU_VIDEO_GROUP=" "$SCRIPT_DIR/.env" | cut -d'=' -f2)
            if [ "$ENV_VIDEO" != "$VIDEO_GID" ]; then
                sed -i.bak "s/^GPU_VIDEO_GROUP=.*/GPU_VIDEO_GROUP=$VIDEO_GID/" "$SCRIPT_DIR/.env"
                rm -f "$SCRIPT_DIR/.env.bak"
                print_info "  Updated GPU_VIDEO_GROUP=$VIDEO_GID"
            fi
        fi
        if [ -n "$RENDER_GID" ]; then
            local ENV_RENDER=$(grep "^GPU_RENDER_GROUP=" "$SCRIPT_DIR/.env" | cut -d'=' -f2)
            if [ "$ENV_RENDER" != "$RENDER_GID" ]; then
                sed -i.bak "s/^GPU_RENDER_GROUP=.*/GPU_RENDER_GROUP=$RENDER_GID/" "$SCRIPT_DIR/.env"
                rm -f "$SCRIPT_DIR/.env.bak"
                print_info "  Updated GPU_RENDER_GROUP=$RENDER_GID"
            fi
        fi
    fi

    # Summary
    echo ""
    print_header "Installation Summary"

    if [ $ERRORS -eq 0 ]; then
        print_success "Installation completed successfully!"
        echo ""
        print_info "Next steps:"
        echo "  1. Review and edit .env file if needed"
        echo "  2. Run: ./stack.sh stack start"
        echo "  3. Wait 2-3 minutes for services to start"
        echo "  4. Access via: http://localhost (nginx) or http://localhost:8096 (jellyfin)"
        echo ""
        print_info "Service URLs (via nginx):"
        echo "  - Jellyfin:    http://localhost/jellyfin/"
        echo "  - Jellyseerr:  http://localhost/jellyseerr/"
        echo "  - Radarr:      http://localhost/radarr/"
        echo "  - Sonarr:      http://localhost/sonarr/"
        echo "  - Prowlarr:    http://localhost/prowlarr/"
        echo "  - Bazarr:      http://localhost/bazarr/"
        echo "  - qBittorrent: http://localhost/qbittorrent/"
    else
        print_error "Installation completed with $ERRORS error(s)"
        print_warning "Please fix the errors above before starting the stack"
    fi

    return $ERRORS
}

stack_install_check() {
    print_header "Installation Status Check"

    local SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local ALL_OK=1

    echo ""

    # Check Docker
    echo -n "Docker:           "
    if command -v docker &>/dev/null; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${RED}NOT INSTALLED${NC}"
        ALL_OK=0
    fi

    # Check Docker Compose
    echo -n "Docker Compose:   "
    if docker compose version &>/dev/null; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${RED}NOT INSTALLED${NC}"
        ALL_OK=0
    fi

    # Check network
    echo -n "Network:          "
    if docker network inspect jollyroger &>/dev/null; then
        echo -e "${GREEN}OK${NC} (jollyroger)"
    else
        echo -e "${YELLOW}NOT CREATED${NC}"
        ALL_OK=0
    fi

    # Check .env
    echo -n ".env file:        "
    if [ -f "$SCRIPT_DIR/.env" ]; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${YELLOW}MISSING${NC}"
        ALL_OK=0
    fi

    # Check config dirs
    echo -n "Config dirs:      "
    local MISSING_DIRS=0
    local CONFIG_DIRS=("qbittorrent" "radarr" "sonarr" "prowlarr" "bazarr" "jellyfin" "jellyseerr" "nginx/logs")
    for dir in "${CONFIG_DIRS[@]}"; do
        if [ ! -d "$SCRIPT_DIR/config/$dir" ]; then
            MISSING_DIRS=$((MISSING_DIRS + 1))
        fi
    done
    if [ $MISSING_DIRS -eq 0 ]; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${YELLOW}$MISSING_DIRS MISSING${NC}"
        ALL_OK=0
    fi

    # Check downloads path
    echo -n "Downloads path:   "
    local DOWNLOADS_PATH="/media/STORAGE/downloads"
    if [ -f "$SCRIPT_DIR/.env" ]; then
        DOWNLOADS_PATH=$(grep "^DOWNLOADS_PATH=" "$SCRIPT_DIR/.env" | cut -d'=' -f2)
    fi
    if [ -d "$DOWNLOADS_PATH" ]; then
        echo -e "${GREEN}OK${NC} ($DOWNLOADS_PATH)"
    else
        echo -e "${YELLOW}NOT FOUND${NC} ($DOWNLOADS_PATH)"
        ALL_OK=0
    fi

    # Check GPU
    echo -n "GPU devices:      "
    if [ -d "/dev/dri" ]; then
        echo -e "${GREEN}AVAILABLE${NC}"
    else
        echo -e "${YELLOW}NOT FOUND${NC} (optional)"
    fi

    echo ""
    if [ $ALL_OK -eq 1 ]; then
        print_success "All checks passed! Ready to start."
    else
        print_warning "Some items need attention. Run: ./stack.sh stack install"
    fi
}

stack_uninstall() {
    print_header "Uninstall Blackbeard Stack"

    print_error "WARNING: This will remove:"
    echo "  - All running containers"
    echo "  - Docker network 'jollyroger'"
    echo ""
    print_warning "Config directories and .env will NOT be removed"
    echo ""

    read -p "Are you sure? Type 'yes' to confirm: " -r
    echo

    if [[ $REPLY != "yes" ]]; then
        print_info "Uninstall cancelled"
        return 0
    fi

    print_info "Stopping containers..."
    docker compose down 2>/dev/null || true

    print_info "Removing network..."
    docker network rm jollyroger 2>/dev/null || true

    print_success "Uninstall completed"
    print_info "Config directories preserved in ./config/"
    print_info "To fully remove, delete the project directory manually"
}

# ============================================================================
# Backup Functions
# ============================================================================

backup_list_volumes() {
    print_header "Volumes Marked for Backup"
    docker volume ls --filter "label=backup.enable=true" --format "table {{.Name}}\t{{.Driver}}\t{{.Labels}}"
}

backup_volume() {
    local VOLUME_NAME=$1
    local BACKUP_PATH="$BACKUP_DIR/$TIMESTAMP"
    local BACKUP_FILE="$BACKUP_PATH/${VOLUME_NAME}.tar.gz"

    mkdir -p "$BACKUP_PATH"
    print_info "Backing up volume: $VOLUME_NAME"

    docker run --rm \
        -v "$VOLUME_NAME:/data:ro" \
        -v "$BACKUP_PATH:/backup" \
        alpine \
        tar czf "/backup/${VOLUME_NAME}.tar.gz" -C /data .

    if [ $? -eq 0 ]; then
        local SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
        print_success "Backup completed: ${VOLUME_NAME}.tar.gz ($SIZE)"
    else
        print_error "Backup failed: $VOLUME_NAME"
        return 1
    fi
}

backup_all() {
    local BACKUP_PATH="$BACKUP_DIR/$TIMESTAMP"
    mkdir -p "$BACKUP_PATH"

    print_header "Starting Backup Process"

    VOLUMES=$(docker volume ls --filter "label=backup.enable=true" -q)

    if [ -z "$VOLUMES" ]; then
        print_warning "No volumes found with label 'backup.enable=true'"
        return 1
    fi

    local TOTAL=$(echo "$VOLUMES" | wc -l | tr -d ' ')
    print_info "Found $TOTAL volumes to backup"
    print_info "Backup destination: $BACKUP_PATH"

    local COUNT=0
    for VOLUME in $VOLUMES; do
        COUNT=$((COUNT + 1))
        echo ""
        print_info "[$COUNT/$TOTAL] Processing $VOLUME"

        docker run --rm \
            -v "$VOLUME:/data:ro" \
            -v "$BACKUP_PATH:/backup" \
            alpine \
            tar czf "/backup/${VOLUME}.tar.gz" -C /data .

        if [ $? -eq 0 ]; then
            local SIZE=$(du -h "$BACKUP_PATH/${VOLUME}.tar.gz" | cut -f1)
            print_success "Backup completed: ${VOLUME}.tar.gz ($SIZE)"
        else
            print_error "Backup failed: $VOLUME"
        fi
    done

    echo ""
    print_header "Backup Summary"
    print_success "Backup completed successfully"
    print_info "Location: $BACKUP_PATH"
    print_info "Total size: $(du -sh $BACKUP_PATH | cut -f1)"

    echo ""
    print_info "Backup files:"
    ls -lh "$BACKUP_PATH"
}

backup_restore() {
    local BACKUP_FILE=$1
    local VOLUME_NAME=$2

    if [ ! -f "$BACKUP_FILE" ]; then
        print_error "Backup file not found: $BACKUP_FILE"
        return 1
    fi

    if [ -z "$VOLUME_NAME" ]; then
        VOLUME_NAME=$(basename "$BACKUP_FILE" .tar.gz)
    fi

    print_warning "This will REPLACE all data in volume: $VOLUME_NAME"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Restore cancelled"
        return 0
    fi

    print_info "Restoring volume: $VOLUME_NAME from $BACKUP_FILE"

    docker volume create "$VOLUME_NAME" 2>/dev/null || true

    docker run --rm \
        -v "$VOLUME_NAME:/data" \
        -v "$(dirname $BACKUP_FILE):/backup:ro" \
        alpine \
        sh -c "rm -rf /data/* && tar xzf /backup/$(basename $BACKUP_FILE) -C /data"

    if [ $? -eq 0 ]; then
        print_success "Restore completed: $VOLUME_NAME"
    else
        print_error "Restore failed: $VOLUME_NAME"
        return 1
    fi
}

backup_list() {
    print_header "Available Backups"

    if [ ! -d "$BACKUP_DIR" ]; then
        print_warning "No backups found in $BACKUP_DIR"
        return 1
    fi

    for BACKUP in $(ls -dt $BACKUP_DIR/*/ 2>/dev/null); do
        local DATE=$(basename $BACKUP)
        local SIZE=$(du -sh $BACKUP | cut -f1)
        local COUNT=$(ls -1 $BACKUP/*.tar.gz 2>/dev/null | wc -l | tr -d ' ')

        echo ""
        print_info "Backup: $DATE"
        echo "  Size: $SIZE"
        echo "  Files: $COUNT volumes"
        echo "  Location: $BACKUP"
    done
}

backup_cleanup() {
    local KEEP_DAYS=${1:-7}

    print_header "Cleaning Old Backups"
    print_warning "Removing backups older than $KEEP_DAYS days"

    find "$BACKUP_DIR" -maxdepth 1 -type d -mtime +$KEEP_DAYS -exec rm -rf {} \;

    print_success "Cleanup completed"
}

# ============================================================================
# Docker Cleanup Functions
# ============================================================================

docker_disk() {
    print_header "Docker Disk Usage"
    docker system df -v
}

docker_list() {
    print_header "Unused Images"

    echo "Dangling Images (no tag):"
    docker images -f "dangling=true" --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}\t{{.CreatedAt}}" || echo "None found"

    echo ""
    echo "All Images:"
    docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}\t{{.CreatedAt}}"

    echo ""
    echo "Used by containers:"
    docker ps -a --format "{{.Image}}" | sort -u
}

docker_dangling() {
    print_header "Removing Dangling Images"

    DANGLING_COUNT=$(docker images -f "dangling=true" -q | wc -l | tr -d ' ')

    if [ "$DANGLING_COUNT" -gt 0 ]; then
        print_info "Found $DANGLING_COUNT dangling images"
        docker image prune -f
        print_success "Dangling images removed"
    else
        print_info "No dangling images found"
    fi
}

docker_prune_images() {
    print_header "Removing All Unused Images"

    print_warning "This will remove ALL images not used by containers"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker image prune -a -f
        print_success "All unused images removed"
    else
        print_info "Operation cancelled"
    fi
}

docker_prune_old() {
    DAYS=${1:-30}

    print_header "Removing Images Older Than $DAYS Days"

    print_warning "This will remove images created more than $DAYS days ago"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker image prune -a -f --filter "until=${DAYS}d"
        print_success "Old images removed"
    else
        print_info "Operation cancelled"
    fi
}

docker_clean_all() {
    print_header "Complete Docker Cleanup"

    print_error "WARNING: This will remove:"
    echo "  - All stopped containers"
    echo "  - All unused networks"
    echo "  - All unused images"
    echo "  - All build cache"
    echo ""
    print_warning "Volumes will NOT be removed for safety"
    echo ""

    read -p "Are you ABSOLUTELY sure? Type 'yes' to confirm: " -r
    echo

    if [[ $REPLY == "yes" ]]; then
        print_info "Removing stopped containers..."
        docker container prune -f

        print_info "Removing unused networks..."
        docker network prune -f

        print_info "Removing unused images..."
        docker image prune -a -f

        print_info "Removing build cache..."
        docker builder prune -f

        print_success "Complete cleanup finished"
    else
        print_info "Operation cancelled"
    fi
}

docker_protected() {
    print_header "Protected Images (In Use by Containers)"
    docker ps -a --format "table {{.Image}}\t{{.Names}}\t{{.Status}}"
}

# ============================================================================
# Hardware Monitoring Functions
# ============================================================================

hw_get_temp_color() {
    local temp_numeric="$1"

    if awk "BEGIN {exit !($temp_numeric < 45)}"; then
        echo "${GREEN}"
    elif awk "BEGIN {exit !($temp_numeric < 60)}"; then
        echo "${YELLOW}"
    elif awk "BEGIN {exit !($temp_numeric < 75)}"; then
        echo "${ORANGE}"
    elif awk "BEGIN {exit !($temp_numeric < 85)}"; then
        echo "${RED}"
    else
        echo "${BOLD_RED}"
    fi
}

hw_read_temp() {
    local temp_file="$1"
    local label="$2"

    if [[ ! -f "$temp_file" ]]; then
        echo "N/A"
        return 1
    fi

    local temp_millidegrees=$(cat "$temp_file")

    if [[ ! "$temp_millidegrees" =~ ^[0-9]+$ ]]; then
        echo "N/A"
        return 1
    fi

    local temp_celsius=$((temp_millidegrees / 1000))
    local temp_decimal=$((temp_millidegrees % 1000 / 100))
    local temp_value="${temp_celsius}.${temp_decimal}"
    local color=$(hw_get_temp_color "$temp_value")
    echo -e "${color}${temp_value}${NC}°C"
}

hw_temp() {
    local target=${1:-all}

    case "$target" in
        cpu)
            echo -e "CPU: $(hw_read_temp "$CPU_TEMP_PATH" "CPU")"
            ;;
        gpu)
            echo -e "GPU: $(hw_read_temp "$GPU_TEMP_PATH" "GPU")"
            ;;
        all|*)
            local cpu_temp=$(hw_read_temp "$CPU_TEMP_PATH" "CPU")
            local gpu_temp=$(hw_read_temp "$GPU_TEMP_PATH" "GPU")
            echo -e "CPU: $cpu_temp | GPU: $gpu_temp"
            ;;
    esac
}

hw_temp_monitor() {
    local interval=${1:-2}

    print_info "Monitoring temperature every ${interval}s (Ctrl+C to stop)"
    echo ""

    while true; do
        local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        local cpu_temp=$(hw_read_temp "$CPU_TEMP_PATH" "CPU")
        local gpu_temp=$(hw_read_temp "$GPU_TEMP_PATH" "GPU")
        echo -e "[$timestamp] CPU: $cpu_temp | GPU: $gpu_temp"
        sleep "$interval"
    done
}

hw_gpu_status() {
    print_header "GPU/VPU Status (RK3566)"

    if [[ -d "$GPU_FREQ_PATH" ]]; then
        local gpu_cur=$(cat "$GPU_FREQ_PATH/cur_freq" 2>/dev/null || echo "0")
        local gpu_max=$(cat "$GPU_FREQ_PATH/max_freq" 2>/dev/null || echo "1")
        local gpu_pct=$((gpu_cur * 100 / gpu_max))

        printf "GPU Mali:  %3d MHz / %d MHz (%d%%)\n" $((gpu_cur/1000000)) $((gpu_max/1000000)) $gpu_pct
    else
        print_warning "GPU frequency path not found: $GPU_FREQ_PATH"
    fi

    echo ""
    echo "VPU/RGA Clocks:"
    if sudo cat /sys/kernel/debug/clk/clk_summary 2>/dev/null | grep -qE "vpu|rga"; then
        sudo cat /sys/kernel/debug/clk/clk_summary 2>/dev/null | grep -E "vpu|rga" | awk '{printf "  %-20s %d MHz\n", $1, $4/1000000}'
    else
        print_warning "VPU/RGA clock info not available"
    fi

    echo ""
    echo "VPU Interrupts:"
    if grep -qE "hantro|fdea|fdee" /proc/interrupts 2>/dev/null; then
        grep -E "hantro|fdea|fdee" /proc/interrupts 2>/dev/null | awk '{printf "  %-10s %s\n", $NF, $2}'
    else
        print_warning "VPU interrupt info not available"
    fi
}

hw_gpu_monitor() {
    local interval=${1:-2}

    print_info "Monitoring GPU every ${interval}s (Ctrl+C to stop)"
    echo ""

    while true; do
        clear
        echo "╔══════════════════════════════════════════════════════════════╗"
        echo "║           Orange Pi 3B - GPU/VPU Monitor                     ║"
        echo "╚══════════════════════════════════════════════════════════════╝"
        echo ""

        if [[ -d "$GPU_FREQ_PATH" ]]; then
            local gpu_cur=$(cat "$GPU_FREQ_PATH/cur_freq" 2>/dev/null || echo "0")
            local gpu_max=$(cat "$GPU_FREQ_PATH/max_freq" 2>/dev/null || echo "1")
            local gpu_pct=$((gpu_cur * 100 / gpu_max))
            printf "GPU Mali:  %3d MHz / %d MHz (%d%%)\n" $((gpu_cur/1000000)) $((gpu_max/1000000)) $gpu_pct
        fi

        echo ""
        echo "VPU/RGA Clocks:"
        sudo cat /sys/kernel/debug/clk/clk_summary 2>/dev/null | grep -E "vpu|rga" | awk '{printf "  %-20s %d MHz\n", $1, $4/1000000}'

        echo ""
        echo "VPU Interrupts:"
        grep -E "hantro|fdea|fdee" /proc/interrupts 2>/dev/null | awk '{printf "  %-10s %s\n", $NF, $2}'

        sleep "$interval"
    done
}

hw_full() {
    print_header "Hardware Status"

    echo "Temperature:"
    hw_temp all
    echo ""

    hw_gpu_status
}

# ============================================================================
# Help Functions
# ============================================================================

show_main_help() {
    cat << 'EOF'
Stack Management CLI - Unified Tool

Usage: ./stack.sh <group> <command> [options]

Groups:
  stack     Stack management (install, start, stop, logs, etc.)
  backup    Volume backup and restore operations
  docker    Docker cleanup operations
  hw        Hardware monitoring (temperature, GPU/VPU)

Run './stack.sh <group> help' for group-specific commands.

Quick Start:
  ./stack.sh stack install            # First-time setup
  ./stack.sh stack start              # Start all services

Examples:
  ./stack.sh stack install            # Setup directories, .env, network
  ./stack.sh stack check              # Verify installation status
  ./stack.sh stack start              # Start the stack
  ./stack.sh stack logs radarr        # View radarr logs
  ./stack.sh backup all               # Backup all labeled volumes
  ./stack.sh docker prune             # Remove unused images
  ./stack.sh hw temp                  # Show CPU/GPU temperature

EOF
}

show_stack_help() {
    cat << 'EOF'
Stack Management Commands

Usage: ./stack.sh stack <command> [options]

Setup Commands:
  install               Install/setup the stack (creates dirs, .env, network)
  check                 Check installation status
  uninstall             Remove containers and network (preserves config)

Runtime Commands:
  start                 Start the entire stack
  stop                  Stop the entire stack
  restart               Restart the entire stack
  status                Show container status
  health                Show health check status
  logs [service]        Show logs (optionally for specific service)
  restart-svc <service> Restart a specific service
  update                Pull new images and recreate containers
  resources             Show resource usage (CPU, memory)
  validate              Validate docker-compose configuration

Examples:
  ./stack.sh stack install            # First-time setup
  ./stack.sh stack check              # Verify installation
  ./stack.sh stack start              # Start all services
  ./stack.sh stack logs               # Show all logs
  ./stack.sh stack logs radarr        # Show radarr logs only
  ./stack.sh stack restart-svc sonarr # Restart sonarr service
  ./stack.sh stack update             # Update all images

EOF
}

show_backup_help() {
    cat << 'EOF'
Backup Management Commands

Usage: ./stack.sh backup <command> [options]

Uses label 'backup.enable=true' to identify important volumes.

Commands:
  volumes               List volumes marked for backup
  all                   Backup all marked volumes
  volume <name>         Backup a specific volume
  restore <file> [name] Restore volume from backup file
  list                  List available backups
  cleanup [days]        Remove backups older than N days (default: 7)

Environment:
  BACKUP_DIR            Backup destination (default: ./backups)

Examples:
  ./stack.sh backup volumes                    # List volumes to backup
  ./stack.sh backup all                        # Backup all marked volumes
  ./stack.sh backup volume radarr-config       # Backup specific volume
  ./stack.sh backup list                       # List available backups
  ./stack.sh backup restore backups/20260114_120000/radarr-config.tar.gz
  BACKUP_DIR=/mnt/nas ./stack.sh backup all   # Backup to NAS

EOF
}

show_docker_help() {
    cat << 'EOF'
Docker Cleanup Commands

Usage: ./stack.sh docker <command> [options]

Commands:
  disk                  Show disk usage
  list                  List unused images
  dangling              Remove dangling images only (safe)
  prune                 Remove all unused images
  prune-old [days]      Remove images older than N days (default: 30)
  clean                 Complete cleanup (containers, networks, images, cache)
  protected             Show protected images (in use)

Examples:
  ./stack.sh docker disk              # Show disk usage
  ./stack.sh docker dangling          # Remove dangling images (safe)
  ./stack.sh docker prune             # Remove all unused images
  ./stack.sh docker prune-old 14      # Remove images older than 14 days
  ./stack.sh docker clean             # Full cleanup (careful!)

EOF
}

show_hw_help() {
    cat << 'EOF'
Hardware Monitoring Commands (OrangePi/RK3566)

Usage: ./stack.sh hw <command> [options]

Commands:
  temp [cpu|gpu]        Show temperature (default: both)
  temp-monitor [sec]    Monitor temperature continuously (default: 2s)
  gpu                   Show GPU/VPU status
  gpu-monitor [sec]     Monitor GPU/VPU continuously (default: 2s)
  status                Show full hardware status

Temperature color coding:
  Green     - Cool: < 45°C
  Yellow    - Normal: 45-59°C
  Orange    - Warm: 60-74°C
  Red       - Hot: 75-84°C
  Bold Red  - Critical: ≥85°C

Examples:
  ./stack.sh hw temp                  # Show CPU and GPU temperature
  ./stack.sh hw temp cpu              # Show CPU temperature only
  ./stack.sh hw temp-monitor          # Monitor temp every 2 seconds
  ./stack.sh hw temp-monitor 5        # Monitor temp every 5 seconds
  ./stack.sh hw gpu                   # Show GPU/VPU status
  ./stack.sh hw gpu-monitor           # Monitor GPU continuously
  ./stack.sh hw status                # Full hardware status

EOF
}

# ============================================================================
# Main Command Router
# ============================================================================

GROUP=${1:-help}
COMMAND=${2:-help}
ARG1=${3:-}
ARG2=${4:-}

case "$GROUP" in
    # Stack commands
    stack)
        case "$COMMAND" in
            install)        stack_install ;;
            check)          stack_install_check ;;
            uninstall)      stack_uninstall ;;
            start)          stack_start ;;
            stop)           stack_stop ;;
            restart)        stack_restart ;;
            status)         stack_status ;;
            health)         stack_health ;;
            logs)           stack_logs "$ARG1" ;;
            restart-svc)    stack_restart_service "$ARG1" ;;
            update)         stack_update ;;
            resources)      stack_resources ;;
            validate)       stack_validate ;;
            help|--help|-h) show_stack_help ;;
            *)
                print_error "Unknown stack command: $COMMAND"
                echo ""
                show_stack_help
                exit 1
                ;;
        esac
        ;;

    # Backup commands
    backup)
        case "$COMMAND" in
            volumes)        backup_list_volumes ;;
            all)            backup_all ;;
            volume)         backup_volume "$ARG1" ;;
            restore)        backup_restore "$ARG1" "$ARG2" ;;
            list)           backup_list ;;
            cleanup)        backup_cleanup "$ARG1" ;;
            help|--help|-h) show_backup_help ;;
            *)
                print_error "Unknown backup command: $COMMAND"
                echo ""
                show_backup_help
                exit 1
                ;;
        esac
        ;;

    # Docker cleanup commands
    docker)
        case "$COMMAND" in
            disk)           docker_disk ;;
            list)           docker_list ;;
            dangling)       docker_dangling ;;
            prune)          docker_prune_images ;;
            prune-old)      docker_prune_old "$ARG1" ;;
            clean)          docker_clean_all ;;
            protected)      docker_protected ;;
            help|--help|-h) show_docker_help ;;
            *)
                print_error "Unknown docker command: $COMMAND"
                echo ""
                show_docker_help
                exit 1
                ;;
        esac
        ;;

    # Hardware monitoring commands
    hw)
        case "$COMMAND" in
            temp)           hw_temp "$ARG1" ;;
            temp-monitor)   hw_temp_monitor "$ARG1" ;;
            gpu)            hw_gpu_status ;;
            gpu-monitor)    hw_gpu_monitor "$ARG1" ;;
            status)         hw_full ;;
            help|--help|-h) show_hw_help ;;
            *)
                print_error "Unknown hw command: $COMMAND"
                echo ""
                show_hw_help
                exit 1
                ;;
        esac
        ;;

    # Main help
    help|--help|-h)
        show_main_help
        ;;

    *)
        print_error "Unknown group: $GROUP"
        echo ""
        show_main_help
        exit 1
        ;;
esac
