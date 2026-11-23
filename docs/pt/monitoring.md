# Monitoramento

## Saúde

`GET /health` é público e pode ficar exposto para um balanceador: mostra só contagens,
nunca nomes de sessão nem números.

```json
{
  "status": "ok",
  "version": "v1.1.0",
  "instances": 3,
  "sessions": {"connected": 2, "qr": 1}
}
```

Crie um alerta quando uma sessão que deveria estar `connected` ficar em outro estado.

## Métricas do Prometheus

`GET /metrics` (exige a chave de API) devolve contadores no formato texto do Prometheus:

| Métrica | Tipo | Significado |
|---|---|---|
| `arcanum_sessions{status}` | gauge | Sessões por status. |
| `arcanum_messages_received_total` | counter | Mensagens recebidas e encaminhadas ao webhook. |
| `arcanum_messages_sent_total` | counter | Mensagens enviadas pela API. |
| `arcanum_messages_send_errors_total` | counter | Envios que falharam depois das tentativas. |
| `arcanum_webhook_delivered_total` | counter | Entregas de webhook respondidas com `2xx`. |
| `arcanum_webhook_failed_total` | counter | Entregas de webhook que falharam. |
| `arcanum_http_requests_total{code}` | counter | Respostas da API por código HTTP. |

```yaml title="prometheus.yml"
scrape_configs:
  - job_name: arcanum
    metrics_path: /metrics
    authorization:
      credentials: <sua GATEWAY_API_KEY>
    static_configs:
      - targets: ["arcanum:3100"]
```

### Alertas úteis

```yaml
groups:
  - name: arcanum
    rules:
      - alert: ArcanumSessaoCaiu
        expr: arcanum_sessions{status="connected"} < 1
        for: 5m
      - alert: ArcanumWebhookFalhando
        expr: rate(arcanum_webhook_failed_total[5m]) > 0
        for: 10m
      - alert: ArcanumPedidosSemChave
        expr: rate(arcanum_http_requests_total{code="401"}[5m]) > 1
        for: 5m
```

Um fluxo constante de `401` costuma ser alguém sondando a API, ou um cliente com a chave
antiga.

## Logs

Use `GATEWAY_LOG_FORMAT=json` para mandar os logs para Loki, CloudWatch, Datadog ou
qualquer coletor:

```json
{"time":"2026-11-23T18:28:16Z","level":"INFO","msg":"arcanum gateway listening","addr":":3100","version":"v1.1.0","auth":true}
```

Use `GATEWAY_LOG_LEVEL=debug` por um tempo para investigar entregas de webhook.
