// Package metrics guarda contadores do gateway e os expoe no formato texto do Prometheus
// em GET /metrics. Sem dependencia externa: so contadores atomicos.
package metrics

import (
	"fmt"
	"io"
	"sort"
	"sync"
	"sync/atomic"
)

var (
	MessagesReceived  atomic.Int64 // mensagens recebidas do WhatsApp e entregues ao webhook
	WebhookDelivered  atomic.Int64 // POSTs de webhook com resposta 2xx
	WebhookFailed     atomic.Int64 // POSTs de webhook com erro de rede ou resposta nao-2xx
	MessagesSent      atomic.Int64 // envios pela API concluidos
	MessagesSendError atomic.Int64 // envios pela API que falharam

	httpMu       sync.Mutex
	httpRequests = map[int]int64{} // status HTTP -> quantidade
)

// HTTPRequest conta uma resposta da API pelo codigo de status.
func HTTPRequest(status int) {
	httpMu.Lock()
	httpRequests[status]++
	httpMu.Unlock()
}

// Write escreve todas as metricas. sessions vem do gerenciador: status -> quantidade.
func Write(w io.Writer, sessions map[string]int) {
	gauge(w, "arcanum_sessions", "Sessions by connection status.")
	keys := make([]string, 0, len(sessions))
	for k := range sessions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "arcanum_sessions{status=%q} %d\n", k, sessions[k])
	}

	counter(w, "arcanum_messages_received_total", "Incoming WhatsApp messages forwarded to the webhook.", MessagesReceived.Load())
	counter(w, "arcanum_messages_sent_total", "Messages sent through the API.", MessagesSent.Load())
	counter(w, "arcanum_messages_send_errors_total", "Messages that failed to send through the API.", MessagesSendError.Load())
	counter(w, "arcanum_webhook_delivered_total", "Webhook deliveries answered with 2xx.", WebhookDelivered.Load())
	counter(w, "arcanum_webhook_failed_total", "Webhook deliveries that failed (network error or non-2xx).", WebhookFailed.Load())

	fmt.Fprintln(w, "# HELP arcanum_http_requests_total API responses by status code.")
	fmt.Fprintln(w, "# TYPE arcanum_http_requests_total counter")
	httpMu.Lock()
	codes := make([]int, 0, len(httpRequests))
	for c := range httpRequests {
		codes = append(codes, c)
	}
	sort.Ints(codes)
	for _, c := range codes {
		fmt.Fprintf(w, "arcanum_http_requests_total{code=\"%d\"} %d\n", c, httpRequests[c])
	}
	httpMu.Unlock()
}

// Snapshot devolve os contadores para o painel (JSON).
func Snapshot() map[string]int64 {
	return map[string]int64{
		"messagesReceived":  MessagesReceived.Load(),
		"messagesSent":      MessagesSent.Load(),
		"messagesSendError": MessagesSendError.Load(),
		"webhookDelivered":  WebhookDelivered.Load(),
		"webhookFailed":     WebhookFailed.Load(),
	}
}

func gauge(w io.Writer, name, help string) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n", name, help, name)
}

func counter(w io.Writer, name, help string, v int64) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n%s %d\n", name, help, name, name, v)
}
