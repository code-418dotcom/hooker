package telegram

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/code-418dotcom/hooker/internal/config"
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
	ListGroups(ctx context.Context) ([]string, error)
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
			if update.CallbackQuery != nil {
				if update.CallbackQuery.From.ID != b.adminID {
					continue
				}
				go b.handleCallback(ctx, update.CallbackQuery)
				continue
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

	b.sendWithKeyboard(ctx, msg.Chat.ID, reply)
}

// handleCallback processes an inline keyboard button press.
func (b *Bot) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	// Acknowledge the callback to remove the loading indicator.
	callback := tgbotapi.NewCallback(cb.ID, "")
	if _, err := b.api.Request(callback); err != nil {
		log.Printf("callback ack error: %v", err)
	}

	cmd := Parse(cb.Data)
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
		reply = "Unknown command."
	}

	if err != nil {
		reply = "Error: " + err.Error()
	}

	b.sendWithKeyboard(ctx, cb.Message.Chat.ID, reply)
}

// buildKeyboard builds an inline keyboard with common commands and group shortcuts.
func (b *Bot) buildKeyboard(ctx context.Context) *tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📋 Status", "/list"),
			tgbotapi.NewInlineKeyboardButtonData("▶️ Start All", "/start all"),
			tgbotapi.NewInlineKeyboardButtonData("⏹ Stop All", "/stop all"),
		},
	}

	groups, err := b.ops.ListGroups(ctx)
	if err != nil {
		log.Printf("list groups error: %v", err)
	}
	for _, g := range groups {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("▶️ "+g, "/start group "+g),
			tgbotapi.NewInlineKeyboardButtonData("⏹ "+g, "/stop group "+g),
		})
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return &kb
}

// sendWithKeyboard sends a message with an inline keyboard of available commands.
func (b *Bot) sendWithKeyboard(ctx context.Context, chatID int64, text string) {
	m := tgbotapi.NewMessage(chatID, text)
	m.ParseMode = tgbotapi.ModeMarkdown
	m.ReplyMarkup = b.buildKeyboard(ctx)
	if _, err := b.api.Send(m); err != nil {
		log.Printf("send error: %v", err)
	}
}
