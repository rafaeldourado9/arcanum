// Command gateway starts the Arcanum WhatsApp API HTTP server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rafaeldourado9/arcanum/internal/api"
	"github.com/rafaeldourado9/arcanum/internal/audit"
	"github.com/rafaeldourado9/arcanum/internal/config"
	"github.com/rafaeldourado9/arcanum/internal/instance"
	"github.com/rafaeldourado9/arcanum/internal/keys"
	"github.com/rafaeldourado9/arcanum/internal/logbuf"
)

func main() {
	if len(os.Args) > 1 {
		os.Exit(subcommand(os.Args[1]))
	}

	cfg := config.Load()
	logs := logbuf.New(1000)
	setupLogging(cfg, logs)

	// 0700: o banco de cada sessao e a "senha" do numero; ninguem alem do gateway le.
	if err := os.MkdirAll(cfg.DBPath, 0o700); err != nil {
		log.Fatalf("failed to create GATEWAY_DB_PATH %q: %v", cfg.DBPath, err)
	}

	switch {
	case cfg.AuthDisabled:
		slog.Warn("AUTHENTICATION DISABLED (GATEWAY_AUTH_DISABLED=true): anyone who reaches this port controls your WhatsApp sessions. Use only on localhost.")
	case cfg.APIKey == "":
		key, err := api.GenerateAPIKey()
		if err != nil {
			log.Fatalf("generate API key: %v", err)
		}
		cfg.APIKey = key
		// Mostrada uma vez, no boot: sem GATEWAY_API_KEY a chave muda a cada restart.
		slog.Warn("GATEWAY_API_KEY not set: generated a temporary key for this run. Set GATEWAY_API_KEY to keep it across restarts.", "api_key", key)
	case len(cfg.APIKey) < 32:
		slog.Warn("GATEWAY_API_KEY is short; use at least 32 random characters (e.g. `openssl rand -hex 32`)")
	}

	if cfg.DashboardEnabled && !cfg.AuthDisabled {
		if cfg.DashboardPassword != "" {
			slog.Info("dashboard enabled at / (login with GATEWAY_DASHBOARD_PASSWORD)")
		} else {
			slog.Info("dashboard enabled at / (login with the API key; set GATEWAY_DASHBOARD_PASSWORD for a separate password)")
		}
	}

	// Chaves de escopo e auditoria ficam em <dados>/.arcanum: o gerenciador le so arquivos soltos
	// da pasta de dados (um keys.json la viraria uma sessao chamada "keys").
	stateDir := filepath.Join(cfg.DBPath, ".arcanum")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		log.Fatalf("failed to create %s: %v", stateDir, err)
	}
	keyStore, err := keys.Open(filepath.Join(stateDir, "keys.json"))
	if err != nil {
		log.Fatalf("open API key store: %v", err)
	}
	auditLog, err := audit.Open(filepath.Join(stateDir, "audit.log"))
	if err != nil {
		log.Fatalf("open audit log: %v", err)
	}
	defer auditLog.Close()

	mgr := instance.NewManager(cfg)
	addr := ":" + strconv.Itoa(cfg.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           api.NewServerWith(mgr, cfg, api.Deps{Logs: logs, Keys: keyStore, Audit: auditLog}),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute, // upload de midia grande
		// Envio com anti-ban espera "digitando..." e o limite por minuto antes de responder.
		WriteTimeout:   5 * time.Minute,
		IdleTimeout:    2 * time.Minute,
		MaxHeaderBytes: 64 << 10,
	}

	go func() {
		slog.Info("arcanum gateway listening", "addr", addr, "db_path", cfg.DBPath, "version", api.Version, "auth", !cfg.AuthDisabled)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Desligamento limpo (docker stop, Ctrl+C): para de aceitar requisicoes, espera as que
	// estao em andamento e fecha cada sessao, para o SQLite nao ficar pela metade.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Warn("http shutdown", "error", err)
	}
	mgr.Shutdown()
	slog.Info("bye")
}

// setupLogging troca o log padrao por slog (texto ou JSON). O log.Printf antigo do resto do
// codigo passa pelo mesmo handler, entao tudo sai no mesmo formato.
func setupLogging(cfg *config.Config, logs *logbuf.Buffer) {
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	if strings.EqualFold(cfg.LogFormat, "json") {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	// Copia de cada linha para o painel (logs ao vivo), com segredos mascarados.
	slog.SetDefault(slog.New(logbuf.NewHandler(h, logs)))
}

// subcommand: `arcanum healthcheck` (usado pelo HEALTHCHECK da imagem, que nao tem shell nem
// wget) e `arcanum version`.
func subcommand(name string) int {
	switch name {
	case "version", "--version", "-v":
		fmt.Println("arcanum", api.Version)
		return 0
	case "healthcheck":
		port := 3100
		if v := os.Getenv("GATEWAY_PORT"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 65535 {
				fmt.Fprintln(os.Stderr, "unhealthy: invalid GATEWAY_PORT")
				return 1
			}
			port = n
		}
		client := &http.Client{Timeout: 4 * time.Second}
		// #nosec G704 -- destino fixo (127.0.0.1) e porta validada como numero acima
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err != nil {
			fmt.Fprintln(os.Stderr, "unhealthy:", err)
			return 1
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintln(os.Stderr, "unhealthy: HTTP", resp.StatusCode)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (available: version, healthcheck). Configure the gateway with GATEWAY_* environment variables.\n", name)
		return 2
	}
}
