# Blackbeard Media Stack 🏴‍☠️

Blackbeard is a comprehensive Docker-based home media server solution that provides automated media acquisition, management, and streaming capabilities. This collection of services creates a complete self-hosted entertainment ecosystem.

**Version 2.0** - Refatoração completa com health checks, gestão de recursos e melhorias de confiabilidade.

## 📦 Serviços Inclusos

| Serviço | Porta | Função |
|---------|-------|--------|
| **qBittorrent** | 5080 | Cliente de download torrent com interface VueTorrent |
| **Radarr** | 7878 | Gerenciador de filmes com busca automatizada |
| **Sonarr** | 8989 | Gerenciador de séries com tracking de episódios |
| **Prowlarr** | 9696 | Gerenciador de indexadores integrado |
| **Bazarr** | 6767 | Gerenciador de legendas automático |
| **Jellyfin** | 8096 | Servidor de mídia com hardware acceleration |
| **Jellyseerr** | 5055 | Interface de requisições de mídia |
| **Profilarr** | 6868 | Gerenciador de perfis de qualidade |
| **FlareSolverr** | 8191 | Proxy para bypass de Cloudflare |
| **Nginx** | 80/443 | Reverse proxy unificado |

## 🚀 Instalação Rápida

### Pré-requisitos
- Docker Engine 20.10+
- Docker Compose 2.0+
- 8GB+ RAM recomendado
- GPU para hardware acceleration (opcional)

### Passo a Passo

```bash
# 1. Criar a rede Docker
docker network create jollyroger

# 2. Copiar e configurar variáveis de ambiente
cp .env.example .env
nano .env  # Ajustar PUID, PGID, paths, etc.

# 3. Iniciar o stack (recomendado)
./manage-stack.sh start

# OU usando docker compose diretamente
docker compose up -d

# 4. Verificar status e health
./manage-stack.sh health
```

### Configurações Importantes no .env

Execute `id` no terminal para obter PUID e PGID:
```bash
PUID=1000              # Seu user ID
PGID=1000              # Seu group ID
TZ=America/Sao_Paulo   # Seu timezone
DOWNLOADS_PATH=/media/STORAGE/downloads  # Path de downloads
```

## 🎮 Gerenciamento do Stack

### Script de Gerenciamento (Recomendado)

```bash
./manage-stack.sh start          # Iniciar todos os serviços
./manage-stack.sh stop           # Parar todos os serviços
./manage-stack.sh restart        # Reiniciar todos os serviços
./manage-stack.sh status         # Ver status
./manage-stack.sh health         # Ver health checks
./manage-stack.sh logs [service] # Ver logs
./manage-stack.sh restart-svc <service>  # Reiniciar um serviço
./manage-stack.sh update         # Atualizar imagens
./manage-stack.sh backup         # Backup de configurações
./manage-stack.sh resources      # Ver uso de recursos
./manage-stack.sh help           # Ajuda completa
```

## 📖 Configuração Inicial dos Serviços

### 1. qBittorrent (5080)
- **Usuário:** `admin`
- **Senha:** Execute `docker logs qbittorrent` para ver
- **IMPORTANTE:** Altere a senha em Tools → Options → WebUI
- Configure downloads para `/downloads`

### 2. Prowlarr (9696) → Configure Primeiro!
- Adicione seus indexadores favoritos
- Em Settings → Apps, adicione Radarr e Sonarr
- Indexadores sincronizam automaticamente

### 3. Radarr (7878) & Sonarr (8989)
- Adicione qBittorrent como download client:
  - Host: `qbittorrent`, Port: `5080`
  - Category: `movies` (Radarr) ou `tv` (Sonarr)
- Root Folders:
  - Radarr: `/downloads/movies`
  - Sonarr: `/downloads/tv`

### 4. Bazarr (6767)
- Conecte ao Radarr e Sonarr
- Configure providers de legendas
- Defina idiomas (PT-BR, EN, etc.)

### 5. Jellyfin (8096)
- Complete o setup wizard
- Adicione bibliotecas:
  - Filmes: `/downloads/movies`
  - Séries: `/downloads/tv`
- Configure hardware acceleration (Settings → Playback)

### 6. Jellyseerr (5055)
- Conecte ao Jellyfin (URL: `http://jellyfin:8096`)
- Conecte ao Radarr e Sonarr
- Configure permissões de usuários

### 7. Profilarr (6868)
- Conecte ao Radarr e Sonarr
- Configure perfis de qualidade automatizados

## 🔧 Configuração Avançada

### Hardware Acceleration (Jellyfin)

Jellyfin está configurado para GPU via `/dev/dri`:

```bash
# Verificar dispositivos
ls -la /dev/dri

# Verificar grupos (ajuste no .env se necessário)
getent group video   # Normalmente 44
getent group render  # Normalmente 105
```

### Ajustar Limites de Recursos

Edite `.env` para ajustar CPU/memória:

```bash
# Reduzir Jellyfin
JELLYFIN_CPU_LIMIT=2.0
JELLYFIN_MEM_LIMIT=2g

# Aumentar qBittorrent
QBITTORRENT_CPU_LIMIT=4.0
QBITTORRENT_MEM_LIMIT=4g
```

### Nginx Reverse Proxy

Configure `config/nginx/nginx.conf` para proxy reverso unificado.

Exemplo de configuração:
```nginx
server {
    listen 80;
    server_name radarr.example.com;

    location / {
        proxy_pass http://radarr:7878;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 🔄 Atualizações Automáticas (Watchtower)

Todos os serviços possuem label `com.centurylinklabs.watchtower.enable=true`.

Para habilitar atualizações automáticas, adicione Watchtower:

```yaml
watchtower:
  image: containrrr/watchtower
  container_name: watchtower
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock
  environment:
    - WATCHTOWER_CLEANUP=true
    - WATCHTOWER_LABEL_ENABLE=true
    - WATCHTOWER_SCHEDULE=0 0 4 * * *  # 4 AM diariamente
    - TZ=${TZ:-America/Sao_Paulo}
  restart: unless-stopped
```

## 💾 Backup

### Automático
```bash
./manage-stack.sh backup
```
Cria backup em `backups/YYYYMMDD_HHMMSS/`

### Manual
```bash
docker compose down
tar -czf media-stack-backup-$(date +%Y%m%d).tar.gz config/ docker-compose.yml .env
docker compose up -d
```

**Volumes importantes:**
- `config/` - Todas as configurações
- `.env` - Variáveis de ambiente
- `docker-compose.yml` - Definição do stack

## 🐛 Troubleshooting

### Containers não iniciam na ordem
```bash
./manage-stack.sh health  # Verificar health checks
docker compose logs <service>  # Ver logs específicos
```

### Problemas de permissão
```bash
id  # Verificar PUID/PGID
sudo chown -R $PUID:$PGID ./config
```

### Jellyfin não detecta GPU
```bash
ls -la /dev/dri  # Verificar dispositivo
id  # Verificar se está nos grupos video e render
sudo usermod -aG video,render $USER  # Adicionar aos grupos
```

### Serviço não fica "healthy"
```bash
docker inspect <container> | grep -A 20 Health
# Aumentar start_period no docker-compose.yml se necessário
```

### qBittorrent não conecta
```bash
docker compose logs qbittorrent
docker exec qbittorrent curl -f http://localhost:5080
```

## 📊 Monitoramento

```bash
# Uso de recursos em tempo real
./manage-stack.sh resources

# Logs em tempo real
./manage-stack.sh logs          # Todos
./manage-stack.sh logs radarr   # Específico

# Health checks
./manage-stack.sh health
```

## 🔐 Segurança

### Checklist de Segurança
- ✅ Altere senha padrão do qBittorrent
- ✅ Configure VPN para torrents
- ✅ Use HTTPS no nginx (Let's Encrypt)
- ✅ Backup regular das configurações
- ✅ Mantenha containers atualizados
- ✅ Configure firewall apropriadamente

### Expor Apenas via Nginx
Para maior segurança, remova portas diretas e acesse tudo via nginx:

```yaml
radarr:
  ports: []  # Sem exposição direta
```

Acesse via: `http://your-server/radarr`

## 📚 Documentação

- [REFACTORING_CHANGES.md](REFACTORING_CHANGES.md) - Detalhes completos da refatoração v2.0
- [.env.example](.env.example) - Todas as variáveis disponíveis
- `./manage-stack.sh help` - Ajuda do script de gerenciamento

## 🆕 Novidades v2.0

### Melhorias Críticas
✅ Corrigido erro `dependent_on` → `depends_on`
✅ Health checks em todos os serviços
✅ Dependências com condição `service_healthy`
✅ Limites de CPU e memória configuráveis

### Novas Features
✅ Arquivo `.env` para configuração centralizada
✅ Script de gerenciamento `manage-stack.sh`
✅ Labels para backup e Watchtower
✅ Tmpfs para cache do Jellyfin
✅ Hostnames definidos
✅ Organização por categorias
✅ UMASK padronizado (002)

Veja [REFACTORING_CHANGES.md](REFACTORING_CHANGES.md) para detalhes completos.

## 🏴‍☠️ Filosofia

O nome "Blackbeard" reflete o espírito libertário do auto-hospedagem, proporcionando liberdade e controle total sobre sua biblioteca de entretenimento.

---

**Última atualização:** 2026-01-12
**Versão:** 2.0.0
**Status:** ✅ Pronto para produção
