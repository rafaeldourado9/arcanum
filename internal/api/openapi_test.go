package api

import (
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/rafaeldourado9/arcanum/internal/config"
	"github.com/rafaeldourado9/arcanum/internal/instance"
)

// A spec ja listou 18 rotas que nao existiam. Este teste prende spec e roteador juntos:
// rota nova sem documentacao, ou documentacao de rota inexistente, quebra o CI.
func TestOpenAPIMatchesRoutes(t *testing.T) {
	cfg := &config.Config{DBPath: t.TempDir(), APIKey: testAPIKey}
	router, ok := NewServer(instance.NewManager(cfg), cfg).(chi.Routes)
	if !ok {
		t.Fatal("NewServer no longer returns a chi router")
	}

	real := map[string]bool{}
	_ = chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if route == "/docs" || route == "/docs/openapi.json" {
			return nil // a propria documentacao
		}
		real[strings.ToLower(method)+" "+route] = true
		return nil
	})

	spec := map[string]bool{}
	for path, ops := range buildPaths() {
		for method := range ops.(map[string]any) {
			spec[method+" "+path] = true
		}
	}

	var missing, extra []string
	for r := range real {
		if !spec[r] {
			missing = append(missing, r)
		}
	}
	for s := range spec {
		if !real[s] {
			extra = append(extra, s)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 {
		t.Errorf("routes without OpenAPI docs: %v", missing)
	}
	if len(extra) > 0 {
		t.Errorf("OpenAPI documents routes that do not exist: %v", extra)
	}
}
