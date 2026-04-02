package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// Ops provides Docker container operations.
type Ops struct {
	cli *client.Client
}

// NewOps creates a new Ops instance.
func NewOps(c *client.Client) *Ops {
	return &Ops{cli: c}
}

// isProtected checks if a container has the hooker.protect=true label.
// Protected containers are skipped by bulk operations like StopAll and RestartAll.
func isProtected(c types.Container) bool {
	return c.Labels["hooker.protect"] == "true"
}

// List returns a formatted string of all containers and their state.
func (o *Ops) List(ctx context.Context) (string, error) {
	containers, err := o.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}

	if len(containers) == 0 {
		return "No containers found.", nil
	}

	var lines []string
	for _, c := range containers {
		name := c.Names[0]
		if strings.HasPrefix(name, "/") {
			name = name[1:]
		}
		state := c.State
		line := fmt.Sprintf("`%s` (%s)", name, state)
		lines = append(lines, line)
	}

	return "Containers:\n" + strings.Join(lines, "\n"), nil
}

// Start starts a container by name.
func (o *Ops) Start(ctx context.Context, name string) (string, error) {
	if err := o.cli.ContainerStart(ctx, name, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}
	return fmt.Sprintf("Started `%s`", name), nil
}

// Stop stops a container by name.
func (o *Ops) Stop(ctx context.Context, name string) (string, error) {
	if err := o.cli.ContainerStop(ctx, name, container.StopOptions{}); err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}
	return fmt.Sprintf("Stopped `%s`", name), nil
}

// Restart restarts a container by name.
func (o *Ops) Restart(ctx context.Context, name string) (string, error) {
	if err := o.cli.ContainerRestart(ctx, name, container.StopOptions{}); err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}
	return fmt.Sprintf("Restarted `%s`", name), nil
}

// StartAll starts all stopped containers.
func (o *Ops) StartAll(ctx context.Context) (string, error) {
	containers, err := o.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}

	var results []string
	for _, c := range containers {
		if c.State == "running" {
			continue
		}
		name := c.Names[0]
		if strings.HasPrefix(name, "/") {
			name = name[1:]
		}
		if err := o.cli.ContainerStart(ctx, c.ID, container.StartOptions{}); err != nil {
			results = append(results, fmt.Sprintf("`%s`: error (%v)", name, err))
		} else {
			results = append(results, fmt.Sprintf("`%s`: started", name))
		}
	}

	if len(results) == 0 {
		return "All containers are already running.", nil
	}

	return "Started containers:\n" + strings.Join(results, "\n"), nil
}

// StopAll stops all running containers except those with hooker.protect=true label.
func (o *Ops) StopAll(ctx context.Context) (string, error) {
	containers, err := o.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}

	var results []string
	for _, c := range containers {
		if c.State != "running" {
			continue
		}
		if isProtected(c) {
			name := c.Names[0]
			if strings.HasPrefix(name, "/") {
				name = name[1:]
			}
			results = append(results, fmt.Sprintf("`%s`: protected (skipped)", name))
			continue
		}
		name := c.Names[0]
		if strings.HasPrefix(name, "/") {
			name = name[1:]
		}
		if err := o.cli.ContainerStop(ctx, c.ID, container.StopOptions{}); err != nil {
			results = append(results, fmt.Sprintf("`%s`: error (%v)", name, err))
		} else {
			results = append(results, fmt.Sprintf("`%s`: stopped", name))
		}
	}

	if len(results) == 0 {
		return "All containers are already stopped.", nil
	}

	return "Stopped containers:\n" + strings.Join(results, "\n"), nil
}

// RestartAll restarts all containers except those with hooker.protect=true label.
func (o *Ops) RestartAll(ctx context.Context) (string, error) {
	containers, err := o.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}

	var results []string
	for _, c := range containers {
		if isProtected(c) {
			name := c.Names[0]
			if strings.HasPrefix(name, "/") {
				name = name[1:]
			}
			results = append(results, fmt.Sprintf("`%s`: protected (skipped)", name))
			continue
		}
		name := c.Names[0]
		if strings.HasPrefix(name, "/") {
			name = name[1:]
		}
		if err := o.cli.ContainerRestart(ctx, c.ID, container.StopOptions{}); err != nil {
			results = append(results, fmt.Sprintf("`%s`: error (%v)", name, err))
		} else {
			results = append(results, fmt.Sprintf("`%s`: restarted", name))
		}
	}

	if len(results) == 0 {
		return "No containers to restart.", nil
	}

	return "Restarted containers:\n" + strings.Join(results, "\n"), nil
}

// ListGroups returns all distinct hooker.group label values across containers.
func (o *Ops) ListGroups(ctx context.Context) ([]string, error) {
	containers, err := o.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("docker: %w", err)
	}

	seen := make(map[string]bool)
	var groups []string
	for _, c := range containers {
		if g, ok := c.Labels["hooker.group"]; ok && g != "" && !seen[g] {
			seen[g] = true
			groups = append(groups, g)
		}
	}
	return groups, nil
}

// StartGroup starts all containers with the given group label.
func (o *Ops) StartGroup(ctx context.Context, tag string) (string, error) {
	filter := filters.NewArgs(filters.Arg("label", "hooker.group="+tag))
	containers, err := o.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: filter})
	if err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}

	if len(containers) == 0 {
		return fmt.Sprintf("No containers found with group label `%s`", tag), nil
	}

	var results []string
	for _, c := range containers {
		if c.State == "running" {
			name := c.Names[0]
			if strings.HasPrefix(name, "/") {
				name = name[1:]
			}
			results = append(results, fmt.Sprintf("`%s`: already running", name))
			continue
		}
		name := c.Names[0]
		if strings.HasPrefix(name, "/") {
			name = name[1:]
		}
		if err := o.cli.ContainerStart(ctx, c.ID, container.StartOptions{}); err != nil {
			results = append(results, fmt.Sprintf("`%s`: error (%v)", name, err))
		} else {
			results = append(results, fmt.Sprintf("`%s`: started", name))
		}
	}

	return fmt.Sprintf("Group `%s`:\n%s", tag, strings.Join(results, "\n")), nil
}

// StopGroup stops all containers with the given group label.
func (o *Ops) StopGroup(ctx context.Context, tag string) (string, error) {
	filter := filters.NewArgs(filters.Arg("label", "hooker.group="+tag))
	containers, err := o.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: filter})
	if err != nil {
		return "", fmt.Errorf("docker: %w", err)
	}

	if len(containers) == 0 {
		return fmt.Sprintf("No containers found with group label `%s`", tag), nil
	}

	var results []string
	for _, c := range containers {
		if c.State != "running" {
			name := c.Names[0]
			if strings.HasPrefix(name, "/") {
				name = name[1:]
			}
			results = append(results, fmt.Sprintf("`%s`: already stopped", name))
			continue
		}
		name := c.Names[0]
		if strings.HasPrefix(name, "/") {
			name = name[1:]
		}
		if err := o.cli.ContainerStop(ctx, c.ID, container.StopOptions{}); err != nil {
			results = append(results, fmt.Sprintf("`%s`: error (%v)", name, err))
		} else {
			results = append(results, fmt.Sprintf("`%s`: stopped", name))
		}
	}

	return fmt.Sprintf("Group `%s`:\n%s", tag, strings.Join(results, "\n")), nil
}
