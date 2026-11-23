# Webhooks

O Arcanum envia cada mensagem recebida no WhatsApp para sua aplicação, em JSON. Você
escolhe a URL, o formato e um segredo para assinar cada entrega.

## Configurar

=== "Uma URL para todas as sessões (env)"

    ```bash
    GATEWAY_WEBHOOK_FORWARD_URL=https://seu-app.exemplo.com/webhooks/whatsapp/{instance}
    GATEWAY_WEBHOOK_FORMAT=simple        # ou "meta"
    GATEWAY_META_APP_SECRET=troque-isto  # segredo do HMAC (muito recomendado)
    ```

    `{instance}` é trocado pelo nome da sessão, então sua aplicação sabe em qual número a
    mensagem chegou (`/webhooks/whatsapp/vendas`).

=== "Por sessão (API)"

    ```bash
    curl -X POST http://localhost:3100/api/events/webhook/vendas \
      -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
      -d '{"url": "https://seu-app.exemplo.com/webhooks/whatsapp/vendas", "enabled": true}'
    ```

    `GET` na mesma rota devolve a configuração atual. `"enabled": false` pausa as entregas
    sem perder a URL.

A URL tem que ser `http` ou `https`. Endereços de metadados da nuvem e link-local
(`169.254.169.254`, ...) são recusados, conferidos de novo na hora de conectar, e
redirecionamentos não são seguidos. Redes privadas são permitidas, então um webhook para
outro container (`http://api:8000/...`) funciona.

A configuração de cada sessão fica salva em disco e sobrevive a reinícios. O
[painel](dashboard.md) mostra, edita e exporta.

## Teste e histórico

```bash
# manda um {"event":"test","instance":"vendas","timestamp":"..."} assinado e devolve o resultado
curl -X POST -H "X-Api-Key: $ARCANUM_KEY" http://localhost:3100/api/events/webhook/vendas/test
```

```json
{"time":"2026-11-23T19:13:03Z","event":"test","status":200,"ok":true,"durationMs":84}
```

`GET /api/events/webhook/{nome}/deliveries` lista as últimas 100 entregas (evento, status
HTTP, duração, erro). O corpo não é guardado, então nenhum dado pessoal fica salvo.

## Exportar e importar

```bash
curl -H "X-Api-Key: $ARCANUM_KEY" http://localhost:3100/api/events/webhooks/export -o webhooks.json
# no servidor novo, depois de criar as sessões:
curl -X POST -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  --data @webhooks.json http://localhost:3100/api/events/webhooks/import
```

A importação só atualiza sessões que existem; nomes desconhecidos e URLs inválidas voltam em
`skipped`, com o motivo.

## Formato `simple` (padrão)

Plano e fácil de ler. Exemplo de mensagem de texto:

```json
{
  "from": "5511999999999@s.whatsapp.net",
  "from_me": false,
  "push_name": "Maria",
  "type": "text",
  "text": "Oi, meu pedido já saiu?",
  "message_id": "3EB0C431C26A1916E2F5",
  "timestamp": "1764000000"
}
```

| Campo | Observação |
|---|---|
| `from` | O **contato** da conversa, em JID. Sempre o telefone (`@s.whatsapp.net`) quando o WhatsApp deixa o Arcanum descobrir; senão, `@lid`. Responda para esse valor como veio. |
| `from_me` | `true` quando a mensagem foi enviada **pelo próprio número conectado**, inclusive o que um humano digita no celular. Sempre presente. |
| `push_name` | Nome de exibição do contato, quando houver. |
| `type` | `text`, `image`, `audio`, `video`, `document`, `sticker`, `location` ou `unknown`. |
| `text` | Texto da mensagem, ou a legenda de uma mídia. |
| `message_id` | Use para reações e confirmação de leitura. |
| `timestamp` | Segundos Unix, em texto. |
| `media` | Presente em mídias: `mimetype`, `data` (base64), `filename`, `caption`. |
| `location` | Presente em `location`: `latitude`, `longitude`, `name`, `address`. |

??? example "Exemplos de mídia e localização"

    ```json title="imagem"
    {
      "from": "5511999999999@s.whatsapp.net", "from_me": false, "type": "image",
      "text": "a peça quebrada", "message_id": "3EB0...", "timestamp": "1764000000",
      "media": {"mimetype": "image/jpeg", "data": "/9j/4AAQSk...", "filename": "", "caption": "a peça quebrada"}
    }
    ```

    ```json title="localização"
    {
      "from": "5511999999999@s.whatsapp.net", "from_me": false, "type": "location",
      "message_id": "3EB0...", "timestamp": "1764000000",
      "location": {"latitude": -23.5614, "longitude": -46.6559, "name": "Matriz"}
    }
    ```

## Formato `meta`

Mesmo envelope da **Cloud API do WhatsApp** (Meta): código escrito para a API oficial lê as
mensagens do Arcanum com o mesmo parser. Use `GATEWAY_WEBHOOK_FORMAT=meta`.

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
          "text": {"body": "Oi, meu pedido já saiu?"}
        }]
      }
    }]
  }]
}
```

## Outros eventos

Estes são planos nos dois formatos e trazem o campo `event`:

### Digitando

Enviado enquanto o contato digita, para o seu bot esperar antes de responder.

```json
{"event": "typing", "from": "5511999999999@s.whatsapp.net", "state": "composing"}
```

`state` é `composing` ou `paused`.

### Falhas de envio

Um texto da fila que ainda falhou depois de 3 tentativas:

```json
{"event": "send_failed", "to": "5511999999999", "text": "Seu pedido foi enviado."}
```

## Atendimento humano com `from_me`

Quando alguém responde um cliente pelo celular, o Arcanum encaminha essa mensagem com
`from_me: true` e `from` igual ao **cliente**. Use isso para pausar o bot naquela conversa:

```python
if payload["from_me"]:
    pausar_bot(conversa=payload["from"], minutos=30)
    return
```

## Conferir a assinatura

Com `GATEWAY_META_APP_SECRET` definido, toda entrega traz:

```http
X-Hub-Signature-256: sha256=<HMAC-SHA256 em hex do corpo bruto>
```

Calcule o HMAC sobre os **bytes brutos** recebidos (antes de ler o JSON) e compare em tempo
constante. Recuse o pedido se não bater.

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
        esperado = "sha256=" + hmac.new(SECRET, body, hashlib.sha256).hexdigest()
        if not hmac.compare_digest(esperado, x_hub_signature_256):
            raise HTTPException(status_code=401, detail="assinatura inválida")

        payload = await request.json()
        if payload.get("event") == "typing":
            return {"ok": True}
        if payload.get("from_me"):
            return {"ok": True}           # um humano respondeu pelo celular

        enfileirar_resposta(instance, payload)  # trabalho pesado vai para uma fila
        return {"ok": True}
    ```

=== "Node.js (Express)"

    ```js
    import crypto from "node:crypto";
    import express from "express";

    const SECRET = process.env.ARCANUM_WEBHOOK_SECRET;
    const app = express();

    // corpo bruto: a assinatura é sobre os bytes exatos que o Arcanum mandou
    app.post("/webhooks/whatsapp/:instance", express.raw({ type: "application/json" }), (req, res) => {
      const esperado = "sha256=" + crypto.createHmac("sha256", SECRET).update(req.body).digest("hex");
      const recebido = req.get("X-Hub-Signature-256") || "";
      const ok = recebido.length === esperado.length &&
        crypto.timingSafeEqual(Buffer.from(recebido), Buffer.from(esperado));
      if (!ok) return res.status(401).send("assinatura inválida");

      const payload = JSON.parse(req.body);
      if (!payload.event && !payload.from_me) enfileirarResposta(req.params.instance, payload);
      res.json({ ok: true });
    });

    app.listen(8000);
    ```

=== "Go"

    ```go
    func whatsapp(w http.ResponseWriter, r *http.Request) {
        body, err := io.ReadAll(io.LimitReader(r.Body, 70<<20))
        if err != nil {
            http.Error(w, "erro de leitura", http.StatusBadRequest)
            return
        }
        mac := hmac.New(sha256.New, []byte(os.Getenv("ARCANUM_WEBHOOK_SECRET")))
        mac.Write(body)
        esperado := "sha256=" + hex.EncodeToString(mac.Sum(nil))
        if !hmac.Equal([]byte(esperado), []byte(r.Header.Get("X-Hub-Signature-256"))) {
            http.Error(w, "assinatura inválida", http.StatusUnauthorized)
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
            go tratarMensagem(r.PathValue("instance"), msg)
        }
        w.WriteHeader(http.StatusOK)
    }
    ```

=== "PHP"

    ```php
    <?php
    $body = file_get_contents('php://input');
    $esperado = 'sha256=' . hash_hmac('sha256', $body, getenv('ARCANUM_WEBHOOK_SECRET'));
    if (!hash_equals($esperado, $_SERVER['HTTP_X_HUB_SIGNATURE_256'] ?? '')) {
        http_response_code(401);
        exit('assinatura inválida');
    }
    $payload = json_decode($body, true);
    // ...
    http_response_code(200);
    ```

## Como as entregas funcionam

- Cada evento é enviado **uma vez**, com timeout de 30 segundos. Qualquer resposta `2xx`
  conta como entregue; o resto vai para o log e para o contador
  `arcanum_webhook_failed_total` ([métricas](monitoring.md)).
- **Responda rápido.** Devolva `200` na hora e faça o trabalho (chamada de IA, gravação no
  banco) numa fila em segundo plano. Um endpoint lento não atrasa o resto do Arcanum, mas um
  timeout significa que o evento não é reenviado.
- Mensagens com mais de 10 minutos (o acúmulo que o WhatsApp entrega depois de muito tempo
  offline, ou logo após o pareamento) não são encaminhadas, para o bot não responder
  conversas antigas.
- A lista `events` é guardada na configuração; hoje todos os eventos são entregues. Filtro
  por evento e reenvio automático estão no roadmap.

## Testar na sua máquina

Exponha sua aplicação local com um túnel e aponte o webhook para ele:

```bash
# terminal 1: sua aplicação na porta 8000
# terminal 2:
cloudflared tunnel --url http://localhost:8000   # ou: ngrok http 8000
```

Use a URL HTTPS que aparecer como webhook e mande uma mensagem para o número conectado.
