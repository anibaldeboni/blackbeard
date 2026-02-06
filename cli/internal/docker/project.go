package docker

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
)

// LoadProject loads a compose project from the given compose file
// with the given .env file for variable interpolation.
func LoadProject(ctx context.Context, composeFile string, envFile string) (*types.Project, error) {
	options, err := cli.NewProjectOptions(
		[]string{composeFile},
		cli.WithOsEnv,
		cli.WithDotEnv,
	)
	if err != nil {
		return nil, err
	}

	project, err := options.LoadProject(ctx)
	if err != nil {
		return nil, err
	}

	return project, nil
}

// ConfigDirsFromProject extracts bind-mount device paths from the project's
// volume definitions and returns them relative to configBasePath.
// Only volumes whose resolved device path falls under configBasePath are included.
func ConfigDirsFromProject(project *types.Project, configBasePath string) []string {
	absBase, _ := filepath.Abs(configBasePath)

	var dirs []string
	seen := map[string]bool{}

	for _, vol := range project.Volumes {
		device, ok := vol.DriverOpts["device"]
		if !ok {
			continue
		}
		if vol.DriverOpts["type"] != "none" || vol.DriverOpts["o"] != "bind" {
			continue
		}

		absDevice, _ := filepath.Abs(device)

		if !strings.HasPrefix(absDevice, absBase+string(filepath.Separator)) {
			continue
		}

		rel, err := filepath.Rel(absBase, absDevice)
		if err != nil || rel == "." {
			continue
		}

		if !seen[rel] {
			seen[rel] = true
			dirs = append(dirs, rel)
		}
	}

	return dirs
}
