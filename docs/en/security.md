# Security

Arcanum controls real WhatsApp accounts. Whoever reaches the API, or copies a session
file, can read and send messages as that number. Treat it like a database.

## Checklist before going live

- [x] `GATEWAY_API_KEY` set to 32+ random characters.
- [x] `GATEWAY_AUTH_DISABLED` **not** set.
- [x] `GATEWAY_DASHBOARD_PASSWORD` set (or `GATEWAY_DASHBOARD=false` if you do not use the dashboard).
- [x] Port 3100 not open to the internet: only a reverse proxy with HTTPS reaches it.
- [x] `GATEWAY_META_APP_SECRET` set, and your webhook verifies `X-Hub-Signature-256`.
- [x] Data volume backed up **encrypted**, with restricted access.
- [x] The API key lives in your backend only, never in front-end code.
- [x] Updates applied regularly (new releases fix security issues and keep up with WhatsApp).

## What Arcanum does for you

| Area | Protection |
|---|---|
| Authentication | API key on every route except `/health`, `/docs` and the dashboard sign-in; constant-time comparison; random key generated when none is set, so it is never open by accident. |
| Input | Session names restricted to `A-Z a-z 0-9 - _` (they become file names: no path traversal). Request size limit and HTTP timeouts. |
| Webhooks (SSRF) | Only `http`/`https`; cloud metadata and link-local addresses refused, checked again at connection time (DNS rebinding); redirects not followed; no credentials in the URL. |
| Media | Base64 only: Arcanum never downloads URLs on your behalf. |
| Privacy | Message text, phone numbers and contact names are not logged at `info` level. |
| HTTP headers | `nosniff`, `X-Frame-Options: DENY`, strict `Content-Security-Policy`, `Cache-Control: no-store`, `Referrer-Policy: no-referrer`. |
| Dashboard | Separate password, `HttpOnly` + `SameSite=Strict` session cookie, anti-CSRF header on every request, sign-in lockout after 10 failures, strict CSP, sessions stored as hashes in memory. See [Dashboard](dashboard.md). |
| Container | Distroless image (no shell, no package manager), unprivileged user (UID 65532), works with a read-only root filesystem and no capabilities; reproducible build with `go mod verify`; signed with Cosign, with SBOM and provenance. |
| Files | Session files and settings created with permission `600`, data folder `700`. |
| systemd | The provided unit runs sandboxed: no new privileges, read-only system, private `/tmp`, no capabilities, filtered system calls. |
| Shutdown | Graceful on `SIGTERM`: session files closed cleanly. |
| Supply chain | Every change runs the race detector, `go vet`, `staticcheck`, `govulncheck`, `gosec` and a Trivy image scan. Dependabot updates dependencies weekly. |

## Hosting for several customers (scoped keys)

You can run Arcanum on a server and give each integrator (AI agent, third-party system,
another company) access to **only their own sessions**, without handing over the master key.
The integrator never needs to know who your other customers are: they only see their sessions.

```bash
# 1. You (admin, master key) create the session and its key
curl -X POST http://localhost:3100/api/instance/create -H "X-Api-Key: $GATEWAY_API_KEY" \
  -d '{"instanceName":"acme-shop","webhook":"https://agent.example.com/whatsapp"}'

curl -X POST http://localhost:3100/api/keys -H "X-Api-Key: $GATEWAY_API_KEY" \
  -d '{"name":"acme-agent","instances":["acme-shop"],"readOnly":false}'
# response: {"secret":"arc_..."}  <- shown ONCE; hand it to the integrator

# 2. A signing secret for this session only (the integrator verifies X-Hub-Signature-256 with it)
curl -X POST http://localhost:3100/api/events/webhook/acme-shop/secret -H "X-Api-Key: $GATEWAY_API_KEY"
```

| Key | What it reaches |
|---|---|
| Master (`GATEWAY_API_KEY`) and dashboard | Everything. |
| Scoped (`arc_...`) | Only the listed sessions: send, configure the webhook and read their state. Another customer's session answers `404`, same as one that does not exist. It cannot create or delete sessions, manage keys, or read metrics, logs or audit. |
| Scoped `readOnly` | Queries only: state, settings, deliveries, lookups. No sending, no configuration changes, no starting a connection. |

- **Revoking** is immediate: `DELETE /api/keys/{id}`. The key stays listed as revoked.
- Only the key **hash** is stored (`600` file at `<data>/.arcanum/keys.json`).
- **Per-session webhook secret:** each session can have its own; the API never shows it again
  (only `hasSecret`). Without one, the global `GATEWAY_META_APP_SECRET` applies.
- **Lockout:** 20 wrong keys from one IP within 15 minutes lock that IP out (`429`) for 15
  minutes. Requests with no key do not count (the dashboard asks before login). Behind a proxy
  every request comes from the proxy IP, so the limit becomes global.
- **Audit:** `GET /api/audit` (admin only) lists who did what: login, create/delete/reconnect
  a session, change settings or webhook, create/revoke a key and denied access. Stored at
  `<data>/.arcanum/audit.log` (`600`, rotates at 5 MB). It never contains keys, secrets, phone
  numbers or message text.

## Reporting a vulnerability

Please do not open a public issue. Use **Security > Report a vulnerability** on the
[GitHub repository](https://github.com/rafaeldourado9/arcanum/security). You get an answer
within 72 hours. Details in
[SECURITY.md](https://github.com/rafaeldourado9/arcanum/blob/main/SECURITY.md).

## Privacy and compliance

Messages pass through Arcanum in memory on their way to your webhook; Arcanum does not keep
a message history. Session files hold encryption keys and contact identifiers. If you serve
people in Brazil (LGPD) or Europe (GDPR), you are the data controller: document the purpose,
secure the server and the webhook endpoint, and honour opt-out requests.
