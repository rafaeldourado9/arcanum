# Sending messages

All send routes take the session name in the path and a JSON body. `number` accepts a
phone number with country code (`5511999999999`, punctuation is ignored) or a full JID
received in a webhook (`5511999999999@s.whatsapp.net`, `123456789@lid`).

## Common answers

| Code | Meaning |
|---|---|
| `200` | Sent. Body has `messageId`. |
| `202` | Text queued (see below). |
| `400` | Missing or invalid field. |
| `404` | Session does not exist. |
| `429` | Per-minute limit reached: `{"error":"rate limit exceeded","current":15,"limit":15}`. |
| `503` | Session is not connected: `{"error":"not connected","status":"qr"}`. |

## Text

```bash
curl -X POST http://localhost:3100/api/message/sendText/sales \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"number": "5511999999999", "text": "Your order has shipped."}'
```

```json
{"ok": true, "status": "queued"}
```

Text is **queued**: the answer is immediate and the [anti-ban layer](anti-ban.md) sends it
a few seconds later (typing indicator, human pause). Messages to the same chat go out in
the order you sent them. If a message still fails after 3 attempts, your webhook receives
a [`send_failed` event](webhooks.md#send-failures).

=== "Python"

    ```python
    arcanum.post("/api/message/sendText/sales",
                 json={"number": "5511999999999", "text": "Your order has shipped."})
    ```

=== "Node.js"

    ```js
    await fetch("http://localhost:3100/api/message/sendText/sales", {
      method: "POST",
      headers: { "X-Api-Key": process.env.ARCANUM_KEY, "Content-Type": "application/json" },
      body: JSON.stringify({ number: "5511999999999", text: "Your order has shipped." }),
    });
    ```

=== "Go"

    ```go
    body := strings.NewReader(`{"number":"5511999999999","text":"Your order has shipped."}`)
    req, _ := http.NewRequest("POST", base+"/api/message/sendText/sales", body)
    req.Header.Set("X-Api-Key", key)
    req.Header.Set("Content-Type", "application/json")
    resp, err := http.DefaultClient.Do(req)
    ```

## Media

Image, audio, video or document. `media` is the **file content in base64**; Arcanum does
not download URLs (that keeps it from being used to fetch internal addresses).

```bash
curl -X POST http://localhost:3100/api/message/sendMedia/sales \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d "{
        \"number\": \"5511999999999\",
        \"mediatype\": \"document\",
        \"mimetype\": \"application/pdf\",
        \"fileName\": \"invoice-1042.pdf\",
        \"caption\": \"Your invoice\",
        \"media\": \"$(base64 -w0 invoice-1042.pdf)\"
      }"
```

| Field | Notes |
|---|---|
| `mediatype` | `image`, `audio`, `video` or `document` (default). |
| `mimetype` | e.g. `image/jpeg`, `audio/ogg`, `video/mp4`, `application/pdf`. |
| `caption` | Shown under images, videos and documents. |
| `fileName` | Name shown for documents. |

Audio shows "recording audio..." before it is sent. Requests are limited to
`GATEWAY_MAX_BODY_MB` (64 MB by default), which includes the base64 overhead (~33%).

=== "Python"

    ```python
    import base64, pathlib

    data = base64.b64encode(pathlib.Path("invoice-1042.pdf").read_bytes()).decode()
    arcanum.post("/api/message/sendMedia/sales", json={
        "number": "5511999999999", "mediatype": "document",
        "mimetype": "application/pdf", "fileName": "invoice-1042.pdf", "media": data,
    })
    ```

=== "Node.js"

    ```js
    import { readFile } from "node:fs/promises";

    const media = (await readFile("invoice-1042.pdf")).toString("base64");
    await fetch(`${base}/api/message/sendMedia/sales`, {
      method: "POST",
      headers: { "X-Api-Key": key, "Content-Type": "application/json" },
      body: JSON.stringify({ number: "5511999999999", mediatype: "document",
        mimetype: "application/pdf", fileName: "invoice-1042.pdf", media }),
    });
    ```

## Location

```bash
curl -X POST http://localhost:3100/api/message/sendLocation/sales \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"number": "5511999999999", "name": "Head office",
       "address": "Av. Paulista, 1000", "latitude": -23.5614, "longitude": -46.6559}'
```

## Contact card

```bash
curl -X POST http://localhost:3100/api/message/sendContact/sales \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"number": "5511999999999",
       "contact": [{"fullName": "Support", "phoneNumber": "5511888888888"}]}'
```

## Reaction

Use the message `id` and chat from the webhook. An empty `reaction` removes it.

```bash
curl -X POST http://localhost:3100/api/message/sendReaction/sales \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"key": {"remoteJid": "5511999999999@s.whatsapp.net", "id": "3EB0C431C26A1916E2F5"},
       "reaction": "👍"}'
```

## Poll

```bash
curl -X POST http://localhost:3100/api/message/sendPoll/sales \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"number": "5511999999999", "name": "Best day for the meeting?",
       "selectableCount": 1, "values": ["Monday", "Tuesday", "Wednesday"]}'
```

## Other chat actions

| Route | Body | Does |
|---|---|---|
| `POST /api/chat/markMessageAsRead/{name}` | `{"readMessages":[{"remoteJid":"...","id":"..."}]}` | Blue ticks. |
| `POST /api/chat/checkWhatsAppNumbers/{name}` | `{"numbers":["5511..."]}` | Which numbers have WhatsApp. |
| `POST /api/chat/updateProfileName/{name}` | `{"name":"Acme Support"}` | Display name. |
| `POST /api/chat/updateProfileStatus/{name}` | `{"status":"Replies in minutes"}` | "About" text. |
| `POST /api/chat/updateProfilePicture/{name}` | `{"picture":"<base64 jpeg>"}` | Profile photo. |
| `POST /api/instance/setPresence/{name}` | `{"presence":"available"}` | Online / offline. |

!!! info "Planned"
    `archiveChat` (returns `501`) and `findChats`, `findContacts`, `findMessages` (return
    empty lists) need a message store and are on the roadmap.
