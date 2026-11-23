# Primeiros passos

Em cinco minutos você terá o Arcanum rodando, um número de WhatsApp conectado e a primeira
mensagem enviada.

## 1. Rode o Arcanum

=== "npx"

    ```bash
    export ARCANUM_KEY="$(openssl rand -hex 32)"
    echo "Sua chave de API: $ARCANUM_KEY"

    GATEWAY_API_KEY="$ARCANUM_KEY" npx arcanum-gateway
    ```

    Precisa de Node.js 18+. Baixa o binário uma vez (com checksum conferido) e roda.

=== "Docker"

    ```bash
    export ARCANUM_KEY="$(openssl rand -hex 32)"
    echo "Sua chave de API: $ARCANUM_KEY"

    docker run -d --name arcanum -p 3100:3100 \
      -e GATEWAY_API_KEY="$ARCANUM_KEY" \
      -v arcanum-data:/data \
      ghcr.io/rafaeldourado9/arcanum:latest
    ```

=== "Docker Compose"

    ```yaml title="compose.yaml"
    services:
      arcanum:
        image: ghcr.io/rafaeldourado9/arcanum:latest
        restart: unless-stopped
        ports:
          - "127.0.0.1:3100:3100"   # só esta máquina; para a internet use um proxy reverso
        environment:
          GATEWAY_API_KEY: ${ARCANUM_KEY:?defina ARCANUM_KEY}
          GATEWAY_WEBHOOK_FORWARD_URL: https://seu-app.exemplo.com/webhooks/whatsapp/{instance}
          GATEWAY_META_APP_SECRET: ${ARCANUM_WEBHOOK_SECRET}
        volumes:
          - arcanum-data:/data

    volumes:
      arcanum-data:
    ```

    ```bash
    ARCANUM_KEY="$(openssl rand -hex 32)" docker compose up -d
    ```

=== "Pelo código-fonte"

    Precisa de Go 1.26+ e de um compilador C (o driver do SQLite usa CGO).

    ```bash
    git clone https://github.com/rafaeldourado9/arcanum.git
    cd arcanum
    go build -o arcanum ./cmd/gateway
    GATEWAY_API_KEY="$(openssl rand -hex 32)" ./arcanum
    ```

Confira se subiu (esta rota é pública):

```bash
curl http://localhost:3100/health
```

```json
{"instances":0,"sessions":{},"status":"ok","version":"v1.1.0"}
```

!!! tip "Prefere clicar?"
    Abra **http://localhost:3100** e entre no [painel](dashboard.md): ele cria sessões, mostra
    o QR code e configura webhooks. Acrescente `-e GATEWAY_DASHBOARD_PASSWORD=...` ao comando
    acima para entrar com uma senha em vez da chave de API. Outros jeitos de instalar (script,
    systemd, Windows, Kubernetes) estão em [Instalação](installation.md).

!!! tip "Esqueceu a chave?"
    Se você subir o Arcanum sem `GATEWAY_API_KEY`, ele gera uma chave temporária e mostra no
    log (`docker logs arcanum`). Ela muda a cada reinício, então defina a sua.

## 2. Crie uma sessão

Uma sessão (instância) é um número de WhatsApp. Dê um nome com letras, números, `-` ou `_`.

```bash
curl -X POST http://localhost:3100/api/instance/create \
  -H "X-Api-Key: $ARCANUM_KEY" \
  -H "Content-Type: application/json" \
  -d '{"instanceName": "vendas"}'
```

## 3. Conecte o número

=== "QR code"

    Abra o QR code como imagem e leia em **WhatsApp > Aparelhos conectados > Conectar um
    aparelho**:

    ```bash
    curl -H "X-Api-Key: $ARCANUM_KEY" \
      "http://localhost:3100/api/instance/connect/vendas?format=png" -o qr.png
    ```

    Sem `?format=png`, a mesma rota devolve um JSON com o QR code em data URL, pronto para
    um `<img src>` na sua própria tela.

=== "Código de pareamento"

    Sem câmera por perto? Peça um código de 8 caracteres e digite em **WhatsApp > Aparelhos
    conectados > Conectar com número de telefone**:

    ```bash
    curl -X POST http://localhost:3100/api/instance/pair/vendas \
      -H "X-Api-Key: $ARCANUM_KEY" \
      -H "Content-Type: application/json" \
      -d '{"phone": "5511999999999"}'
    ```

Depois confira o status até aparecer `connected`:

```bash
curl -H "X-Api-Key: $ARCANUM_KEY" \
  http://localhost:3100/api/instance/connectionState/vendas
```

## 4. Envie a primeira mensagem

```bash
curl -X POST http://localhost:3100/api/message/sendText/vendas \
  -H "X-Api-Key: $ARCANUM_KEY" \
  -H "Content-Type: application/json" \
  -d '{"number": "5511999999999", "text": "Olá, aqui é o Arcanum!"}'
```

```json
{"ok": true, "status": "queued"}
```

A resposta vem na hora (`202 Accepted`). A mensagem sai alguns segundos depois, após o
"digitando..." e uma pausa humana: é a [camada anti-ban](anti-ban.md) trabalhando.

## 5. Receba mensagens

Aponte um webhook para sua aplicação e cada mensagem recebida chega como um POST:

```bash
curl -X POST http://localhost:3100/api/events/webhook/vendas \
  -H "X-Api-Key: $ARCANUM_KEY" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://seu-app.exemplo.com/webhooks/whatsapp", "enabled": true}'
```

Veja em [Webhooks](webhooks.md) o formato e como conferir a assinatura.

## E agora

- [Documentação interativa](api-reference.md): todas as rotas, com botão "Try it out".
- [Produção](deployment.md): HTTPS, backup e atualização.
- [Segurança](security.md): a lista para conferir antes de ir para o ar.
