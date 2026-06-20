package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port              int
	WebhookForwardURL string
	WebhookSecret     string
	DBPath            string

	MinDelayMs       int
	MaxDelayMs       int
	TypingDurationMs int
	MsPerChar        int
	MaxTypingMs      int
	RateLimitPerMin  int
}

func Load() *Config {
	return &Config{
		Port:              envInt("GATEWAY_PORT", 3100),
		WebhookForwardURL: envStr("GATEWAY_WEBHOOK_FORWARD_URL", "http://api:8000/api/v1/whatsapp/webhook"),
		WebhookSecret:     envStr("GATEWAY_META_APP_SECRET", ""),
		DBPath:            envStr("GATEWAY_DB_PATH", "./data"),

		MinDelayMs:       envInt("GATEWAY_MIN_DELAY_MS", 1500),
		MaxDelayMs:       envInt("GATEWAY_MAX_DELAY_MS", 4000),
		TypingDurationMs: envInt("GATEWAY_TYPING_DURATION_MS", 2000),
		MsPerChar:        envInt("GATEWAY_MS_PER_CHAR", 50),
		MaxTypingMs:      envInt("GATEWAY_MAX_TYPING_MS", 8000),
		RateLimitPerMin:  envInt("GATEWAY_RATE_LIMIT_PER_MIN", 15),
	}
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
