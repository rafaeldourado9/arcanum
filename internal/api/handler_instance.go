package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rafaeldourado9/arcanum/internal/instance"
	"github.com/rafaeldourado9/arcanum/internal/provider"
	qrcode "github.com/skip2/go-qrcode"
)

// QR generation runs in a background goroutine after Connect() returns (see
// WhatsmeowProvider.connectWithQR), so the code is rarely ready immediately —
// poll briefly instead of forcing the caller to make a second request.
const (
	qrPollTimeout  = 5 * time.Second
	qrPollInterval = 150 * time.Millisecond
)

func waitForQR(inst *instance.Instance) string {
	deadline := time.Now().Add(qrPollTimeout)
	for {
		qr := inst.Provider.QRCode()
		if qr != "" || inst.Provider.Status() == provider.StatusConnected {
			return qr
		}
		if time.Now().After(deadline) {
			return inst.Provider.QRCode()
		}
		time.Sleep(qrPollInterval)
	}
}

func writeQRResponse(
	w http.ResponseWriter, r *http.Request, instanceName string, status provider.ConnectionStatus, qr string,
) {
	if qr == "" {
		writeJSON(w, 200, map[string]any{"instanceName": instanceName, "status": status, "qr": ""})
		return
	}

	png, err := qrcode.Encode(qr, qrcode.Medium, 300)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	if r.URL.Query().Get("format") == "png" {
		w.Header().Set("Content-Type", "image/png")
		// #nosec G705 -- bytes PNG gerados pelo proprio gateway, Content-Type fixo e nosniff global
		_, _ = w.Write(png)
		return
	}

	writeJSON(w, 200, map[string]any{
		"instanceName": instanceName,
		"status":       status,
		"qr":           "data:image/png;base64," + b64encode(png),
	})
}

// phoneOf returns the paired number when the provider knows it (whatsmeow does).
func phoneOf(p provider.WhatsAppProvider) string {
	if numbered, ok := p.(interface{ PhoneNumber() string }); ok {
		return numbered.PhoneNumber()
	}
	return ""
}

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
			status := http.StatusBadRequest
			if errors.Is(err, instance.ErrExists) {
				status = http.StatusConflict
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}

		info, _ := mgr.Info(inst.Name)
		writeJSON(w, 201, info)
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

		qr := waitForQR(inst)
		writeQRResponse(w, r, inst.Name, inst.Provider.Status(), qr)
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

		if inst.Provider.Status() == provider.StatusConnected {
			writeJSON(w, 200, map[string]any{
				"instanceName": inst.Name, "status": provider.StatusConnected, "qr": "", "number": phoneOf(inst.Provider),
			})
			return
		}
		writeQRResponse(w, r, inst.Name, inst.Provider.Status(), inst.Provider.QRCode())
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
			if inst.Provider.Status() == provider.StatusConnected {
				msg = "Already authenticated"
			}
			writeJSON(w, 200, map[string]any{"status": inst.Provider.Status(), "message": msg})
			return
		}

		writeQRResponse(w, r, inst.Name, inst.Provider.Status(), qr)
	}
}

func handleFetchInstances(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, mgr.List())
	}
}
