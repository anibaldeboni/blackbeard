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

# Function to detect host timezone (Linux only)
get_host_timezone() {
    local timezone=""
    
    # Try multiple methods to get the timezone on Linux
    if [[ -f /etc/timezone ]]; then
        # Debian/Ubuntu/Armbian
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

# Function to detect SBC information and system capabilities
detect_sbc_info() {
    print_status "Detecting SBC hardware and capabilities..."
    
    # Detect architecture
    SYSTEM_ARCH=$(dpkg --print-architecture 2>/dev/null || uname -m)
    
    # Detect SBC model
    SBC_MODEL="Unknown"
    if [[ -f /proc/device-tree/model ]]; then
        SBC_MODEL=$(tr -d '\0' < /proc/device-tree/model 2>/dev/null || echo "Unknown")
    elif [[ -f /sys/firmware/devicetree/base/model ]]; then
        SBC_MODEL=$(tr -d '\0' < /sys/firmware/devicetree/base/model 2>/dev/null || echo "Unknown")
    fi
    
    # Detect total RAM in MB
    TOTAL_RAM_MB=$(free -m | awk '/^Mem:/ {print $2}')
    
    # Detect CPU cores
    CPU_CORES=$(nproc)
    
    # Detect if running on ARM
    IS_ARM=false
    if [[ "$SYSTEM_ARCH" =~ ^(arm|aarch64)$ ]] || [[ "$(uname -m)" =~ ^(arm|aarch64)$ ]]; then
        IS_ARM=true
    fi
    
    # Set resource limits based on available RAM
    if [[ $TOTAL_RAM_MB -le 1024 ]]; then
        RESOURCE_PROFILE="minimal"  # 1GB or less
        MAX_CONCURRENT_SERVICES=2
    elif [[ $TOTAL_RAM_MB -le 2048 ]]; then
        RESOURCE_PROFILE="low"      # 1-2GB
        MAX_CONCURRENT_SERVICES=3
    elif [[ $TOTAL_RAM_MB -le 4096 ]]; then
        RESOURCE_PROFILE="medium"   # 2-4GB
        MAX_CONCURRENT_SERVICES=5
    else
        RESOURCE_PROFILE="high"     # 4GB+
        MAX_CONCURRENT_SERVICES=9
    fi
    
    # Auto-detect user IDs
    DETECTED_PUID=$(id -u)
    DETECTED_PGID=$(id -g)
    
    print_status "System Information:"
    print_status "  - SBC Model: $SBC_MODEL"
    print_status "  - Architecture: $SYSTEM_ARCH"
    print_status "  - ARM System: $IS_ARM"
    print_status "  - RAM: ${TOTAL_RAM_MB}MB"
    print_status "  - CPU Cores: $CPU_CORES"
    print_status "  - Resource Profile: $RESOURCE_PROFILE"
    print_status "  - User ID: $DETECTED_PUID:$DETECTED_PGID"
}

# Function to verify ARM compatibility and suggest alternatives
verify_arm_compatibility() {
    if [[ "$IS_ARM" == "true" ]]; then
        print_status "Verifying ARM compatibility for container images..."
        
        # Define ARM-compatible image alternatives
        declare -A ARM_IMAGES
        ARM_IMAGES[qbittorrent]="lscr.io/linuxserver/qbittorrent:latest"
        ARM_IMAGES[radarr]="lscr.io/linuxserver/radarr:latest"
        ARM_IMAGES[sonarr]="lscr.io/linuxserver/sonarr:latest"
        ARM_IMAGES[prowlarr]="lscr.io/linuxserver/prowlarr:latest"
        ARM_IMAGES[jellyseerr]="fallenbagel/jellyseerr:latest"
        ARM_IMAGES[jellyfin]="lscr.io/linuxserver/jellyfin:latest"
        ARM_IMAGES[flaresolverr]="ghcr.io/flaresolverr/flaresolverr:latest"
        ARM_IMAGES[nginx]="nginx:alpine"
        ARM_IMAGES[bazarr]="lscr.io/linuxserver/bazarr:latest"
        
        print_status "Using ARM-compatible image versions:"
        for service in "${!ARM_IMAGES[@]}"; do
            print_status "  - $service: ${ARM_IMAGES[$service]}"
        done
        
        # Warning for resource-intensive services on lower-end SBCs
        if [[ "$RESOURCE_PROFILE" == "minimal" || "$RESOURCE_PROFILE" == "low" ]]; then
            print_warning "Resource Warning for ${RESOURCE_PROFILE} profile:"
            print_warning "  - Consider disabling Jellyfin transcoding for better performance"
            print_warning "  - qBittorrent may need connection limits reduced"
            print_warning "  - Monitor system temperature during operation"
        fi
    else
        print_status "Non-ARM system detected, using standard image versions"
    fi
}

# Function to calculate resource limits based on system capabilities
calculate_resource_limits() {
    print_status "Calculating optimal resource limits for $RESOURCE_PROFILE profile..."
    
    # Calculate memory limits based on available RAM and service priority
    case "$RESOURCE_PROFILE" in
        "minimal")  # 1GB or less
            # Very conservative limits
            MEMORY_LIMIT_HIGH="256m"     # Jellyfin, qBittorrent
            MEMORY_LIMIT_MEDIUM="128m"   # Radarr, Sonarr, Prowlarr
            MEMORY_LIMIT_LOW="64m"       # Bazarr, Jellyseerr, FlareSolverr
            MEMORY_LIMIT_NGINX="32m"     # Nginx
            CPU_LIMIT="1.0"              # Max 1 CPU core
            ;;
        "low")      # 1-2GB
            MEMORY_LIMIT_HIGH="512m"
            MEMORY_LIMIT_MEDIUM="256m"
            MEMORY_LIMIT_LOW="128m"
            MEMORY_LIMIT_NGINX="64m"
            CPU_LIMIT="1.5"
            ;;
        "medium")   # 2-4GB
            MEMORY_LIMIT_HIGH="1g"
            MEMORY_LIMIT_MEDIUM="512m"
            MEMORY_LIMIT_LOW="256m"
            MEMORY_LIMIT_NGINX="128m"
            CPU_LIMIT="2.0"
            ;;
        "high")     # 4GB+
            MEMORY_LIMIT_HIGH="2g"
            MEMORY_LIMIT_MEDIUM="1g"
            MEMORY_LIMIT_LOW="512m"
            MEMORY_LIMIT_NGINX="256m"
            CPU_LIMIT="3.0"
            ;;
    esac
    
    # Calculate disk I/O limits for SD card protection
    if [[ -f /sys/block/mmcblk0/queue/rotational ]] && [[ "$(cat /sys/block/mmcblk0/queue/rotational)" == "0" ]]; then
        # SD card detected - apply I/O limits
        IO_WEIGHT="100"  # Lower I/O priority
        print_status "SD card detected - applying I/O limits for longevity"
    else
        IO_WEIGHT="500"  # Normal I/O priority
    fi
    
    print_status "Resource limits configured:"
    print_status "  - High priority: ${MEMORY_LIMIT_HIGH}, CPU: ${CPU_LIMIT}"
    print_status "  - Medium priority: ${MEMORY_LIMIT_MEDIUM}"
    print_status "  - Low priority: ${MEMORY_LIMIT_LOW}"
    print_status "  - I/O weight: ${IO_WEIGHT}"
}

# Function to build resource limit arguments for containers
build_resource_args() {
    local priority="$1"
    local memory_limit=""
    local cpu_limit="$CPU_LIMIT"
    
    case "$priority" in
        "high")
            memory_limit="$MEMORY_LIMIT_HIGH"
            ;;
        "medium")
            memory_limit="$MEMORY_LIMIT_MEDIUM"
            ;;
        "low")
            memory_limit="$MEMORY_LIMIT_LOW"
            ;;
        "nginx")
            memory_limit="$MEMORY_LIMIT_NGINX"
            cpu_limit="0.5"
            ;;
        *)
            memory_limit="$MEMORY_LIMIT_MEDIUM"
            ;;
    esac
    
    echo "--memory=$memory_limit --cpus=$cpu_limit --blkio-weight=$IO_WEIGHT"
}

# Centralized service configurations
declare -A SERVICE_CONFIGS

# Initialize service configurations
init_service_configs() {
    # qBittorrent configuration
    SERVICE_CONFIGS["qbittorrent_image"]="lscr.io/linuxserver/qbittorrent:latest"
    SERVICE_CONFIGS["qbittorrent_priority"]="high"
    SERVICE_CONFIGS["qbittorrent_ports"]="-p 5080:5080 -p 6881:6881 -p 6881:6881/udp"
    SERVICE_CONFIGS["qbittorrent_volumes"]="-v ${SCRIPT_DIR}/config/qbittorrent:/config -v ${DOWNLOADS_DIR}:/downloads"
    SERVICE_CONFIGS["qbittorrent_env"]="-e WEBUI_PORT=5080 -e DOCKER_MODS=ghcr.io/vuetorrent/vuetorrent-lsio-mod:latest"
    SERVICE_CONFIGS["qbittorrent_extra"]=""
    
    # Radarr configuration
    SERVICE_CONFIGS["radarr_image"]="lscr.io/linuxserver/radarr:latest"
    SERVICE_CONFIGS["radarr_priority"]="medium"
    SERVICE_CONFIGS["radarr_ports"]="-p 7878:7878"
    SERVICE_CONFIGS["radarr_volumes"]="-v ${SCRIPT_DIR}/config/radarr:/config -v ${DOWNLOADS_DIR}:/downloads"
    SERVICE_CONFIGS["radarr_env"]=""
    SERVICE_CONFIGS["radarr_extra"]=""
    
    # Sonarr configuration
    SERVICE_CONFIGS["sonarr_image"]="lscr.io/linuxserver/sonarr:latest"
    SERVICE_CONFIGS["sonarr_priority"]="medium"
    SERVICE_CONFIGS["sonarr_ports"]="-p 8989:8989"
    SERVICE_CONFIGS["sonarr_volumes"]="-v ${SCRIPT_DIR}/config/sonarr:/config -v ${DOWNLOADS_DIR}:/downloads"
    SERVICE_CONFIGS["sonarr_env"]=""
    SERVICE_CONFIGS["sonarr_extra"]=""
    
    # Prowlarr configuration
    SERVICE_CONFIGS["prowlarr_image"]="lscr.io/linuxserver/prowlarr:latest"
    SERVICE_CONFIGS["prowlarr_priority"]="medium"
    SERVICE_CONFIGS["prowlarr_ports"]="-p 9696:9696"
    SERVICE_CONFIGS["prowlarr_volumes"]="-v ${SCRIPT_DIR}/config/prowlarr:/config"
    SERVICE_CONFIGS["prowlarr_env"]=""
    SERVICE_CONFIGS["prowlarr_extra"]=""
    
    # Jellyseerr configuration
    SERVICE_CONFIGS["jellyseerr_image"]="fallenbagel/jellyseerr:latest"
    SERVICE_CONFIGS["jellyseerr_priority"]="low"
    SERVICE_CONFIGS["jellyseerr_ports"]="-p 5055:5055"
    SERVICE_CONFIGS["jellyseerr_volumes"]="-v ${SCRIPT_DIR}/config/jellyseerr:/app/config"
    SERVICE_CONFIGS["jellyseerr_env"]=""
    SERVICE_CONFIGS["jellyseerr_extra"]=""
    
    # Jellyfin configuration (special handling for GPU)
    SERVICE_CONFIGS["jellyfin_image"]="lscr.io/linuxserver/jellyfin:latest"
    SERVICE_CONFIGS["jellyfin_priority"]="high"
    SERVICE_CONFIGS["jellyfin_ports"]="-p 8096:8096 -p 7359:7359/udp -p 8920:8920"
    SERVICE_CONFIGS["jellyfin_volumes"]="-v ${SCRIPT_DIR}/config/jellyfin:/config -v ${DOWNLOADS_DIR}:/downloads"
    SERVICE_CONFIGS["jellyfin_env"]=""
    SERVICE_CONFIGS["jellyfin_extra"]=""  # GPU args will be added dynamically
    
    # FlareSolverr configuration
    SERVICE_CONFIGS["flaresolverr_image"]="ghcr.io/flaresolverr/flaresolverr:latest"
    SERVICE_CONFIGS["flaresolverr_priority"]="low"
    SERVICE_CONFIGS["flaresolverr_ports"]="-p ${PORT:-8191}:8191"
    SERVICE_CONFIGS["flaresolverr_volumes"]=""
    SERVICE_CONFIGS["flaresolverr_env"]="-e LOG_LEVEL=${LOG_LEVEL:-info} -e LOG_HTML=${LOG_HTML:-false} -e CAPTCHA_SOLVER=${CAPTCHA_SOLVER:-none}"
    SERVICE_CONFIGS["flaresolverr_extra"]=""
    
    # Nginx configuration
    SERVICE_CONFIGS["nginx_image"]="nginx:alpine"
    SERVICE_CONFIGS["nginx_priority"]="nginx"
    SERVICE_CONFIGS["nginx_ports"]="-p 8080:80"
    SERVICE_CONFIGS["nginx_volumes"]="-v ${SCRIPT_DIR}/config/nginx/nginx.conf:/etc/nginx/nginx.conf:ro -v ${SCRIPT_DIR}/config/nginx/logs:/var/log/nginx"
    SERVICE_CONFIGS["nginx_env"]=""
    SERVICE_CONFIGS["nginx_extra"]=""
    
    # Bazarr configuration
    SERVICE_CONFIGS["bazarr_image"]="lscr.io/linuxserver/bazarr:latest"
    SERVICE_CONFIGS["bazarr_priority"]="low"
    SERVICE_CONFIGS["bazarr_ports"]="-p 6767:6767"
    SERVICE_CONFIGS["bazarr_volumes"]="-v ${SCRIPT_DIR}/config/bazarr:/config -v ${DOWNLOADS_DIR}:/downloads"
    SERVICE_CONFIGS["bazarr_env"]=""
    SERVICE_CONFIGS["bazarr_extra"]=""
}

# Generic function to run any container based on service configuration
run_container() {
    local service_name="$1"
    
    # Initialize configurations if not done
    if [[ -z "${SERVICE_CONFIGS["${service_name}_image"]}" ]]; then
        init_service_configs
    fi
    
    print_header "Starting $service_name service"
    stop_container "$service_name"
    
    # Get service configuration
    local image="${SERVICE_CONFIGS["${service_name}_image"]}"
    local priority="${SERVICE_CONFIGS["${service_name}_priority"]}"
    local ports="${SERVICE_CONFIGS["${service_name}_ports"]}"
    local volumes="${SERVICE_CONFIGS["${service_name}_volumes"]}"
    local env="${SERVICE_CONFIGS["${service_name}_env"]}"
    local extra="${SERVICE_CONFIGS["${service_name}_extra"]}"
    
    # Build resource arguments
    local resource_args
    resource_args=$(build_resource_args "$priority")
    
    # Build standard environment variables (for LinuxServer containers)
    local standard_env=""
    if [[ "$service_name" != "nginx" && "$service_name" != "flaresolverr" ]]; then
        standard_env="-e PUID=$DETECTED_PUID -e PGID=$DETECTED_PGID -e UMASK=002 -e TZ=$HOST_TIMEZONE"
    elif [[ "$service_name" == "flaresolverr" ]]; then
        standard_env="-e TZ=$HOST_TIMEZONE"
    fi
    
    # Special handling for Jellyfin GPU
    if [[ "$service_name" == "jellyfin" ]]; then
        extra=$(get_jellyfin_gpu_args)
    fi
    
    # Run the container
    podman run -d \
        --name "$service_name" \
        --network "$NETWORK_NAME" \
        --restart unless-stopped \
        $resource_args \
        $standard_env \
        $env \
        $volumes \
        $ports \
        $extra \
        "$image"
    
    print_status "$service_name container started"
    
    # Special messages
    if [[ "$service_name" == "qbittorrent" ]]; then
        print_warning "Default credentials - Username: admin, check logs for password: podman logs qbittorrent"
    elif [[ "$service_name" == "nginx" ]]; then
        print_status "Nginx container started in rootless mode on port 8080"
        print_status "Configure your system to forward port 80 -> 8080 if needed"
    fi
}

# Function to get Jellyfin GPU arguments
get_jellyfin_gpu_args() {
    local gpu_args=""
    local gpu_type="none"
    
    # Check for various ARM GPU types
    if [[ -e /dev/dri ]]; then
        # Generic DRM devices (works for most modern SBCs)
        gpu_args="--device /dev/dri:/dev/dri --group-add 44 --group-add 105"
        gpu_type="DRM"
        print_status "GPU acceleration enabled: Generic DRM"
    elif [[ -e /dev/mali0 ]] || [[ -e /dev/mali1 ]]; then
        # ARM Mali GPU (common in many SBCs)
        gpu_args="--device /dev/mali0:/dev/mali0 --group-add 44"
        gpu_type="Mali"
        print_status "GPU acceleration enabled: ARM Mali"
    elif [[ -e /dev/video10 ]] || [[ -e /dev/video11 ]]; then
        # Video4Linux devices (some SBCs expose hardware codecs this way)
        gpu_args="--device /dev/video10:/dev/video10 --device /dev/video11:/dev/video11 --group-add 44"
        gpu_type="V4L2"
        print_status "GPU acceleration enabled: Video4Linux"
    elif [[ "$SBC_MODEL" == *"Raspberry Pi"* ]]; then
        # Raspberry Pi specific GPU handling
        if [[ -e /dev/vchiq ]]; then
            gpu_args="--device /dev/vchiq:/dev/vchiq --group-add 44"
            gpu_type="VideoCore"
            print_status "GPU acceleration enabled: VideoCore (Raspberry Pi)"
        fi
    else
        print_warning "No GPU acceleration available for this SBC"
        if [[ "$RESOURCE_PROFILE" == "minimal" || "$RESOURCE_PROFILE" == "low" ]]; then
            print_warning "Transcoding will be CPU-only - consider disabling it for better performance"
        fi
    fi
    
    echo "$gpu_args"
}

# Function to check system health before starting services
check_system_health() {
    print_status "Performing system health checks..."
    
    local warnings=0
    
    # Check available disk space
    local available_space_gb
    available_space_gb=$(df "${SCRIPT_DIR}" | awk 'NR==2 {print int($4/1024/1024)}')
    if [[ $available_space_gb -lt 5 ]]; then
        print_error "Critical: Only ${available_space_gb}GB disk space available"
        print_error "At least 5GB recommended for container operations"
        ((warnings++))
    elif [[ $available_space_gb -lt 10 ]]; then
        print_warning "Warning: Only ${available_space_gb}GB disk space available"
        print_warning "Consider freeing up space for optimal performance"
        ((warnings++))
    else
        print_status "Disk space: ${available_space_gb}GB available"
    fi
    
    # Check memory usage
    local available_mem_mb
    available_mem_mb=$(free -m | awk '/^Mem:/ {print $7}')
    local mem_usage_percent
    mem_usage_percent=$(free | awk '/^Mem:/ {printf "%.0f", ($3/$2)*100}')
    
    if [[ $mem_usage_percent -gt 80 ]]; then
        print_warning "High memory usage: ${mem_usage_percent}% (${available_mem_mb}MB available)"
        print_warning "Consider stopping other services before starting containers"
        ((warnings++))
    else
        print_status "Memory usage: ${mem_usage_percent}% (${available_mem_mb}MB available)"
    fi
    
    # Check CPU temperature (if available)
    local temp_celsius=""
    if [[ -f /sys/class/thermal/thermal_zone0/temp ]]; then
        local temp_millis
        temp_millis=$(cat /sys/class/thermal/thermal_zone0/temp)
        temp_celsius=$((temp_millis / 1000))
        
        if [[ $temp_celsius -gt 70 ]]; then
            print_warning "High CPU temperature: ${temp_celsius}°C"
            print_warning "Consider improving cooling before heavy workloads"
            ((warnings++))
        elif [[ $temp_celsius -gt 60 ]]; then
            print_warning "Moderate CPU temperature: ${temp_celsius}°C"
        else
            print_status "CPU temperature: ${temp_celsius}°C"
        fi
    fi
    
    # Check load average
    local load_avg
    load_avg=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | sed 's/,//')
    local load_threshold
    load_threshold=$(echo "$CPU_CORES * 0.7" | bc -l | cut -d. -f1)
    
    if (( $(echo "$load_avg > $load_threshold" | bc -l) )); then
        print_warning "High system load: ${load_avg} (threshold: ${load_threshold})"
        print_warning "System may be under stress"
        ((warnings++))
    else
        print_status "System load: ${load_avg}"
    fi
    
    # Summary
    if [[ $warnings -eq 0 ]]; then
        print_status "System health: All checks passed"
        return 0
    else
        print_warning "System health: ${warnings} warning(s) detected"
        if [[ "$RESOURCE_PROFILE" == "minimal" && $warnings -gt 2 ]]; then
            print_error "Multiple issues detected on minimal system - consider addressing before proceeding"
            return 1
        fi
        return 0
    fi
}

# Function to monitor system load during service startup
monitor_system_load() {
    local max_wait=30
    local wait_count=0
    
    while [[ $wait_count -lt $max_wait ]]; do
        local current_load
        current_load=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | sed 's/,//')
        local load_threshold
        load_threshold=$(echo "$CPU_CORES * 1.5" | bc -l | cut -d. -f1)
        
        if (( $(echo "$current_load < $load_threshold" | bc -l) )); then
            return 0  # Load is acceptable
        fi
        
        print_warning "High load detected: ${current_load}, waiting..."
        sleep 2
        ((wait_count++))
    done
    
    print_warning "Load still high after waiting, proceeding anyway"
    return 1
}

# Function to start services sequentially with intelligent resource management
start_services_sequentially() {
    print_header "Starting services sequentially based on resource profile: $RESOURCE_PROFILE"
    
    # Define service startup order and priorities
    local high_priority=("nginx" "prowlarr")
    local medium_priority=("qbittorrent" "radarr" "sonarr")
    local low_priority=("jellyseerr" "bazarr" "flaresolverr")
    local resource_intensive=("jellyfin")
    
    # Start high priority services first (lightweight, essential)
    for service in "${high_priority[@]}"; do
        if [[ " ${SERVICES[*]} " =~ \ ${service}\  ]]; then
            print_status "Starting high priority service: $service"
            "run_$service"
            
            # Short wait for service to initialize
            sleep 3
            monitor_system_load
        fi
    done
    
    # Start medium priority services with load monitoring
    for service in "${medium_priority[@]}"; do
        if [[ " ${SERVICES[*]} " =~ \ ${service}\  ]]; then
            # Wait for system load to stabilize on minimal systems
            if [[ "$RESOURCE_PROFILE" == "minimal" ]]; then
                monitor_system_load
            fi
            
            print_status "Starting medium priority service: $service"
            "run_$service"
            
            # Longer wait between medium priority services
            sleep 5
        fi
    done
    
    # Start low priority services
    for service in "${low_priority[@]}"; do
        if [[ " ${SERVICES[*]} " =~ \ ${service}\  ]]; then
            if [[ "$RESOURCE_PROFILE" == "minimal" || "$RESOURCE_PROFILE" == "low" ]]; then
                monitor_system_load
            fi
            
            print_status "Starting low priority service: $service"
            "run_$service"
            sleep 3
        fi
    done
    
    # Start resource-intensive services last with careful monitoring
    for service in "${resource_intensive[@]}"; do
        if [[ " ${SERVICES[*]} " =~ \ ${service}\  ]]; then
            # Always monitor load before starting Jellyfin
            monitor_system_load
            
            print_status "Starting resource-intensive service: $service"
            "run_$service"
            
            # Give Jellyfin extra time to initialize
            sleep 8
        fi
    done
    
    print_header "Sequential startup completed"
    print_status "All requested services are starting up"
    print_status "Monitor with: podman ps"
}

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
    # Check if running on Linux
    if [[ "$OSTYPE" != "linux-gnu"* ]]; then
        print_error "This script only supports Linux systems. Please install Podman manually."
        exit 1
    fi
    
    # Check for Debian-based system (apt-get)
    if command -v apt-get &> /dev/null; then
        print_status "Installing Podman on ARM/Armbian system..."
        
        # Detect architecture
        local arch
        arch=$(dpkg --print-architecture)
        print_status "Detected architecture: $arch"
        
        # Update package list
        sudo apt-get update
        
        # Install required packages
        sudo apt-get install -y \
            software-properties-common \
            ca-certificates \
            curl \
            gnupg \
            lsb-release \
            apt-transport-https
        
        # Try to install from default repositories first (newer Armbian versions have Podman)
        if apt-cache search podman | grep -q "^podman "; then
            print_status "Installing Podman from default repositories..."
            sudo apt-get install -y podman
        else
            print_status "Adding Podman repository for Debian-based ARM systems..."
            
            # Use Debian repository for ARM systems
            local debian_version
            if [[ "$VERSION_CODENAME" ]]; then
                debian_version="$VERSION_CODENAME"
            elif [[ "$UBUNTU_CODENAME" ]]; then
                debian_version="$UBUNTU_CODENAME"
            else
                # Fallback for Armbian
                debian_version="bookworm"  # Default to Debian 12
            fi
            
            print_status "Using Debian codename: $debian_version"
            
            # Add the repository
            echo "deb https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/Debian_12/ /" | sudo tee /etc/apt/sources.list.d/devel:kubic:libcontainers:stable.list
            
            # Add the GPG key
            curl -fsSL https://download.opensuse.org/repositories/devel:kubic:libcontainers:stable/Debian_12/Release.key | gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/devel_kubic_libcontainers_stable.gpg > /dev/null
            
            # Update and install Podman
            sudo apt-get update
            sudo apt-get install -y podman
        fi
        
        print_status "Podman installation completed"
        
        # Configure Podman for rootless operation
        print_status "Configuring Podman for rootless operation..."
        
        # Set up subuid and subgid if not already configured
        if ! grep -q "^$USER:" /etc/subuid 2>/dev/null; then
            echo "$USER:100000:65536" | sudo tee -a /etc/subuid
            print_status "Configured subuid for user $USER"
        fi
        
        if ! grep -q "^$USER:" /etc/subgid 2>/dev/null; then
            echo "$USER:100000:65536" | sudo tee -a /etc/subgid
            print_status "Configured subgid for user $USER"
        fi
        
        # Initialize Podman for the current user
        if command -v podman &> /dev/null; then
            podman system migrate 2>/dev/null || true
            print_status "Podman configuration completed"
        fi
        
    else
        print_error "This script requires a Debian-based Linux distribution with apt-get."
        print_error "Supported systems: Armbian, Ubuntu, Debian, etc."
        print_error "Please install Podman manually or use a supported distribution."
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

# Function to setup firewall for nginx (UFW - Armbian default)
setup_nginx_firewall() {
    if command -v ufw &> /dev/null; then
        print_status "Configuring UFW firewall for nginx..."
        
        # Check if UFW is active
        if ! sudo ufw status | grep -q "Status: active"; then
            print_warning "UFW is not active. Enabling UFW..."
            sudo ufw --force enable
        fi
        
        # Check and add rules for HTTP traffic
        if ! sudo ufw status | grep -q "80/tcp"; then
            print_status "Adding HTTP port 80 to UFW..."
            sudo ufw allow 80/tcp comment "HTTP for nginx"
        else
            print_status "Port 80/tcp already allowed in UFW"
        fi
        
        # Check and add rules for nginx container port
        if ! sudo ufw status | grep -q "8080/tcp"; then
            print_status "Adding port 8080/tcp for nginx container..."
            sudo ufw allow 8080/tcp comment "Nginx container port"
        else
            print_status "Port 8080/tcp already allowed in UFW"
        fi
        
        # Note about port forwarding (iptables rules would be needed for actual forwarding)
        print_status "UFW rules configured for nginx"
        print_warning "For production use, configure port forwarding: 80 -> 8080"
        print_warning "Run: sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080"
        
    elif command -v firewall-cmd &> /dev/null; then
        # Fallback to firewalld if available (some custom Armbian setups)
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
        
        # Only reload if changes were made
        if [[ "$needs_reload" == "true" ]]; then
            print_status "Reloading firewall configuration..."
            sudo firewall-cmd --reload
            print_status "Firewall rules configured: ports 80, 8080"
        else
            print_status "All firewall rules already configured, no changes needed"
        fi
    else
        print_warning "No supported firewall found (UFW or firewalld)"
        print_warning "Please manually configure firewall to allow ports 80 and 8080"
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

# Function to create systemd service for a container
create_systemd_service() {
    local service_name="$1"
    
    print_status "Creating systemd service for $service_name"
    
    # Generate systemd unit file for the container
    podman generate systemd --new --files --name "$service_name" --restart-policy=always
    
    # Move the service file to systemd directory
    local service_file="container-${service_name}.service"
    if [[ -f "$service_file" ]]; then
        sudo mv "$service_file" "/etc/systemd/system/"
        
        # Reload systemd and enable the service
        sudo systemctl daemon-reload
        sudo systemctl enable "$service_file"
        
        print_status "Systemd service created and enabled: $service_file"
    else
        print_error "Failed to generate systemd service for $service_name"
    fi
}

# Function to enable systemd services for all containers
enable_systemd_services() {
    print_header "Enabling systemd services for auto-start on boot"
    
    # Enable lingering for the current user (allows user services to run without login)
    sudo loginctl enable-linger "$USER"
    
    for service in "${SERVICES[@]}"; do
        if podman container exists "$service" 2>/dev/null; then
            create_systemd_service "$service"
        else
            print_warning "Container $service not found, skipping systemd service creation"
        fi
    done
    
    print_status "All systemd services enabled. Containers will auto-start on boot."
    print_status "You can manage services with: sudo systemctl start/stop/restart container-<name>.service"
}

# Function to disable systemd services
disable_systemd_services() {
    print_header "Disabling systemd services"
    
    for service in "${SERVICES[@]}"; do
        local service_file="container-${service}.service"
        if systemctl is-enabled "$service_file" &>/dev/null; then
            print_status "Disabling systemd service: $service_file"
            sudo systemctl disable "$service_file"
            sudo systemctl stop "$service_file" 2>/dev/null || true
            sudo rm -f "/etc/systemd/system/$service_file"
        fi
    done
    
    sudo systemctl daemon-reload
    print_status "All systemd services disabled"
}

# Function to check systemd services status
check_systemd_status() {
    print_header "Systemd Services Status"
    
    echo "Service              Status     Active"
    echo "------------------------------------"
    for service in "${SERVICES[@]}"; do
        local service_file="container-${service}.service"
        if systemctl list-unit-files "$service_file" &>/dev/null; then
            local status
            local active
            status=$(systemctl is-enabled "$service_file" 2>/dev/null || echo "disabled")
            active=$(systemctl is-active "$service_file" 2>/dev/null || echo "inactive")
            printf "%-20s %-10s %-10s\n" "$service" "$status" "$active"
        else
            printf "%-20s %-10s %-10s\n" "$service" "not-found" "inactive"
        fi
    done
}

# Function to run qBittorrent
run_qbittorrent() {
    run_container "qbittorrent"
}

# Function to run Radarr
run_radarr() {
    run_container "radarr"
}

# Function to run Sonarr
run_sonarr() {
    run_container "sonarr"
}

# Function to run Prowlarr
run_prowlarr() {
    run_container "prowlarr"
}

# Function to run Jellyseerr
run_jellyseerr() {
    run_container "jellyseerr"
}

# Function to run Jellyfin
run_jellyfin() {
    run_container "jellyfin"
}

# Function to run FlareSolverr
run_flaresolverr() {
    run_container "flaresolverr"
}

# Function to run Nginx (rootless mode)
run_nginx() {
    # Setup firewall before starting nginx
    setup_nginx_firewall
    
    run_container "nginx"
}

# Function to run Bazarr
run_bazarr() {
    run_container "bazarr"
}

# Function to run all services using intelligent sequential startup
run_all_services() {
    # Calculate resource limits based on system
    calculate_resource_limits
    
    # Check system health before starting
    if ! check_system_health; then
        if [[ "$RESOURCE_PROFILE" == "minimal" ]]; then
            print_error "System health issues detected on minimal profile. Aborting startup."
            print_error "Please resolve issues before starting services."
            exit 1
        fi
    fi
    
    # Use sequential startup for better resource management
    start_services_sequentially
}

# Function to show service status
show_status() {
    print_header "Blackbeard Services Status"
    podman ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" --filter "network=$NETWORK_NAME"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [service_name|all|status|stop|enable-autostart|disable-autostart|autostart-status]"
    echo ""
    echo "Available services:"
    for service in "${SERVICES[@]}"; do
        echo "  - $service"
    done
    echo ""
    echo "Commands:"
    echo "  all                - Start all services"
    echo "  status             - Show running containers status"
    echo "  stop               - Stop all services"
    echo "  enable-autostart   - Enable auto-start on boot via systemd"
    echo "  disable-autostart  - Disable auto-start on boot"
    echo "  autostart-status   - Show auto-start services status"
    echo ""
    echo "Examples:"
    echo "  $0 nginx              # Start only nginx"
    echo "  $0 all                # Start all services"
    echo "  $0 enable-autostart   # Enable auto-start on boot"
    echo "  $0 status             # Show containers status"
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
    
    # Detect SBC and system information
    detect_sbc_info
    
    # Verify ARM compatibility
    verify_arm_compatibility
    
    # Calculate resource limits
    calculate_resource_limits
    
    # Initialize service configurations
    init_service_configs
    
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
        "enable-autostart")
            enable_systemd_services
            ;;
        "disable-autostart")
            disable_systemd_services
            ;;
        "autostart-status")
            check_systemd_status
            ;;
        "qbittorrent"|"radarr"|"sonarr"|"prowlarr"|"jellyseerr"|"jellyfin"|"flaresolverr"|"nginx"|"bazarr")
            # Ensure resource limits are calculated for individual services
            if [[ -z "$MEMORY_LIMIT_HIGH" ]]; then
                calculate_resource_limits
            fi
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
