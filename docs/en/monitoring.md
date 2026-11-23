# Monitoring

## Health

`GET /health` is public and safe to expose to a load balancer: it shows counts only, never
session names or numbers.

```json
{
  "status": "ok",
  "version": "v1.1.0",
  "instances": 3,
  "sessions": {"connected": 2, "qr": 1}
}
```

Alert when a session you expect to be `connected` stays in another state.

## Prometheus metrics

`GET /metrics` (requires the API key) returns counters in the Prometheus text format:

| Metric | Type | Meaning |
|---|---|---|
| `arcanum_sessions{status}` | gauge | Sessions by status. |
| `arcanum_messages_received_total` | counter | Incoming messages forwarded to the webhook. |
| `arcanum_messages_sent_total` | counter | Messages sent through the API. |
| `arcanum_messages_send_errors_total` | counter | Sends that failed after retries. |
| `arcanum_webhook_delivered_total` | counter | Webhook deliveries answered with `2xx`. |
| `arcanum_webhook_failed_total` | counter | Webhook deliveries that failed. |
| `arcanum_http_requests_total{code}` | counter | API responses by HTTP status. |

```yaml title="prometheus.yml"
scrape_configs:
  - job_name: arcanum
    metrics_path: /metrics
    authorization:
      credentials: <your GATEWAY_API_KEY>
    static_configs:
      - targets: ["arcanum:3100"]
```

### Useful alerts

```yaml
groups:
  - name: arcanum
    rules:
      - alert: ArcanumSessionDown
        expr: arcanum_sessions{status="connected"} < 1
        for: 5m
      - alert: ArcanumWebhookFailing
        expr: rate(arcanum_webhook_failed_total[5m]) > 0
        for: 10m
      - alert: ArcanumUnauthorizedRequests
        expr: rate(arcanum_http_requests_total{code="401"}[5m]) > 1
        for: 5m
```

A steady flow of `401` usually means someone is probing the API, or a client is using an
old key.

## Logs

Set `GATEWAY_LOG_FORMAT=json` to ship logs to Loki, CloudWatch, Datadog or any collector:

```json
{"time":"2026-11-23T18:28:16Z","level":"INFO","msg":"arcanum gateway listening","addr":":3100","version":"v1.1.0","auth":true}
```

Use `GATEWAY_LOG_LEVEL=debug` temporarily to troubleshoot webhook deliveries.
