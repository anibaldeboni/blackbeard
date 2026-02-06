package stack

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
)

// RunInstall performs the 8-step installation process.
func RunInstall(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer, skipConfirm bool) error {
	p.Header("Blackbeard Media Stack - Installation")

	errors := 0
	p.Println("")
	p.Info(fmt.Sprintf("Installation directory: %s", cfg.ProjectDir))
	p.Println("")

	// Step 1: Check Docker
	p.Info("[1/8] Checking Docker...")
	if clients != nil {
		ping, err := clients.Engine.Ping(ctx)
		if err != nil {
			p.Error("Docker is not running or not accessible!")
			p.Info("Install Docker: https://docs.docker.com/engine/install/")
			errors++
		} else {
			p.Success(fmt.Sprintf("Docker connected (API v%s)", ping.APIVersion))
		}
	} else {
		p.Error("Docker is not available!")
		errors++
	}

	// Step 2: Check Docker Compose (implicit - if clients initialized, compose is available)
	p.Info("[2/8] Checking Docker Compose...")
	if clients != nil {
		p.Success("Docker Compose available (via SDK)")
	} else {
		p.Error("Docker Compose is not available!")
		errors++
	}

	// Step 3: Create Docker network
	p.Info("[3/8] Creating Docker network...")
	if err := ensureNetwork(ctx, cfg, clients, p); err != nil {
		p.Error(fmt.Sprintf("Failed to create network '%s'", cfg.NetworkName))
		errors++
	}

	// Step 4: Create .env file
	p.Info("[4/8] Setting up environment file...")
	if cfg.EnvFileExists() {
		p.Success(".env file already exists")
	} else if cfg.EnvExampleExists() {
		data, err := os.ReadFile(cfg.EnvExample)
		if err != nil {
			p.Error(fmt.Sprintf("reading .env.example: %s", err))
			errors++
		} else {
			if err := os.WriteFile(cfg.EnvFile, data, 0644); err != nil {
				p.Error(fmt.Sprintf("creating .env: %s", err))
				errors++
			} else {
				p.Success("Created .env from .env.example")
				p.Warning("Remember to edit .env with your settings!")
			}
		}
	} else {
		p.Error(".env.example not found!")
		errors++
	}

	// Step 5: Detect user/group IDs
	p.Info("[5/8] Detecting user configuration...")
	currentUser, _ := user.Current()
	currentUID := os.Getuid()
	currentGID := os.Getgid()
	p.Info(fmt.Sprintf("  Your UID: %d", currentUID))
	p.Info(fmt.Sprintf("  Your GID: %d", currentGID))
	_ = currentUser

	if cfg.EnvFileExists() {
		envData, err := os.ReadFile(cfg.EnvFile)
		if err == nil {
			content := string(envData)
			envPUID := extractEnvValue(content, "PUID")
			envPGID := extractEnvValue(content, "PGID")

			if envPUID != strconv.Itoa(currentUID) || envPGID != strconv.Itoa(currentGID) {
				if ui.ConfirmYesNo("PUID/PGID in .env differ from current user. Update?", skipConfirm) {
					content = replaceEnvValue(content, "PUID", strconv.Itoa(currentUID))
					content = replaceEnvValue(content, "PGID", strconv.Itoa(currentGID))
					if err := os.WriteFile(cfg.EnvFile, []byte(content), 0644); err != nil {
						p.Error(fmt.Sprintf("updating .env: %s", err))
					} else {
						p.Success(fmt.Sprintf("Updated PUID=%d, PGID=%d", currentUID, currentGID))
					}
				}
			} else {
				p.Success("PUID/PGID already configured correctly")
			}
		}
	}

	// Step 6: Create config directories (discovered from docker-compose.yml)
	p.Info("[6/8] Creating config directories...")
	_, _, dirErrors := EnsureConfigDirs(ctx, cfg, p)
	errors += dirErrors

	// Always ensure backups dir exists too
	backupsPath := filepath.Join(cfg.ProjectDir, "backups")
	if _, err := os.Stat(backupsPath); os.IsNotExist(err) {
		if err := os.MkdirAll(backupsPath, 0755); err != nil {
			p.Error(fmt.Sprintf("creating backups: %s", err))
			errors++
		} else {
			p.Success("Created backups")
		}
	} else {
		p.Info("  backups (exists)")
	}

	// Step 7: Check downloads directory
	p.Info("[7/8] Setting up downloads directory...")
	if _, err := os.Stat(cfg.DownloadsPath); err == nil {
		p.Success(fmt.Sprintf("Downloads directory exists: %s", cfg.DownloadsPath))
	} else {
		p.Warning(fmt.Sprintf("Downloads directory not found: %s", cfg.DownloadsPath))
		if ui.ConfirmYesNo("Create downloads directory?", skipConfirm) {
			if err := os.MkdirAll(cfg.DownloadsPath, 0755); err != nil {
				p.Warning("Could not create directory (may need sudo)")
				p.Info(fmt.Sprintf("Run: sudo mkdir -p %s && sudo chown %d:%d %s",
					cfg.DownloadsPath, currentUID, currentGID, cfg.DownloadsPath))
			} else {
				p.Success(fmt.Sprintf("Created %s", cfg.DownloadsPath))
			}
		}
	}

	// Step 8: Check GPU devices
	p.Info("[8/8] Checking GPU devices (optional)...")
	gpuAvailable := false
	if _, err := os.Stat("/dev/dri"); err == nil {
		p.Success("/dev/dri available (GPU rendering)")
		gpuAvailable = true
	}
	for _, dev := range []string{"/dev/video0", "/dev/video1", "/dev/video2"} {
		if _, err := os.Stat(dev); err == nil {
			p.Success(fmt.Sprintf("%s available (VPU)", dev))
		}
	}
	if !gpuAvailable {
		p.Warning("No GPU devices found (hardware transcoding disabled)")
	}

	// Detect GPU groups and update .env
	if gpuAvailable && cfg.EnvFileExists() {
		envData, _ := os.ReadFile(cfg.EnvFile)
		content := string(envData)
		updated := false

		videoGID := lookupGroupID("video")
		renderGID := lookupGroupID("render")

		if videoGID != "" {
			envVideo := extractEnvValue(content, "GPU_VIDEO_GROUP")
			if envVideo != videoGID {
				content = replaceEnvValue(content, "GPU_VIDEO_GROUP", videoGID)
				updated = true
				p.Info(fmt.Sprintf("  Updated GPU_VIDEO_GROUP=%s", videoGID))
			}
		}
		if renderGID != "" {
			envRender := extractEnvValue(content, "GPU_RENDER_GROUP")
			if envRender != renderGID {
				content = replaceEnvValue(content, "GPU_RENDER_GROUP", renderGID)
				updated = true
				p.Info(fmt.Sprintf("  Updated GPU_RENDER_GROUP=%s", renderGID))
			}
		}
		if updated {
			os.WriteFile(cfg.EnvFile, []byte(content), 0644)
		}
	}

	// Summary
	p.Println("")
	p.Header("Installation Summary")

	if errors == 0 {
		p.Success("Installation completed successfully!")
		p.Println("")
		p.Info("Next steps:")
		p.Println("  1. Review and edit .env file if needed")
		p.Println("  2. Run: flint stack start")
		p.Println("  3. Wait 2-3 minutes for services to start")
		p.Println("  4. Access via: http://localhost (nginx) or http://localhost:8096 (jellyfin)")
		p.Println("")
		p.Info("Service URLs (via nginx):")
		p.Println("  - Jellyfin:    http://localhost/jellyfin/")
		p.Println("  - Jellyseerr:  http://localhost/jellyseerr/")
		p.Println("  - Radarr:      http://localhost/radarr/")
		p.Println("  - Sonarr:      http://localhost/sonarr/")
		p.Println("  - Prowlarr:    http://localhost/prowlarr/")
		p.Println("  - Bazarr:      http://localhost/bazarr/")
		p.Println("  - qBittorrent: http://localhost/qbittorrent/")
	} else {
		p.Error(fmt.Sprintf("Installation completed with %d error(s)", errors))
		p.Warning("Please fix the errors above before starting the stack")
	}

	if errors > 0 {
		return fmt.Errorf("installation had %d error(s)", errors)
	}
	return nil
}

func extractEnvValue(content, key string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, key+"=") {
			return strings.TrimPrefix(line, key+"=")
		}
	}
	return ""
}

func replaceEnvValue(content, key, value string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			lines[i] = key + "=" + value
			return strings.Join(lines, "\n")
		}
	}
	return content
}

func lookupGroupID(name string) string {
	group, err := user.LookupGroup(name)
	if err != nil {
		return ""
	}
	return group.Gid
}
