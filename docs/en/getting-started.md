# Getting started

In five minutes you will have Arcanum running, a WhatsApp number connected and your first
message sent.

## 1. Run Arcanum

=== "npx"

    ```bash
    export ARCANUM_KEY="$(openssl rand -hex 32)"
    echo "Your API key: $ARCANUM_KEY"

    GATEWAY_API_KEY="$ARCANUM_KEY" npx github:rafaeldourado9/arcanum
    ```

    Needs Node.js 18+. Downloads the binary once (checksum verified) and runs it.

=== "Docker"

    ```bash
    export ARCANUM_KEY="$(openssl rand -hex 32)"
    echo "Your API key: $ARCANUM_KEY"

    docker run -d --name arcanum -p 3100:3100 \
      -e GATEWAY_API_KEY="$ARCANUM_KEY" \
      -v arcanum-data:/data \
      ghcr.io/rafaeldourado9/arcanum:latest
    ```

=== "Docker Compose"

    ```yaml title="compose.yaml"
    services:
      arcanum:
        image: ghcr.io/rafaeldourado9/arcanum:latest
        restart: unless-stopped
        ports:
          - "127.0.0.1:3100:3100"   # only this machine; use a reverse proxy for the internet
        environment:
          GATEWAY_API_KEY: ${ARCANUM_KEY:?set ARCANUM_KEY}
          GATEWAY_WEBHOOK_FORWARD_URL: https://your-app.example.com/webhooks/whatsapp/{instance}
          GATEWAY_META_APP_SECRET: ${ARCANUM_WEBHOOK_SECRET}
        volumes:
          - arcanum-data:/data

    volumes:
      arcanum-data:
    ```

    ```bash
    ARCANUM_KEY="$(openssl rand -hex 32)" docker compose up -d
    ```

=== "From source"

    Requires Go 1.26+ and a C compiler (the SQLite driver uses CGO).

    ```bash
    git clone https://github.com/rafaeldourado9/arcanum.git
    cd arcanum
    go build -o arcanum ./cmd/gateway
    GATEWAY_API_KEY="$(openssl rand -hex 32)" ./arcanum
    ```

Check that it is up (this route is public):

```bash
curl http://localhost:3100/health
```

```json
{"instances":0,"sessions":{},"status":"ok","version":"v1.1.0"}
```

!!! tip "Prefer clicking?"
    Open **http://localhost:3100** and sign in to the [dashboard](dashboard.md): it creates
    sessions, shows the QR code and configures webhooks. Add
    `-e GATEWAY_DASHBOARD_PASSWORD=...` to the command above to use a password instead of the
    API key. Other ways to install (script, systemd, Windows, Kubernetes) are in
    [Installation](installation.md).

!!! tip "Forgot the key?"
    If you start Arcanum without `GATEWAY_API_KEY`, it generates a temporary key and prints
    it in the log (`docker logs arcanum`). It changes on every restart, so set your own.

## 2. Create a session

A session (instance) is one WhatsApp number. Give it a name made of letters, digits,
`-` or `_`.

```bash
curl -X POST http://localhost:3100/api/instance/create \
  -H "X-Api-Key: $ARCANUM_KEY" \
  -H "Content-Type: application/json" \
  -d '{"instanceName": "sales"}'
```

## 3. Connect the number

=== "QR code"

    Open the QR code as an image and scan it with **WhatsApp > Linked devices > Link a
    device**:

    ```bash
    curl -H "X-Api-Key: $ARCANUM_KEY" \
      "http://localhost:3100/api/instance/connect/sales?format=png" -o qr.png
    ```

    Without `?format=png` the same route returns JSON with the QR code as a data URL,
    ready for an `<img src>` in your own UI.

=== "Pairing code"

    No camera at hand? Ask for an 8-character code and type it in **WhatsApp > Linked
    devices > Link with phone number instead**:

    ```bash
    curl -X POST http://localhost:3100/api/instance/pair/sales \
      -H "X-Api-Key: $ARCANUM_KEY" \
      -H "Content-Type: application/json" \
      -d '{"phone": "5511999999999"}'
    ```

Then check the status until it says `connected`:

```bash
curl -H "X-Api-Key: $ARCANUM_KEY" \
  http://localhost:3100/api/instance/connectionState/sales
```

## 4. Send your first message

```bash
curl -X POST http://localhost:3100/api/message/sendText/sales \
  -H "X-Api-Key: $ARCANUM_KEY" \
  -H "Content-Type: application/json" \
  -d '{"number": "5511999999999", "text": "Hello from Arcanum!"}'
```

```json
{"ok": true, "status": "queued"}
```

The answer comes right away (`202 Accepted`). The message goes out a few seconds later,
after the typing indicator and a human-like pause: that is the [anti-ban layer](anti-ban.md)
at work.

## 5. Receive messages

Point a webhook at your application and every incoming message arrives as a POST:

```bash
curl -X POST http://localhost:3100/api/events/webhook/sales \
  -H "X-Api-Key: $ARCANUM_KEY" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://your-app.example.com/webhooks/whatsapp", "enabled": true}'
```

See [Webhooks](webhooks.md) for the payload and how to verify the signature.

## What next

- [Interactive API docs](api-reference.md): every route, with a "Try it out" button.
- [Deployment](deployment.md): HTTPS, backups and updates.
- [Security](security.md): the checklist before going live.
