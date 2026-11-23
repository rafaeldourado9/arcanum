package logbuf

import (
	"io"
	"log/slog"
	"testing"
)

func TestBufferKeepsLastLinesAndRedactsSecrets(t *testing.T) {
	buf := New(3)
	log := slog.New(NewHandler(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}), buf))

	ch, cancel := buf.Subscribe()
	defer cancel()

	log.Debug("one")
	log.Info("two", "instance", "sales")
	log.Warn("three", "api_key", "abc123", "webhook_secret", "s3cr3t")
	log.Error("four")

	got := buf.Recent(0, slog.LevelDebug)
	if len(got) != 3 || got[0].Msg != "two" || got[2].Msg != "four" {
		t.Fatalf("ring buffer = %+v, want two/three/four", got)
	}
	if got[1].Attrs["api_key"] != "[redacted]" || got[1].Attrs["webhook_secret"] != "[redacted]" {
		t.Fatalf("secrets not redacted: %+v", got[1].Attrs)
	}
	if got[0].Attrs["instance"] != "sales" {
		t.Fatalf("normal attr lost: %+v", got[0].Attrs)
	}
	if warn := buf.Recent(0, slog.LevelWarn); len(warn) != 2 {
		t.Fatalf("level filter: got %d, want 2", len(warn))
	}
	if limited := buf.Recent(1, slog.LevelDebug); len(limited) != 1 || limited[0].Msg != "four" {
		t.Fatalf("limit: %+v", limited)
	}
	if first := <-ch; first.Msg != "one" {
		t.Fatalf("subscriber first = %q", first.Msg)
	}
}
