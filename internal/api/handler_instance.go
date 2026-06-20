package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rafaeldourado9/arcanum/internal/instance"
	qrcode "github.com/skip2/go-qrcode"
)

func b64encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

type createInstanceReq struct {
	InstanceName  string   `json:"instanceName"`
	Webhook       string   `json:"webhook"`
	WebhookEvents []string `json:"webhookEvents"`
}

func handleCreateInstance(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createInstanceReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InstanceName == "" {
			writeJSON(w, 400, map[string]string{"error": "Missing 'instanceName'"})
			return
		}

		inst, err := mgr.Create(req.InstanceName, req.Webhook, req.WebhookEvents)
		if err != nil {
			writeJSON(w, 409, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, 201, map[string]any{
			"instanceName": inst.Name,
			"status":       inst.Provider.Status(),
			"webhook":      inst.WebhookConfig,
		})
	}
}

func handleConnectInstance(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "instance")
		inst, err := mgr.Connect(name)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, 200, map[string]any{
			"instanceName": inst.Name,
			"status":       inst.Provider.Status(),
			"qr":           inst.Provider.QRCode(),
		})
	}
}

func handleConnectionState(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "instance")
		inst, err := mgr.Get(name)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}

		resp := map[string]any{
			"instanceName": inst.Name,
			"status":       inst.Provider.Status(),
		}
		if qr := inst.Provider.QRCode(); qr != "" {
			resp["qr"] = qr
		}
		writeJSON(w, 200, resp)
	}
}

func handleLogoutInstance(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "instance")
		if err := mgr.Logout(name); err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "instanceName": name})
	}
}

func handleDeleteInstance(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "instance")
		if err := mgr.Delete(name); err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "instanceName": name})
	}
}

func handleRestartInstance(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "instance")
		if err := mgr.Logout(name); err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		inst, err := mgr.Connect(name)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "instanceName": name, "status": inst.Provider.Status()})
	}
}

type setPresenceReq struct {
	Presence string `json:"presence"`
}

func handleSetPresence(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "instance")
		inst, err := mgr.Get(name)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}

		var req setPresenceReq
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Presence == "" {
			req.Presence = "available"
		}

		_ = inst.Provider.SendPresence("", req.Presence)
		writeJSON(w, 200, map[string]any{"ok": true, "presence": req.Presence})
	}
}

func handlePairInstance(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "instance")
		inst, err := mgr.Get(name)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}

		var req struct {
			Phone string `json:"phone"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Phone == "" {
			writeJSON(w, 400, map[string]string{"error": "Missing 'phone'"})
			return
		}

		code, err := inst.Provider.PairPhone(req.Phone)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, 200, map[string]any{
			"ok":   true,
			"code": code,
			"hint": "Enter this code in WhatsApp > Linked Devices > Link a Device",
		})
	}
}

func handleGetQR(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "instance")
		inst, err := mgr.Get(name)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}

		qr := inst.Provider.QRCode()
		if qr == "" {
			msg := "No QR code available"
			if inst.Provider.Status() == "connected" {
				msg = "Already authenticated"
			}
			writeJSON(w, 200, map[string]any{"status": inst.Provider.Status(), "message": msg})
			return
		}

		if r.URL.Query().Get("format") == "png" {
			png, err := qrcode.Encode(qr, qrcode.Medium, 300)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Write(png)
			return
		}

		png, err := qrcode.Encode(qr, qrcode.Medium, 300)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		dataURL := "data:image/png;base64," + b64encode(png)
		writeJSON(w, 200, map[string]any{"qr": dataURL, "status": "qr"})
	}
}

func handleFetchInstances(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, mgr.List())
	}
}
