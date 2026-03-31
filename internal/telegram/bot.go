package telegram

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sjanus/hooker/internal/config"
)

// DockerOps defines the interface for Docker operations.
// This allows testing without a real Docker daemon.
type DockerOps interface {
	List(ctx context.Context) (string, error)
	Start(ctx context.Context, name string) (string, error)
	Stop(ctx context.Context, name string) (string, error)
	Restart(ctx context.Context, name string) (string, error)
	StartAll(ctx context.Context) (string, error)
	StopAll(ctx context.Context) (string, error)
	RestartAll(ctx context.Context) (string, error)
	StartGroup(ctx context.Context, tag string) (string, error)
	StopGroup(ctx context.Context, tag string) (string, error)
}

// Bot is the Telegram bot that polls for messages and dispatches commands.
type Bot struct {
	api     *tgbotapi.BotAPI
	adminID int64
	ops     DockerOps
	stopCh  chan struct{}
}

// NewBot creates a new Bot instance.
func NewBot(cfg *config.Config, ops DockerOps) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("telegram: %w", err)
	}
	return &Bot{
		api:     api,
		adminID: cfg.AdminID,
		ops:     ops,
		stopCh:  make(chan struct{}),
	}, nil
}

// Start begins the long-polling loop, blocking until Stop is called or context is cancelled.
func (b *Bot) Start(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.stopCh:
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			if update.Message == nil {
				continue
			}
			// Silent drop for non-admin users.
			if update.Message.From.ID != b.adminID {
				continue
			}
			go b.handle(ctx, update.Message)
		}
	}
}

// Stop signals the bot to stop polling and exit gracefully.
func (b *Bot) Stop() {
	b.api.StopReceivingUpdates()
	close(b.stopCh)
}

// handle processes a single message from the admin user.
func (b *Bot) handle(ctx context.Context, msg *tgbotapi.Message) {
	cmd := Parse(msg.Text)
	var reply string
	var err error

	switch cmd.Type {
	case CmdList:
		reply, err = b.ops.List(ctx)
	case CmdStart:
		reply, err = b.ops.Start(ctx, cmd.Name)
	case CmdStop:
		reply, err = b.ops.Stop(ctx, cmd.Name)
	case CmdRestart:
		reply, err = b.ops.Restart(ctx, cmd.Name)
	case CmdStartAll:
		reply, err = b.ops.StartAll(ctx)
	case CmdStopAll:
		reply, err = b.ops.StopAll(ctx)
	case CmdRestartAll:
		reply, err = b.ops.RestartAll(ctx)
	case CmdStartGroup:
		reply, err = b.ops.StartGroup(ctx, cmd.Name)
	case CmdStopGroup:
		reply, err = b.ops.StopGroup(ctx, cmd.Name)
	default:
		reply = `Unknown command. Supported commands:
/list or /status — List all containers
/start <name> — Start a container
/stop <name> — Stop a container
/restart <name> — Restart a container
/start all — Start all containers
/stop all — Stop all containers
/restart all — Restart all containers
/start group <tag> — Start containers with label hooker.group=<tag>
/stop group <tag> — Stop containers with label hooker.group=<tag>`
	}

	if err != nil {
		reply = "Error: " + err.Error()
	}

	b.send(msg.Chat.ID, reply)
}

// send sends a message back to the Telegram chat.
func (b *Bot) send(chatID int64, text string) {
	m := tgbotapi.NewMessage(chatID, text)
	m.ParseMode = tgbotapi.ModeMarkdown
	if _, err := b.api.Send(m); err != nil {
		log.Printf("send error: %v", err)
	}
}
