# API reference

Every running Arcanum serves its interactive documentation:

- **Swagger UI**: `http://<your-gateway>/docs`. Click **Authorize**, paste your API key and
  use **Try it out** on any route.
- **OpenAPI 3.1 spec**: `http://<your-gateway>/docs/openapi.json`, ready to import into
  Postman, Insomnia, Bruno or a client generator.

The spec is tested against the router on every build, so it always lists exactly the routes
that exist.

## All routes

| Method | Route | Description |
|---|---|---|
| `GET` | `/health` | Status, version, sessions by status. **Public.** |
| `GET` | `/metrics` | Prometheus metrics. |
| `GET` | `/api/stats` | Dashboard stats (JSON). |
| `GET` | `/api/logs` | Recent log lines (`?limit=&level=`). |
| `GET` | `/api/logs/stream` | Live log (Server-Sent Events). |
| `POST` | `/api/auth/login` | Dashboard sign-in (session cookie). **Public.** |
| `POST` | `/api/auth/logout` | Dashboard sign-out. **Public.** |
| `GET` | `/api/auth/me` | Dashboard session status. **Public.** |
| `POST` | `/api/instance/create` | Create a session. |
| `GET` | `/api/instance/fetchInstances` | List sessions. |
| `GET` | `/api/instance/connect/{name}` | Connect and get the QR code. |
| `GET` | `/api/instance/connectionState/{name}` | Status and current QR code. |
| `GET` | `/api/instance/qr/{name}` | Current QR code (`?format=png` for an image). |
| `POST` | `/api/instance/pair/{name}` | Pairing code instead of QR. |
| `POST` | `/api/instance/restart/{name}` | Reconnect. |
| `DELETE` | `/api/instance/logout/{name}` | Disconnect (stays paired). |
| `DELETE` | `/api/instance/delete/{name}` | Remove the session. |
| `POST` | `/api/instance/setPresence/{name}` | Online / offline. |
| `POST` | `/api/message/sendText/{name}` | Text (queued, anti-ban). |
| `POST` | `/api/message/sendMedia/{name}` | Image, audio, video, document (base64). |
| `POST` | `/api/message/sendLocation/{name}` | Map pin. |
| `POST` | `/api/message/sendContact/{name}` | Contact card. |
| `POST` | `/api/message/sendReaction/{name}` | Emoji reaction. |
| `POST` | `/api/message/sendPoll/{name}` | Poll. |
| `POST` | `/api/chat/markMessageAsRead/{name}` | Read receipts. |
| `POST` | `/api/chat/checkWhatsAppNumbers/{name}` | Which numbers have WhatsApp. |
| `POST` | `/api/chat/updateProfileName/{name}` | Display name. |
| `POST` | `/api/chat/updateProfileStatus/{name}` | "About" text. |
| `POST` | `/api/chat/updateProfilePicture/{name}` | Profile photo. |
| `POST` | `/api/chat/archiveChat/{name}` | Planned (`501`). |
| `POST` | `/api/chat/findChats/{name}` | Planned (empty list). |
| `POST` | `/api/chat/findContacts/{name}` | Planned (empty list). |
| `POST` | `/api/chat/findMessages/{name}` | Planned (empty list). |
| `GET` | `/api/events/webhook/{name}` | Webhook configuration. |
| `POST` | `/api/events/webhook/{name}` | Change webhook URL / events / on-off. |
| `POST` | `/api/events/webhook/{name}/test` | Send a signed test event. |
| `GET` | `/api/events/webhook/{name}/deliveries` | Last 100 deliveries. |
| `GET` | `/api/events/webhooks/export` | All webhook configurations (JSON). |
| `POST` | `/api/events/webhooks/import` | Apply an exported file. |
| `GET` | `/api/settings/{name}` | Session settings. |
| `POST` | `/api/settings/{name}` | Update settings. |

## Errors

Errors are JSON with an `error` field:

```json
{"error": "invalid instance name \"../x\": use 1-64 letters, digits, '-' or '_'"}
```

| Code | When |
|---|---|
| `400` | Invalid body or field. |
| `401` | Missing or invalid API key. |
| `404` | Session not found. |
| `409` | Session name already exists. |
| `413` | Request larger than `GATEWAY_MAX_BODY_MB`. |
| `429` | Per-minute send limit reached. |
| `501` | Route planned but not implemented yet. |
| `503` | Session not connected. |
