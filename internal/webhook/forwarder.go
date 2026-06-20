// Package webhook encaminha mensagens recebidas do WhatsApp para um backend externo via HTTP POST.
// O payload segue o formato da Meta WhatsApp Cloud API, permitindo que qualquer backend
// que ja processe webhooks do Meta funcione sem alteracao.
// Opcionalmente assina o payload com HMAC-SHA256 (header X-Hub-Signature-256).
package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rafaeldourado9/arcanum/internal/media"
	"github.com/rafaeldourado9/arcanum/internal/provider"
)

type Forwarder struct {
	url    string
	secret string
	client *http.Client
}

func NewForwarder(url, secret string) *Forwarder {
	return &Forwarder{
		url:    url,
		secret: secret,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (f *Forwarder) Forward(msg provider.IncomingMessage) {
	messagePayload := map[string]any{
		"from":      msg.From,
		"id":        msg.MessageID,
		"timestamp": fmt.Sprintf("%d", msg.Timestamp),
		"type":      msg.Type,
	}

	if msg.Text != "" {
		messagePayload["text"] = map[string]string{"body": msg.Text}
	}

	if msg.Media != nil {
		mediaPayload := map[string]any{
			"mimetype": msg.Media.MimeType,
			"data":     media.ToBase64(msg.Media.Data),
		}
		if msg.Media.Filename != "" {
			mediaPayload["filename"] = msg.Media.Filename
		}
		if msg.Media.Caption != "" {
			mediaPayload["caption"] = msg.Media.Caption
		}
		messagePayload["media"] = mediaPayload
	}

	contacts := []map[string]any{{"wa_id": msg.From}}
	if msg.PushName != "" {
		contacts[0]["profile"] = map[string]string{"name": msg.PushName}
	}

	payload := map[string]any{
		"object": "whatsapp_business_account",
		"entry": []map[string]any{
			{
				"id": "gateway",
				"changes": []map[string]any{
					{
						"value": map[string]any{
							"messaging_product": "whatsapp",
							"metadata": map[string]string{
								"display_phone_number": "gateway",
								"phone_number_id":      "gateway",
							},
							"contacts": contacts,
							"messages": []map[string]any{messagePayload},
						},
						"field": "messages",
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[webhook] marshal error: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, f.url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[webhook] request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	if f.secret != "" {
		mac := hmac.New(sha256.New, []byte(f.secret))
		mac.Write(body)
		sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Hub-Signature-256", sig)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		log.Printf("[webhook] forward failed: %v", err)
		return
	}
	resp.Body.Close()

	log.Printf("[webhook] forwarded %s from %s -> HTTP %d", msg.Type, msg.From, resp.StatusCode)
}
