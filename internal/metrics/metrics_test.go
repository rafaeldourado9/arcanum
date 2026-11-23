package metrics

import (
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	MessagesReceived.Add(3)
	WebhookFailed.Add(1)
	HTTPRequest(200)
	HTTPRequest(200)
	HTTPRequest(401)

	var b strings.Builder
	Write(&b, map[string]int{"connected": 2, "qr": 1})
	out := b.String()

	for _, want := range []string{
		`arcanum_sessions{status="connected"} 2`,
		`arcanum_sessions{status="qr"} 1`,
		"arcanum_messages_received_total 3",
		"arcanum_webhook_failed_total 1",
		`arcanum_http_requests_total{code="200"} 2`,
		`arcanum_http_requests_total{code="401"} 1`,
		"# TYPE arcanum_messages_sent_total counter",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
