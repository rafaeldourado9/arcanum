package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rafaeldourado9/arcanum/internal/instance"
	"github.com/rafaeldourado9/arcanum/internal/logbuf"
	"github.com/rafaeldourado9/arcanum/internal/metrics"
)

var startedAt = time.Now()

// handleStats: visao geral do painel (contadores, sessoes por status, tempo no ar).
func handleStats(mgr *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"version":       Version,
			"uptimeSeconds": int64(time.Since(startedAt).Seconds()),
			"sessions":      sessionsByStatus(mgr),
			"counters":      metrics.Snapshot(),
		})
	}
}

// handleLogs: ultimas linhas (?limit=200&level=warn), sem dado pessoal e com segredos
// mascarados (ver logbuf).
func handleLogs(buf *logbuf.Buffer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if buf == nil {
			writeJSON(w, 200, map[string]any{"entries": []any{}})
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 1000 {
			limit = 200
		}
		writeJSON(w, 200, map[string]any{"entries": buf.Recent(limit, logbuf.ParseLevel(r.URL.Query().Get("level")))})
	}
}

// handleLogStream: logs ao vivo por Server-Sent Events. Tira o prazo de escrita do servidor
// (5 min) desta conexao, que fica aberta, e manda um comentario a cada 25 s para proxies nao
// derrubarem a conexao parada.
func handleLogStream(buf *logbuf.Buffer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if buf == nil {
			http.Error(w, "logs unavailable", http.StatusNotFound)
			return
		}
		rc := http.NewResponseController(w)
		_ = rc.SetWriteDeadline(time.Time{})

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Accel-Buffering", "no") // nginx: nao segurar o stream
		w.WriteHeader(http.StatusOK)
		_ = rc.Flush()

		ch, cancel := buf.Subscribe()
		defer cancel()
		heartbeat := time.NewTicker(25 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-heartbeat.C:
				if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
					return
				}
			case e := <-ch:
				b, _ := json.Marshal(e)
				if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
					return
				}
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}
