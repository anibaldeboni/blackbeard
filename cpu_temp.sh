#!/bin/bash
# filepath: cpu_temp.sh

# Temperature sensor paths for OrangePi
readonly CPU_TEMP_PATH="/sys/devices/virtual/thermal/thermal_zone1/temp"
readonly GPU_TEMP_PATH="/sys/devices/virtual/thermal/thermal_zone0/temp"

# ANSI color codes
readonly COLOR_RESET='\033[0m'
readonly COLOR_GREEN='\033[0;32m'
readonly COLOR_YELLOW='\033[0;33m'
readonly COLOR_ORANGE='\033[0;91m'
readonly COLOR_RED='\033[0;31m'
readonly COLOR_BOLD_RED='\033[1;31m'

get_temperature_color() {
    local temp_numeric="$1"
      
    # RK3566 temperature thresholds based on typical ARM CPU operating ranges
    # Using awk for float comparison (more portable than bc)
    if awk "BEGIN {exit !($temp_numeric < 45)}"; then
        echo "${COLOR_GREEN}"          # Cool: < 45°C (Green)
    elif awk "BEGIN {exit !($temp_numeric < 60)}"; then
        echo "${COLOR_YELLOW}"         # Normal: 45-59°C (Yellow)
    elif awk "BEGIN {exit !($temp_numeric < 75)}"; then
        echo "${COLOR_ORANGE}"         # Warm: 60-74°C (Orange)
    elif awk "BEGIN {exit !($temp_numeric < 85)}"; then
        echo "${COLOR_RED}"            # Hot: 75-84°C (Red)
    else
        echo "${COLOR_BOLD_RED}"       # Critical: ≥85°C (Bold Red)
    fi
}

get_temperature() {
    local temp_file="$1"
    local label="$2"
    
    if [[ ! -f "$temp_file" ]]; then
        echo "Error: ${label} temperature file not found: $temp_file"
        exit 1
    fi
    
    local temp_millidegrees=$(cat "$temp_file")
    
    if [[ ! "$temp_millidegrees" =~ ^[0-9]+$ ]]; then
        echo "Error: Invalid ${label} temperature reading: $temp_millidegrees"
        exit 1
    fi
    
    local temp_celsius=$((temp_millidegrees / 1000))
    local temp_decimal=$((temp_millidegrees % 1000 / 100))
    local temp_value="${temp_celsius}.${temp_decimal}"
    local color=$(get_temperature_color "$temp_value")
    echo -e "${color}${temp_value}${COLOR_RESET}°C"
}

get_cpu_temperature() {
    get_temperature "$CPU_TEMP_PATH" "CPU"
}

get_gpu_temperature() {
    get_temperature "$GPU_TEMP_PATH" "GPU"
}

show_both_temperatures() {
    local cpu_temp=$(get_cpu_temperature)
    local gpu_temp=$(get_gpu_temperature)
    echo -e "CPU: $cpu_temp | GPU: $gpu_temp"
}

monitor_temperature() {
    local interval=${1:-2}  # Default: 2 seconds

    echo "Monitoring temperature every ${interval}s (Ctrl+C to stop)"
    echo ""
    
    while true; do
        local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        local temps=$(show_both_temperatures)
        echo -e "[$timestamp] $temps"
        sleep "$interval"
    done
}

show_help() {
    echo "Usage: $0 [OPTION] [INTERVAL]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help"
    echo "  -c, --cpu      Show CPU temperature only"
    echo "  -g, --gpu      Show GPU temperature only"
    echo "  -m, --monitor  Monitor both temperatures continuously"
    echo ""
    echo "Color coding for RK3566 temperatures:"
    echo -e "  ${COLOR_GREEN}Green${COLOR_RESET}   - Cool: < 45°C"
    echo -e "  ${COLOR_YELLOW}Yellow${COLOR_RESET}  - Normal: 45-59°C"
    echo -e "  ${COLOR_ORANGE}Orange${COLOR_RESET}  - Warm: 60-74°C"
    echo -e "  ${COLOR_RED}Red${COLOR_RESET}     - Hot: 75-84°C"
    echo -e "  ${COLOR_BOLD_RED}Bold Red${COLOR_RESET} - Critical: ≥85°C"
    echo ""
    echo "Examples:"
    echo "  $0                   # Show both CPU and GPU temperatures"
    echo "  $0 -c                # Show CPU temperature only"
    echo "  $0 -g                # Show GPU temperature only"
    echo "  $0 -m                # Monitor both with 2s interval"
    echo "  $0 -m 5              # Monitor both with 5s interval"
}

case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    -c|--cpu)
        echo -e "CPU: $(get_cpu_temperature)"
        ;;
    -g|--gpu)
        echo -e "GPU: $(get_gpu_temperature)"
        ;;
    -m|--monitor)
        monitor_temperature "${2:-2}"
        ;;
    *)
        show_both_temperatures
        ;;
esac
