package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/code-418dotcom/hooker/internal/config"
)

// FakeOps is a mock implementation of DockerOps for testing.
type FakeOps struct {
	listCalls    int
	startCalls   map[string]int
	stopCalls    map[string]int
	restartCalls map[string]int
	err          error // if set, all calls return this error
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
	if f.err != nil {
		return "", f.err
	}
	return "container1 running\ncontainer2 stopped", nil
}

func (f *FakeOps) Start(ctx context.Context, name string) (string, error) {
	f.startCalls[name]++
	if f.err != nil {
		return "", f.err
	}
	return "Started " + name, nil
}

func (f *FakeOps) Stop(ctx context.Context, name string) (string, error) {
	f.stopCalls[name]++
	if f.err != nil {
		return "", f.err
	}
	return "Stopped " + name, nil
}

func (f *FakeOps) Restart(ctx context.Context, name string) (string, error) {
	f.restartCalls[name]++
	if f.err != nil {
		return "", f.err
	}
	return "Restarted " + name, nil
}

func (f *FakeOps) StartAll(ctx context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "Started all", nil
}

func (f *FakeOps) StopAll(ctx context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "Stopped all", nil
}

func (f *FakeOps) RestartAll(ctx context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "Restarted all", nil
}

func (f *FakeOps) StartGroup(ctx context.Context, tag string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "Started group " + tag, nil
}

func (f *FakeOps) StopGroup(ctx context.Context, tag string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "Stopped group " + tag, nil
}

func (f *FakeOps) ListGroups(ctx context.Context) ([]string, error) {
	return []string{"web", "db"}, nil
}

// TestNewBot verifies that NewBot returns an error with an invalid token.
func TestNewBot(t *testing.T) {
	cfg := &config.Config{
		BotToken: "test-token",
		AdminID:  12345,
	}
	ops := NewFakeOps()

	_, err := NewBot(cfg, ops)
	if err == nil {
		t.Fatal("expected error with invalid test token")
	}
}

// TestGroupCaching verifies that the group cache avoids repeated Docker API calls.
func TestGroupCaching(t *testing.T) {
	ops := NewFakeOps()
	bot := &Bot{
		adminID: 12345,
		ops:     ops,
		stopCh:  make(chan struct{}),
	}
	ctx := context.Background()

	// First call should populate cache.
	groups := bot.cachedGroups(ctx)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// Second call within TTL should return cached result (ops.ListGroups not called again).
	ops2 := NewFakeOps() // fresh ops that would return different data
	bot.ops = ops2
	groups2 := bot.cachedGroups(ctx)
	if len(groups2) != 2 {
		t.Fatalf("expected cached 2 groups, got %d", len(groups2))
	}

	// After expiring cache, should refresh.
	bot.mu.Lock()
	bot.groupCachedAt = time.Now().Add(-time.Minute)
	bot.mu.Unlock()
	groups3 := bot.cachedGroups(ctx)
	if len(groups3) != 2 {
		t.Fatalf("expected refreshed 2 groups, got %d", len(groups3))
	}
}

// TestDispatchMethod tests the bot's dispatch method directly.
func TestDispatchMethod(t *testing.T) {
	ops := NewFakeOps()
	bot := &Bot{
		adminID: 12345,
		ops:     ops,
		stopCh:  make(chan struct{}),
	}
	ctx := context.Background()

	t.Run("list command", func(t *testing.T) {
		reply := bot.dispatch(ctx, Command{Type: CmdList})
		if reply == "" {
			t.Error("expected non-empty reply for list")
		}
	})

	t.Run("start single", func(t *testing.T) {
		reply := bot.dispatch(ctx, Command{Type: CmdStart, Name: "nginx"})
		if reply != "Started nginx" {
			t.Errorf("unexpected reply: %q", reply)
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		reply := bot.dispatch(ctx, Command{Type: CmdUnknown})
		if reply == "" {
			t.Error("expected help text for unknown command")
		}
	})

	t.Run("invalid name rejected", func(t *testing.T) {
		reply := bot.dispatch(ctx, Command{Type: CmdStart, Name: "../bad"})
		if reply != "Invalid name. Use only letters, numbers, dashes, underscores, and dots (max 128 chars)." {
			t.Errorf("expected validation error, got: %q", reply)
		}
	})

	t.Run("error returns sanitized message", func(t *testing.T) {
		errOps := NewFakeOps()
		errOps.err = errors.New("docker: connection refused to /var/run/docker.sock")
		errBot := &Bot{
			adminID: 12345,
			ops:     errOps,
			stopCh:  make(chan struct{}),
		}
		reply := errBot.dispatch(ctx, Command{Type: CmdList})
		if reply != "Something went wrong. Check the server logs for details." {
			t.Errorf("expected sanitized error, got: %q", reply)
		}
	})
}

// TestIsBulkCommand verifies bulk command detection.
func TestIsBulkCommand(t *testing.T) {
	bulk := []CommandType{CmdStartAll, CmdStopAll, CmdRestartAll, CmdStartGroup, CmdStopGroup}
	for _, ct := range bulk {
		if !isBulkCommand(ct) {
			t.Errorf("expected %d to be bulk command", ct)
		}
	}

	notBulk := []CommandType{CmdList, CmdStart, CmdStop, CmdRestart, CmdUnknown}
	for _, ct := range notBulk {
		if isBulkCommand(ct) {
			t.Errorf("expected %d to not be bulk command", ct)
		}
	}
}

// TestRateLimiting verifies the bulk operation cooldown.
func TestRateLimiting(t *testing.T) {
	bot := &Bot{
		adminID: 12345,
		ops:     NewFakeOps(),
		stopCh:  make(chan struct{}),
	}

	// First call should succeed.
	if msg := bot.checkBulkCooldown(); msg != "" {
		t.Errorf("first bulk call should not be rate limited, got: %q", msg)
	}

	// Immediate second call should be rate limited.
	if msg := bot.checkBulkCooldown(); msg == "" {
		t.Error("second immediate bulk call should be rate limited")
	}

	// After waiting, should succeed again.
	bot.mu.Lock()
	bot.lastBulk = time.Now().Add(-10 * time.Second)
	bot.mu.Unlock()

	if msg := bot.checkBulkCooldown(); msg != "" {
		t.Errorf("call after cooldown should not be rate limited, got: %q", msg)
	}
}
