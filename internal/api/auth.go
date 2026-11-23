package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/rafaeldourado9/arcanum/internal/dashauth"
)

// publicPaths nao pedem chave: checagem de saude (orquestradores e load balancers), a
// documentacao e as rotas de login do painel.
var publicPaths = map[string]bool{
	"/health":            true,
	"/docs":              true,
	"/docs/openapi.json": true,
	"/api/auth/login":    true,
	"/api/auth/logout":   true,
	"/api/auth/me":       true,
}

// needsAuth: a API e as metricas. O resto (arquivos do painel: HTML, JS, CSS, fonte) e
// publico, porque nao tem dado nenhum; os dados vem da API, que exige login.
func needsAuth(path string) bool {
	if publicPaths[path] {
		return false
	}
	return strings.HasPrefix(path, "/api/") || path == "/metrics"
}

// requireAuth aceita a chave de API (X-Api-Key ou Bearer) ou, para o painel, cookie de
// sessao + header anti-CSRF. A chave e comparada em tempo constante.
func requireAuth(key string, sessions *dashauth.Auth) func(http.Handler) http.Handler {
	want := []byte(key)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !needsAuth(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			if got := []byte(apiKeyFrom(r)); len(got) > 0 && subtle.ConstantTimeCompare(got, want) == 1 {
				next.ServeHTTP(w, r)
				return
			}
			if sessions != nil && sessions.Authenticated(r) {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("WWW-Authenticate", `Bearer realm="arcanum"`)
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "missing or invalid API key (send X-Api-Key or Authorization: Bearer)",
			})
		})
	}
}

// requireAPIKey: so a chave (usado nos testes do middleware).
func requireAPIKey(key string) func(http.Handler) http.Handler { return requireAuth(key, nil) }

func apiKeyFrom(r *http.Request) string {
	if k := strings.TrimSpace(r.Header.Get("X-Api-Key")); k != "" {
		return k
	}
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

// GenerateAPIKey cria uma chave aleatoria de 32 bytes (64 caracteres hex), usada quando o
// operador nao define GATEWAY_API_KEY.
func GenerateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// limitBody corta requisicoes maiores que o limite antes de alguem ler o corpo inteiro na
// memoria (midia em base64 vem no JSON).
func limitBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// ── login do painel ──

func handleLogin(sessions *dashauth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if sessions == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "dashboard login is disabled"})
			return
		}
		ip := dashauth.ClientIP(r)
		if sessions.Blocked(ip) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many failed attempts, try again in 15 minutes"})
			return
		}
		var req struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing password"})
			return
		}
		token, ok := sessions.Login(ip, req.Password)
		if !ok {
			time.Sleep(400 * time.Millisecond) // encarece tentativa automatizada
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "wrong password"})
			return
		}
		dashauth.SetCookie(w, r, token)
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": true})
	}
}

func handleLogout(sessions *dashauth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if sessions != nil {
			sessions.Logout(dashauth.FromRequest(r))
		}
		dashauth.SetCookie(w, r, "")
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
	}
}

// handleMe diz ao painel se ha sessao (sem expor nada alem disso).
func handleMe(sessions *dashauth.Auth, authDisabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok := authDisabled || (sessions != nil && sessions.Valid(dashauth.FromRequest(r)))
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": ok, "version": Version})
	}
}
