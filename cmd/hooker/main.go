package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/code-418dotcom/hooker/internal/config"
	"github.com/code-418dotcom/hooker/internal/docker"
	"github.com/code-418dotcom/hooker/internal/telegram"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.MustLoad()

	dc, err := docker.NewClient()
	if err != nil {
		slog.Error("failed to create docker client", "error", err)
		os.Exit(1)
	}
	defer dc.Close()

	ops := docker.NewOps(dc)

	bot, err := telegram.NewBot(cfg, ops)
	if err != nil {
		slog.Error("failed to create bot", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go bot.Start(ctx)
	slog.Info("bot started, polling for messages")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	received := <-sig

	slog.Info("shutdown signal received", "signal", received.String())
	cancel()
	bot.Stop()
	slog.Info("bot stopped")
}
