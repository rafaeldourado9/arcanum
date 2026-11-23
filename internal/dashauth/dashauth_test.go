package dashauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginSessionAndLockout(t *testing.T) {
	a := New("correct horse battery staple")
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return now }

	if _, ok := a.Login("1.2.3.4", "wrong"); ok {
		t.Fatal("wrong password accepted")
	}
	tok, ok := a.Login("1.2.3.4", "correct horse battery staple")
	if !ok || len(tok) != 64 {
		t.Fatalf("login failed: ok=%v tok=%q", ok, tok)
	}
	if !a.Valid(tok) {
		t.Fatal("fresh session not valid")
	}

	// cookie sozinho nao basta: precisa do header anti-CSRF
	r := httptest.NewRequest("GET", "/api/instance/fetchInstances", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: tok})
	if a.Authenticated(r) {
		t.Fatal("authenticated without CSRF header")
	}
	r.Header.Set(CSRFHeader, CSRFValue)
	if !a.Authenticated(r) {
		t.Fatal("not authenticated with cookie + header")
	}

	a.Logout(tok)
	if a.Valid(tok) {
		t.Fatal("session valid after logout")
	}

	tok2, _ := a.Login("1.2.3.4", "correct horse battery staple")
	now = now.Add(SessionTTL + time.Minute)
	if a.Valid(tok2) {
		t.Fatal("session valid after TTL")
	}

	for i := 0; i < maxFailures; i++ {
		a.Login("9.9.9.9", "nope")
	}
	if !a.Blocked("9.9.9.9") {
		t.Fatal("not blocked after too many failures")
	}
	if a.Blocked("1.2.3.4") {
		t.Fatal("other IP blocked")
	}
	now = now.Add(failureWindow + time.Minute)
	if a.Blocked("9.9.9.9") {
		t.Fatal("still blocked after window")
	}
}
