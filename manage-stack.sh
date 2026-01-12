#!/bin/bash

# ============================================================================
# Media Stack Management Script
# ============================================================================
# This script helps manage the media stack with common operations

set -e

COMPOSE_FILE="docker-compose.yml"
ENV_FILE=".env"
ENV_EXAMPLE=".env.example"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
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

validate_config() {
    print_header "Validating Configuration"
    if docker compose config --quiet; then
        print_success "Docker Compose configuration is valid"
    else
        print_error "Docker Compose configuration has errors"
        exit 1
    fi
}

show_status() {
    print_header "Stack Status"
    docker compose ps
}

show_health() {
    print_header "Health Status"
    docker compose ps --format "table {{.Name}}\t{{.Status}}\t{{.State}}"
}

show_logs() {
    SERVICE=${1:-}
    if [ -z "$SERVICE" ]; then
        docker compose logs -f --tail=100
    else
        docker compose logs -f --tail=100 "$SERVICE"
    fi
}

start_stack() {
    check_env_file
    check_network
    validate_config

    print_header "Starting Media Stack"
    docker compose up -d
    print_success "Stack started successfully"

    echo ""
    print_warning "Waiting for services to be healthy (this may take a few minutes)..."
    sleep 10
    show_health
}

stop_stack() {
    print_header "Stopping Media Stack"
    docker compose down
    print_success "Stack stopped successfully"
}

restart_stack() {
    stop_stack
    echo ""
    start_stack
}

restart_service() {
    SERVICE=$1
    if [ -z "$SERVICE" ]; then
        print_error "Please specify a service name"
        exit 1
    fi

    print_header "Restarting $SERVICE"
    docker compose restart "$SERVICE"
    print_success "$SERVICE restarted successfully"
}

update_stack() {
    print_header "Updating Stack Images"
    docker compose pull
    print_success "Images updated successfully"

    echo ""
    print_warning "Recreating containers with new images..."
    docker compose up -d --force-recreate
    print_success "Stack updated successfully"
}

backup_configs() {
    BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"
    print_header "Backing up configurations"

    mkdir -p "$BACKUP_DIR"

    # Backup compose file and env
    cp "$COMPOSE_FILE" "$BACKUP_DIR/"
    cp "$ENV_FILE" "$BACKUP_DIR/"

    # Backup config directories
    if [ -d "config" ]; then
        cp -r config "$BACKUP_DIR/"
    fi

    print_success "Backup created at $BACKUP_DIR"
}

show_resources() {
    print_header "Resource Usage"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"
}

show_help() {
    cat << EOF
Media Stack Management Script

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    start           Start the entire stack
    stop            Stop the entire stack
    restart         Restart the entire stack
    status          Show container status
    health          Show health check status
    logs [service]  Show logs (optionally for specific service)
    restart-svc     Restart a specific service
    update          Pull new images and recreate containers
    backup          Backup configurations
    resources       Show resource usage
    validate        Validate docker-compose configuration
    help            Show this help message

Examples:
    $0 start                    # Start all services
    $0 logs                     # Show all logs
    $0 logs radarr              # Show radarr logs only
    $0 restart-svc sonarr       # Restart sonarr service
    $0 update                   # Update all images

EOF
}

# Main script
case "${1:-help}" in
    start)
        start_stack
        ;;
    stop)
        stop_stack
        ;;
    restart)
        restart_stack
        ;;
    status)
        show_status
        ;;
    health)
        show_health
        ;;
    logs)
        show_logs "$2"
        ;;
    restart-svc)
        restart_service "$2"
        ;;
    update)
        update_stack
        ;;
    backup)
        backup_configs
        ;;
    resources)
        show_resources
        ;;
    validate)
        validate_config
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
