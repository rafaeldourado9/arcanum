package api

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/rafaeldourado9/arcanum/internal/audit"
	"github.com/rafaeldourado9/arcanum/internal/dashauth"
	"github.com/rafaeldourado9/arcanum/internal/keys"
)

// Quem chama. Tres tipos:
//   - admin: a chave mestra (GATEWAY_API_KEY) e o painel logado. Veem e fazem tudo.
//   - chave de escopo (arc_...): so as sessoes listadas e, se readOnly, so consulta. Serve
//     para entregar a um integrador (agente de IA, sistema de terceiros) acesso as sessoes dele
//     sem dar acesso as dos outros.
type principal struct {
	id       string // "admin", "panel" ou "key:<id>"
	admin    bool
	readOnly bool
	key      keys.Key
}

type ctxKey struct{}

type auditTargetKey struct{}

// setAuditTarget deixa o handler informar a sessao afetada quando ela nao esta no caminho.
func setAuditTarget(r *http.Request, name string) {
	if p, ok := r.Context().Value(auditTargetKey{}).(*string); ok {
		*p = name
	}
}

// scopeFrom devolve quem fez o pedido. Sem autenticacao ligada (GATEWAY_AUTH_DISABLED) ou em
// teste de handler isolado nao ha ninguem no contexto: vale como admin.
func scopeFrom(r *http.Request) principal {
	if p, ok := r.Context().Value(ctxKey{}).(principal); ok {
		return p
	}
	return principal{id: "admin", admin: true}
}

// canSee diz se o principal pode ver a sessao.
func (p principal) canSee(instance string) bool {
	return p.admin || p.key.Allows(instance)
}

type authDeps struct {
	sessions *dashauth.Auth
	keys     *keys.Store
	audit    *audit.Log
}

type authGuard struct {
	master []byte
	authDeps
	fails *failLimiter
}

func newAuthGuard(master string, d authDeps) *authGuard {
	return &authGuard{master: []byte(master), authDeps: d, fails: newFailLimiter(20, 15*time.Minute)}
}

// failLimiter bloqueia o IP que erra a chave de API muitas vezes seguidas. Atras de proxy
// todos os pedidos chegam do IP do proxy (X-Forwarded-For nao e confiavel): o limite vira
// global, o que ainda protege contra adivinhacao.
type failLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	fails  map[string][]time.Time
	now    func() time.Time
}

func newFailLimiter(max int, window time.Duration) *failLimiter {
	return &failLimiter{max: max, window: window, fails: map[string][]time.Time{}, now: time.Now}
}

func (f *failLimiter) recent(ip string) []time.Time {
	cut := f.now().Add(-f.window)
	kept := f.fails[ip][:0]
	for _, t := range f.fails[ip] {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(f.fails, ip)
		return nil
	}
	f.fails[ip] = kept
	return kept
}

func (f *failLimiter) blocked(ip string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.recent(ip)) >= f.max
}

// add registra um erro e diz se foi ele que fechou o bloqueio.
func (f *failLimiter) add(ip string) (justBlocked bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.fails) > 10000 { // varre IPs antigos: o mapa nao cresce sem limite
		for k := range f.fails {
			f.recent(k)
		}
	}
	n := len(f.recent(ip)) + 1
	f.fails[ip] = append(f.fails[ip], f.now())
	return n == f.max
}

func (g *authGuard) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !needsAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		ip := dashauth.ClientIP(r)
		if g.fails.blocked(ip) {
			w.Header().Set("Retry-After", "900")
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many failed attempts, try again in 15 minutes"})
			return
		}

		p, state := g.identify(r)
		switch state {
		case identFailed:
			if g.fails.add(ip) {
				g.audit.Record(audit.Event{Actor: "-", IP: ip, Action: "auth.blocked", Detail: "too many wrong API keys"})
			}
			g.audit.Record(audit.Event{Actor: "-", IP: ip, Action: "auth.failed", Target: r.URL.Path, Status: http.StatusUnauthorized})
			unauthorized(w)
			return
		case identNone:
			unauthorized(w) // sem credencial: nao conta como tentativa (o painel pergunta antes do login)
			return
		}

		if status, msg := g.authorize(p, r); status != 0 {
			g.audit.Record(audit.Event{Actor: p.id, IP: ip, Action: "auth.denied", Target: r.URL.Path, Status: status, Detail: r.Method})
			writeJSON(w, status, map[string]string{"error": msg})
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, p))
		action, auditable := auditAction(r)
		if !auditable {
			next.ServeHTTP(w, r)
			return
		}
		var bodyTarget string
		r = r.WithContext(context.WithValue(r.Context(), auditTargetKey{}, &bodyTarget))
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		target, _ := instanceFromPath(r.URL.Path)
		if target == "" {
			target = bodyTarget // ex.: instance.create, cujo nome vem no corpo
		}
		g.audit.Record(audit.Event{Actor: p.id, IP: ip, Action: action, Target: target, Status: status})
	})
}

type identState int

const (
	identOK identState = iota
	identNone
	identFailed
)

// identify descobre quem chama: chave mestra (comparada em tempo constante), chave de escopo
// ou sessao do painel (cookie + header anti-CSRF).
func (g *authGuard) identify(r *http.Request) (principal, identState) {
	if presented := apiKeyFrom(r); presented != "" {
		if len(g.master) > 0 && subtle.ConstantTimeCompare([]byte(presented), g.master) == 1 {
			return principal{id: "admin", admin: true}, identOK
		}
		if k, ok := g.keys.Authenticate(presented); ok {
			return principal{id: "key:" + k.ID, readOnly: k.ReadOnly, key: k}, identOK
		}
		return principal{}, identFailed
	}
	if g.sessions != nil && g.sessions.Authenticated(r) {
		return principal{id: "panel", admin: true}, identOK
	}
	return principal{}, identNone
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="arcanum"`)
	writeJSON(w, http.StatusUnauthorized, map[string]string{
		"error": "missing or invalid API key (send X-Api-Key or Authorization: Bearer)",
	})
}

// readOnlyPosts sao POSTs que so consultam: uma chave somente-leitura pode usa-los.
var readOnlyPosts = map[string]bool{
	"findChats": true, "findContacts": true, "findMessages": true, "checkWhatsAppNumbers": true,
}

// authorize decide se o principal pode fazer o pedido. Devolve 0 se pode.
//
// Chave de escopo so alcanca rotas que tem uma sessao no caminho, e so as dela. Sessao de
// outro integrador responde 404 (igual a uma que nao existe), para nao revelar que existe.
// Fica de fora, por ser administracao: criar/apagar sessao, chaves, auditoria, metricas, logs,
// estatisticas e importar/exportar webhooks.
func (g *authGuard) authorize(p principal, r *http.Request) (int, string) {
	if p.admin {
		return 0, ""
	}
	if r.URL.RawPath != "" && r.URL.RawPath != r.URL.Path {
		return http.StatusBadRequest, "invalid path"
	}
	if r.URL.Path == "/api/instance/fetchInstances" && r.Method == http.MethodGet {
		return 0, "" // a lista sai filtrada pelo handler
	}
	name, ok := instanceFromPath(r.URL.Path)
	if !ok {
		return http.StatusForbidden, "this API key cannot use this endpoint"
	}
	if !p.key.Allows(name) {
		return http.StatusNotFound, "instance not found"
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) >= 3 && parts[1] == "instance" && parts[2] == "delete" {
		return http.StatusForbidden, "this API key cannot delete a session"
	}
	if p.readOnly {
		switch {
		case r.Method == http.MethodGet && len(parts) >= 3 && parts[1] == "instance" && parts[2] == "connect":
			return http.StatusForbidden, "this API key is read-only"
		case r.Method == http.MethodGet || r.Method == http.MethodHead:
		case r.Method == http.MethodPost && len(parts) >= 3 && parts[1] == "chat" && readOnlyPosts[parts[2]]:
		default:
			return http.StatusForbidden, "this API key is read-only"
		}
	}
	return 0, ""
}

// instanceFromPath extrai a sessao do caminho das rotas por sessao.
func instanceFromPath(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 || parts[0] != "api" {
		return "", false
	}
	switch parts[1] {
	case "instance", "message", "chat":
		if len(parts) >= 4 && parts[3] != "" {
			return parts[3], true
		}
	case "settings":
		if len(parts) == 3 && parts[2] != "" {
			return parts[2], true
		}
	case "events":
		if len(parts) >= 4 && parts[2] == "webhook" && parts[3] != "" {
			return parts[3], true
		}
	}
	return "", false
}

// auditAction diz se o pedido entra na auditoria e com que nome. Entram so as acoes que mudam
// estado ou dao acesso (nao o envio de mensagem nem as consultas, que seriam milhares).
func auditAction(r *http.Request) (string, bool) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "api" {
		return "", false
	}
	switch parts[1] {
	case "instance":
		switch parts[2] {
		case "create", "logout", "delete", "restart", "pair", "setPresence":
			return "instance." + parts[2], r.Method != http.MethodGet
		case "connect":
			return "instance.connect", true
		}
	case "settings":
		return "settings.update", r.Method == http.MethodPost
	case "events":
		if r.Method != http.MethodPost && parts[2] != "webhooks" {
			return "", false
		}
		switch {
		case parts[2] == "webhooks" && len(parts) >= 4:
			return "webhook." + parts[3], true
		case parts[2] == "webhook" && len(parts) >= 5:
			return "webhook." + parts[4], true
		case parts[2] == "webhook":
			return "webhook.update", true
		}
	case "chat":
		if strings.HasPrefix(parts[2], "updateProfile") {
			return "profile.update", r.Method == http.MethodPost
		}
	}
	return "", false
}
