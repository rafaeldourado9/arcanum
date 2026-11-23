# Enviando mensagens

Todas as rotas de envio recebem o nome da sessão no caminho e um corpo JSON. O campo
`number` aceita o telefone com código do país (`5511999999999`; pontuação é ignorada) ou um
JID completo recebido no webhook (`5511999999999@s.whatsapp.net`, `123456789@lid`).

## Respostas comuns

| Código | Significado |
|---|---|
| `200` | Enviada. O corpo traz `messageId`. |
| `202` | Texto na fila (veja abaixo). |
| `400` | Campo faltando ou inválido. |
| `404` | A sessão não existe. |
| `429` | Limite por minuto atingido: `{"error":"rate limit exceeded","current":15,"limit":15}`. |
| `503` | A sessão não está conectada: `{"error":"not connected","status":"qr"}`. |

## Texto

```bash
curl -X POST http://localhost:3100/api/message/sendText/vendas \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"number": "5511999999999", "text": "Seu pedido foi enviado."}'
```

```json
{"ok": true, "status": "queued"}
```

O texto vai para uma **fila**: a resposta é imediata e a [camada anti-ban](anti-ban.md)
envia alguns segundos depois ("digitando...", pausa humana). Mensagens para a mesma
conversa saem na ordem em que você mandou. Se uma mensagem ainda falhar depois de 3
tentativas, seu webhook recebe um [evento `send_failed`](webhooks.md#falhas-de-envio).

=== "Python"

    ```python
    arcanum.post("/api/message/sendText/vendas",
                 json={"number": "5511999999999", "text": "Seu pedido foi enviado."})
    ```

=== "Node.js"

    ```js
    await fetch("http://localhost:3100/api/message/sendText/vendas", {
      method: "POST",
      headers: { "X-Api-Key": process.env.ARCANUM_KEY, "Content-Type": "application/json" },
      body: JSON.stringify({ number: "5511999999999", text: "Seu pedido foi enviado." }),
    });
    ```

=== "Go"

    ```go
    body := strings.NewReader(`{"number":"5511999999999","text":"Seu pedido foi enviado."}`)
    req, _ := http.NewRequest("POST", base+"/api/message/sendText/vendas", body)
    req.Header.Set("X-Api-Key", key)
    req.Header.Set("Content-Type", "application/json")
    resp, err := http.DefaultClient.Do(req)
    ```

## Mídia

Imagem, áudio, vídeo ou documento. `media` é o **conteúdo do arquivo em base64**; o Arcanum
não baixa URLs (isso impede que ele seja usado para acessar endereços internos).

```bash
curl -X POST http://localhost:3100/api/message/sendMedia/vendas \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d "{
        \"number\": \"5511999999999\",
        \"mediatype\": \"document\",
        \"mimetype\": \"application/pdf\",
        \"fileName\": \"boleto-1042.pdf\",
        \"caption\": \"Seu boleto\",
        \"media\": \"$(base64 -w0 boleto-1042.pdf)\"
      }"
```

| Campo | Observação |
|---|---|
| `mediatype` | `image`, `audio`, `video` ou `document` (padrão). |
| `mimetype` | Ex.: `image/jpeg`, `audio/ogg`, `video/mp4`, `application/pdf`. |
| `caption` | Aparece embaixo de imagens, vídeos e documentos. |
| `fileName` | Nome mostrado nos documentos. |

Áudio mostra "gravando áudio..." antes de ser enviado. O pedido tem limite de
`GATEWAY_MAX_BODY_MB` (64 MB por padrão), já contando o acréscimo do base64 (~33%).

=== "Python"

    ```python
    import base64, pathlib

    data = base64.b64encode(pathlib.Path("boleto-1042.pdf").read_bytes()).decode()
    arcanum.post("/api/message/sendMedia/vendas", json={
        "number": "5511999999999", "mediatype": "document",
        "mimetype": "application/pdf", "fileName": "boleto-1042.pdf", "media": data,
    })
    ```

=== "Node.js"

    ```js
    import { readFile } from "node:fs/promises";

    const media = (await readFile("boleto-1042.pdf")).toString("base64");
    await fetch(`${base}/api/message/sendMedia/vendas`, {
      method: "POST",
      headers: { "X-Api-Key": key, "Content-Type": "application/json" },
      body: JSON.stringify({ number: "5511999999999", mediatype: "document",
        mimetype: "application/pdf", fileName: "boleto-1042.pdf", media }),
    });
    ```

## Localização

```bash
curl -X POST http://localhost:3100/api/message/sendLocation/vendas \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"number": "5511999999999", "name": "Matriz",
       "address": "Av. Paulista, 1000", "latitude": -23.5614, "longitude": -46.6559}'
```

## Contato

```bash
curl -X POST http://localhost:3100/api/message/sendContact/vendas \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"number": "5511999999999",
       "contact": [{"fullName": "Suporte", "phoneNumber": "5511888888888"}]}'
```

## Reação

Use o `id` da mensagem e a conversa que vieram no webhook. `reaction` vazio remove a reação.

```bash
curl -X POST http://localhost:3100/api/message/sendReaction/vendas \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"key": {"remoteJid": "5511999999999@s.whatsapp.net", "id": "3EB0C431C26A1916E2F5"},
       "reaction": "👍"}'
```

## Enquete

```bash
curl -X POST http://localhost:3100/api/message/sendPoll/vendas \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"number": "5511999999999", "name": "Melhor dia para a reunião?",
       "selectableCount": 1, "values": ["Segunda", "Terça", "Quarta"]}'
```

## Outras ações

| Rota | Corpo | Faz |
|---|---|---|
| `POST /api/chat/markMessageAsRead/{nome}` | `{"readMessages":[{"remoteJid":"...","id":"..."}]}` | Tiques azuis. |
| `POST /api/chat/checkWhatsAppNumbers/{nome}` | `{"numbers":["5511..."]}` | Quais números têm WhatsApp. |
| `POST /api/chat/updateProfileName/{nome}` | `{"name":"Suporte Acme"}` | Nome de exibição. |
| `POST /api/chat/updateProfileStatus/{nome}` | `{"status":"Respondemos em minutos"}` | Texto do "recado". |
| `POST /api/chat/updateProfilePicture/{nome}` | `{"picture":"<jpeg em base64>"}` | Foto do perfil. |
| `POST /api/instance/setPresence/{nome}` | `{"presence":"available"}` | Online / offline. |

!!! info "Planejado"
    `archiveChat` (responde `501`) e `findChats`, `findContacts`, `findMessages` (devolvem
    listas vazias) dependem de um armazenamento de mensagens e estão no roadmap.
