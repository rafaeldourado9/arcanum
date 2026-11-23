# Troubleshooting

## The QR code does not appear

1. Check the logs: `docker logs arcanum`. Look for lines starting with `[whatsmeow]`.
2. A line like `QR event "err-client-outdated"` means WhatsApp no longer accepts this
   version of the client library. **Update Arcanum** (`docker compose pull && docker
   compose up -d`).
3. Make sure the session exists (`fetchInstances`) and call `connect` again; the QR code
   rotates every ~20 seconds.

## `401 missing or invalid API key`

The key is missing or different from `GATEWAY_API_KEY`. If you never set one, Arcanum
generated a temporary key at startup: find it in the log, then set your own.

## `503 not connected`

The session is not `connected`. Check `connectionState`. If it is `qr`, scan the code. If
it keeps going back to `connecting`, check the server's internet connection.

## `429 rate limit exceeded`

The session reached `GATEWAY_RATE_LIMIT_PER_MIN`. Wait and retry, or spread the sends. See
[Anti-ban](anti-ban.md) before raising the limit.

## Messages arrive late

That is the anti-ban layer: a pause, then "typing..." for a time that grows with the
message length. Expected total: roughly 4-13 seconds per message with the defaults.

## My webhook receives nothing

- Is the webhook `enabled`? `GET /api/events/webhook/{name}`.
- Is the URL reachable **from the Arcanum container**? `localhost` inside a container is the
  container itself: use the service name (`http://api:8000`) or the host address.
- Does your endpoint answer `2xx` within 30 seconds? Check
  `arcanum_webhook_failed_total` and the logs with `GATEWAY_LOG_LEVEL=debug`.
- Group, status and broadcast messages are not forwarded.

## The reply goes to a strange number ending in `@lid`

WhatsApp sometimes hides the phone number behind an internal ID (`@lid`). Arcanum resolves
it to the phone number when it can. When it cannot, reply to the `from` value exactly as you
received it: the message still arrives.

## Sessions disappeared after a restart

The data volume was not persisted. Mount `/data` as a named volume or a host folder
(`-v arcanum-data:/data`), and never delete it: it holds the logged-in sessions.

## Still stuck?

Open an issue on [GitHub](https://github.com/rafaeldourado9/arcanum/issues) with the
Arcanum version (`/health`), what you did, what you expected, and the relevant log lines.
Remove API keys and phone numbers before posting.
