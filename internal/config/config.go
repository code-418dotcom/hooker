package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds runtime configuration for the bot.
type Config struct {
	BotToken string
	AdminID  int64
}

// MustLoad loads configuration from environment variables.
// Panics if required variables are missing or invalid.
func MustLoad() *Config {
	token := mustGetenv("TELEGRAM_BOT_TOKEN")
	adminStr := mustGetenv("TELEGRAM_ADMIN_ID")

	adminID, err := strconv.ParseInt(adminStr, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("invalid TELEGRAM_ADMIN_ID: %q must be an integer: %v", adminStr, err))
	}

	return &Config{
		BotToken: token,
		AdminID:  adminID,
	}
}

// mustGetenv returns the value of the environment variable key.
// Panics if the variable is not set.
func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}
