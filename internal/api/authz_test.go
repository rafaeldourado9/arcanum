package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rafaeldourado9/arcanum/internal/audit"
	"github.com/rafaeldourado9/arcanum/internal/config"
	"github.com/rafaeldourado9/arcanum/internal/instance"
	"github.com/rafaeldourado9/arcanum/internal/keys"
)

type authEnv struct {
	h     http.Handler
	mgr   *instance.Manager
	keys  *keys.Store
	audit *audit.Log
}

func newAuthEnv(t *testing.T) *authEnv {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{DBPath: dir, RateLimitPerMin: 100, APIKey: testAPIKey}
	ks, err := keys.Open(filepath.Join(dir, "keys.json"))
	if err != nil {
		t.Fatal(err)
	}
	al, err := audit.Open(filepath.Join(dir, "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(al.Close)
	mgr := instance.NewManager(cfg)
	for _, n := range []string{"loja", "suporte"} {
		if _, err := mgr.Create(n, "", nil); err != nil {
			t.Fatal(err)
		}
	}
	return &authEnv{h: NewServerWith(mgr, cfg, Deps{Keys: ks, Audit: al}), mgr: mgr, keys: ks, audit: al}
}

func (e *authEnv) do(method, path, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if key != "" {
		req.Header.Set("X-Api-Key", key)
	}
	req.RemoteAddr = "203.0.113.7:5555"
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func (e *authEnv) newKey(t *testing.T, instances []string, readOnly bool) string {
	t.Helper()
	secret, _, err := e.keys.Create("integrador", instances, readOnly)
	if err != nil {
		t.Fatal(err)
	}
	return secret
}

func TestScopedKeySeesOnlyItsInstances(t *testing.T) {
	e := newAuthEnv(t)
	k := e.newKey(t, []string{"loja"}, false)

	rec := e.do("GET", "/api/instance/fetchInstances", k, "")
	if rec.Code != 200 {
		t.Fatalf("fetchInstances = %d", rec.Code)
	}
	var list []instance.InstanceInfo
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 || list[0].Name != "loja" {
		t.Fatalf("scoped key saw %+v, want only loja", list)
	}

	// A chave mestra ve as duas.
	_ = json.Unmarshal(e.do("GET", "/api/instance/fetchInstances", testAPIKey, "").Body.Bytes(), &list)
	if len(list) != 2 {
		t.Fatalf("admin saw %d instances, want 2", len(list))
	}

	cases := []struct {
		name, method, path, body string
		want                     int
	}{
		{"own settings", "GET", "/api/settings/loja", "", 200},
		{"own webhook", "GET", "/api/events/webhook/loja", "", 200},
		{"other instance looks like it does not exist", "GET", "/api/settings/suporte", "", 404},
		{"other instance, message route", "POST", "/api/message/sendText/suporte", `{}`, 404},
		{"cannot create sessions", "POST", "/api/instance/create", `{"instanceName":"x"}`, 403},
		{"cannot delete sessions", "DELETE", "/api/instance/delete/loja", "", 403},
		{"no stats", "GET", "/api/stats", "", 403},
		{"no metrics", "GET", "/metrics", "", 403},
		{"no logs", "GET", "/api/logs", "", 403},
		{"no key admin", "GET", "/api/keys", "", 403},
		{"no audit", "GET", "/api/audit", "", 403},
		{"no webhook export", "GET", "/api/events/webhooks/export", "", 403},
		{"no webhook import", "POST", "/api/events/webhooks/import", `{"webhooks":{}}`, 403},
		{"encoded path is refused", "GET", "/api/settings/lo%6Aa", "", 400},
	}
	for _, tc := range cases {
		if got := e.do(tc.method, tc.path, k, tc.body).Code; got != tc.want {
			t.Errorf("%s: %s %s = %d, want %d", tc.name, tc.method, tc.path, got, tc.want)
		}
	}
}

func TestReadOnlyKey(t *testing.T) {
	e := newAuthEnv(t)
	k := e.newKey(t, []string{"loja"}, true)

	for _, tc := range []struct {
		name, method, path string
		want               int // 0 = qualquer coisa menos 403
	}{
		{"read settings", "GET", "/api/settings/loja", 200},
		{"change settings", "POST", "/api/settings/loja", 403},
		{"change webhook", "POST", "/api/events/webhook/loja", 403},
		{"rotate webhook secret", "POST", "/api/events/webhook/loja/secret", 403},
		{"send message", "POST", "/api/message/sendText/loja", 403},
		{"connect (starts pairing)", "GET", "/api/instance/connect/loja", 403},
		{"restart", "POST", "/api/instance/restart/loja", 403},
		{"find chats", "POST", "/api/chat/findChats/loja", 0},
	} {
		got := e.do(tc.method, tc.path, k, `{}`).Code
		if tc.want == 0 {
			if got == 403 {
				t.Errorf("%s: read-only query was forbidden", tc.name)
			}
			continue
		}
		if got != tc.want {
			t.Errorf("%s: %s %s = %d, want %d", tc.name, tc.method, tc.path, got, tc.want)
		}
	}
}

func TestKeyLifecycleThroughAPI(t *testing.T) {
	e := newAuthEnv(t)

	rec := e.do("POST", "/api/keys", testAPIKey, `{"name":"agente","instances":["loja"],"readOnly":false}`)
	if rec.Code != 201 {
		t.Fatalf("create key = %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		Key    keys.Key `json:"key"`
		Secret string   `json:"secret"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if !strings.HasPrefix(out.Secret, keys.Prefix) || out.Key.ID == "" {
		t.Fatalf("bad create response: %s", rec.Body)
	}

	// A listagem nunca devolve o segredo nem o hash.
	list := e.do("GET", "/api/keys", testAPIKey, "").Body.String()
	if strings.Contains(list, out.Secret) || strings.Contains(strings.ToLower(list), "hash") {
		t.Fatalf("key listing leaks secret material: %s", list)
	}

	if got := e.do("GET", "/api/settings/loja", out.Secret, "").Code; got != 200 {
		t.Fatalf("new key rejected: %d", got)
	}
	if got := e.do("POST", "/api/keys", out.Secret, `{"name":"x","instances":["loja"]}`).Code; got != 403 {
		t.Fatalf("scoped key created a key: %d", got)
	}

	if got := e.do("DELETE", "/api/keys/"+out.Key.ID, testAPIKey, "").Code; got != 200 {
		t.Fatalf("revoke = %d", got)
	}
	if got := e.do("GET", "/api/settings/loja", out.Secret, "").Code; got != 401 {
		t.Fatalf("revoked key still works: %d", got)
	}
	if got := e.do("DELETE", "/api/keys/key_naoexiste", testAPIKey, "").Code; got != 404 {
		t.Fatalf("revoking unknown key = %d, want 404", got)
	}
}

func TestWrongKeysAreBlockedAfterTooManyAttempts(t *testing.T) {
	e := newAuthEnv(t)
	for i := 0; i < 20; i++ {
		if got := e.do("GET", "/api/instance/fetchInstances", "errada", "").Code; got != 401 {
			t.Fatalf("attempt %d = %d, want 401", i, got)
		}
	}
	// Bloqueado: ate a chave certa espera (senao da para adivinhar sem custo).
	if got := e.do("GET", "/api/instance/fetchInstances", testAPIKey, "").Code; got != 429 {
		t.Fatalf("after 20 failures = %d, want 429", got)
	}
}

func TestMissingCredentialsDoNotCountAsAttacks(t *testing.T) {
	e := newAuthEnv(t)
	// O painel pergunta sem credencial antes do login: isso nao pode trancar o dono para fora.
	for i := 0; i < 40; i++ {
		if got := e.do("GET", "/api/instance/fetchInstances", "", "").Code; got != 401 {
			t.Fatalf("no key = %d, want 401", got)
		}
	}
	if got := e.do("GET", "/api/instance/fetchInstances", testAPIKey, "").Code; got != 200 {
		t.Fatalf("owner locked out by anonymous requests: %d", got)
	}
}

func TestAuditTrail(t *testing.T) {
	e := newAuthEnv(t)
	k := e.newKey(t, []string{"loja"}, true)

	e.do("POST", "/api/instance/create", testAPIKey, `{"instanceName":"nova"}`)
	e.do("POST", "/api/settings/loja", k, `{"rejectCalls":true}`) // negado: so leitura
	e.do("GET", "/api/instance/fetchInstances", "chute-errado", "")

	actions := map[string]audit.Event{}
	for _, ev := range e.audit.Tail(50) {
		actions[ev.Action] = ev
	}
	if ev, ok := actions["instance.create"]; !ok || ev.Actor != "admin" || ev.Target != "nova" || ev.Status != 201 {
		t.Errorf("missing/wrong instance.create event: %+v", ev)
	}
	if ev, ok := actions["auth.denied"]; !ok || !strings.HasPrefix(ev.Actor, "key:") {
		t.Errorf("missing/wrong auth.denied event: %+v", ev)
	}
	if ev, ok := actions["auth.failed"]; !ok || ev.Actor != "-" || ev.IP != "203.0.113.7" {
		t.Errorf("missing/wrong auth.failed event: %+v", ev)
	}

	// Nenhum segredo vai para o log.
	b, _ := json.Marshal(e.audit.Tail(50))
	for _, secret := range []string{testAPIKey, k, "chute-errado"} {
		if strings.Contains(string(b), secret) {
			t.Fatalf("audit log leaks a credential")
		}
	}

	// So o admin le a auditoria.
	if got := e.do("GET", "/api/audit", testAPIKey, "").Code; got != 200 {
		t.Fatalf("admin cannot read audit: %d", got)
	}
}

func TestWebhookSecretNeverExposed(t *testing.T) {
	e := newAuthEnv(t)

	rec := e.do("POST", "/api/events/webhook/loja/secret", testAPIKey, "")
	if rec.Code != 200 {
		t.Fatalf("rotate = %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		Secret string `json:"secret"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if !strings.HasPrefix(out.Secret, "whsec_") {
		t.Fatalf("unexpected secret %q", out.Secret)
	}

	for _, path := range []string{"/api/events/webhook/loja", "/api/instance/fetchInstances", "/api/events/webhooks/export"} {
		body := e.do("GET", path, testAPIKey, "").Body.String()
		if strings.Contains(body, out.Secret) {
			t.Fatalf("%s leaks the webhook secret", path)
		}
	}
	if !strings.Contains(e.do("GET", "/api/events/webhook/loja", testAPIKey, "").Body.String(), `"hasSecret":true`) {
		t.Fatal("hasSecret not reported")
	}

	if got := e.do("POST", "/api/events/webhook/loja", testAPIKey, `{"secret":"curto"}`).Code; got != 400 {
		t.Fatalf("short secret = %d, want 400", got)
	}
	if got := e.do("POST", "/api/events/webhook/loja", testAPIKey, `{"secret":""}`).Code; got != 200 {
		t.Fatalf("clearing secret = %d, want 200", got)
	}
	if !strings.Contains(e.do("GET", "/api/events/webhook/loja", testAPIKey, "").Body.String(), `"hasSecret":false`) {
		t.Fatal("secret not cleared")
	}
}
