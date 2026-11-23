// Package dashauth faz o login do painel: senha -> sessao em cookie HttpOnly.
//
// Decisoes de seguranca:
//   - Sessao guardada so em memoria, pelo hash do token (vazar a memoria nao entrega cookies
//     validos prontos; reiniciar o gateway desloga todo mundo).
//   - Cookie HttpOnly (JavaScript nao le), SameSite=Strict (outro site nao usa o cookie) e
//     Secure quando a conexao e HTTPS.
//   - Pedido autenticado por cookie tambem precisa do header X-Requested-With: arcanum
//     (protecao extra contra CSRF: formulario de outro site nao consegue mandar header).
//   - Senha comparada em tempo constante; 10 erros por IP em 15 minutos bloqueiam o login.
package dashauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	CookieName    = "arcanum_session"
	CSRFHeader    = "X-Requested-With"
	CSRFValue     = "arcanum"
	SessionTTL    = 12 * time.Hour
	maxFailures   = 10
	failureWindow = 15 * time.Minute
)

type Auth struct {
	password [32]byte // sha256 da senha: comparacao de tamanho fixo

	mu       sync.Mutex
	sessions map[[32]byte]time.Time // hash do token -> expira em
	failures map[string][]time.Time // ip -> erros recentes
	now      func() time.Time
}

// New cria o login com a senha do painel.
func New(password string) *Auth {
	return &Auth{
		password: sha256.Sum256([]byte(password)),
		sessions: map[[32]byte]time.Time{},
		failures: map[string][]time.Time{},
		now:      time.Now,
	}
}

// Blocked diz se o IP errou demais e esta temporariamente bloqueado.
func (a *Auth) Blocked(ip string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.recentFailures(ip)) >= maxFailures
}

func (a *Auth) recentFailures(ip string) []time.Time {
	cut := a.now().Add(-failureWindow)
	kept := a.failures[ip][:0]
	for _, t := range a.failures[ip] {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(a.failures, ip)
		return nil
	}
	a.failures[ip] = kept
	return kept
}

// Login confere a senha e, se certa, devolve um token de sessao novo.
func (a *Auth) Login(ip, password string) (string, bool) {
	got := sha256.Sum256([]byte(password))
	if subtle.ConstantTimeCompare(got[:], a.password[:]) != 1 {
		a.mu.Lock()
		a.failures[ip] = append(a.recentFailures(ip), a.now())
		a.mu.Unlock()
		return "", false
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", false
	}
	token := hex.EncodeToString(raw)
	a.mu.Lock()
	delete(a.failures, ip)
	a.gc()
	a.sessions[sha256.Sum256([]byte(token))] = a.now().Add(SessionTTL)
	a.mu.Unlock()
	return token, true
}

// Logout invalida o token.
func (a *Auth) Logout(token string) {
	a.mu.Lock()
	delete(a.sessions, sha256.Sum256([]byte(token)))
	a.mu.Unlock()
}

// Valid diz se o token e de uma sessao ativa.
func (a *Auth) Valid(token string) bool {
	if token == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.sessions[sha256.Sum256([]byte(token))]
	return ok && a.now().Before(exp)
}

func (a *Auth) gc() {
	now := a.now()
	for k, exp := range a.sessions {
		if !now.Before(exp) {
			delete(a.sessions, k)
		}
	}
}

// FromRequest devolve o token do cookie, se houver.
func FromRequest(r *http.Request) string {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// Authenticated: cookie de sessao valido E header anti-CSRF presente.
func (a *Auth) Authenticated(r *http.Request) bool {
	return r.Header.Get(CSRFHeader) == CSRFValue && a.Valid(FromRequest(r))
}

// SetCookie grava o cookie de sessao (ou apaga, com token vazio).
func SetCookie(w http.ResponseWriter, r *http.Request, token string) {
	// #nosec G124 -- HttpOnly e SameSite=Strict sempre; Secure so em HTTPS de proposito: com
	// Secure fixo o login nao funcionaria em http:// na rede local (o painel avisa para usar
	// HTTPS fora do localhost).
	c := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		MaxAge:   int(SessionTTL.Seconds()),
	}
	if token == "" {
		c.MaxAge = -1
	}
	http.SetCookie(w, c)
}

// ClientIP e o IP da conexao (nao confia em X-Forwarded-For, que qualquer um forja). Atras
// de proxy, todos os pedidos vem do IP do proxy: o limite vira global, o que ainda protege.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
