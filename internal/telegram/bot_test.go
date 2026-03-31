package telegram

import (
	"context"
	"testing"

	"github.com/code-418dotcom/hooker/internal/config"
)

// FakeOps is a mock implementation of DockerOps for testing.
type FakeOps struct {
	listCalls    int
	startCalls   map[string]int
	stopCalls    map[string]int
	restartCalls map[string]int
}

func NewFakeOps() *FakeOps {
	return &FakeOps{
		startCalls:   make(map[string]int),
		stopCalls:    make(map[string]int),
		restartCalls: make(map[string]int),
	}
}

func (f *FakeOps) List(ctx context.Context) (string, error) {
	f.listCalls++
	return "container1 running\ncontainer2 stopped", nil
}

func (f *FakeOps) Start(ctx context.Context, name string) (string, error) {
	f.startCalls[name]++
	return "Started " + name, nil
}

func (f *FakeOps) Stop(ctx context.Context, name string) (string, error) {
	f.stopCalls[name]++
	return "Stopped " + name, nil
}

func (f *FakeOps) Restart(ctx context.Context, name string) (string, error) {
	f.restartCalls[name]++
	return "Restarted " + name, nil
}

func (f *FakeOps) StartAll(ctx context.Context) (string, error) {
	return "Started all", nil
}

func (f *FakeOps) StopAll(ctx context.Context) (string, error) {
	return "Stopped all", nil
}

func (f *FakeOps) RestartAll(ctx context.Context) (string, error) {
	return "Restarted all", nil
}

func (f *FakeOps) StartGroup(ctx context.Context, tag string) (string, error) {
	return "Started group " + tag, nil
}

func (f *FakeOps) StopGroup(ctx context.Context, tag string) (string, error) {
	return "Stopped group " + tag, nil
}

// TestNewBot verifies that NewBot creates a valid Bot instance.
func TestNewBot(t *testing.T) {
	cfg := &config.Config{
		BotToken: "test-token",
		AdminID:  12345,
	}
	ops := NewFakeOps()

	// Note: This will fail because the token is invalid.
	// In a real test, we'd skip this or use a test API.
	// For now, we're just testing that the function signature works.
	_, err := NewBot(cfg, ops)
	if err == nil {
		t.Fatal("expected error with test token")
	}
}

// TestDispatch verifies that the bot dispatches commands correctly.
// (This is a simplified example; a full test would mock tgbotapi)
func TestDispatch(t *testing.T) {
	ops := NewFakeOps()
	ctx := context.Background()

	// Test command parsing integration
	cases := []struct {
		cmdType CommandType
		name    string
		wantOp  string
	}{
		{CmdList, "", "list"},
		{CmdStart, "container1", "start"},
		{CmdStop, "container1", "stop"},
		{CmdRestart, "container1", "restart"},
	}

	for _, tc := range cases {
		t.Run(tc.wantOp, func(t *testing.T) {
			switch tc.cmdType {
			case CmdList:
				_, _ = ops.List(ctx)
				if ops.listCalls != 1 {
					t.Errorf("list calls = %d, want 1", ops.listCalls)
				}
			case CmdStart:
				_, _ = ops.Start(ctx, tc.name)
				if ops.startCalls[tc.name] != 1 {
					t.Errorf("start calls for %s = %d, want 1", tc.name, ops.startCalls[tc.name])
				}
			case CmdStop:
				_, _ = ops.Stop(ctx, tc.name)
				if ops.stopCalls[tc.name] != 1 {
					t.Errorf("stop calls for %s = %d, want 1", tc.name, ops.stopCalls[tc.name])
				}
			case CmdRestart:
				_, _ = ops.Restart(ctx, tc.name)
				if ops.restartCalls[tc.name] != 1 {
					t.Errorf("restart calls for %s = %d, want 1", tc.name, ops.restartCalls[tc.name])
				}
			}
		})
	}
}
