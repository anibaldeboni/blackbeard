#!/bin/bash
# gpu-monitor.sh - Monitoramento GPU/VPU para RK3566

while true; do
  clear
  echo "╔══════════════════════════════════════════════════════════════╗"
  echo "║           Orange Pi 3B - GPU/VPU Monitor                     ║"
  echo "╚══════════════════════════════════════════════════════════════╝"
  echo ""
  
  # GPU Mali
  GPU_CUR=$(cat /sys/class/devfreq/fde60000.gpu/cur_freq 2>/dev/null)
  GPU_MAX=$(cat /sys/class/devfreq/fde60000.gpu/max_freq 2>/dev/null)
  GPU_PCT=$((GPU_CUR * 100 / GPU_MAX))
  printf "GPU Mali:  %3d MHz / %d MHz (%d%%)\n" $((GPU_CUR/1000000)) $((GPU_MAX/1000000)) $GPU_PCT
  
  # VPU/RGA clocks
  echo ""
  echo "VPU/RGA Clocks:"
  sudo cat /sys/kernel/debug/clk/clk_summary 2>/dev/null | grep -E "vpu|rga" | awk '{printf "  %-20s %d MHz\n", $1, $4/1000000}'
  
  # VPU Interrupts (indica atividade)
  echo ""
  echo "VPU Interrupts:"
  grep -E "hantro|fdea|fdee" /proc/interrupts 2>/dev/null | awk '{printf "  %-10s %s\n", $NF, $2}'
  
  sleep 2
done
