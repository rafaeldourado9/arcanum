package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/rafaeldourado9/arcanum/internal/audit"
	"github.com/rafaeldourado9/arcanum/internal/dashauth"
	"github.com/rafaeldourado9/arcanum/internal/keys"
)

func noKeyStore(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "API keys are not available (key store not initialised)"})
}

func handleListKeys(store *keys.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			noKeyStore(w)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"keys": store.List()})
	}
}

type createKeyReq struct {
	Name      string   `json:"name"`
	Instances []string `json:"instances"`
	ReadOnly  bool     `json:"readOnly"`
}

// handleCreateKey cria uma chave de escopo. O segredo vem so nesta resposta.
func handleCreateKey(store *keys.Store, aud *audit.Log) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			noKeyStore(w)
			return
		}
		var req createKeyReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
			return
		}
		secret, k, err := store.Create(req.Name, req.Instances, req.ReadOnly)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		aud.Record(audit.Event{Actor: scopeFrom(r).id, IP: dashauth.ClientIP(r), Action: "key.create", Target: k.ID, Status: http.StatusCreated})
		writeJSON(w, http.StatusCreated, map[string]any{
			"key":    k,
			"secret": secret,
			"notice": "Copy the secret now: it is shown only once and cannot be recovered.",
		})
	}
}

func handleRevokeKey(store *keys.Store, aud *audit.Log) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			noKeyStore(w)
			return
		}
		k, err := store.Revoke(chi.URLParam(r, "id"))
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, keys.ErrNotFound) {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		aud.Record(audit.Event{Actor: scopeFrom(r).id, IP: dashauth.ClientIP(r), Action: "key.revoke", Target: k.ID, Status: http.StatusOK})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "key": k})
	}
}

// handleAudit devolve os ultimos eventos de auditoria (mais novo primeiro).
func handleAudit(aud *audit.Log) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 100
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		if limit > 500 {
			limit = 500
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": aud.Tail(limit)})
	}
}
