package instance

import (
	"sync"
	"testing"
	"time"

	"github.com/rafaeldourado9/arcanum/internal/config"
	"github.com/rafaeldourado9/arcanum/internal/provider"
)

// behaviorProvider captura os callbacks registrados e anota o que o gerenciador pediu.
type behaviorProvider struct {
	provider.WhatsAppProvider

	mu        sync.Mutex
	onMessage provider.MessageHandler
	onCall    provider.CallHandler
	reads     []string
	rejected  []string
	online    bool
}

func (b *behaviorProvider) OnMessage(h provider.MessageHandler) { b.onMessage = h }
func (b *behaviorProvider) OnPresence(provider.PresenceHandler) {}
func (b *behaviorProvider) OnCall(h provider.CallHandler)       { b.onCall = h }
func (b *behaviorProvider) Status() provider.ConnectionStatus   { return provider.StatusConnected }
func (b *behaviorProvider) PhoneNumber() string                 { return "" }
func (b *behaviorProvider) Close() error                        { return nil }
func (b *behaviorProvider) SetAlwaysOnline(on bool)             { b.mu.Lock(); b.online = on; b.mu.Unlock() }
func (b *behaviorProvider) RejectCall(from, callID string) error {
	b.mu.Lock()
	b.rejected = append(b.rejected, callID)
	b.mu.Unlock()
	return nil
}
func (b *behaviorProvider) MarkAsRead(id, from string) error {
	b.mu.Lock()
	b.reads = append(b.reads, id)
	b.mu.Unlock()
	return nil
}
func (b *behaviorProvider) snapshot() (reads, rejected []string, online bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.reads...), append([]string(nil), b.rejected...), b.online
}

func behaviorManager(t *testing.T) (*Manager, *behaviorProvider) {
	t.Helper()
	readReceiptDelay = func() time.Duration { return 0 }
	t.Cleanup(func() {
		readReceiptDelay = func() time.Duration { return time.Second }
	})
	bp := &behaviorProvider{}
	cfg := &config.Config{DBPath: t.TempDir(), RateLimitPerMin: 15, WebhookFormat: "simple"}
	m := newManager(cfg, func(string) provider.WhatsAppProvider { return bp })
	if _, err := m.Create("s", "", nil); err != nil {
		t.Fatal(err)
	}
	return m, bp
}

func eventually(t *testing.T, cond func() bool) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

func TestAutoReadReceipts(t *testing.T) {
	m, bp := behaviorManager(t)

	// Padrao de instancia nova: confirmacao automatica ligada.
	bp.onMessage(provider.IncomingMessage{MessageID: "m1", From: "5511@s.whatsapp.net"})
	eventually(t, func() bool { r, _, _ := bp.snapshot(); return len(r) == 1 && r[0] == "m1" })

	// Mensagem do proprio numero (atendente no celular) nao e confirmada.
	bp.onMessage(provider.IncomingMessage{MessageID: "mine", From: "5511@s.whatsapp.net", FromMe: true})

	// Desligada: nada e confirmado.
	if _, err := m.UpdateSettings("s", func(s *Settings) error { s.ReadMessages = false; return nil }); err != nil {
		t.Fatal(err)
	}
	bp.onMessage(provider.IncomingMessage{MessageID: "m2", From: "5511@s.whatsapp.net"})
	time.Sleep(100 * time.Millisecond)
	if r, _, _ := bp.snapshot(); len(r) != 1 {
		t.Fatalf("reads = %v, want only m1", r)
	}
}

func TestRejectCallsFollowsSetting(t *testing.T) {
	m, bp := behaviorManager(t)

	bp.onCall("5511@s.whatsapp.net", "call-1")
	if _, rej, _ := bp.snapshot(); len(rej) != 0 {
		t.Fatalf("call rejected with the setting off: %v", rej)
	}

	if _, err := m.UpdateSettings("s", func(s *Settings) error { s.RejectCalls = true; return nil }); err != nil {
		t.Fatal(err)
	}
	bp.onCall("5511@s.whatsapp.net", "call-2")
	if _, rej, _ := bp.snapshot(); len(rej) != 1 || rej[0] != "call-2" {
		t.Fatalf("rejected = %v, want [call-2]", rej)
	}
}

func TestAlwaysOnlineFollowsSetting(t *testing.T) {
	m, bp := behaviorManager(t)

	if _, _, on := bp.snapshot(); on {
		t.Fatal("always online should start off")
	}
	if _, err := m.UpdateSettings("s", func(s *Settings) error { s.AlwaysOnline = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, _, on := bp.snapshot(); !on {
		t.Fatal("always online not applied to the provider")
	}
	if _, err := m.UpdateSettings("s", func(s *Settings) error { s.AlwaysOnline = false; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, _, on := bp.snapshot(); on {
		t.Fatal("always online not turned off")
	}
}
