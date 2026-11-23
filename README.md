<p align="center">
  <img src="web/public/logo.svg" width="96" alt="Arcanum">
</p>

<h1 align="center">Arcanum</h1>

<p align="center">
  <strong>Open-source, self-hosted WhatsApp gateway in Go.</strong><br>
  Multi-session, built-in anti-ban, signed webhooks and a web dashboard. One static binary, no browser, no Node.
</p>

<p align="center">
  <a href="https://github.com/rafaeldourado9/arcanum/actions/workflows/ci.yml"><img src="https://github.com/rafaeldourado9/arcanum/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/rafaeldourado9/arcanum/releases"><img src="https://img.shields.io/github/v/release/rafaeldourado9/arcanum?sort=semver" alt="Release"></a>
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go 1.26">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-black.svg" alt="MIT"></a>
</p>

<p align="center">
  <a href="README.pt-BR.md"><b>Português (BR)</b></a> ·
  <a href="https://arcanum.rafaeldourado.tech"><b>Documentation</b></a> ·
  <a href="https://arcanum.rafaeldourado.tech/getting-started/">Getting started</a> ·
  <a href="https://arcanum.rafaeldourado.tech/webhooks/">Webhooks</a> ·
  <a href="SECURITY.md">Security</a>
</p>

---

<p align="center">
  <img src="docs/assets/screenshots/overview.jpg" alt="Arcanum dashboard: overview" width="880">
</p>

<table>
  <tr>
    <td><img src="docs/assets/screenshots/connect-qr.jpg" alt="Connect a number with the QR code"></td>
    <td><img src="docs/assets/screenshots/session.jpg" alt="Session page with webhook test and deliveries"></td>
  </tr>
  <tr>
    <td><img src="docs/assets/screenshots/logs.jpg" alt="Live logs"></td>
    <td><img src="docs/assets/screenshots/overview-dark.jpg" alt="Dark theme"></td>
  </tr>
</table>

## Why Arcanum

- **Light**: a single Go binary on [whatsmeow](https://github.com/tulir/whatsmeow). Starts in
  milliseconds, small memory footprint, runs on a Raspberry Pi.
- **Anti-ban built in**: typing indicator sized to the message, human-like delay, per-minute
  limit and one message at a time per chat, on every send.
- **Many numbers, one API**: each session has its own webhook, settings and rate limit, and
  survives restarts.
- **Webhooks you can trust**: HMAC-SHA256 signatures, Meta Cloud API compatible format or a
  simple flat one, and a `fromMe` flag for human takeover.
- **Dashboard included**: manage sessions, scan QR codes, test webhooks and watch live logs
  in the browser, light and dark, English and Portuguese.
- **Secure by default**: API key on every route, password-protected dashboard, SSRF-safe
  webhooks, no personal data in logs, distroless non-root image, vulnerability scans in CI.
- **Installs anywhere**: `npx`, Docker, a one-line script, systemd, Kubernetes, or a single
  binary for Linux, macOS and Windows.

## Install

| | Command |
|---|---|
| **npx** (Node.js 18+) | `npx arcanum-gateway` |
| **New project** (Compose + generated keys) | `npm create arcanum` |
| **Docker** | `docker run -p 3100:3100 -v arcanum-data:/data ghcr.io/rafaeldourado9/arcanum` |
| **Linux / macOS script** | `curl -fsSL https://arcanum.rafaeldourado.tech/install.sh \| sh` |
| **systemd service** | `curl -fsSL https://arcanum.rafaeldourado.tech/install.sh \| sh -s -- --systemd` |
| **Binary** (Linux, macOS, Windows) | [Releases](https://github.com/rafaeldourado9/arcanum/releases) |

Then open **http://localhost:3100**. Details: [Installation](https://arcanum.rafaeldourado.tech/installation/).

## Quick start

```bash
export ARCANUM_KEY="$(openssl rand -hex 32)"

docker run -d --name arcanum -p 3100:3100 \
  -e GATEWAY_API_KEY="$ARCANUM_KEY" \
  -e GATEWAY_DASHBOARD_PASSWORD="choose-a-long-password" \
  -v arcanum-data:/data \
  ghcr.io/rafaeldourado9/arcanum:latest

# open http://localhost:3100 for the dashboard, or use the API:
# create a session and get the QR code
curl -X POST localhost:3100/api/instance/create \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"instanceName": "sales"}'
curl -H "X-Api-Key: $ARCANUM_KEY" "localhost:3100/api/instance/connect/sales?format=png" -o qr.png

# after scanning qr.png in WhatsApp > Linked devices:
curl -X POST localhost:3100/api/message/sendText/sales \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"number": "5511999999999", "text": "Hello from Arcanum!"}'
```

Open **http://localhost:3100** for the dashboard, or **/docs** for the interactive API.

Have Node.js? **`npx arcanum-gateway`** runs it with no Docker and no install, and
**`npm create arcanum`** scaffolds a Compose project with generated keys.

No Docker or Node? `curl -fsSL https://arcanum.rafaeldourado.tech/install.sh | sh` (add
`-s -- --systemd` to install it as a service), or grab a binary for Linux, macOS or Windows
from [Releases](https://github.com/rafaeldourado9/arcanum/releases). All options:
[Installation](https://arcanum.rafaeldourado.tech/installation/).

**Learn by doing:** four short [tutorials](https://arcanum.rafaeldourado.tech/tutorials/)
([em português](https://arcanum.rafaeldourado.tech/pt/tutorials/)) build a real bot from the
ready-made project in [`examples/faq-bot`](examples/faq-bot): automatic answers, human
takeover, notifications from your system and moving servers.

The full guide (sessions, every message type, webhook payloads with signature
verification in Python, Node.js, Go and PHP, production setup with HTTPS, monitoring and
troubleshooting) is at **[arcanum.rafaeldourado.tech](https://arcanum.rafaeldourado.tech)**,
in English and Portuguese. The same pages live in [`docs/`](docs/) in this repository.

## Features

| | |
|---|---|
| Sessions | QR code (JSON or PNG) or pairing code, auto-reconnect, persisted across restarts |
| Messages | Text, image, audio, video, document, location, contact, reaction, poll |
| Chat | Read receipts, number lookup, profile name, about and photo |
| Webhooks | `simple` or `meta` format, HMAC signature, typing and send-failure events, `fromMe` |
| Dashboard | Sessions and QR codes, webhook test and delivery history, export/import, live logs |
| Operations | `/health`, Prometheus `/metrics`, JSON logs, graceful shutdown, health check |
| Security | API key, SSRF protection, request limits, security headers, non-root image |

On the roadmap: per-event webhook filtering and retries, message store
(`findChats`, `findMessages`), groups, and official SDKs for Python, Node.js and Go.

## Configuration

Everything is set through environment variables. The essentials:

| Variable | Default | |
|---|---|---|
| `GATEWAY_API_KEY` | generated | Required key. Use `openssl rand -hex 32`. |
| `GATEWAY_WEBHOOK_FORWARD_URL` | | Webhook for every session (`{instance}` is replaced). |
| `GATEWAY_META_APP_SECRET` | | HMAC secret for webhook signatures. |
| `GATEWAY_WEBHOOK_FORMAT` | `simple` | `simple` or `meta`. |
| `GATEWAY_DASHBOARD_PASSWORD` | | Dashboard password (without it, sign in with the API key). |

See [`.env.example`](.env.example) and the
[configuration reference](https://arcanum.rafaeldourado.tech/configuration/).

## Development

```bash
go test ./...                         # Go 1.26+, no C compiler needed (pure-Go SQLite)
(cd web && npm ci && npm run build)   # dashboard, embedded into the binary
go run ./cmd/gateway
```

No Go installed? Run the tests in Docker:

```bash
docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine go test ./...
```

Docs preview: `pip install -r requirements-docs.txt && mkdocs serve`.

Contributions are welcome: see [CONTRIBUTING.md](CONTRIBUTING.md). Report security issues
privately as described in [SECURITY.md](SECURITY.md).

## Disclaimer

Arcanum is not affiliated with, endorsed by or connected to WhatsApp or Meta. It uses the
same protocol as WhatsApp Web. Automated messaging can get numbers banned: follow
[WhatsApp's terms](https://www.whatsapp.com/legal/terms-of-service) and only message
people who expect to hear from you.

## License

[MIT](LICENSE)
