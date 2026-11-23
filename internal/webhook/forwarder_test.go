package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rafaeldourado9/arcanum/internal/provider"
)

func metaMessage(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()
	entry := payload["entry"].([]map[string]any)[0]
	change := entry["changes"].([]map[string]any)[0]
	value := change["value"].(map[string]any)
	return value["messages"].([]map[string]any)[0]
}

func TestMetaPayloadCarriesLocationLikeTheMetaAPI(t *testing.T) {
	f := NewForwarder("", "meta", "")
	msg := provider.IncomingMessage{
		MessageID: "m1",
		From:      "5583999990000@s.whatsapp.net",
		Type:      "location",
		Location:  &provider.Location{Latitude: -7.1195, Longitude: -34.845, Address: "Av. Epitacio"},
	}
	message := metaMessage(t, f.buildMetaPayload(msg))
	if message["type"] != "location" {
		t.Fatalf("type = %v", message["type"])
	}
	loc := message["location"].(map[string]any)
	if loc["latitude"] != -7.1195 || loc["longitude"] != -34.845 || loc["address"] != "Av. Epitacio" {
		t.Fatalf("location = %v", loc)
	}
	if _, ok := loc["name"]; ok {
		t.Fatalf("name vazio nao deveria ir no payload: %v", loc)
	}
}

func TestMetaPayloadWithoutLocationHasNoLocationKey(t *testing.T) {
	f := NewForwarder("", "meta", "")
	message := metaMessage(t, f.buildMetaPayload(provider.IncomingMessage{MessageID: "m2", Type: "text", Text: "oi"}))
	if _, ok := message["location"]; ok {
		t.Fatalf("mensagem de texto nao carrega location: %v", message)
	}
}

func TestReportSendFailureIsSignedLikeTheMessages(t *testing.T) {
	var got map[string]any
	var signature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		signature = r.Header.Get("X-Hub-Signature-256")
		_ = json.Unmarshal(body, &got)
		mac := hmac.New(sha256.New, []byte("segredo"))
		mac.Write(body)
		if signature != "sha256="+hex.EncodeToString(mac.Sum(nil)) {
			t.Errorf("assinatura invalida: %s", signature)
		}
	}))
	defer server.Close()

	NewForwarder(server.URL, "meta", "segredo").ReportSendFailure("5583999990000", "oi")

	if got["event"] != "send_failed" || got["to"] != "5583999990000" || got["text"] != "oi" {
		t.Fatalf("payload = %v", got)
	}
}

func TestReportTypingSendsComposingAndPaused(t *testing.T) {
	var got []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		got = append(got, payload)
	}))
	defer server.Close()

	fwd := NewForwarder(server.URL, "meta", "segredo")
	fwd.ReportTyping("5567999990000@s.whatsapp.net", true)
	fwd.ReportTyping("5567999990000@s.whatsapp.net", false)

	if len(got) != 2 || got[0]["event"] != "typing" || got[0]["state"] != "composing" ||
		got[1]["state"] != "paused" || got[0]["from"] != "5567999990000@s.whatsapp.net" {
		t.Fatalf("payloads = %v", got)
	}
}
