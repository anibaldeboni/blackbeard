while true; do
  # Lê o tempo total de atividade e o tempo total de operação
  BUSY=$(cat /sys/class/devfreq/fde60000.gpu/trans_stat | grep "time" -A 1 | tail -n 1)
  # Como trans_stat é complexo, usamos a variação da frequência como proxy
  FREQ=$(cat /sys/class/devfreq/fde60000.gpu/cur_freq)
  MAX=$(cat /sys/class/devfreq/fde60000.gpu/max_freq)
  
  # Calcula a carga teórica baseada na frequência atual vs máxima
  # (No governo simple_ondemand, a freq é proporcional à carga)
  PERC=$(echo "scale=2; ($FREQ / $MAX) * 100" | bc)
  
  echo -ne "GPU Freq: $((FREQ / 1000000)) MHz | Carga Estimada: $PERC% \r"
  sleep 1
done
