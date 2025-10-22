#!/bin/bash
# filepath: cpu_temp.sh

get_temperature() {
    local temp_file="/sys/devices/virtual/thermal/thermal_zone0/temp"
    
    if [[ ! -f "$temp_file" ]]; then
        echo "Error: temperature file not found: $temp_file"
        exit 1
    fi
    
    local temp_millidegrees=$(cat "$temp_file")
    
    if [[ ! "$temp_millidegrees" =~ ^[0-9]+$ ]]; then
        echo "Error: Invalid temperature reading: $temp_millidegrees"
        exit 1
    fi
    
    local temp_celsius=$((temp_millidegrees / 1000))
    local temp_decimal=$((temp_millidegrees % 1000 / 100))
    
    echo "${temp_celsius}.${temp_decimal}°C"
}

monitor_temperature() {
    local interval=${1:-2}  # Default: 2 seconds

    echo "Monitoring temperature every ${interval}s (Ctrl+C to stop)"
    echo ""
    
    while true; do
        local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        local temperature=$(get_temperature)
        echo "[$timestamp] Temperature: $temperature"
        sleep "$interval"
    done
}

show_help() {
    echo "Usage: $0 [OPTION] [INTERVAL]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help"
    echo "  -m, --monitor  Monitor continuously"
    echo ""
    echo "Examples:"
    echo "  $0                   # Read temperature once"
    echo "  $0 -m                # Monitor with 2s interval"
    echo "  $0 -m 5              # Monitor with 5s interval"
}

case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    -m|--monitor)
        monitor_temperature "${2:-2}"
        ;;
    *)
        get_temperature
        ;;
esac
