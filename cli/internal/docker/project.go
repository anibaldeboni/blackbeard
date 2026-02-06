package docker

import (
	"context"

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
