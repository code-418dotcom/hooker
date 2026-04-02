package docker

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

type mockAction struct {
	id     string
	action string
}

type mockDockerClient struct {
	containers  []types.Container
	err         error
	actions     []mockAction
	startErr    error
	stopErr     error
	restartErr  error
	inspectData map[string]types.ContainerJSON
	inspectErr  error
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

func (m *mockDockerClient) ContainerInspect(_ context.Context, id string) (types.ContainerJSON, error) {
	if m.inspectErr != nil {
		return types.ContainerJSON{}, m.inspectErr
	}
	if data, ok := m.inspectData[id]; ok {
		return data, nil
	}
	return types.ContainerJSON{Config: &container.Config{Labels: map[string]string{}}}, nil
}

// newTestOps creates an Ops with a mock client for testing.
func newTestOps(mock *mockDockerClient) *Ops {
	return &Ops{cli: mock}
}

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

func TestContainerNameEmpty(t *testing.T) {
	c := types.Container{ID: "abc123"}
	got := containerName(c)
	if got != "abc123" {
		t.Errorf("containerName with no names = %q, want %q", got, "abc123")
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

func TestList(t *testing.T) {
	mock := &mockDockerClient{
		containers: []types.Container{
			{Names: []string{"/nginx"}, State: "running"},
			{Names: []string{"/postgres"}, State: "exited"},
		},
	}
	ops := newTestOps(mock)
	result, err := ops.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "nginx") || !strings.Contains(result, "postgres") {
		t.Errorf("expected both containers in output, got: %s", result)
	}
}

func TestListEmpty(t *testing.T) {
	mock := &mockDockerClient{}
	ops := newTestOps(mock)
	result, err := ops.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "No containers found." {
		t.Errorf("expected empty message, got: %s", result)
	}
}

func TestListError(t *testing.T) {
	mock := &mockDockerClient{err: errors.New("connection refused")}
	ops := newTestOps(mock)
	_, err := ops.List(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStartSingle(t *testing.T) {
	mock := &mockDockerClient{}
	ops := newTestOps(mock)
	result, err := ops.Start(context.Background(), "nginx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "nginx") {
		t.Errorf("expected nginx in result, got: %s", result)
	}
	if len(mock.actions) != 1 || mock.actions[0].action != "start" {
		t.Errorf("expected 1 start action, got: %v", mock.actions)
	}
}

func TestStopProtectedSingle(t *testing.T) {
	mock := &mockDockerClient{
		inspectData: map[string]types.ContainerJSON{
			"protected-app": {Config: &container.Config{Labels: map[string]string{"hooker.protect": "true"}}},
		},
	}
	ops := newTestOps(mock)
	result, err := ops.Stop(context.Background(), "protected-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "protected") {
		t.Errorf("expected protection message, got: %s", result)
	}
	if len(mock.actions) != 0 {
		t.Errorf("expected no stop action on protected container, got: %v", mock.actions)
	}
}

func TestStopUnprotectedSingle(t *testing.T) {
	mock := &mockDockerClient{}
	ops := newTestOps(mock)
	result, err := ops.Stop(context.Background(), "nginx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Stopped") {
		t.Errorf("expected stopped message, got: %s", result)
	}
	if len(mock.actions) != 1 || mock.actions[0].action != "stop" {
		t.Errorf("expected 1 stop action, got: %v", mock.actions)
	}
}

func TestRestartProtectedSingle(t *testing.T) {
	mock := &mockDockerClient{
		inspectData: map[string]types.ContainerJSON{
			"critical-db": {Config: &container.Config{Labels: map[string]string{"hooker.protect": "true"}}},
		},
	}
	ops := newTestOps(mock)
	result, err := ops.Restart(context.Background(), "critical-db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "protected") {
		t.Errorf("expected protection message, got: %s", result)
	}
	if len(mock.actions) != 0 {
		t.Errorf("expected no restart action on protected container, got: %v", mock.actions)
	}
}

func TestStartAllSkipsRunningAndProtected(t *testing.T) {
	mock := &mockDockerClient{
		containers: []types.Container{
			{ID: "1", Names: []string{"/running"}, State: "running", Labels: map[string]string{}},
			{ID: "2", Names: []string{"/stopped"}, State: "exited", Labels: map[string]string{}},
			{ID: "3", Names: []string{"/protected-stopped"}, State: "exited", Labels: map[string]string{"hooker.protect": "true"}},
		},
	}
	ops := newTestOps(mock)
	result, err := ops.StartAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should only start container 2 (stopped, unprotected).
	if len(mock.actions) != 1 || mock.actions[0].id != "2" {
		t.Errorf("expected only container 2 started, got actions: %v", mock.actions)
	}
	if !strings.Contains(result, "protected (skipped)") {
		t.Errorf("expected protection skip message, got: %s", result)
	}
}

func TestStopAllSkipsProtected(t *testing.T) {
	mock := &mockDockerClient{
		containers: []types.Container{
			{ID: "1", Names: []string{"/app"}, State: "running", Labels: map[string]string{}},
			{ID: "2", Names: []string{"/critical"}, State: "running", Labels: map[string]string{"hooker.protect": "true"}},
		},
	}
	ops := newTestOps(mock)
	result, err := ops.StopAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.actions) != 1 || mock.actions[0].id != "1" {
		t.Errorf("expected only container 1 stopped, got: %v", mock.actions)
	}
	if !strings.Contains(result, "protected (skipped)") {
		t.Errorf("expected protection skip, got: %s", result)
	}
}

func TestBulkErrorSanitized(t *testing.T) {
	mock := &mockDockerClient{
		containers: []types.Container{
			{ID: "1", Names: []string{"/app"}, State: "running", Labels: map[string]string{}},
		},
		stopErr: errors.New("cannot stop: /var/run/docker.sock permission denied"),
	}
	ops := newTestOps(mock)
	result, err := ops.StopAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should say "failed", not include the raw error with socket path.
	if strings.Contains(result, "/var/run/docker.sock") {
		t.Errorf("raw error leaked to user: %s", result)
	}
	if !strings.Contains(result, "failed") {
		t.Errorf("expected 'failed' in result, got: %s", result)
	}
}

func TestListGroups(t *testing.T) {
	mock := &mockDockerClient{
		containers: []types.Container{
			{Labels: map[string]string{"hooker.group": "web"}},
			{Labels: map[string]string{"hooker.group": "web"}},
			{Labels: map[string]string{"hooker.group": "db"}},
			{Labels: map[string]string{}},
		},
	}
	ops := newTestOps(mock)
	groups, err := ops.ListGroups(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
}
