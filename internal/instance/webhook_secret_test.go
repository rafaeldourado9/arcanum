package instance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// A assinatura de cada sessao usa o segredo dela (e nao o global), o segredo sobrevive ao
// reinicio e nunca sai nas respostas da API (so "hasSecret").
func TestPerInstanceWebhookSecret(t *testing.T) {
	dir := t.TempDir()
	made := map[string]*fakeProvider{}
	m := testManager(t, dir, made)
	m.cfg.WebhookSecret = "global-secret-global-secret"

	if _, err := m.Create("loja", "", nil); err != nil {
		t.Fatal(err)
	}
	secret, err := m.RotateWebhookSecret("loja")
	if err != nil || !strings.HasPrefix(secret, "whsec_") {
		t.Fatalf("rotate: %q %v", secret, err)
	}
	inst, _ := m.Get("loja")
	if !inst.WebhookConfig().HasSecret {
		t.Fatal("HasSecret should be true")
	}

	// Entrega de teste: confere com qual segredo a assinatura foi feita.
	var gotSig string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-Hub-Signature-256")
		gotBody, _ = io.ReadAll(r.Body)
	}))
	defer srv.Close()
	url := srv.URL
	if _, err := m.UpdateWebhook("loja", WebhookUpdate{URL: &url}); err != nil {
		t.Fatal(err)
	}
	inst.Forwarder().Test("loja")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(gotBody)
	if want := "sha256=" + hex.EncodeToString(mac.Sum(nil)); gotSig != want {
		t.Fatalf("signature %q does not match the instance secret (%q)", gotSig, want)
	}

	// Sobrevive ao reinicio, em arquivo privado.
	raw, err := os.ReadFile(m.metaFile("loja"))
	if err != nil || !strings.Contains(string(raw), secret) {
		t.Fatalf("secret not persisted: %v", err)
	}
	m2 := testManager(t, dir, map[string]*fakeProvider{})
	m2.restorePersisted()
	again, err := m2.Get("loja")
	if err != nil || !again.WebhookConfig().HasSecret {
		t.Fatal("secret lost after restart")
	}

	// Apagar volta ao global.
	empty := ""
	if _, err := m.UpdateWebhook("loja", WebhookUpdate{Secret: &empty}); err != nil {
		t.Fatal(err)
	}
	if inst.WebhookConfig().HasSecret {
		t.Fatal("secret should be cleared")
	}
	short := "curto"
	if _, err := m.UpdateWebhook("loja", WebhookUpdate{Secret: &short}); err == nil {
		t.Fatal("short secret accepted")
	}
}
