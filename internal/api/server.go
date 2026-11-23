package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"strings"

	"github.com/rafaeldourado9/arcanum/internal/config"
	"github.com/rafaeldourado9/arcanum/internal/dashauth"
	"github.com/rafaeldourado9/arcanum/internal/dashboard"
	"github.com/rafaeldourado9/arcanum/internal/instance"
	"github.com/rafaeldourado9/arcanum/internal/logbuf"
	"github.com/rafaeldourado9/arcanum/internal/metrics"
)

// Version e preenchida no build (-ldflags "-X .../internal/api.Version=v1.1.0").
var Version = "dev"

// Deps sao pecas opcionais do servidor (os testes montam sem elas).
type Deps struct {
	Logs *logbuf.Buffer // linhas de log para o painel
}

func NewServer(mgr *instance.Manager, cfg *config.Config) http.Handler {
	return NewServerWith(mgr, cfg, Deps{})
}

func NewServerWith(mgr *instance.Manager, cfg *config.Config, deps Deps) http.Handler {
	// Login do painel: senha propria ou, sem ela, a propria chave de API.
	var sessions *dashauth.Auth
	if cfg.DashboardEnabled && !cfg.AuthDisabled {
		password := cfg.DashboardPassword
		if password == "" {
			password = cfg.APIKey
		}
		if password != "" {
			sessions = dashauth.New(password)
		}
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	// Sem middleware.RealIP: ele confia em X-Forwarded-For de qualquer um (falsificavel) e o
	// gateway nao usa o IP do cliente para nada.
	r.Use(countResponses)
	r.Use(securityHeaders)
	if !cfg.AuthDisabled {
		r.Use(requireAuth(cfg.APIKey, sessions))
	}
	maxBody := cfg.MaxBodyMB
	if maxBody <= 0 {
		maxBody = 64 // Config montada sem Load(): limite 0 recusaria todo pedido
	}
	r.Use(limitBody(int64(maxBody) << 20))

	// System
	r.Get("/health", handleHealthMulti(mgr))
	r.Get("/metrics", handleMetrics(mgr))
	r.Get("/docs", handleDocs())
	r.Get("/docs/openapi.json", handleOpenAPI())
	r.Get("/api/stats", handleStats(mgr))
	r.Get("/api/logs", handleLogs(deps.Logs))
	r.Get("/api/logs/stream", handleLogStream(deps.Logs))

	// Dashboard login
	r.Post("/api/auth/login", handleLogin(sessions))
	r.Post("/api/auth/logout", handleLogout(sessions))
	r.Get("/api/auth/me", handleMe(sessions, cfg.AuthDisabled))

	// Instance
	r.Post("/api/instance/create", handleCreateInstance(mgr))
	r.Get("/api/instance/connect/{instance}", handleConnectInstance(mgr))
	r.Get("/api/instance/connectionState/{instance}", handleConnectionState(mgr))
	r.Delete("/api/instance/logout/{instance}", handleLogoutInstance(mgr))
	r.Delete("/api/instance/delete/{instance}", handleDeleteInstance(mgr))
	r.Post("/api/instance/restart/{instance}", handleRestartInstance(mgr))
	r.Post("/api/instance/setPresence/{instance}", handleSetPresence(mgr))
	r.Get("/api/instance/fetchInstances", handleFetchInstances(mgr))
	r.Get("/api/instance/qr/{instance}", handleGetQR(mgr))
	r.Post("/api/instance/pair/{instance}", handlePairInstance(mgr))

	// Message
	r.Post("/api/message/sendText/{instance}", handleSendText(mgr, cfg))
	r.Post("/api/message/sendMedia/{instance}", handleSendMedia(mgr, cfg))
	r.Post("/api/message/sendLocation/{instance}", handleSendLocation(mgr))
	r.Post("/api/message/sendContact/{instance}", handleSendContact(mgr))
	r.Post("/api/message/sendReaction/{instance}", handleSendReaction(mgr))
	r.Post("/api/message/sendPoll/{instance}", handleSendPoll(mgr))

	// Chat
	r.Post("/api/chat/archiveChat/{instance}", handleArchiveChat(mgr))
	r.Post("/api/chat/findChats/{instance}", handleFindChats(mgr))
	r.Post("/api/chat/findContacts/{instance}", handleFindContacts(mgr))
	r.Post("/api/chat/findMessages/{instance}", handleFindMessages(mgr))
	r.Post("/api/chat/markMessageAsRead/{instance}", handleMarkAsRead(mgr))
	r.Post("/api/chat/updateProfileName/{instance}", handleUpdateProfileName(mgr))
	r.Post("/api/chat/updateProfilePicture/{instance}", handleUpdateProfilePicture(mgr))
	r.Post("/api/chat/updateProfileStatus/{instance}", handleUpdateProfileStatus(mgr))
	r.Post("/api/chat/checkWhatsAppNumbers/{instance}", handleCheckNumbers(mgr))

	// Events
	r.Get("/api/events/webhook/{instance}", handleGetWebhook(mgr))
	r.Post("/api/events/webhook/{instance}", handleSetWebhook(mgr))
	r.Post("/api/events/webhook/{instance}/test", handleTestWebhook(mgr))
	r.Get("/api/events/webhook/{instance}/deliveries", handleWebhookDeliveries(mgr))
	r.Get("/api/events/webhooks/export", handleExportWebhooks(mgr))
	r.Post("/api/events/webhooks/import", handleImportWebhooks(mgr))

	// Settings
	r.Get("/api/settings/{instance}", handleGetSettings(mgr))
	r.Post("/api/settings/{instance}", handleSetSettings(mgr))

	// Painel web: tudo que nao e API, metricas ou docs.
	if cfg.DashboardEnabled {
		r.NotFound(func(w http.ResponseWriter, req *http.Request) {
			if !isDashboardPath(req.URL.Path) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}
			dashboard.Handler().ServeHTTP(w, req)
		})
	}

	return r
}

// isDashboardPath: caminhos do painel (paginas e arquivos), nunca a API.
func isDashboardPath(p string) bool {
	return !strings.HasPrefix(p, "/api/") && p != "/api" && p != "/metrics" && p != "/health" && !strings.HasPrefix(p, "/docs")
}

// handleHealthMulti e publico: so contagens, nenhum nome de sessao nem numero.
func handleHealthMulti(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions := sessionsByStatus(mgr)
		total := 0
		for _, n := range sessions {
			total += n
		}
		writeJSON(w, 200, map[string]any{
			"status":    "ok",
			"version":   Version,
			"instances": total,
			"sessions":  sessions,
		})
	}
}

func handleMetrics(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		metrics.Write(w, sessionsByStatus(mgr))
	}
}

func sessionsByStatus(mgr *instance.Manager) map[string]int {
	out := map[string]int{}
	for _, inst := range mgr.List() {
		out[string(inst.Status)]++
	}
	return out
}

// securityHeaders: o navegador nao adivinha tipo de conteudo, a API nao abre em iframe
// (clickjacking) e nenhuma resposta fica em cache compartilhado (QR code e dados de sessao).
// Na /docs a CSP libera so o Swagger UI fixado no unpkg.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		switch {
		case r.URL.Path == "/docs":
			h.Set("Content-Security-Policy", "default-src 'none'; script-src 'self' 'unsafe-inline' https://unpkg.com; style-src 'self' 'unsafe-inline' https://unpkg.com; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'")
		case isDashboardPath(r.URL.Path):
			// Painel: tudo do proprio gateway; QR code vem como data: URL.
			h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'; object-src 'none'")
		default:
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		}
		next.ServeHTTP(w, r)
	})
}

// countResponses conta cada resposta pelo codigo HTTP para o /metrics.
func countResponses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		metrics.HTTPRequest(status)
	})
}
