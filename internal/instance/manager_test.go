package instance

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rafaeldourado9/arcanum/internal/config"
	"github.com/rafaeldourado9/arcanum/internal/provider"
)

// fakeProvider implementa so o que o gerenciador usa; o resto vem da interface embutida
// (nil) e quebraria o teste se fosse chamado sem querer.
type fakeProvider struct {
	provider.WhatsAppProvider
	unlinked, closed bool
}

func (f *fakeProvider) OnMessage(provider.MessageHandler)   {}
func (f *fakeProvider) OnPresence(provider.PresenceHandler) {}
func (f *fakeProvider) Status() provider.ConnectionStatus   { return provider.StatusDisconnected }
func (f *fakeProvider) PhoneNumber() string                 { return "" }
func (f *fakeProvider) Unlink() error                       { f.unlinked = true; return nil }
func (f *fakeProvider) Close() error                        { f.closed = true; return nil }
func (f *fakeProvider) Connect() error                      { return nil }

func testManager(t *testing.T, dir string, made map[string]*fakeProvider) *Manager {
	t.Helper()
	cfg := &config.Config{DBPath: dir, RateLimitPerMin: 15, WebhookFormat: "simple"}
	m := newManager(cfg, func(dbPath string) provider.WhatsAppProvider {
		fp := &fakeProvider{}
		made[filepath.Base(dbPath)] = fp
		return fp
	})
	return m
}

func TestConfigSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	made := map[string]*fakeProvider{}
	m := testManager(t, dir, made)

	if _, err := m.Create("sales", "", nil); err != nil {
		t.Fatal(err)
	}
	url := "https://example.com/hook"
	enabled := true
	if _, err := m.UpdateWebhook("sales", WebhookUpdate{URL: &url, Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.UpdateSettings("sales", func(s *Settings) error { s.GroupsIgnore = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(filepath.Join(dir, "sales.json")); err != nil || st.Mode().Perm()&0o077 != 0 && st.Mode().Perm() != 0o666 {
		// no Windows a permissao aparece como 0666; no Linux tem que ser 0600
		if err != nil {
			t.Fatalf("sales.json not written: %v", err)
		}
	}

	// "reinicio": gerenciador novo lendo a mesma pasta
	m2 := testManager(t, dir, map[string]*fakeProvider{})
	m2.restorePersisted()
	info, err := m2.Info("sales")
	if err != nil {
		t.Fatalf("instance not restored: %v", err)
	}
	if info.Webhook.URL != url || !info.Webhook.Enabled || !info.Settings.GroupsIgnore {
		t.Fatalf("config lost on restart: %+v", info)
	}
}

func TestDeleteRemovesFilesAndUnlinks(t *testing.T) {
	dir := t.TempDir()
	made := map[string]*fakeProvider{}
	m := testManager(t, dir, made)
	if _, err := m.Create("sales", "", nil); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"sales.db", "sales.db-wal"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if err := m.Delete("sales"); err != nil {
		t.Fatal(err)
	}
	fp := made["sales.db"]
	if !fp.unlinked || !fp.closed {
		t.Fatalf("unlinked=%v closed=%v, want both", fp.unlinked, fp.closed)
	}
	left, _ := os.ReadDir(dir)
	if len(left) != 0 {
		t.Fatalf("files left after delete: %v", left)
	}
	if _, err := m.Get("sales"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after delete = %v, want ErrNotFound", err)
	}

	// reinicio nao traz de volta
	m2 := testManager(t, dir, map[string]*fakeProvider{})
	m2.restorePersisted()
	if len(m2.List()) != 0 {
		t.Fatal("deleted instance came back after restart")
	}
}

func TestWebhookHistoryKeptWhenURLChanges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	defer srv.Close()

	m := testManager(t, t.TempDir(), map[string]*fakeProvider{})
	if _, err := m.Create("sales", srv.URL, nil); err != nil {
		t.Fatal(err)
	}
	if d, _ := m.TestWebhook("sales"); !d.OK || d.Status != 204 {
		t.Fatalf("test delivery = %+v", d)
	}
	other := srv.URL + "/other"
	if _, err := m.UpdateWebhook("sales", WebhookUpdate{URL: &other}); err != nil {
		t.Fatal(err)
	}
	list, _ := m.Deliveries("sales")
	if len(list) != 1 || list[0].Event != "test" {
		t.Fatalf("history lost after URL change: %+v", list)
	}

	bad := "http://169.254.169.254/x"
	if _, err := m.UpdateWebhook("sales", WebhookUpdate{URL: &bad}); err == nil {
		t.Fatal("metadata URL accepted")
	}

	res := m.ImportWebhooks(map[string]WebhookConfig{
		"sales":   {URL: srv.URL, Enabled: true},
		"missing": {URL: srv.URL},
	})
	if len(res.Updated) != 1 || res.Skipped["missing"] == "" {
		t.Fatalf("import result = %+v", res)
	}
}
