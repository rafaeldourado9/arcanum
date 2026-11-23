package api

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rafaeldourado9/arcanum/internal/metrics"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

var sendErrCodeRe = regexp.MustCompile(`error (\d{3})`)

// permanentSendCodes are WhatsApp server error codes that will never succeed on
// retry: 401 unauthorized, 403 forbidden, 404 not-found, 463 recipient
// unreachable / not a valid account.
var permanentSendCodes = map[string]bool{"401": true, "403": true, "404": true, "463": true}

// isPermanentSendError reports whether a send failure is permanent and must not
// be retried. Retrying a permanent error (e.g. 463 when the recipient isn't a
// real WhatsApp account) just burns the remaining attempts and floods the log
// with identical failures.
func isPermanentSendError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "not on whatsapp") {
		return true
	}
	if m := sendErrCodeRe.FindStringSubmatch(msg); m != nil && permanentSendCodes[m[1]] {
		return true
	}
	return false
}

// sendWithRetry runs a send operation with a few short retries. Sends happen
// in a background goroutine (see handleSendText) with no caller waiting on
// the result, so a transient failure (e.g. a WhatsApp usync query timeout)
// would otherwise silently drop the message — this gives it a few chances to
// go through before giving up. The last error comes back so the caller can tell
// the backend the message was not delivered.
func sendWithRetry(to string, send func() error) error {
	const maxAttempts = 3
	backoff := 500 * time.Millisecond

	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err = send(); err == nil {
			metrics.MessagesSent.Add(1)
			return nil
		}
		log.Printf("[send] attempt %d/%d to %s failed: %v", attempt, maxAttempts, to, err)
		if isPermanentSendError(err) {
			log.Printf("[send] permanent failure to %s — not retrying: %v", to, err)
			metrics.MessagesSendError.Add(1)
			return err
		}
		if attempt < maxAttempts {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	log.Printf("[send] giving up on %s after %d attempts: %v", to, maxAttempts, err)
	metrics.MessagesSendError.Add(1)
	return err
}
