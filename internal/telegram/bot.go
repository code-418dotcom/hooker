package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

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

// bulkCooldown is the minimum interval between bulk/destructive operations.
const bulkCooldown = 5 * time.Second

// Bot is the Telegram bot that polls for messages and dispatches commands.
type Bot struct {
	api     *tgbotapi.BotAPI
	adminID int64
	ops     DockerOps
	stopCh  chan struct{}
	wg      sync.WaitGroup

	mu       sync.Mutex
	lastBulk time.Time
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
					b.notifyUnauthorized(update.CallbackQuery.From, "callback")
					continue
				}
				b.wg.Add(1)
				go b.safeHandleCallback(ctx, update.CallbackQuery)
				continue
			}
			if update.Message == nil {
				continue
			}
			if update.Message.From.ID != b.adminID {
				b.notifyUnauthorized(update.Message.From, "message")
				continue
			}
			b.wg.Add(1)
			go b.safeHandle(ctx, update.Message)
		}
	}
}

// Stop signals the bot to stop polling, waits for in-flight handlers to finish
// (up to 10 seconds), then returns.
func (b *Bot) Stop() {
	b.api.StopReceivingUpdates()
	close(b.stopCh)

	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("all handlers finished")
	case <-time.After(10 * time.Second):
		slog.Warn("shutdown timed out waiting for handlers")
	}
}

// safeHandle wraps handle with panic recovery and WaitGroup tracking.
func (b *Bot) safeHandle(ctx context.Context, msg *tgbotapi.Message) {
	defer b.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in message handler", "panic", r, "user_id", msg.From.ID, "text", msg.Text)
		}
	}()
	b.handle(ctx, msg)
}

// safeHandleCallback wraps handleCallback with panic recovery and WaitGroup tracking.
func (b *Bot) safeHandleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	defer b.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in callback handler", "panic", r, "user_id", cb.From.ID, "data", cb.Data)
		}
	}()
	b.handleCallback(ctx, cb)
}

// notifyUnauthorized logs the attempt and sends a warning to the admin via Telegram.
func (b *Bot) notifyUnauthorized(user *tgbotapi.User, kind string) {
	slog.Warn("unauthorized access attempt",
		"user_id", user.ID,
		"username", user.UserName,
		"first_name", user.FirstName,
		"last_name", user.LastName,
		"kind", kind,
	)

	text := fmt.Sprintf("⚠️ Unauthorized %s from user `%d` (@%s, %s %s)",
		kind, user.ID, user.UserName, user.FirstName, user.LastName)
	msg := tgbotapi.NewMessage(b.adminID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := b.api.Send(msg); err != nil {
		slog.Error("failed to send unauthorized alert to admin", "error", err)
	}
}

// checkBulkCooldown returns an error message if a bulk operation was performed too recently.
func (b *Bot) checkBulkCooldown() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if since := time.Since(b.lastBulk); since < bulkCooldown {
		remaining := bulkCooldown - since
		return fmt.Sprintf("Rate limited — wait %.0f seconds before the next bulk operation.", remaining.Seconds())
	}
	b.lastBulk = time.Now()
	return ""
}

// isBulkCommand returns true if the command type is a bulk/destructive operation.
func isBulkCommand(t CommandType) bool {
	switch t {
	case CmdStartAll, CmdStopAll, CmdRestartAll, CmdStartGroup, CmdStopGroup:
		return true
	}
	return false
}

// handle processes a single message from the admin user.
func (b *Bot) handle(ctx context.Context, msg *tgbotapi.Message) {
	cmd := Parse(msg.Text)

	slog.Info("command received",
		"cmd", msg.Text,
		"user_id", msg.From.ID,
		"type", cmd.Type,
	)

	reply := b.dispatch(ctx, cmd)
	b.sendWithKeyboard(ctx, msg.Chat.ID, reply)
}

// handleCallback processes an inline keyboard button press.
func (b *Bot) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	// Acknowledge the callback to remove the loading indicator.
	callback := tgbotapi.NewCallback(cb.ID, "")
	if _, err := b.api.Request(callback); err != nil {
		slog.Error("callback ack failed", "error", err)
	}

	cmd := Parse(cb.Data)

	slog.Info("callback received",
		"data", cb.Data,
		"user_id", cb.From.ID,
		"type", cmd.Type,
	)

	reply := b.dispatch(ctx, cmd)
	b.sendWithKeyboard(ctx, cb.Message.Chat.ID, reply)
}

// dispatch executes a parsed command and returns the reply text.
func (b *Bot) dispatch(ctx context.Context, cmd Command) string {
	// Validate names before sending to Docker.
	if cmd.Name != "" && !isValidName(cmd.Name) {
		return "Invalid name. Use only letters, numbers, dashes, underscores, and dots (max 128 chars)."
	}

	// Rate-limit bulk operations.
	if isBulkCommand(cmd.Type) {
		if msg := b.checkBulkCooldown(); msg != "" {
			return msg
		}
	}

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
		slog.Error("command failed", "cmd", cmd.Type, "name", cmd.Name, "error", err)
		reply = "Something went wrong. Check the server logs for details."
	}

	return reply
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
		slog.Error("failed to list groups for keyboard", "error", err)
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
		slog.Error("failed to send message", "error", err, "chat_id", chatID)
	}
}
