# Configuration

Arcanum is configured only through environment variables, all prefixed with `GATEWAY_`.

## Server and security

| Variable | Default | Description |
|---|---|---|
| `GATEWAY_PORT` | `3100` | HTTP port. |
| `GATEWAY_API_KEY` | *(generated)* | Key required on every route. Use 32+ random characters (`openssl rand -hex 32`). If empty, a temporary key is generated on each start and printed in the log. |
| `GATEWAY_AUTH_DISABLED` | `false` | Turns authentication off. Local development only. |
| `GATEWAY_MAX_BODY_MB` | `64` | Largest request accepted, in MB (base64 media counts here). |
| `GATEWAY_DB_PATH` | `./data` (`/data` in Docker) | Folder with one SQLite file (`<name>.db`) and one settings file (`<name>.json`) per session. Created with permission `700`, files `600`. |

## Dashboard

| Variable | Default | Description |
|---|---|---|
| `GATEWAY_DASHBOARD` | `true` | Serves the [web dashboard](dashboard.md) at `/`. `false` turns it off. |
| `GATEWAY_DASHBOARD_PASSWORD` | *(empty)* | Dashboard sign-in password. Empty: sign in with the API key. Set it, long and random. |

## Webhooks

| Variable | Default | Description |
|---|---|---|
| `GATEWAY_WEBHOOK_FORWARD_URL` | *(empty)* | Default webhook URL for every session. `{instance}` is replaced by the session name. |
| `GATEWAY_WEBHOOK_FORMAT` | `simple` | `simple` (flat JSON) or `meta` (WhatsApp Cloud API envelope). See [Webhooks](webhooks.md). |
| `GATEWAY_META_APP_SECRET` | *(empty)* | Secret for the `X-Hub-Signature-256` HMAC. Without it, deliveries are not signed. |

## Anti-ban

| Variable | Default | Description |
|---|---|---|
| `GATEWAY_RATE_LIMIT_PER_MIN` | `15` | Messages per minute per session. |
| `GATEWAY_MIN_DELAY_MS` | `1500` | Shortest pause before typing. |
| `GATEWAY_MAX_DELAY_MS` | `4000` | Longest pause before typing. |
| `GATEWAY_TYPING_DURATION_MS` | `2000` | Shortest typing time. |
| `GATEWAY_MS_PER_CHAR` | `50` | Typing time per character. |
| `GATEWAY_MAX_TYPING_MS` | `8000` | Longest typing time. |

## Logging

| Variable | Default | Description |
|---|---|---|
| `GATEWAY_LOG_FORMAT` | `text` | `text` for people, `json` for log collectors (Loki, CloudWatch, Datadog...). |
| `GATEWAY_LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error`. |

Logs never contain message text, phone numbers or contact names at `info` level. At
`debug`, the sender of each webhook delivery is logged to help troubleshooting.

## Example `.env`

```bash title=".env"
GATEWAY_API_KEY=3f9c...64-hex-characters...
GATEWAY_WEBHOOK_FORWARD_URL=https://your-app.example.com/webhooks/whatsapp/{instance}
GATEWAY_WEBHOOK_FORMAT=simple
GATEWAY_META_APP_SECRET=another-long-random-value
GATEWAY_LOG_FORMAT=json
```

!!! danger "Never commit `.env`"
    It holds the keys to every connected number. The repository `.gitignore` already
    excludes it.
