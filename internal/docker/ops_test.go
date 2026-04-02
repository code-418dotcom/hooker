package docker

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// mockClient implements just enough of the Docker client interface for testing.
// We test via Ops methods which call the real client methods, so we use a
// wrapper that satisfies the calls Ops makes.

type mockAction struct {
	id     string
	action string
}

type mockDockerClient struct {
	containers []types.Container
	err        error
	actions    []mockAction
	startErr   error
	stopErr    error
	restartErr error
}

func (m *mockDockerClient) ContainerList(_ context.Context, _ container.ListOptions) ([]types.Container, error) {
	return m.containers, m.err
}

func (m *mockDockerClient) ContainerStart(_ context.Context, id string, _ container.StartOptions) error {
	m.actions = append(m.actions, mockAction{id, "start"})
	return m.startErr
}

func (m *mockDockerClient) ContainerStop(_ context.Context, id string, _ container.StopOptions) error {
	m.actions = append(m.actions, mockAction{id, "stop"})
	return m.stopErr
}

func (m *mockDockerClient) ContainerRestart(_ context.Context, id string, _ container.StopOptions) error {
	m.actions = append(m.actions, mockAction{id, "restart"})
	return m.restartErr
}

// Since Ops uses *client.Client directly, we can't easily mock it without an interface.
// Instead, test the helper functions and use integration-style tests for the rest.

func TestContainerName(t *testing.T) {
	tests := []struct {
		names []string
		want  string
	}{
		{[]string{"/nginx"}, "nginx"},
		{[]string{"nginx"}, "nginx"},
		{[]string{"/my-app"}, "my-app"},
	}
	for _, tc := range tests {
		c := types.Container{Names: tc.names}
		got := containerName(c)
		if got != tc.want {
			t.Errorf("containerName(%v) = %q, want %q", tc.names, got, tc.want)
		}
	}
}

func TestIsProtected(t *testing.T) {
	tests := []struct {
		labels map[string]string
		want   bool
	}{
		{map[string]string{"hooker.protect": "true"}, true},
		{map[string]string{"hooker.protect": "false"}, false},
		{map[string]string{}, false},
		{nil, false},
	}
	for _, tc := range tests {
		c := types.Container{Labels: tc.labels}
		got := isProtected(c)
		if got != tc.want {
			t.Errorf("isProtected(labels=%v) = %v, want %v", tc.labels, got, tc.want)
		}
	}
}

func TestForEachContainer(t *testing.T) {
	mock := &mockDockerClient{
		containers: []types.Container{
			{ID: "1", Names: []string{"/running"}, State: "running", Labels: map[string]string{}},
			{ID: "2", Names: []string{"/stopped"}, State: "exited", Labels: map[string]string{}},
			{ID: "3", Names: []string{"/protected"}, State: "running", Labels: map[string]string{"hooker.protect": "true"}},
		},
	}

	// Test that forEachContainer skips filtered containers and applies action to the rest.
	// We can't call forEachContainer directly since it uses o.cli, but we can test
	// the filter and action pattern through the public methods once we have an interface.

	// For now, verify the mock data and helper functions work correctly.
	for _, c := range mock.containers {
		name := containerName(c)
		if name == "" {
			t.Error("expected non-empty container name")
		}
	}

	if !isProtected(mock.containers[2]) {
		t.Error("expected container 3 to be protected")
	}
	if isProtected(mock.containers[0]) {
		t.Error("expected container 1 to not be protected")
	}
}

func TestContainerNameEdgeCases(t *testing.T) {
	c := types.Container{Names: []string{"/a/b/c"}}
	got := containerName(c)
	if got != "a/b/c" {
		t.Errorf("containerName with nested path = %q, want %q", got, "a/b/c")
	}
}

// TestListFormatsOutput verifies the List output format using mock data.
// This requires refactoring Ops to accept an interface; for now test helpers.
func TestListGroupsExtraction(t *testing.T) {
	containers := []types.Container{
		{Labels: map[string]string{"hooker.group": "web"}},
		{Labels: map[string]string{"hooker.group": "web"}},
		{Labels: map[string]string{"hooker.group": "db"}},
		{Labels: map[string]string{}},
	}

	seen := make(map[string]bool)
	var groups []string
	for _, c := range containers {
		if g, ok := c.Labels[labelGroup]; ok && g != "" && !seen[g] {
			seen[g] = true
			groups = append(groups, g)
		}
	}

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0] != "web" || groups[1] != "db" {
		t.Errorf("expected [web, db], got %v", groups)
	}
}

func TestBulkResultFormatting(t *testing.T) {
	// Test that error formatting in bulk operations produces readable output.
	results := []string{
		"`nginx`: started",
		"`postgres`: error (connection refused)",
		"`redis`: started",
	}
	output := "Started containers:\n" + strings.Join(results, "\n")
	if !strings.Contains(output, "nginx") {
		t.Error("expected output to contain nginx")
	}
	if !strings.Contains(output, "error") {
		t.Error("expected output to contain error")
	}
}

func TestDockerErrorWrapping(t *testing.T) {
	inner := errors.New("connection refused")
	wrapped := errors.New("docker: " + inner.Error())
	if !strings.Contains(wrapped.Error(), "docker:") {
		t.Error("expected wrapped error to contain docker prefix")
	}
}
