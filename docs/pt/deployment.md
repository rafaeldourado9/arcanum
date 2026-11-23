# Produção

Uma instalação de produção tem três partes: o Arcanum numa porta privada, um proxy reverso
com HTTPS na frente e backup dos arquivos de sessão.

## Docker Compose com HTTPS (Caddy)

O Caddy emite e renova o certificado TLS sozinho.

```yaml title="compose.yaml"
services:
  arcanum:
    image: ghcr.io/rafaeldourado9/arcanum:latest
    restart: unless-stopped
    environment:
      GATEWAY_API_KEY: ${ARCANUM_KEY:?defina ARCANUM_KEY}
      GATEWAY_WEBHOOK_FORWARD_URL: ${ARCANUM_WEBHOOK_URL}
      GATEWAY_META_APP_SECRET: ${ARCANUM_WEBHOOK_SECRET}
      GATEWAY_LOG_FORMAT: json
    volumes:
      - arcanum-data:/data
    # sem "ports": só o Caddy alcança, pela rede interna

  caddy:
    image: caddy:2
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    command: caddy reverse-proxy --from wa.exemplo.com --to arcanum:3100
    volumes:
      - caddy-data:/data

volumes:
  arcanum-data:
  caddy-data:
```

```bash title=".env"
ARCANUM_KEY=...                 # openssl rand -hex 32
ARCANUM_WEBHOOK_URL=https://seu-app.exemplo.com/webhooks/whatsapp/{instance}
ARCANUM_WEBHOOK_SECRET=...      # openssl rand -hex 32
```

Aponte o DNS de `wa.exemplo.com` para o servidor e rode `docker compose up -d`.

??? example "Nginx no lugar do Caddy"

    ```nginx
    server {
        listen 443 ssl http2;
        server_name wa.exemplo.com;

        ssl_certificate     /etc/letsencrypt/live/wa.exemplo.com/fullchain.pem;
        ssl_certificate_key /etc/letsencrypt/live/wa.exemplo.com/privkey.pem;

        client_max_body_size 64m;          # igual a GATEWAY_MAX_BODY_MB

        location / {
            proxy_pass http://127.0.0.1:3100;
            proxy_set_header Host $host;
            proxy_read_timeout 300s;
        }
    }
    ```

## Dimensionamento

Cada sessão usa por volta de 20 a 50 MB de memória. Uma máquina de 1 GB roda dezenas de
sessões com folga. O uso de CPU é mínimo, exceto no envio de mídias grandes.

## Backup

Cada `<sessão>.db` no volume de dados **é** uma sessão de WhatsApp logada. Faça backup do
volume, criptografado, e restaure para trazer as sessões de volta sem ler QR code.

```bash
# para por alguns segundos, para a cópia sair consistente
docker compose stop arcanum
docker run --rm -v arcanum-data:/data -v "$PWD":/backup alpine \
  tar czf /backup/arcanum-$(date +%F).tgz -C /data .
docker compose start arcanum
```

Criptografe o arquivo (por exemplo com `age` ou `gpg`) antes de tirá-lo do servidor.

## Atualizar

```bash
docker compose pull arcanum
docker compose up -d arcanum
```

As sessões reconectam sozinhas depois do reinício. O Arcanum desliga de forma limpa no
`SIGTERM`: termina os pedidos em andamento e fecha cada arquivo de sessão direito. Atualize
com frequência: de tempos em tempos o WhatsApp deixa de aceitar versões antigas do cliente.

## Checagem de saúde

A imagem já tem um `HEALTHCHECK` em `/health`. Para Kubernetes ou um balanceador:

```yaml
livenessProbe:
  httpGet: { path: /health, port: 3100 }
  periodSeconds: 30
```
