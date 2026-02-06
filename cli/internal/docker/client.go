package docker

import (
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	dockerclient "github.com/docker/docker/client"
)

// Clients bundles both Docker clients needed by the application.
type Clients struct {
	Engine  dockerclient.APIClient
	Compose api.Service
	cli     *command.DockerCli
}

// NewClients creates Docker Engine client + Compose service.
func NewClients() (*Clients, error) {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return nil, err
	}

	if err := dockerCli.Initialize(flags.NewClientOptions()); err != nil {
		return nil, err
	}

	composeService := compose.NewComposeService(dockerCli)

	return &Clients{
		Engine:  dockerCli.Client(),
		Compose: composeService,
		cli:     dockerCli,
	}, nil
}

// Close releases underlying resources.
func (c *Clients) Close() error {
	if c.Engine != nil {
		return c.Engine.Close()
	}
	return nil
}
