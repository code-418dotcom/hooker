package docker

import (
	"github.com/docker/docker/client"
)

// NewClient creates a new Docker client.
// It respects the DOCKER_HOST environment variable and defaults to the Unix socket.
func NewClient() (*client.Client, error) {
	return client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
}
