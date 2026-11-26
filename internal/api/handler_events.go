package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rafaeldourado9/arcanum/internal/instance"
)

// notFoundOr400 responde 404 para instancia inexistente e 400 para o resto (validacao).
func notFoundOr400(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, instance.ErrNotFound) {
		status = http.StatusNotFound
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func handleGetWebhook(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inst, err := mgr.Get(chi.URLParam(r, "instance"))
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, inst.WebhookConfig())
	}
}

type setWebhookReq struct {
	URL     *string  `json:"url"`
	Events  []string `json:"events"`
	Enabled *bool    `json:"enabled"`
	// Secret: segredo proprio de assinatura desta sessao ("" volta ao segredo global).
	Secret *string `json:"secret"`
}

// handleRotateWebhookSecret gera um segredo novo de assinatura para a sessao. O valor aparece
// so nesta resposta; depois a API mostra apenas "hasSecret".
func handleRotateWebhookSecret(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret, err := mgr.RotateWebhookSecret(chi.URLParam(r, "instance"))
		if err != nil {
			notFoundOr400(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{
			"secret": secret,
			"notice": "Copy the secret now: it is shown only once. Use it to verify X-Hub-Signature-256 on your side.",
		})
	}
}

func handleSetWebhook(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req setWebhookReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, 400, map[string]string{"error": "Invalid JSON"})
			return
		}
		cfg, err := mgr.UpdateWebhook(chi.URLParam(r, "instance"), instance.WebhookUpdate{
			URL: req.URL, Events: req.Events, Enabled: req.Enabled, Secret: req.Secret,
		})
		if err != nil {
			notFoundOr400(w, err)
			return
		}
		writeJSON(w, 200, cfg)
	}
}

// handleTestWebhook manda um evento "test" assinado e devolve o resultado da entrega.
func handleTestWebhook(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d, err := mgr.TestWebhook(chi.URLParam(r, "instance"))
		if err != nil {
			notFoundOr400(w, err)
			return
		}
		writeJSON(w, 200, d)
	}
}

// handleWebhookDeliveries lista as ultimas entregas (sem corpo: so evento e resultado).
func handleWebhookDeliveries(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := mgr.Deliveries(chi.URLParam(r, "instance"))
		if err != nil {
			notFoundOr400(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"deliveries": list})
	}
}

func handleExportWebhooks(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="arcanum-webhooks.json"`)
		writeJSON(w, 200, map[string]any{"version": 1, "webhooks": mgr.ExportWebhooks()})
	}
}

func handleImportWebhooks(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Webhooks map[string]instance.WebhookConfig `json:"webhooks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Webhooks == nil {
			writeJSON(w, 400, map[string]string{"error": `expected {"webhooks": {"<instance>": {"url": "...", "events": [...], "enabled": true}}}`})
			return
		}
		writeJSON(w, 200, mgr.ImportWebhooks(req.Webhooks))
	}
}
