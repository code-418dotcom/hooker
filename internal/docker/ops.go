package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

const (
	// singleTimeout is the timeout for operations on a single container.
	singleTimeout = 30 * time.Second
	// bulkTimeout is the timeout for operations that touch multiple containers.
	bulkTimeout = 2 * time.Minute

	labelProtect = "hooker.protect"
	labelGroup   = "hooker.group"

	stateRunning = "running"
)

// Ops provides Docker container operations.
type Ops struct {
	cli *client.Client
}

// NewOps creates a new Ops instance.
func NewOps(c *client.Client) *Ops {
	return &Ops{cli: c}
}

// containerName extracts the clean name from a Docker container.
func containerName(c types.Container) string {
	if len(c.Names) == 0 {
		return c.ID
	}
	name := c.Names[0]
	if strings.HasPrefix(name, "/") {
		name = name[1:]
	}
	return name
}

// isProtected checks if a container has the hooker.protect=true label.
func isProtected(c types.Container) bool {
	return c.Labels[labelProtect] == "true"
}

// List returns a formatted string of all containers and their state.
func (o *Ops) List(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, bulkTimeout)
	defer cancel()

	containers, err := o.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}

	if len(containers) == 0 {
		return "No containers found.", nil
	}

	var lines []string
	for _, c := range containers {
		line := fmt.Sprintf("`%s` (%s)", containerName(c), c.State)
		lines = append(lines, line)
	}

	return "Containers:\n" + strings.Join(lines, "\n"), nil
}

// Start starts a container by name.
func (o *Ops) Start(ctx context.Context, name string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, singleTimeout)
	defer cancel()

	if err := o.cli.ContainerStart(ctx, name, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}
	return fmt.Sprintf("Started `%s`", name), nil
}

// Stop stops a container by name.
func (o *Ops) Stop(ctx context.Context, name string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, singleTimeout)
	defer cancel()

	if err := o.cli.ContainerStop(ctx, name, container.StopOptions{}); err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}
	return fmt.Sprintf("Stopped `%s`", name), nil
}

// Restart restarts a container by name.
func (o *Ops) Restart(ctx context.Context, name string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, singleTimeout)
	defer cancel()

	if err := o.cli.ContainerRestart(ctx, name, container.StopOptions{}); err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}
	return fmt.Sprintf("Restarted `%s`", name), nil
}

// forEachContainer lists containers with the given options and applies fn to each one
// that passes the filter. Returns a formatted result summary.
func (o *Ops) forEachContainer(
	ctx context.Context,
	opts container.ListOptions,
	filter func(types.Container) (skip bool, reason string),
	action func(ctx context.Context, c types.Container) error,
	verb string,
) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, bulkTimeout)
	defer cancel()

	containers, err := o.cli.ContainerList(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}

	var results []string
	for _, c := range containers {
		name := containerName(c)
		if skip, reason := filter(c); skip {
			if reason != "" {
				results = append(results, fmt.Sprintf("`%s`: %s", name, reason))
			}
			continue
		}
		if err := action(ctx, c); err != nil {
			results = append(results, fmt.Sprintf("`%s`: error (%v)", name, err))
		} else {
			results = append(results, fmt.Sprintf("`%s`: %s", name, verb))
		}
	}

	return strings.Join(results, "\n"), nil
}

// StartAll starts all stopped containers.
func (o *Ops) StartAll(ctx context.Context) (string, error) {
	result, err := o.forEachContainer(ctx,
		container.ListOptions{All: true},
		func(c types.Container) (bool, string) {
			if c.State == stateRunning {
				return true, ""
			}
			return false, ""
		},
		func(ctx context.Context, c types.Container) error {
			return o.cli.ContainerStart(ctx, c.ID, container.StartOptions{})
		},
		"started",
	)
	if err != nil {
		return "", err
	}
	if result == "" {
		return "All containers are already running.", nil
	}
	return "Started containers:\n" + result, nil
}

// StopAll stops all running containers except those with hooker.protect=true label.
func (o *Ops) StopAll(ctx context.Context) (string, error) {
	result, err := o.forEachContainer(ctx,
		container.ListOptions{All: true},
		func(c types.Container) (bool, string) {
			if c.State != stateRunning {
				return true, ""
			}
			if isProtected(c) {
				return true, "protected (skipped)"
			}
			return false, ""
		},
		func(ctx context.Context, c types.Container) error {
			return o.cli.ContainerStop(ctx, c.ID, container.StopOptions{})
		},
		"stopped",
	)
	if err != nil {
		return "", err
	}
	if result == "" {
		return "All containers are already stopped.", nil
	}
	return "Stopped containers:\n" + result, nil
}

// RestartAll restarts all containers except those with hooker.protect=true label.
func (o *Ops) RestartAll(ctx context.Context) (string, error) {
	result, err := o.forEachContainer(ctx,
		container.ListOptions{All: true},
		func(c types.Container) (bool, string) {
			if isProtected(c) {
				return true, "protected (skipped)"
			}
			return false, ""
		},
		func(ctx context.Context, c types.Container) error {
			return o.cli.ContainerRestart(ctx, c.ID, container.StopOptions{})
		},
		"restarted",
	)
	if err != nil {
		return "", err
	}
	if result == "" {
		return "No containers to restart.", nil
	}
	return "Restarted containers:\n" + result, nil
}

// ListGroups returns all distinct hooker.group label values across containers.
func (o *Ops) ListGroups(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, singleTimeout)
	defer cancel()

	containers, err := o.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("docker: %w", err)
	}

	seen := make(map[string]bool)
	var groups []string
	for _, c := range containers {
		if g, ok := c.Labels[labelGroup]; ok && g != "" && !seen[g] {
			seen[g] = true
			groups = append(groups, g)
		}
	}
	return groups, nil
}

// StartGroup starts all containers with the given group label.
func (o *Ops) StartGroup(ctx context.Context, tag string) (string, error) {
	f := filters.NewArgs(filters.Arg("label", labelGroup+"="+tag))
	result, err := o.forEachContainer(ctx,
		container.ListOptions{All: true, Filters: f},
		func(c types.Container) (bool, string) {
			if c.State == stateRunning {
				return true, "already running"
			}
			return false, ""
		},
		func(ctx context.Context, c types.Container) error {
			return o.cli.ContainerStart(ctx, c.ID, container.StartOptions{})
		},
		"started",
	)
	if err != nil {
		return "", err
	}
	if result == "" {
		return fmt.Sprintf("No containers found with group label `%s`", tag), nil
	}
	return fmt.Sprintf("Group `%s`:\n%s", tag, result), nil
}

// StopGroup stops all containers with the given group label.
func (o *Ops) StopGroup(ctx context.Context, tag string) (string, error) {
	f := filters.NewArgs(filters.Arg("label", labelGroup+"="+tag))
	result, err := o.forEachContainer(ctx,
		container.ListOptions{All: true, Filters: f},
		func(c types.Container) (bool, string) {
			if c.State != stateRunning {
				return true, "already stopped"
			}
			return false, ""
		},
		func(ctx context.Context, c types.Container) error {
			return o.cli.ContainerStop(ctx, c.ID, container.StopOptions{})
		},
		"stopped",
	)
	if err != nil {
		return "", err
	}
	if result == "" {
		return fmt.Sprintf("No containers found with group label `%s`", tag), nil
	}
	return fmt.Sprintf("Group `%s`:\n%s", tag, result), nil
}
