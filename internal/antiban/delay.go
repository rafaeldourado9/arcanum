package antiban

import (
	"math/rand"
	"time"
	"unicode/utf8"

	"github.com/rafaeldourado9/arcanum/internal/config"
	"github.com/rafaeldourado9/arcanum/internal/provider"
)

func HumanizedSend(p provider.WhatsAppProvider, to string, text string, cfg *config.Config) {
	preDelay := randBetween(cfg.MinDelayMs, cfg.MaxDelayMs)
	time.Sleep(time.Duration(preDelay) * time.Millisecond)

	_ = p.MarkAsRead("", to)

	_ = p.SendPresence(to, "composing")

	charCount := utf8.RuneCountInString(text)
	charBased := charCount * cfg.MsPerChar
	typingMs := clamp(charBased, cfg.TypingDurationMs, cfg.MaxTypingMs)
	typingMs += randBetween(0, 1000)
	time.Sleep(time.Duration(typingMs) * time.Millisecond)

	_ = p.SendPresence(to, "paused")
	time.Sleep(time.Duration(randBetween(200, 600)) * time.Millisecond)
}

func randBetween(min, max int) int {
	if max <= min {
		return min
	}
	return min + rand.Intn(max-min+1)
}

func clamp(val, lo, hi int) int {
	if val < lo {
		return lo
	}
	if val > hi {
		return hi
	}
	return val
}
