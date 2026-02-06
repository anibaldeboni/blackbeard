package stack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/docker/api/types/network"
	"github.com/fatih/color"
)

// RunCheck verifies the installation status.
func RunCheck(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer) error {
	p.Header("Installation Status Check")
	p.Println("")

	allOK := true
	ok := color.New(color.FgGreen).SprintFunc()
	warn := color.New(color.FgYellow).SprintFunc()
	fail := color.New(color.FgRed).SprintFunc()

	// Check Docker
	fmt.Printf("Docker:           ")
	if clients != nil {
		if _, err := clients.Engine.Ping(ctx); err == nil {
			fmt.Println(ok("OK"))
		} else {
			fmt.Println(fail("NOT RUNNING"))
			allOK = false
		}
	} else {
		fmt.Println(fail("NOT INSTALLED"))
		allOK = false
	}

	// Check Docker Compose
	fmt.Printf("Docker Compose:   ")
	if clients != nil {
		fmt.Println(ok("OK"))
	} else {
		fmt.Println(fail("NOT INSTALLED"))
		allOK = false
	}

	// Check network
	fmt.Printf("Network:          ")
	if clients != nil {
		if _, err := clients.Engine.NetworkInspect(ctx, cfg.NetworkName, network.InspectOptions{}); err == nil {
			fmt.Printf("%s (%s)\n", ok("OK"), cfg.NetworkName)
		} else {
			fmt.Println(warn("NOT CREATED"))
			allOK = false
		}
	} else {
		fmt.Println(warn("UNKNOWN"))
		allOK = false
	}

	// Check .env
	fmt.Printf(".env file:        ")
	if cfg.EnvFileExists() {
		fmt.Println(ok("OK"))
	} else {
		fmt.Println(warn("MISSING"))
		allOK = false
	}

	// Check config dirs (discovered from docker-compose.yml)
	fmt.Printf("Config dirs:      ")
	project, loadErr := dkr.LoadProject(ctx, cfg.ComposeFile, cfg.EnvFile)
	if loadErr != nil {
		fmt.Println(warn("CANNOT READ COMPOSE"))
		allOK = false
	} else {
		configDirs := dkr.ConfigDirsFromProject(project, cfg.ConfigBasePath)
		missing := 0
		for _, dir := range configDirs {
			fullPath := filepath.Join(cfg.ConfigBasePath, dir)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				missing++
			}
		}
		if missing == 0 {
			fmt.Printf("%s (%d dirs)\n", ok("OK"), len(configDirs))
		} else {
			fmt.Printf("%s\n", warn(fmt.Sprintf("%d MISSING", missing)))
			allOK = false
		}
	}

	// Check downloads path
	fmt.Printf("Downloads path:   ")
	if _, err := os.Stat(cfg.DownloadsPath); err == nil {
		fmt.Printf("%s (%s)\n", ok("OK"), cfg.DownloadsPath)
	} else {
		fmt.Printf("%s (%s)\n", warn("NOT FOUND"), cfg.DownloadsPath)
		allOK = false
	}

	// Check GPU
	fmt.Printf("GPU devices:      ")
	if _, err := os.Stat("/dev/dri"); err == nil {
		fmt.Println(ok("AVAILABLE"))
	} else {
		fmt.Printf("%s (optional)\n", warn("NOT FOUND"))
	}

	p.Println("")
	if allOK {
		p.Success("All checks passed! Ready to start.")
	} else {
		p.Warning("Some items need attention. Run: flint stack install")
	}

	return nil
}
