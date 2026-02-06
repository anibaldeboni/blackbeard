package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/anibalnet/blackbeard/cli/internal/backup"
	"github.com/anibalnet/blackbeard/cli/internal/cleanup"
	"github.com/anibalnet/blackbeard/cli/internal/config"
	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/hw"
	"github.com/anibalnet/blackbeard/cli/internal/stack"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

// CLI is the root command structure for flint.
type CLI struct {
	ProjectDir string           `help:"Path to blackbeard project root." short:"p" env:"BLACKBEARD_DIR" type:"path"`
	NoColor    bool             `help:"Disable colored output." env:"NO_COLOR"`
	Yes        bool             `help:"Skip confirmation prompts." short:"y"`
	Version    kong.VersionFlag `help:"Show version."`

	Stack  StackCmd  `cmd:"" help:"Stack management (install, start, stop, logs, etc.)."`
	Backup BackupCmd `cmd:"" help:"Volume backup and restore operations."`
	Docker DockerCmd `cmd:"" help:"Docker cleanup operations."`
	Hw     HwCmd     `cmd:"" help:"Hardware monitoring (temperature, GPU/VPU)."`
}

// Ctx is the shared context passed to all command Run methods via Kong bindings.
type Ctx struct {
	context.Context
	Config  *config.Config
	Clients *dkr.Clients
	Printer *ui.Printer
	Yes     bool
}

// --- Stack commands ---

type StackCmd struct {
	Install   StackInstallCmd   `cmd:"" help:"Install/setup the stack (creates dirs, .env, network)."`
	Check     StackCheckCmd     `cmd:"" help:"Check installation status."`
	Uninstall StackUninstallCmd `cmd:"" help:"Remove containers and network (preserves config)."`
	Start     StackStartCmd     `cmd:"" help:"Start the entire stack."`
	Stop      StackStopCmd      `cmd:"" help:"Stop the entire stack."`
	Restart   StackRestartCmd   `cmd:"" help:"Restart the stack or a specific service."`
	Status    StackStatusCmd    `cmd:"" help:"Show container status."`
	Health    StackHealthCmd    `cmd:"" help:"Show health check status."`
	Logs      StackLogsCmd      `cmd:"" help:"Show logs (optionally for a specific service)."`
	Update    StackUpdateCmd    `cmd:"" help:"Pull new images and recreate containers."`
	Resources StackResourcesCmd `cmd:"" help:"Show resource usage (CPU, memory)."`
	Validate  StackValidateCmd  `cmd:"" help:"Validate docker-compose configuration."`
}

type StackInstallCmd struct{}

func (cmd *StackInstallCmd) Run(ctx *Ctx) error {
	return stack.RunInstall(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer, ctx.Yes)
}

type StackCheckCmd struct{}

func (cmd *StackCheckCmd) Run(ctx *Ctx) error {
	return stack.RunCheck(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer)
}

type StackUninstallCmd struct{}

func (cmd *StackUninstallCmd) Run(ctx *Ctx) error {
	return stack.RunUninstall(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer, ctx.Yes)
}

type StackStartCmd struct{}

func (cmd *StackStartCmd) Run(ctx *Ctx) error {
	return stack.RunStart(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer)
}

type StackStopCmd struct{}

func (cmd *StackStopCmd) Run(ctx *Ctx) error {
	return stack.RunStop(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer)
}

type StackRestartCmd struct {
	Service string `arg:"" optional:"" help:"Service to restart. If omitted, restarts entire stack."`
}

func (cmd *StackRestartCmd) Run(ctx *Ctx) error {
	if cmd.Service == "" {
		return stack.RunRestart(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer)
	}
	return stack.RunRestartService(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer, cmd.Service)
}

type StackStatusCmd struct{}

func (cmd *StackStatusCmd) Run(ctx *Ctx) error {
	return stack.RunStatus(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer)
}

type StackHealthCmd struct{}

func (cmd *StackHealthCmd) Run(ctx *Ctx) error {
	return stack.RunHealth(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer)
}

type StackLogsCmd struct {
	Service string `arg:"" optional:"" help:"Service to show logs for. If omitted, shows all."`
}

func (cmd *StackLogsCmd) Run(ctx *Ctx) error {
	return stack.RunLogs(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer, cmd.Service)
}

type StackUpdateCmd struct{}

func (cmd *StackUpdateCmd) Run(ctx *Ctx) error {
	return stack.RunUpdate(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer)
}

type StackResourcesCmd struct{}

func (cmd *StackResourcesCmd) Run(ctx *Ctx) error {
	return stack.RunResources(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer)
}

type StackValidateCmd struct{}

func (cmd *StackValidateCmd) Run(ctx *Ctx) error {
	return stack.RunValidate(ctx.Context, ctx.Config, ctx.Printer)
}

// --- Backup commands ---

type BackupCmd struct {
	Volumes BackupVolumesCmd `cmd:"" help:"List volumes marked for backup."`
	All     BackupAllCmd     `cmd:"" help:"Backup all marked volumes."`
	Volume  BackupVolumeCmd  `cmd:"" help:"Backup a specific volume."`
	Restore BackupRestoreCmd `cmd:"" help:"Restore volume from backup file."`
	List    BackupListCmd    `cmd:"" help:"List available backups."`
	Cleanup BackupCleanupCmd `cmd:"" help:"Remove backups older than N days."`
}

type BackupVolumesCmd struct{}

func (cmd *BackupVolumesCmd) Run(ctx *Ctx) error {
	return backup.RunListVolumes(ctx.Context, ctx.Clients, ctx.Printer)
}

type BackupAllCmd struct{}

func (cmd *BackupAllCmd) Run(ctx *Ctx) error {
	return backup.RunBackupAll(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer)
}

type BackupVolumeCmd struct {
	Name string `arg:"" help:"Volume name to backup."`
}

func (cmd *BackupVolumeCmd) Run(ctx *Ctx) error {
	return backup.RunBackupVolume(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer, cmd.Name)
}

type BackupRestoreCmd struct {
	File string `arg:"" help:"Path to backup tar.gz file."`
	Name string `arg:"" optional:"" help:"Volume name to restore to. Defaults to filename without extension."`
}

func (cmd *BackupRestoreCmd) Run(ctx *Ctx) error {
	return backup.RunRestore(ctx.Context, ctx.Config, ctx.Clients, ctx.Printer, cmd.File, cmd.Name, ctx.Yes)
}

type BackupListCmd struct{}

func (cmd *BackupListCmd) Run(ctx *Ctx) error {
	return backup.RunListBackups(ctx.Context, ctx.Config, ctx.Printer)
}

type BackupCleanupCmd struct {
	Days int `arg:"" optional:"" default:"7" help:"Remove backups older than this many days."`
}

func (cmd *BackupCleanupCmd) Run(ctx *Ctx) error {
	return backup.RunCleanup(ctx.Context, ctx.Config, ctx.Printer, cmd.Days)
}

// --- Docker cleanup commands ---

type DockerCmd struct {
	Disk      DockerDiskCmd      `cmd:"" help:"Show Docker disk usage."`
	List      DockerListCmd      `cmd:"" help:"List images (dangling, all, used)."`
	Dangling  DockerDanglingCmd  `cmd:"" help:"Remove dangling images only (safe)."`
	Prune     DockerPruneCmd     `cmd:"" help:"Remove all unused images."`
	PruneOld  DockerPruneOldCmd  `cmd:"prune-old" help:"Remove images older than N days."`
	Clean     DockerCleanCmd     `cmd:"" help:"Complete cleanup (containers, networks, images, cache)."`
	Protected DockerProtectedCmd `cmd:"" help:"Show protected images (in use by containers)."`
}

type DockerDiskCmd struct{}

func (cmd *DockerDiskCmd) Run(ctx *Ctx) error {
	return cleanup.RunDisk(ctx.Context, ctx.Clients, ctx.Printer)
}

type DockerListCmd struct{}

func (cmd *DockerListCmd) Run(ctx *Ctx) error {
	return cleanup.RunListImages(ctx.Context, ctx.Clients, ctx.Printer)
}

type DockerDanglingCmd struct{}

func (cmd *DockerDanglingCmd) Run(ctx *Ctx) error {
	return cleanup.RunDangling(ctx.Context, ctx.Clients, ctx.Printer)
}

type DockerPruneCmd struct{}

func (cmd *DockerPruneCmd) Run(ctx *Ctx) error {
	return cleanup.RunPruneImages(ctx.Context, ctx.Clients, ctx.Printer, ctx.Yes)
}

type DockerPruneOldCmd struct {
	Days int `arg:"" optional:"" default:"30" help:"Remove images older than this many days."`
}

func (cmd *DockerPruneOldCmd) Run(ctx *Ctx) error {
	return cleanup.RunPruneOld(ctx.Context, ctx.Clients, ctx.Printer, cmd.Days, ctx.Yes)
}

type DockerCleanCmd struct{}

func (cmd *DockerCleanCmd) Run(ctx *Ctx) error {
	return cleanup.RunCleanAll(ctx.Context, ctx.Clients, ctx.Printer, ctx.Yes)
}

type DockerProtectedCmd struct{}

func (cmd *DockerProtectedCmd) Run(ctx *Ctx) error {
	return cleanup.RunProtected(ctx.Context, ctx.Clients, ctx.Printer)
}

// --- Hardware monitoring commands ---

type HwCmd struct {
	Temp        HwTempCmd        `cmd:"" help:"Show temperature (CPU, GPU, or both)."`
	TempMonitor HwTempMonitorCmd `cmd:"temp-monitor" help:"Monitor temperature continuously."`
	Gpu         HwGpuCmd         `cmd:"" help:"Show GPU/VPU status."`
	GpuMonitor  HwGpuMonitorCmd  `cmd:"gpu-monitor" help:"Monitor GPU/VPU continuously."`
	Status      HwStatusCmd      `cmd:"" help:"Show full hardware status."`
}

type HwTempCmd struct {
	Target string `arg:"" optional:"" default:"all" enum:"all,cpu,gpu" help:"Sensor to read: cpu, gpu, or all."`
}

func (cmd *HwTempCmd) Run(ctx *Ctx) error {
	return hw.RunTemp(ctx.Printer, cmd.Target)
}

type HwTempMonitorCmd struct {
	Interval int `arg:"" optional:"" default:"2" help:"Update interval in seconds."`
}

func (cmd *HwTempMonitorCmd) Run(ctx *Ctx) error {
	return hw.RunTempMonitor(ctx.Context, ctx.Printer, cmd.Interval)
}

type HwGpuCmd struct{}

func (cmd *HwGpuCmd) Run(ctx *Ctx) error {
	return hw.RunGPUStatus(ctx.Printer)
}

type HwGpuMonitorCmd struct {
	Interval int `arg:"" optional:"" default:"2" help:"Update interval in seconds."`
}

func (cmd *HwGpuMonitorCmd) Run(ctx *Ctx) error {
	return hw.RunGPUMonitor(ctx.Context, ctx.Printer, cmd.Interval)
}

type HwStatusCmd struct{}

func (cmd *HwStatusCmd) Run(ctx *Ctx) error {
	return hw.RunFullStatus(ctx.Printer)
}

// --- main ---

func main() {
	cli := CLI{}
	kongCtx := kong.Parse(&cli,
		kong.Name("flint"),
		kong.Description("Blackbeard Media Stack CLI - manage your homelab Docker stack."),
		kong.Vars{"version": fmt.Sprintf("flint %s (commit: %s, built: %s)", version, commit, buildTime)},
		kong.UsageOnError(),
	)

	printer := ui.NewPrinter(cli.NoColor)

	// Resolve project directory
	projectDir, err := config.ResolveProjectDir(cli.ProjectDir)
	if err != nil {
		// hw commands don't need project dir
		if kongCtx.Command() != "hw temp" &&
			kongCtx.Command() != "hw temp-monitor" &&
			kongCtx.Command() != "hw gpu" &&
			kongCtx.Command() != "hw gpu-monitor" &&
			kongCtx.Command() != "hw status" {
			printer.Error(err.Error())
			os.Exit(1)
		}
		// Use cwd as fallback for hw commands
		projectDir, _ = os.Getwd()
	}

	// Load configuration
	cfg, err := config.Load(projectDir)
	if err != nil {
		printer.Error(fmt.Sprintf("loading config: %s", err))
		os.Exit(1)
	}

	// Initialize Docker clients (lazy - only when needed)
	var clients *dkr.Clients
	cmd := kongCtx.Command()

	needsDocker := true
	switch {
	case cmd == "hw temp", cmd == "hw temp-monitor",
		cmd == "hw gpu", cmd == "hw gpu-monitor",
		cmd == "hw status",
		cmd == "backup list", cmd == "backup cleanup",
		cmd == "stack validate":
		needsDocker = false
	}

	if needsDocker {
		clients, err = dkr.NewClients()
		if err != nil {
			printer.Error(fmt.Sprintf("connecting to Docker: %s", err))
			os.Exit(1)
		}
		defer clients.Close()
	}

	ctx := &Ctx{
		Context: context.Background(),
		Config:  cfg,
		Clients: clients,
		Printer: printer,
		Yes:     cli.Yes,
	}

	if err := kongCtx.Run(ctx); err != nil {
		printer.Error(err.Error())
		os.Exit(1)
	}
}
