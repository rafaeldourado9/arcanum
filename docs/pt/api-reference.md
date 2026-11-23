# Referência da API

Todo Arcanum em execução serve a própria documentação interativa:

- **Swagger UI**: `http://<seu-gateway>/docs`. Clique em **Authorize**, cole sua chave de
  API e use **Try it out** em qualquer rota.
- **Especificação OpenAPI 3.1**: `http://<seu-gateway>/docs/openapi.json`, pronta para
  importar no Postman, Insomnia, Bruno ou num gerador de cliente.

A especificação é testada contra o roteador a cada build, então lista sempre exatamente as
rotas que existem.

## Todas as rotas

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/health` | Status, versão e sessões por status. **Pública.** |
| `GET` | `/metrics` | Métricas do Prometheus. |
| `GET` | `/api/stats` | Estatísticas do painel (JSON). |
| `GET` | `/api/logs` | Últimas linhas de log (`?limit=&level=`). |
| `GET` | `/api/logs/stream` | Log ao vivo (Server-Sent Events). |
| `POST` | `/api/auth/login` | Login do painel (cookie de sessão). **Pública.** |
| `POST` | `/api/auth/logout` | Sair do painel. **Pública.** |
| `GET` | `/api/auth/me` | Situação da sessão do painel. **Pública.** |
| `POST` | `/api/instance/create` | Cria uma sessão. |
| `GET` | `/api/instance/fetchInstances` | Lista as sessões. |
| `GET` | `/api/instance/connect/{nome}` | Conecta e devolve o QR code. |
| `GET` | `/api/instance/connectionState/{nome}` | Status e QR code atual. |
| `GET` | `/api/instance/qr/{nome}` | QR code atual (`?format=png` para imagem). |
| `POST` | `/api/instance/pair/{nome}` | Código de pareamento no lugar do QR. |
| `POST` | `/api/instance/restart/{nome}` | Reconecta. |
| `DELETE` | `/api/instance/logout/{nome}` | Desconecta (continua pareada). |
| `DELETE` | `/api/instance/delete/{nome}` | Remove a sessão. |
| `POST` | `/api/instance/setPresence/{nome}` | Online / offline. |
| `POST` | `/api/message/sendText/{nome}` | Texto (na fila, com anti-ban). |
| `POST` | `/api/message/sendMedia/{nome}` | Imagem, áudio, vídeo, documento (base64). |
| `POST` | `/api/message/sendLocation/{nome}` | Localização. |
| `POST` | `/api/message/sendContact/{nome}` | Contato. |
| `POST` | `/api/message/sendReaction/{nome}` | Reação com emoji. |
| `POST` | `/api/message/sendPoll/{nome}` | Enquete. |
| `POST` | `/api/chat/markMessageAsRead/{nome}` | Confirmação de leitura. |
| `POST` | `/api/chat/checkWhatsAppNumbers/{nome}` | Quais números têm WhatsApp. |
| `POST` | `/api/chat/updateProfileName/{nome}` | Nome de exibição. |
| `POST` | `/api/chat/updateProfileStatus/{nome}` | Texto do "recado". |
| `POST` | `/api/chat/updateProfilePicture/{nome}` | Foto do perfil. |
| `POST` | `/api/chat/archiveChat/{nome}` | Planejada (`501`). |
| `POST` | `/api/chat/findChats/{nome}` | Planejada (lista vazia). |
| `POST` | `/api/chat/findContacts/{nome}` | Planejada (lista vazia). |
| `POST` | `/api/chat/findMessages/{nome}` | Planejada (lista vazia). |
| `GET` | `/api/events/webhook/{nome}` | Configuração do webhook. |
| `POST` | `/api/events/webhook/{nome}` | Muda URL, eventos ou liga/desliga o webhook. |
| `POST` | `/api/events/webhook/{nome}/test` | Envia um evento de teste assinado. |
| `GET` | `/api/events/webhook/{nome}/deliveries` | Últimas 100 entregas. |
| `GET` | `/api/events/webhooks/export` | Configuração de todos os webhooks (JSON). |
| `POST` | `/api/events/webhooks/import` | Aplica um arquivo exportado. |
| `GET` | `/api/settings/{nome}` | Configurações da sessão. |
| `POST` | `/api/settings/{nome}` | Atualiza as configurações. |

## Erros

Os erros são JSON com o campo `error`:

```json
{"error": "invalid instance name \"../x\": use 1-64 letters, digits, '-' or '_'"}
```

| Código | Quando |
|---|---|
| `400` | Corpo ou campo inválido. |
| `401` | Chave de API faltando ou inválida. |
| `404` | Sessão não encontrada. |
| `409` | Já existe sessão com esse nome. |
| `413` | Pedido maior que `GATEWAY_MAX_BODY_MB`. |
| `429` | Limite de envios por minuto atingido. |
| `501` | Rota planejada, ainda não implementada. |
| `503` | Sessão não conectada. |
