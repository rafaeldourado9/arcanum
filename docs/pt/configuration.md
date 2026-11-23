# Configuração

O Arcanum é configurado só por variáveis de ambiente, todas com o prefixo `GATEWAY_`.

## Servidor e segurança

| Variável | Padrão | Descrição |
|---|---|---|
| `GATEWAY_PORT` | `3100` | Porta HTTP. |
| `GATEWAY_API_KEY` | *(gerada)* | Chave exigida em toda rota. Use 32+ caracteres aleatórios (`openssl rand -hex 32`). Vazia: uma chave temporária é gerada a cada início e mostrada no log. |
| `GATEWAY_AUTH_DISABLED` | `false` | Desliga a autenticação. Só para desenvolvimento local. |
| `GATEWAY_MAX_BODY_MB` | `64` | Maior pedido aceito, em MB (mídia em base64 conta aqui). |
| `GATEWAY_DB_PATH` | `./data` (`/data` no Docker) | Pasta com um arquivo SQLite (`<nome>.db`) e um de configuração (`<nome>.json`) por sessão. Criada com permissão `700`, arquivos `600`. |

## Painel

| Variável | Padrão | Descrição |
|---|---|---|
| `GATEWAY_DASHBOARD` | `true` | Serve o [painel web](dashboard.md) em `/`. `false` desliga. |
| `GATEWAY_DASHBOARD_PASSWORD` | *(vazia)* | Senha do login do painel. Vazia: o login usa a chave de API. Defina, longa e aleatória. |

## Webhooks

| Variável | Padrão | Descrição |
|---|---|---|
| `GATEWAY_WEBHOOK_FORWARD_URL` | *(vazia)* | URL de webhook padrão para todas as sessões. `{instance}` é trocado pelo nome da sessão. |
| `GATEWAY_WEBHOOK_FORMAT` | `simple` | `simple` (JSON plano) ou `meta` (envelope da Cloud API do WhatsApp). Veja [Webhooks](webhooks.md). |
| `GATEWAY_META_APP_SECRET` | *(vazia)* | Segredo do HMAC em `X-Hub-Signature-256`. Sem ele, as entregas não são assinadas. |

## Anti-ban

| Variável | Padrão | Descrição |
|---|---|---|
| `GATEWAY_RATE_LIMIT_PER_MIN` | `15` | Mensagens por minuto por sessão. |
| `GATEWAY_MIN_DELAY_MS` | `1500` | Menor pausa antes de digitar. |
| `GATEWAY_MAX_DELAY_MS` | `4000` | Maior pausa antes de digitar. |
| `GATEWAY_TYPING_DURATION_MS` | `2000` | Menor tempo digitando. |
| `GATEWAY_MS_PER_CHAR` | `50` | Tempo digitando por caractere. |
| `GATEWAY_MAX_TYPING_MS` | `8000` | Maior tempo digitando. |

## Logs

| Variável | Padrão | Descrição |
|---|---|---|
| `GATEWAY_LOG_FORMAT` | `text` | `text` para pessoas, `json` para coletores de log (Loki, CloudWatch, Datadog...). |
| `GATEWAY_LOG_LEVEL` | `info` | `debug`, `info`, `warn` ou `error`. |

No nível `info`, os logs nunca trazem texto de mensagem, telefone ou nome de contato. Em
`debug`, o remetente de cada entrega de webhook aparece, para ajudar a investigar problemas.

## Exemplo de `.env`

```bash title=".env"
GATEWAY_API_KEY=3f9c...64-caracteres-hex...
GATEWAY_WEBHOOK_FORWARD_URL=https://seu-app.exemplo.com/webhooks/whatsapp/{instance}
GATEWAY_WEBHOOK_FORMAT=simple
GATEWAY_META_APP_SECRET=outro-valor-longo-e-aleatorio
GATEWAY_LOG_FORMAT=json
```

!!! danger "Nunca suba o `.env` para o Git"
    Ele guarda as chaves de todos os números conectados. O `.gitignore` do repositório já o
    exclui.
