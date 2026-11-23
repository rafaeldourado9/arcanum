# Problemas comuns

## O QR code não aparece

1. Olhe os logs: `docker logs arcanum`. Procure linhas que começam com `[whatsmeow]`.
2. Uma linha como `QR event "err-client-outdated"` quer dizer que o WhatsApp não aceita mais
   esta versão da biblioteca do cliente. **Atualize o Arcanum** (`docker compose pull &&
   docker compose up -d`).
3. Confira se a sessão existe (`fetchInstances`) e chame `connect` de novo; o QR code muda a
   cada ~20 segundos.

## `401 missing or invalid API key`

A chave está faltando ou é diferente da `GATEWAY_API_KEY`. Se você nunca definiu uma, o
Arcanum gerou uma chave temporária ao subir: procure no log e depois defina a sua.

## `503 not connected`

A sessão não está `connected`. Confira o `connectionState`. Se estiver `qr`, leia o código.
Se ficar voltando para `connecting`, verifique a internet do servidor.

## `429 rate limit exceeded`

A sessão chegou ao `GATEWAY_RATE_LIMIT_PER_MIN`. Espere e tente de novo, ou espalhe os
envios. Leia [Anti-ban](anti-ban.md) antes de aumentar o limite.

## As mensagens demoram a chegar

É a camada anti-ban: uma pausa e depois "digitando..." por um tempo que cresce com o
tamanho da mensagem. Com os padrões, conte com uns 4 a 13 segundos por mensagem.

## Meu webhook não recebe nada

- O webhook está `enabled`? `GET /api/events/webhook/{nome}`.
- A URL é alcançável **de dentro do container do Arcanum**? `localhost` dentro de um
  container é o próprio container: use o nome do serviço (`http://api:8000`) ou o endereço
  da máquina.
- Seu endpoint responde `2xx` em até 30 segundos? Veja `arcanum_webhook_failed_total` e os
  logs com `GATEWAY_LOG_LEVEL=debug`.
- Mensagens de grupo, status e listas de transmissão não são encaminhadas.

## A resposta vai para um número estranho terminado em `@lid`

Às vezes o WhatsApp esconde o telefone atrás de um identificador interno (`@lid`). O
Arcanum troca pelo telefone quando consegue. Quando não consegue, responda para o `from`
exatamente como veio: a mensagem chega do mesmo jeito.

## As sessões sumiram depois de reiniciar

O volume de dados não foi persistido. Monte `/data` como volume nomeado ou pasta da máquina
(`-v arcanum-data:/data`) e nunca apague: é lá que ficam as sessões logadas.

## Continua travado?

Abra uma issue no [GitHub](https://github.com/rafaeldourado9/arcanum/issues) com a versão do
Arcanum (`/health`), o que você fez, o que esperava e as linhas de log relevantes. Tire
chaves de API e telefones antes de publicar.
