package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testAPIKey = "test-key-0123456789abcdef0123456789abcdef"

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestRequireAPIKey(t *testing.T) {
	h := requireAPIKey("secret-key")(okHandler())

	cases := []struct {
		name   string
		path   string
		header map[string]string
		want   int
	}{
		{"no key", "/api/instance/fetchInstances", nil, http.StatusUnauthorized},
		{"wrong key", "/api/instance/fetchInstances", map[string]string{"X-Api-Key": "nope"}, http.StatusUnauthorized},
		{"x-api-key", "/api/instance/fetchInstances", map[string]string{"X-Api-Key": "secret-key"}, http.StatusOK},
		{"bearer", "/api/instance/fetchInstances", map[string]string{"Authorization": "Bearer secret-key"}, http.StatusOK},
		{"bearer lowercase scheme", "/api/instance/fetchInstances", map[string]string{"Authorization": "bearer secret-key"}, http.StatusOK},
		{"basic is not accepted", "/api/instance/fetchInstances", map[string]string{"Authorization": "Basic secret-key"}, http.StatusUnauthorized},
		{"health is public", "/health", nil, http.StatusOK},
		{"docs are public", "/docs", nil, http.StatusOK},
		{"metrics need a key", "/metrics", nil, http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			for k, v := range tc.header {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestGenerateAPIKey(t *testing.T) {
	a, err := GenerateAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := GenerateAPIKey()
	if len(a) != 64 || a == b {
		t.Fatalf("expected two different 64-char keys, got %q and %q", a, b)
	}
}

func TestLimitBody(t *testing.T) {
	h := limitBody(10)(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/message/sendText/x", strings.NewReader(strings.Repeat("a", 20)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
}
