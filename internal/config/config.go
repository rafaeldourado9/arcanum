// Package config carrega a configuracao do Arcanum API a partir de variaveis de ambiente.
// Todas as variaveis usam o prefixo GATEWAY_ e possuem valores padrao seguros para desenvolvimento.
package config

import (
	"os"
	"strconv"
)

// Config armazena todas as configuracoes do gateway.
// Cada campo mapeia para uma variavel de ambiente com prefixo GATEWAY_.
type Config struct {
	// Port define a porta HTTP do servidor (GATEWAY_PORT, padrao: 3100).
	Port int

	// WebhookForwardURL e a URL padrao para encaminhar mensagens recebidas via webhook.
	// Pode ser sobrescrita por instancia via API (GATEWAY_WEBHOOK_FORWARD_URL).
	// Aceita o marcador {instance}, trocado pelo nome da instancia.
	WebhookForwardURL string

	// WebhookFormat define o formato do webhook: "meta" (compativel com Meta Cloud API)
	// ou "simple" (formato simplificado para integracao direta com agent-core).
	// Padrao: "meta" para compatibilidade. Use "simple" para integracoes novas.
	// (GATEWAY_WEBHOOK_FORMAT)
	WebhookFormat string

	// WebhookSecret e o secret usado para assinar webhooks com HMAC-SHA256.
	// Se vazio, os webhooks sao enviados sem assinatura (GATEWAY_META_APP_SECRET).
	WebhookSecret string

	// DBPath e o diretorio onde os arquivos SQLite de sessao sao armazenados.
	// Cada instancia cria um arquivo .db separado (GATEWAY_DB_PATH, padrao: ./data).
	DBPath string

	// MinDelayMs e o delay minimo (ms) antes de enviar uma mensagem — parte do anti-ban.
	MinDelayMs int

	// MaxDelayMs e o delay maximo (ms) antes de enviar uma mensagem — parte do anti-ban.
	MaxDelayMs int

	// TypingDurationMs e a duracao minima (ms) do indicador "digitando..." antes de enviar.
	TypingDurationMs int

	// MsPerChar define quantos milissegundos o "digitando..." dura por caractere da mensagem.
	MsPerChar int

	// MaxTypingMs e a duracao maxima (ms) do indicador "digitando..." independente do tamanho.
	MaxTypingMs int

	// RateLimitPerMin define o numero maximo de mensagens enviadas por minuto por instancia.
	RateLimitPerMin int

	// APIKey e a chave exigida em toda rota da API (header X-Api-Key ou
	// Authorization: Bearer). Vazia: o gateway gera uma ao subir e mostra no log, para
	// nunca ficar aberto por esquecimento (GATEWAY_API_KEY).
	APIKey string

	// AuthDisabled desliga a autenticacao. So para desenvolvimento local, nunca em
	// servidor exposto (GATEWAY_AUTH_DISABLED=true).
	AuthDisabled bool

	// LogFormat: "text" (padrao, legivel) ou "json" (para coletores de log)
	// (GATEWAY_LOG_FORMAT).
	LogFormat string

	// LogLevel: "debug", "info" (padrao), "warn" ou "error" (GATEWAY_LOG_LEVEL).
	LogLevel string

	// MaxBodyMB e o tamanho maximo de uma requisicao, em MB (midia em base64 entra aqui)
	// (GATEWAY_MAX_BODY_MB, padrao: 64).
	MaxBodyMB int

	// DashboardEnabled serve o painel web em / (GATEWAY_DASHBOARD, padrao: true).
	DashboardEnabled bool

	// DashboardPassword e a senha do login do painel. Vazia: o login usa a propria
	// GATEWAY_API_KEY (GATEWAY_DASHBOARD_PASSWORD).
	DashboardPassword string
}

// Load carrega a configuracao a partir das variaveis de ambiente.
// Valores ausentes usam os padroes definidos.
func Load() *Config {
	return &Config{
		Port:              envInt("GATEWAY_PORT", 3100),
		WebhookForwardURL: envStr("GATEWAY_WEBHOOK_FORWARD_URL", ""),
		WebhookFormat:     envStr("GATEWAY_WEBHOOK_FORMAT", "simple"),
		WebhookSecret:     envStr("GATEWAY_META_APP_SECRET", ""),
		DBPath:            envStr("GATEWAY_DB_PATH", "./data"),

		MinDelayMs:       envInt("GATEWAY_MIN_DELAY_MS", 1500),
		MaxDelayMs:       envInt("GATEWAY_MAX_DELAY_MS", 4000),
		TypingDurationMs: envInt("GATEWAY_TYPING_DURATION_MS", 2000),
		MsPerChar:        envInt("GATEWAY_MS_PER_CHAR", 50),
		MaxTypingMs:      envInt("GATEWAY_MAX_TYPING_MS", 8000),
		RateLimitPerMin:  envInt("GATEWAY_RATE_LIMIT_PER_MIN", 15),

		APIKey:       envStr("GATEWAY_API_KEY", ""),
		AuthDisabled: envBool("GATEWAY_AUTH_DISABLED", false),
		LogFormat:    envStr("GATEWAY_LOG_FORMAT", "text"),
		LogLevel:     envStr("GATEWAY_LOG_LEVEL", "info"),
		MaxBodyMB:    envInt("GATEWAY_MAX_BODY_MB", 64),

		DashboardEnabled:  envBool("GATEWAY_DASHBOARD", true),
		DashboardPassword: envStr("GATEWAY_DASHBOARD_PASSWORD", ""),
	}
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
