# Security Policy

## Reporting a vulnerability

Please **do not open a public issue** for security problems.

Report privately through GitHub: **Security → Report a vulnerability** on this repository
(private vulnerability reporting). Include what you found, how to reproduce it and the
impact you expect. You will get an answer within 72 hours, and a fix or mitigation plan
within 14 days for confirmed issues. We credit reporters in the release notes unless you
prefer to stay anonymous.

## Supported versions

Only the latest release receives security fixes.

## Hardening checklist for operators

Arcanum controls real WhatsApp accounts. Treat it like a database:

- **Always set `GATEWAY_API_KEY`** to a long random value (`openssl rand -hex 32`). Without
  it, Arcanum generates a temporary key on every start and prints it in the log.
- **Never set `GATEWAY_AUTH_DISABLED=true`** outside your own machine.
- **Do not expose port 3100 directly to the internet.** Put it behind a reverse proxy with
  TLS (Caddy, Nginx, Traefik) or keep it on a private network.
- **Protect the data volume** (`GATEWAY_DB_PATH`). Each `<instance>.db` file is a logged-in
  WhatsApp session: whoever copies it can impersonate that number. Back it up encrypted.
- **Sign your webhooks**: set `GATEWAY_META_APP_SECRET` and verify the
  `X-Hub-Signature-256` header (HMAC-SHA256 of the raw body) on your side.
- Run the official image: it runs as a non-root user and ships with a health check.

## What Arcanum does for you

- API key required on every route except `/health` and `/docs`, compared in constant time.
- Request size limit, HTTP timeouts and graceful shutdown.
- Instance names are validated (they become file names).
- Webhooks cannot reach cloud metadata / link-local addresses, do not follow redirects and
  are checked again at connection time (DNS rebinding).
- Message content, phone numbers and contact names are not written to the logs.
- CI runs `govulncheck`, `gosec`, `staticcheck`, the race detector and a Trivy image scan on
  every change; Dependabot keeps dependencies current.
