package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rafaeldourado9/arcanum/internal/instance"
)

func handleGetSettings(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inst, err := mgr.Get(chi.URLParam(r, "instance"))
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, inst.Settings())
	}
}

// handleSetSettings aplica so os campos enviados (o JSON e decodificado por cima das
// configuracoes atuais) e grava em disco.
func handleSetSettings(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var raw json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			writeJSON(w, 400, map[string]string{"error": "Invalid JSON"})
			return
		}
		s, err := mgr.UpdateSettings(chi.URLParam(r, "instance"), func(s *instance.Settings) error {
			return json.Unmarshal(raw, s)
		})
		if err != nil {
			notFoundOr400(w, err)
			return
		}
		writeJSON(w, 200, s)
	}
}
