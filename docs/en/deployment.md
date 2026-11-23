# Deployment

A production setup has three parts: Arcanum on a private port, a reverse proxy with HTTPS
in front, and backups of the session files.

## Docker Compose with HTTPS (Caddy)

Caddy gets and renews the TLS certificate automatically.

```yaml title="compose.yaml"
services:
  arcanum:
    image: ghcr.io/rafaeldourado9/arcanum:latest
    restart: unless-stopped
    environment:
      GATEWAY_API_KEY: ${ARCANUM_KEY:?set ARCANUM_KEY}
      GATEWAY_WEBHOOK_FORWARD_URL: ${ARCANUM_WEBHOOK_URL}
      GATEWAY_META_APP_SECRET: ${ARCANUM_WEBHOOK_SECRET}
      GATEWAY_LOG_FORMAT: json
    volumes:
      - arcanum-data:/data
    # no "ports": only Caddy reaches it, through the internal network

  caddy:
    image: caddy:2
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    command: caddy reverse-proxy --from wa.example.com --to arcanum:3100
    volumes:
      - caddy-data:/data

volumes:
  arcanum-data:
  caddy-data:
```

```bash title=".env"
ARCANUM_KEY=...                 # openssl rand -hex 32
ARCANUM_WEBHOOK_URL=https://your-app.example.com/webhooks/whatsapp/{instance}
ARCANUM_WEBHOOK_SECRET=...      # openssl rand -hex 32
```

Point the DNS of `wa.example.com` to the server and run `docker compose up -d`.

??? example "Nginx instead of Caddy"

    ```nginx
    server {
        listen 443 ssl http2;
        server_name wa.example.com;

        ssl_certificate     /etc/letsencrypt/live/wa.example.com/fullchain.pem;
        ssl_certificate_key /etc/letsencrypt/live/wa.example.com/privkey.pem;

        client_max_body_size 64m;          # same as GATEWAY_MAX_BODY_MB

        location / {
            proxy_pass http://127.0.0.1:3100;
            proxy_set_header Host $host;
            proxy_read_timeout 300s;
        }
    }
    ```

## Sizing

Each session uses roughly 20-50 MB of RAM. A 1 GB machine comfortably runs dozens of
sessions. CPU use is negligible except when sending large media.

## Backups

Each `<session>.db` in the data volume **is** a logged-in WhatsApp session. Back the volume
up, encrypted, and restore it to bring sessions back without scanning QR codes.

```bash
# stop for a consistent copy (a few seconds of downtime)
docker compose stop arcanum
docker run --rm -v arcanum-data:/data -v "$PWD":/backup alpine \
  tar czf /backup/arcanum-$(date +%F).tgz -C /data .
docker compose start arcanum
```

Encrypt the archive (for example with `age` or `gpg`) before sending it off the server.

## Updating

```bash
docker compose pull arcanum
docker compose up -d arcanum
```

Sessions reconnect on their own after the restart. Arcanum shuts down gracefully on
`SIGTERM`: it finishes in-flight requests and closes every session file cleanly. Update
regularly: WhatsApp periodically stops accepting old client versions.

## Health checks

The image has a built-in `HEALTHCHECK` on `/health`. For Kubernetes or a load balancer:

```yaml
livenessProbe:
  httpGet: { path: /health, port: 3100 }
  periodSeconds: 30
```
