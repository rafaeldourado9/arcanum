# Webhooks

Arcanum posts every incoming WhatsApp message to your application as JSON. You choose
the URL, the format and a secret to sign each delivery.

## Configure

=== "One URL for every session (env)"

    ```bash
    GATEWAY_WEBHOOK_FORWARD_URL=https://your-app.example.com/webhooks/whatsapp/{instance}
    GATEWAY_WEBHOOK_FORMAT=simple        # or "meta"
    GATEWAY_META_APP_SECRET=change-me    # HMAC secret (strongly recommended)
    ```

    `{instance}` is replaced by the session name, so your app knows which number the
    message arrived on (`/webhooks/whatsapp/sales`).

=== "Per session (API)"

    ```bash
    curl -X POST http://localhost:3100/api/events/webhook/sales \
      -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
      -d '{"url": "https://your-app.example.com/webhooks/whatsapp/sales", "enabled": true}'
    ```

    `GET` on the same route returns the current configuration. `"enabled": false` pauses
    deliveries without losing the URL.

The URL must be `http` or `https`. Cloud metadata and link-local addresses
(`169.254.169.254`, ...) are refused, checked again at connection time, and redirects are
not followed. Private networks are allowed, so a webhook to another container
(`http://api:8000/...`) works.

The configuration of each session is saved to disk and survives restarts. The
[dashboard](dashboard.md) shows it, edits it and exports it.

## Test and history

```bash
# send a signed {"event":"test","instance":"sales","timestamp":"..."} and get the result
curl -X POST -H "X-Api-Key: $ARCANUM_KEY" http://localhost:3100/api/events/webhook/sales/test
```

```json
{"time":"2026-11-23T19:13:03Z","event":"test","status":200,"ok":true,"durationMs":84}
```

`GET /api/events/webhook/{name}/deliveries` lists the last 100 deliveries (event, HTTP
status, duration, error). Bodies are not stored, so no personal data is kept.

## Export and import

```bash
curl -H "X-Api-Key: $ARCANUM_KEY" http://localhost:3100/api/events/webhooks/export -o webhooks.json
# on the new server, after creating the sessions:
curl -X POST -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  --data @webhooks.json http://localhost:3100/api/events/webhooks/import
```

Import only updates sessions that exist; unknown names and invalid URLs are reported as
`skipped` with the reason.

## Payload: `simple` format (default)

Flat and easy to read. Example of a text message:

```json
{
  "from": "5511999999999@s.whatsapp.net",
  "from_me": false,
  "push_name": "Maria",
  "type": "text",
  "text": "Hi, is my order on the way?",
  "message_id": "3EB0C431C26A1916E2F5",
  "timestamp": "1764000000"
}
```

| Field | Notes |
|---|---|
| `from` | The **contact** of the conversation as a JID. Always the phone number (`@s.whatsapp.net`) when WhatsApp lets Arcanum resolve it; `@lid` otherwise. Reply to this value as-is. |
| `from_me` | `true` when the message was sent **by the connected number itself**, including messages a human types on the phone. Always present. |
| `push_name` | Contact's display name, when available. |
| `type` | `text`, `image`, `audio`, `video`, `document`, `sticker`, `location` or `unknown`. |
| `text` | Message text, or the caption of a media message. |
| `message_id` | Use it for reactions and read receipts. |
| `timestamp` | Unix seconds, as a string. |
| `media` | Present for media: `mimetype`, `data` (base64), `filename`, `caption`. |
| `location` | Present for `location`: `latitude`, `longitude`, `name`, `address`. |

??? example "Media and location examples"

    ```json title="image"
    {
      "from": "5511999999999@s.whatsapp.net", "from_me": false, "type": "image",
      "text": "the broken part", "message_id": "3EB0...", "timestamp": "1764000000",
      "media": {"mimetype": "image/jpeg", "data": "/9j/4AAQSk...", "filename": "", "caption": "the broken part"}
    }
    ```

    ```json title="location"
    {
      "from": "5511999999999@s.whatsapp.net", "from_me": false, "type": "location",
      "message_id": "3EB0...", "timestamp": "1764000000",
      "location": {"latitude": -23.5614, "longitude": -46.6559, "name": "Head office"}
    }
    ```

## Payload: `meta` format

Same envelope as the **WhatsApp Cloud API** (Meta), so code written for the official API
can read Arcanum messages with the same parser. Set `GATEWAY_WEBHOOK_FORMAT=meta`.

```json
{
  "object": "whatsapp_business_account",
  "entry": [{
    "id": "gateway",
    "changes": [{
      "field": "messages",
      "value": {
        "messaging_product": "whatsapp",
        "metadata": {"display_phone_number": "gateway", "phone_number_id": "gateway"},
        "contacts": [{"wa_id": "5511999999999@s.whatsapp.net", "profile": {"name": "Maria"}}],
        "messages": [{
          "from": "5511999999999@s.whatsapp.net",
          "from_me": false,
          "id": "3EB0C431C26A1916E2F5",
          "timestamp": "1764000000",
          "type": "text",
          "text": {"body": "Hi, is my order on the way?"}
        }]
      }
    }]
  }]
}
```

## Other events

These are flat in both formats and carry an `event` field:

### Typing

Sent while the contact is typing, so your bot can wait before answering.

```json
{"event": "typing", "from": "5511999999999@s.whatsapp.net", "state": "composing"}
```

`state` is `composing` or `paused`.

### Send failures

A queued text that still failed after 3 attempts:

```json
{"event": "send_failed", "to": "5511999999999", "text": "Your order has shipped."}
```

## Human takeover with `from_me`

When someone answers a customer from the phone, Arcanum forwards that message with
`from_me: true` and `from` set to the **customer**. Use it to pause your bot in that
conversation:

```python
if payload["from_me"]:
    pause_bot(conversation=payload["from"], minutes=30)
    return
```

## Verify the signature

With `GATEWAY_META_APP_SECRET` set, every delivery has:

```http
X-Hub-Signature-256: sha256=<hex HMAC-SHA256 of the raw request body>
```

Compute the HMAC over the **raw bytes** you received (before parsing JSON) and compare in
constant time. Reject the request if it does not match.

=== "Python (FastAPI)"

    ```python
    import hashlib, hmac, os
    from fastapi import FastAPI, Header, HTTPException, Request

    SECRET = os.environ["ARCANUM_WEBHOOK_SECRET"].encode()
    app = FastAPI()

    @app.post("/webhooks/whatsapp/{instance}")
    async def whatsapp(instance: str, request: Request,
                       x_hub_signature_256: str = Header(default="")):
        body = await request.body()
        expected = "sha256=" + hmac.new(SECRET, body, hashlib.sha256).hexdigest()
        if not hmac.compare_digest(expected, x_hub_signature_256):
            raise HTTPException(status_code=401, detail="bad signature")

        payload = await request.json()
        if payload.get("event") == "typing":
            return {"ok": True}
        if payload.get("from_me"):
            return {"ok": True}          # a human answered from the phone

        enqueue_reply(instance, payload)  # do the slow work in a queue
        return {"ok": True}
    ```

=== "Node.js (Express)"

    ```js
    import crypto from "node:crypto";
    import express from "express";

    const SECRET = process.env.ARCANUM_WEBHOOK_SECRET;
    const app = express();

    // raw body: the signature is over the exact bytes Arcanum sent
    app.post("/webhooks/whatsapp/:instance", express.raw({ type: "application/json" }), (req, res) => {
      const expected = "sha256=" + crypto.createHmac("sha256", SECRET).update(req.body).digest("hex");
      const got = req.get("X-Hub-Signature-256") || "";
      const ok = got.length === expected.length &&
        crypto.timingSafeEqual(Buffer.from(got), Buffer.from(expected));
      if (!ok) return res.status(401).send("bad signature");

      const payload = JSON.parse(req.body);
      if (!payload.event && !payload.from_me) queueReply(req.params.instance, payload);
      res.json({ ok: true });
    });

    app.listen(8000);
    ```

=== "Go"

    ```go
    func whatsapp(w http.ResponseWriter, r *http.Request) {
        body, err := io.ReadAll(io.LimitReader(r.Body, 70<<20))
        if err != nil {
            http.Error(w, "read error", http.StatusBadRequest)
            return
        }
        mac := hmac.New(sha256.New, []byte(os.Getenv("ARCANUM_WEBHOOK_SECRET")))
        mac.Write(body)
        expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
        if !hmac.Equal([]byte(expected), []byte(r.Header.Get("X-Hub-Signature-256"))) {
            http.Error(w, "bad signature", http.StatusUnauthorized)
            return
        }

        var msg struct {
            Event  string `json:"event"`
            From   string `json:"from"`
            FromMe bool   `json:"from_me"`
            Type   string `json:"type"`
            Text   string `json:"text"`
        }
        _ = json.Unmarshal(body, &msg)
        if msg.Event == "" && !msg.FromMe {
            go handleMessage(r.PathValue("instance"), msg)
        }
        w.WriteHeader(http.StatusOK)
    }
    ```

=== "PHP"

    ```php
    <?php
    $body = file_get_contents('php://input');
    $expected = 'sha256=' . hash_hmac('sha256', $body, getenv('ARCANUM_WEBHOOK_SECRET'));
    if (!hash_equals($expected, $_SERVER['HTTP_X_HUB_SIGNATURE_256'] ?? '')) {
        http_response_code(401);
        exit('bad signature');
    }
    $payload = json_decode($body, true);
    // ...
    http_response_code(200);
    ```

## Delivery behaviour

- Each event is posted **once**, with a 30-second timeout. Any `2xx` answer counts as
  delivered; anything else is logged and counted in `arcanum_webhook_failed_total`
  ([metrics](monitoring.md)).
- **Answer fast.** Return `200` right away and do the work (AI calls, database writes) in a
  background queue. A slow endpoint delays nothing else in Arcanum, but a timeout means the
  event is not retried.
- Messages older than 10 minutes (the backlog WhatsApp delivers after a long time offline
  or right after pairing) are not forwarded, so your bot does not answer old conversations.
- The `events` list is stored with the configuration; today every event is delivered.
  Per-event filtering and automatic retries are on the roadmap.

## Test locally

Expose your local app with a tunnel and point the webhook to it:

```bash
# terminal 1: your app on :8000
# terminal 2:
cloudflared tunnel --url http://localhost:8000   # or: ngrok http 8000
```

Then set the printed HTTPS URL as the webhook and send a message to the connected number.
