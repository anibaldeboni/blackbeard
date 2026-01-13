#!/bin/bash

# ============================================================================
# Docker Volumes Backup Script
# Usa a label "backup.enable=true" para identificar volumes importantes
# ============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Configurações
BACKUP_DIR="${BACKUP_DIR:-./backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_PATH="$BACKUP_DIR/$TIMESTAMP"

# Criar diretório de backup
mkdir -p "$BACKUP_PATH"

# Função para listar volumes marcados
list_volumes_to_backup() {
    print_header "Volumes Marcados para Backup"

    docker volume ls --filter "label=backup.enable=true" --format "table {{.Name}}\t{{.Driver}}\t{{.Labels}}"
}

# Função para fazer backup de um volume
backup_volume() {
    local VOLUME_NAME=$1
    local BACKUP_FILE="$BACKUP_PATH/${VOLUME_NAME}.tar.gz"

    print_info "Backing up volume: $VOLUME_NAME"

    # Criar container temporário para acessar o volume
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

# Função para fazer backup de todos os volumes marcados
backup_all_labeled() {
    print_header "Starting Backup Process"

    # Obter lista de volumes com label
    VOLUMES=$(docker volume ls --filter "label=backup.enable=true" -q)

    if [ -z "$VOLUMES" ]; then
        print_warning "No volumes found with label 'backup.enable=true'"
        return 1
    fi

    local TOTAL=$(echo "$VOLUMES" | wc -l)
    print_info "Found $TOTAL volumes to backup"
    print_info "Backup destination: $BACKUP_PATH"

    echo ""

    # Fazer backup de cada volume
    local COUNT=0
    for VOLUME in $VOLUMES; do
        COUNT=$((COUNT + 1))
        echo ""
        print_info "[$COUNT/$TOTAL] Processing $VOLUME"
        backup_volume "$VOLUME"
    done

    echo ""
    print_header "Backup Summary"
    print_success "Backup completed successfully"
    print_info "Location: $BACKUP_PATH"
    print_info "Total size: $(du -sh $BACKUP_PATH | cut -f1)"

    # Listar arquivos criados
    echo ""
    print_info "Backup files:"
    ls -lh "$BACKUP_PATH"
}

# Função para restaurar um volume
restore_volume() {
    local BACKUP_FILE=$1
    local VOLUME_NAME=$2

    if [ ! -f "$BACKUP_FILE" ]; then
        print_error "Backup file not found: $BACKUP_FILE"
        return 1
    fi

    if [ -z "$VOLUME_NAME" ]; then
        # Tentar extrair nome do arquivo
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

    # Criar volume se não existir
    docker volume create "$VOLUME_NAME" 2>/dev/null || true

    # Restaurar dados
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

# Função para listar backups disponíveis
list_backups() {
    print_header "Available Backups"

    if [ ! -d "$BACKUP_DIR" ]; then
        print_warning "No backups found in $BACKUP_DIR"
        return 1
    fi

    for BACKUP in $(ls -dt $BACKUP_DIR/*/); do
        local DATE=$(basename $BACKUP)
        local SIZE=$(du -sh $BACKUP | cut -f1)
        local COUNT=$(ls -1 $BACKUP/*.tar.gz 2>/dev/null | wc -l)

        echo ""
        print_info "Backup: $DATE"
        echo "  Size: $SIZE"
        echo "  Files: $COUNT volumes"
        echo "  Location: $BACKUP"
    done
}

# Função para limpar backups antigos
cleanup_old_backups() {
    local KEEP_DAYS=${1:-7}

    print_header "Cleaning Old Backups"
    print_warning "Removing backups older than $KEEP_DAYS days"

    find "$BACKUP_DIR" -maxdepth 1 -type d -mtime +$KEEP_DAYS -exec rm -rf {} \;

    print_success "Cleanup completed"
}

# Menu interativo
show_menu() {
    echo ""
    print_header "Docker Volumes Backup Menu"
    echo "1. List volumes marked for backup"
    echo "2. Backup all marked volumes"
    echo "3. Backup specific volume"
    echo "4. List available backups"
    echo "5. Restore volume from backup"
    echo "6. Cleanup old backups (7+ days)"
    echo "0. Exit"
    echo ""
}

# Parse argumentos
case "${1:-menu}" in
    --list|list)
        list_volumes_to_backup
        ;;
    --backup|backup)
        backup_all_labeled
        ;;
    --backup-volume)
        if [ -z "$2" ]; then
            print_error "Please specify volume name"
            exit 1
        fi
        backup_volume "$2"
        ;;
    --restore)
        if [ -z "$2" ]; then
            print_error "Please specify backup file"
            exit 1
        fi
        restore_volume "$2" "$3"
        ;;
    --list-backups)
        list_backups
        ;;
    --cleanup)
        cleanup_old_backups "${2:-7}"
        ;;
    --help|help|-h)
        echo "Docker Volumes Backup Script"
        echo ""
        echo "Uses label 'backup.enable=true' to identify important volumes"
        echo ""
        echo "Usage: $0 [command] [options]"
        echo ""
        echo "Commands:"
        echo "  list                    List volumes marked for backup"
        echo "  backup                  Backup all marked volumes"
        echo "  backup-volume <name>    Backup specific volume"
        echo "  restore <file> [name]   Restore volume from backup"
        echo "  list-backups            List available backups"
        echo "  cleanup [days]          Remove backups older than N days (default: 7)"
        echo "  help                    Show this help"
        echo ""
        echo "Environment:"
        echo "  BACKUP_DIR              Backup destination (default: ./backups)"
        echo ""
        echo "Examples:"
        echo "  $0 list                           # List volumes to backup"
        echo "  $0 backup                         # Backup all marked volumes"
        echo "  BACKUP_DIR=/mnt/nas $0 backup     # Backup to NAS"
        echo "  $0 restore backups/20260112_120000/radarr-config.tar.gz"
        ;;
    menu|*)
        while true; do
            show_menu
            read -p "Select option: " choice

            case $choice in
                1) list_volumes_to_backup ;;
                2) backup_all_labeled ;;
                3)
                    echo ""
                    read -p "Enter volume name: " vol
                    backup_volume "$vol"
                    ;;
                4) list_backups ;;
                5)
                    echo ""
                    read -p "Enter backup file path: " file
                    read -p "Enter volume name (or leave empty to auto-detect): " vol
                    restore_volume "$file" "$vol"
                    ;;
                6) cleanup_old_backups 7 ;;
                0) print_info "Exiting..."; exit 0 ;;
                *) print_error "Invalid option" ;;
            esac

            echo ""
            read -p "Press Enter to continue..."
        done
        ;;
esac
