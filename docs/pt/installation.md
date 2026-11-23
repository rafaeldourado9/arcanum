# Instalação

O Arcanum é um único binário estático, com o painel embutido. Escolha o jeito que combina com
o seu servidor; todos rodam a mesma coisa.

| Jeito | Melhor para |
|---|---|
| [Docker](#docker) | A maioria dos casos. Um comando, atualização fácil. |
| [Script de instalação](#script-de-instalacao-linux-macos) | Linux ou macOS sem Docker. |
| [Serviço do systemd](#servico-do-systemd) | Servidor Linux, sobe no boot, protegido. |
| [Download do binário](#download-do-binario) | Windows, ou controle manual total. |
| [Kubernetes](#kubernetes) | Clusters. |
| [Pelo código-fonte](#pelo-codigo-fonte) | Quem contribui. |

Seja qual for, abra **`http://<servidor>:3100`** para o [painel](dashboard.md) e **`/docs`**
para a API interativa.

## Docker

```bash
docker run -d --name arcanum -p 3100:3100 \
  -e GATEWAY_API_KEY="$(openssl rand -hex 32)" \
  -e GATEWAY_DASHBOARD_PASSWORD="escolha-uma-senha-longa" \
  -v arcanum-data:/data \
  ghcr.io/rafaeldourado9/arcanum:latest
```

A imagem é **distroless** (sem shell, sem gerenciador de pacotes), roda com usuário sem
privilégio, funciona com sistema de arquivos somente leitura e tem checagem de saúde própria.
É publicada para `linux/amd64` e `linux/arm64`, assinada com Cosign e acompanhada de SBOM.

Arquivos de Compose prontos ficam na pasta
[`deploy/`](https://github.com/rafaeldourado9/arcanum/tree/main/deploy) do repositório:

=== "Só esta máquina"

    ```bash
    curl -fsSLO https://raw.githubusercontent.com/rafaeldourado9/arcanum/main/deploy/docker-compose.yml
    curl -fsSL https://raw.githubusercontent.com/rafaeldourado9/arcanum/main/.env.example -o .env
    # edite o .env: defina GATEWAY_API_KEY e GATEWAY_DASHBOARD_PASSWORD
    docker compose up -d
    ```

=== "Servidor público com HTTPS"

    ```bash
    curl -fsSLO https://raw.githubusercontent.com/rafaeldourado9/arcanum/main/deploy/docker-compose.caddy.yml
    curl -fsSL https://raw.githubusercontent.com/rafaeldourado9/arcanum/main/.env.example -o .env
    echo "ARCANUM_DOMAIN=wa.exemplo.com" >> .env   # o DNS tem que apontar para este servidor
    # edite o .env: defina GATEWAY_API_KEY e GATEWAY_DASHBOARD_PASSWORD
    docker compose -f docker-compose.caddy.yml up -d
    ```

    O Caddy emite e renova o certificado do Let's Encrypt. O Arcanum em si não fica exposto
    em porta nenhuma.

??? note "Conferir a assinatura da imagem"

    ```bash
    cosign verify ghcr.io/rafaeldourado9/arcanum:latest \
      --certificate-identity-regexp 'https://github.com/rafaeldourado9/arcanum/' \
      --certificate-oidc-issuer https://token.actions.githubusercontent.com
    ```

## Script de instalação (Linux, macOS)

```bash
curl -fsSL https://arcanum.rafaeldourado.tech/install.sh | sh
```

O script descobre o sistema e o processador, baixa a versão mais recente, **confere o
SHA-256** contra o `checksums.txt` e instala `/usr/local/bin/arcanum`. Depois:

```bash
GATEWAY_API_KEY="$(openssl rand -hex 32)" GATEWAY_DB_PATH="$HOME/.arcanum" arcanum
```

Opções: `--version v1.1.0` fixa uma versão; `ARCANUM_BIN_DIR=$HOME/.local/bin` instala sem
sudo. Prefere ler antes? Ele é
[curto](https://arcanum.rafaeldourado.tech/install.sh).

## Serviço do systemd

Num servidor Linux, deixe o script montar tudo:

```bash
curl -fsSL https://arcanum.rafaeldourado.tech/install.sh | sh -s -- --systemd
```

Ele cria:

- o usuário de sistema `arcanum` e `/var/lib/arcanum` (permissão `700`) para as sessões;
- `/etc/arcanum/arcanum.env`, legível só pelo root, com uma chave de API e uma senha do painel
  aleatórias (`sudo grep -E 'API_KEY|PASSWORD' /etc/arcanum/arcanum.env` mostra as duas);
- o serviço `arcanum`, ligado no boot.

O serviço roda bem isolado (sem novos privilégios, sistema somente leitura, `/tmp` privado,
sem capabilities, chamadas de sistema restritas). No dia a dia:

```bash
sudo systemctl status arcanum
sudo journalctl -u arcanum -f
sudo systemctl restart arcanum   # depois de editar /etc/arcanum/arcanum.env
```

## Download do binário

Toda [versão](https://github.com/rafaeldourado9/arcanum/releases) tem arquivos para Linux,
macOS e Windows, em `amd64` e `arm64`, mais o `checksums.txt` e um atestado de proveniência
do build.

=== "Windows (PowerShell)"

    ```powershell
    # depois de extrair arcanum_<versão>_windows_amd64.zip
    $bytes = New-Object byte[] 32
    [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    $env:GATEWAY_API_KEY = ($bytes | ForEach-Object { $_.ToString("x2") }) -join ""
    Write-Host "Chave de API: $env:GATEWAY_API_KEY"
    $env:GATEWAY_DB_PATH = "$env:LOCALAPPDATA\Arcanum"
    .\arcanum.exe
    ```

=== "Linux / macOS"

    ```bash
    tar -xzf arcanum_*_linux_amd64.tar.gz
    sha256sum -c --ignore-missing checksums.txt
    GATEWAY_API_KEY="$(openssl rand -hex 32)" ./arcanum
    ```

`arcanum version` mostra a versão; `arcanum healthcheck` sai com `0` quando o gateway na
`GATEWAY_PORT` está saudável.

## Kubernetes

```bash
kubectl create namespace arcanum
kubectl -n arcanum create secret generic arcanum \
  --from-literal=GATEWAY_API_KEY="$(openssl rand -hex 32)" \
  --from-literal=GATEWAY_DASHBOARD_PASSWORD="$(openssl rand -hex 12)"
kubectl -n arcanum apply -f https://raw.githubusercontent.com/rafaeldourado9/arcanum/main/deploy/kubernetes.yaml
```

O manifesto roda **uma réplica** com estratégia `Recreate` (uma sessão de WhatsApp nunca pode
ser aberta por dois pods), volume persistente, sistema de arquivos somente leitura e sem
capabilities. Acrescente seu próprio Ingress com TLS.

## Pelo código-fonte

Precisa de Go 1.26+ e Node.js 22+ (para o painel). Não precisa de compilador C.

```bash
git clone https://github.com/rafaeldourado9/arcanum.git && cd arcanum
(cd web && npm ci && npm run build)        # embute o painel no binário
go build -o arcanum ./cmd/gateway
GATEWAY_API_KEY="$(openssl rand -hex 32)" ./arcanum
```

Sem o passo do painel o binário funciona do mesmo jeito: a `/` explica como gerar.

## Atualizando da 1.0

A versão 1.1 roda com usuário sem privilégio (UID 65532). Um volume de dados criado pela 1.0
pertence ao root; passe para o novo usuário uma vez:

```bash
docker run --rm -v arcanum-data:/data alpine chown -R 65532:65532 /data
```

A 1.1 também exige a chave de API em toda rota: defina `GATEWAY_API_KEY` e envie a chave pela
sua aplicação ([Autenticação](authentication.md)).
