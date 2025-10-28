#!/bin/bash

# Blackbeard Media Stack
# Usage: ./blackbeard.sh [service_name] | ./blackbeard.sh all

set -e

# Color codes for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly NETWORK_NAME="jollyroger"
readonly DOWNLOADS_DIR="/media/STORAGE/downloads"

# Function to detect host timezone
get_host_timezone() {
    local timezone=""
    
    # Try multiple methods to get the timezone
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        timezone=$(readlink /etc/localtime | sed 's|.*/zoneinfo/||')
    elif [[ -f /etc/timezone ]]; then
        # Debian/Ubuntu
        timezone=$(cat /etc/timezone)
    elif [[ -L /etc/localtime ]]; then
        # Most modern Linux distributions
        timezone=$(readlink /etc/localtime | sed 's|.*/zoneinfo/||')
    elif command -v timedatectl &> /dev/null; then
        # systemd systems
        timezone=$(timedatectl show --property=Timezone --value)
    else
        # Fallback
        timezone="UTC"
        print_warning "Could not detect host timezone, using UTC as fallback"
    fi
    
    echo "$timezone"
}

# Set timezone from host
HOST_TIMEZONE=$(get_host_timezone)

# Available services
SERVICES=(
    "qbittorrent"
    "radarr"
    "sonarr"
    "prowlarr"
    "jellyseerr"
    "jellyfin"
    "flaresolverr"
    "nginx"
    "bazarr"
)

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}[BLACKBEARD]${NC} $1"
}

# Function to check if Podman is installed
check_podman() {
    if ! command -v podman &> /dev/null; then
        print_warning "Podman is not installed. Attempting to install..."
        install_podman
    else
        print_status "Podman is already installed"
        podman --version
    fi
}

# Function to install Podman
install_podman() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if command -v brew &> /dev/null; then
            print_status "Installing Podman via Homebrew..."
            brew install podman
            print_status "Initializing Podman machine..."
            podman machine init
            podman machine start
        else
            print_error "Homebrew not found. Please install Homebrew first or install Podman manually."
            exit 1
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux
        if command -v apt-get &> /dev/null; then
            # Debian/Ubuntu
            print_status "Installing Podman via apt..."
            sudo apt-get update
            sudo apt-get install -y podman
        elif command -v yum &> /dev/null; then
            # RHEL/CentOS/Fedora
            print_status "Installing Podman via yum/dnf..."
            sudo yum install -y podman
        elif command -v dnf &> /dev/null; then
            # Fedora
            print_status "Installing Podman via dnf..."
            sudo dnf install -y podman
        else
            print_error "Unsupported Linux distribution. Please install Podman manually."
            exit 1
        fi
    else
        print_error "Unsupported operating system. Please install Podman manually."
        exit 1
    fi
}

# Function to create network if it doesn't exist
create_network() {
    if ! podman network exists "$NETWORK_NAME" 2>/dev/null; then
        print_status "Creating network: $NETWORK_NAME"
        podman network create "$NETWORK_NAME"
    else
        print_status "Network $NETWORK_NAME already exists"
    fi
}

# Function to setup firewall for nginx (Linux only)
setup_nginx_firewall() {
    if [[ "$OSTYPE" == "linux-gnu"* ]] && command -v firewall-cmd &> /dev/null; then
        print_status "Checking firewall configuration for nginx..."
        
        local needs_reload=false
        
        # Check if port 80 is already open
        if ! sudo firewall-cmd --permanent --query-port=80/tcp &>/dev/null; then
            print_status "Adding port 80/tcp to firewall..."
            sudo firewall-cmd --permanent --add-port=80/tcp
            needs_reload=true
        else
            print_status "Port 80/tcp already configured in firewall"
        fi
        
        # Check if port 8080 is already open
        if ! sudo firewall-cmd --permanent --query-port=8080/tcp &>/dev/null; then
            print_status "Adding port 8080/tcp to firewall..."
            sudo firewall-cmd --permanent --add-port=8080/tcp
            needs_reload=true
        else
            print_status "Port 8080/tcp already configured in firewall"
        fi
        
        # Check if port forwarding rule exists
        if ! sudo firewall-cmd --permanent --list-forward-ports | grep -q "port=80:proto=tcp:toport=8080"; then
            print_status "Adding port forwarding rule: 80 -> 8080..."
            sudo firewall-cmd --permanent --add-forward-port=port=80:proto=tcp:toport=8080
            needs_reload=true
        else
            print_status "Port forwarding rule 80 -> 8080 already configured"
        fi
        
        # Only reload if changes were made
        if [[ "$needs_reload" == "true" ]]; then
            print_status "Reloading firewall configuration..."
            sudo firewall-cmd --reload
            print_status "Firewall rules configured: port 80 -> 8080"
        else
            print_status "All firewall rules already configured, no changes needed"
        fi
    else
        print_warning "Firewall configuration skipped (not Linux or firewalld not available)"
        print_warning "Manual port forwarding may be required: 80 -> 8080"
    fi
}

# Function to stop and remove existing container
stop_container() {
    local container_name="$1"
    if podman container exists "$container_name" 2>/dev/null; then
        print_status "Stopping existing container: $container_name"
        podman stop "$container_name" 2>/dev/null || true
        podman rm "$container_name" 2>/dev/null || true
    fi
}

# Function to run qBittorrent
run_qbittorrent() {
    print_header "Starting qBittorrent service"
    stop_container "qbittorrent"
    
    podman run -d \
        --name qbittorrent \
        --network "$NETWORK_NAME" \
        --restart unless-stopped \
        -e PUID=1000 \
        -e PGID=1000 \
        -e UMASK=002 \
        -e TZ="$HOST_TIMEZONE" \
        -e WEBUI_PORT=5080 \
        -e DOCKER_MODS=ghcr.io/vuetorrent/vuetorrent-lsio-mod:latest \
        -v "${SCRIPT_DIR}/config/qbittorrent:/config" \
        -v "${DOWNLOADS_DIR}:/downloads" \
        -p 5080:5080 \
        -p 6881:6881 \
        -p 6881:6881/udp \
        lscr.io/linuxserver/qbittorrent:5.1.2
    
    print_status "qBittorrent container started"
    print_warning "Default credentials - Username: admin, check logs for password: podman logs qbittorrent"
}

# Function to run Radarr
run_radarr() {
    print_header "Starting Radarr service"
    stop_container "radarr"
    
    podman run -d \
        --name radarr \
        --network "$NETWORK_NAME" \
        --restart unless-stopped \
        -e PUID=1000 \
        -e PGID=1000 \
        -e UMASK=002 \
        -e TZ="$HOST_TIMEZONE" \
        -v "${SCRIPT_DIR}/config/radarr:/config" \
        -v "${DOWNLOADS_DIR}:/downloads" \
        -p 7878:7878 \
        lscr.io/linuxserver/radarr:5.27.5
    
    print_status "Radarr container started"
}

# Function to run Sonarr
run_sonarr() {
    print_header "Starting Sonarr service"
    stop_container "sonarr"
    
    podman run -d \
        --name sonarr \
        --network "$NETWORK_NAME" \
        --restart unless-stopped \
        -e PUID=1000 \
        -e PGID=1000 \
        -e UMASK=002 \
        -e TZ="$HOST_TIMEZONE" \
        -v "${SCRIPT_DIR}/config/sonarr:/config" \
        -v "${DOWNLOADS_DIR}:/downloads" \
        -p 8989:8989 \
        linuxserver/sonarr:4.0.15
    
    print_status "Sonarr container started"
}

# Function to run Prowlarr
run_prowlarr() {
    print_header "Starting Prowlarr service"
    stop_container "prowlarr"
    
    podman run -d \
        --name prowlarr \
        --network "$NETWORK_NAME" \
        --restart unless-stopped \
        -e PUID=1000 \
        -e PGID=1000 \
        -e UMASK=002 \
        -e TZ="$HOST_TIMEZONE" \
        -v "${SCRIPT_DIR}/config/prowlarr:/config" \
        -p 9696:9696 \
        linuxserver/prowlarr:2.0.5
    
    print_status "Prowlarr container started"
}

# Function to run Jellyseerr
run_jellyseerr() {
    print_header "Starting Jellyseerr service"
    stop_container "jellyseerr"
    
    podman run -d \
        --name jellyseerr \
        --network "$NETWORK_NAME" \
        --restart unless-stopped \
        -e PUID=1000 \
        -e PGID=1000 \
        -e UMASK=002 \
        -e TZ="$HOST_TIMEZONE" \
        -v "${SCRIPT_DIR}/config/jellyseerr:/app/config" \
        -p 5055:5055 \
        fallenbagel/jellyseerr:2.7.3
    
    print_status "Jellyseerr container started"
}

# Function to run Jellyfin
run_jellyfin() {
    print_header "Starting Jellyfin service"
    stop_container "jellyfin"
    
    # Check if GPU devices exist
    GPU_ARGS=""
    if [[ -e /dev/dri ]]; then
        GPU_ARGS="--device /dev/dri:/dev/dri --group-add 44 --group-add 105"
        print_status "GPU acceleration enabled"
    else
        print_warning "GPU devices not found, starting without hardware acceleration"
    fi
    
    podman run -d \
        --name jellyfin \
        --network "$NETWORK_NAME" \
        --restart unless-stopped \
        -e PUID=1000 \
        -e PGID=1000 \
        -e UMASK=002 \
        -e TZ="$HOST_TIMEZONE" \
        -v "${SCRIPT_DIR}/config/jellyfin:/config" \
        -v "${DOWNLOADS_DIR}:/downloads" \
        -p 8096:8096 \
        -p 7359:7359/udp \
        -p 8920:8920 \
        $GPU_ARGS \
        linuxserver/jellyfin:10.10.7
    
    print_status "Jellyfin container started"
}

# Function to run FlareSolverr
run_flaresolverr() {
    print_header "Starting FlareSolverr service"
    stop_container "flaresolverr"
    
    podman run -d \
        --name flaresolverr \
        --network "$NETWORK_NAME" \
        --restart unless-stopped \
        -e LOG_LEVEL="${LOG_LEVEL:-info}" \
        -e LOG_HTML="${LOG_HTML:-false}" \
        -e CAPTCHA_SOLVER="${CAPTCHA_SOLVER:-none}" \
        -e TZ="$HOST_TIMEZONE" \
        -p "${PORT:-8191}:8191" \
        ghcr.io/flaresolverr/flaresolverr:latest
    
    print_status "FlareSolverr container started"
}

# Function to run Nginx (rootless mode)
run_nginx() {
    print_header "Starting Nginx service (rootless mode)"
    stop_container "nginx"
    
    # Setup firewall before starting nginx
    setup_nginx_firewall
    
    # Run nginx in rootless mode on port 8080
    podman run -d \
        --name nginx \
        --network "$NETWORK_NAME" \
        --restart unless-stopped \
        -v "${SCRIPT_DIR}/config/nginx/nginx.conf:/etc/nginx/nginx.conf:ro" \
        -v "${SCRIPT_DIR}/config/nginx/logs:/var/log/nginx" \
        -p 8080:80 \
        nginx:1.29.2-alpine
    
    print_status "Nginx container started in rootless mode on port 8080"
    print_status "Configure your system to forward port 80 -> 8080 if needed"
}

# Function to run Bazarr
run_bazarr() {
    print_header "Starting Bazarr service"
    stop_container "bazarr"
    
    podman run -d \
        --name bazarr \
        --network "$NETWORK_NAME" \
        --restart unless-stopped \
        -e PUID=1000 \
        -e PGID=1000 \
        -e UMASK=002 \
        -e TZ="$HOST_TIMEZONE" \
        -v "${SCRIPT_DIR}/config/bazarr:/config" \
        -v "${DOWNLOADS_DIR}:/downloads" \
        -p 6767:6767 \
        lscr.io/linuxserver/bazarr:latest
    
    print_status "Bazarr container started"
}

# Function to run all services
run_all_services() {
    print_header "Starting all Blackbeard services"
    
    for service in "${SERVICES[@]}"; do
        "run_$service"
        sleep 2  # Small delay between services
    done
    
    print_header "All services started successfully"
    print_status "You can check container status with: podman ps"
}

# Function to show service status
show_status() {
    print_header "Blackbeard Services Status"
    podman ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" --filter "network=$NETWORK_NAME"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [service_name|all|status|stop]"
    echo ""
    echo "Available services:"
    for service in "${SERVICES[@]}"; do
        echo "  - $service"
    done
    echo ""
    echo "Commands:"
    echo "  all     - Start all services"
    echo "  status  - Show running containers status"
    echo "  stop    - Stop all services"
    echo ""
    echo "Examples:"
    echo "  $0 nginx          # Start only nginx"
    echo "  $0 all            # Start all services"
    echo "  $0 status         # Show containers status"
}

# Function to stop all services
stop_all_services() {
    print_header "Stopping all Blackbeard services"
    
    for service in "${SERVICES[@]}"; do
        stop_container "$service"
    done
    
    print_status "All services stopped"
}

# Main script logic
main() {
    print_header "Blackbeard Media Stack - Podman Migration"
    
    # Check and install Podman if needed
    check_podman
    
    # Create network
    create_network
    
    # Handle command line arguments
    case "${1:-all}" in
        "all")
            run_all_services
            ;;
        "status")
            show_status
            ;;
        "stop")
            stop_all_services
            ;;
        "qbittorrent"|"radarr"|"sonarr"|"prowlarr"|"jellyseerr"|"jellyfin"|"flaresolverr"|"nginx"|"bazarr")
            "run_$1"
            ;;
        "--help"|"-h"|"help")
            show_usage
            ;;
        *)
            print_error "Unknown service: $1"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
