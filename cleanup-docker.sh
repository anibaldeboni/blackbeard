#!/bin/bash

# ============================================================================
# Docker Cleanup Script - Remove Unused Images and Resources
# ============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Function to show disk usage
show_disk_usage() {
    print_header "Docker Disk Usage"
    docker system df -v
}

# Function to list unused images
list_unused_images() {
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

# Function to remove dangling images only
remove_dangling() {
    print_header "Removing Dangling Images"

    DANGLING_COUNT=$(docker images -f "dangling=true" -q | wc -l)

    if [ "$DANGLING_COUNT" -gt 0 ]; then
        print_info "Found $DANGLING_COUNT dangling images"
        docker image prune -f
        print_success "Dangling images removed"
    else
        print_info "No dangling images found"
    fi
}

# Function to remove all unused images
remove_all_unused() {
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

# Function to remove old images (older than X days)
remove_old_images() {
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

# Function to clean everything (containers, images, volumes, networks)
clean_all() {
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

# Function to list and optionally remove specific images
remove_specific() {
    print_header "Remove Specific Images"

    echo "Available images:"
    docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}"

    echo ""
    read -p "Enter image ID or name:tag to remove (or 'cancel'): " IMAGE

    if [[ $IMAGE == "cancel" ]]; then
        print_info "Operation cancelled"
        return
    fi

    if [[ -n $IMAGE ]]; then
        docker rmi "$IMAGE" && print_success "Image removed" || print_error "Failed to remove image"
    fi
}

# Function to show what would be removed (dry-run)
dry_run() {
    print_header "Dry Run - What Would Be Removed"

    echo "Dangling images:"
    docker images -f "dangling=true" -q | wc -l

    echo ""
    echo "All unused images:"
    docker images --format "{{.ID}}" | while read id; do
        if ! docker ps -a --format "{{.Image}}" | grep -q "$id"; then
            docker images --filter "id=$id" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"
        fi
    done

    echo ""
    print_info "Run with --execute to actually remove these"
}

# Function to show protected images (from running stack)
show_protected() {
    print_header "Protected Images (In Use by Containers)"

    docker ps -a --format "table {{.Image}}\t{{.Names}}\t{{.Status}}"
}

# Main menu
show_menu() {
    echo ""
    print_header "Docker Cleanup Menu"
    echo "1. Show disk usage"
    echo "2. List unused images"
    echo "3. Remove dangling images only (safe)"
    echo "4. Remove all unused images"
    echo "5. Remove old images (30+ days)"
    echo "6. Remove specific image"
    echo "7. Complete cleanup (everything)"
    echo "8. Dry run (show what would be removed)"
    echo "9. Show protected images"
    echo "0. Exit"
    echo ""
}

# Parse command line arguments
case "${1:-menu}" in
    --disk-usage|disk)
        show_disk_usage
        ;;
    --list|list)
        list_unused_images
        ;;
    --dangling|dangling)
        remove_dangling
        ;;
    --all|all)
        remove_all_unused
        ;;
    --old|old)
        remove_old_images "${2:-30}"
        ;;
    --clean|clean)
        clean_all
        ;;
    --dry-run|dry)
        dry_run
        ;;
    --protected|protected)
        show_protected
        ;;
    --help|help|-h)
        echo "Docker Cleanup Script"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  disk              Show disk usage"
        echo "  list              List unused images"
        echo "  dangling          Remove dangling images (safe)"
        echo "  all               Remove all unused images"
        echo "  old [days]        Remove images older than N days (default: 30)"
        echo "  clean             Complete cleanup (everything)"
        echo "  dry               Dry run (show what would be removed)"
        echo "  protected         Show protected images"
        echo "  help              Show this help"
        echo ""
        echo "Interactive:"
        echo "  $0                Run interactive menu"
        ;;
    menu|*)
        while true; do
            show_menu
            read -p "Select option: " choice

            case $choice in
                1) show_disk_usage ;;
                2) list_unused_images ;;
                3) remove_dangling ;;
                4) remove_all_unused ;;
                5) remove_old_images 30 ;;
                6) remove_specific ;;
                7) clean_all ;;
                8) dry_run ;;
                9) show_protected ;;
                0) print_info "Exiting..."; exit 0 ;;
                *) print_error "Invalid option" ;;
            esac

            echo ""
            read -p "Press Enter to continue..."
        done
        ;;
esac
