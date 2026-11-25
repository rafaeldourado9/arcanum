# Installation

Arcanum is a single static binary with the dashboard built in. Pick the way that fits your
server; they all run the same thing.

| Method | Best for |
|---|---|
| [Windows](#windows-powershell) | Windows: one PowerShell command, works even when scripts are disabled. |
| [npx](#npx-nodejs) | Anyone with Node.js: one command, nothing else to install. |
| [Docker](#docker) | Most setups. One command, easy updates. |
| [Install script](#install-script-linux-macos) | A Linux or macOS machine without Docker. |
| [systemd service](#systemd-service) | A Linux server, started on boot, hardened. |
| [Binary download](#binary-download) | Windows, or full manual control. |
| [Kubernetes](#kubernetes) | Clusters. |
| [From source](#from-source) | Contributors. |

Whatever you choose, open **`http://<server>:3100`** for the [dashboard](dashboard.md) and
**`/docs`** for the interactive API.

## Windows (PowerShell)

Open **PowerShell** and paste:

```powershell
irm https://arcanumgateway.com.br/install.ps1 | iex
```

It works **even when running scripts is disabled** (the Windows default) and needs no
administrator rights. The installer downloads `arcanum.exe` from the GitHub release, **checks its
SHA-256**, installs it to `%LOCALAPPDATA%\Programs\Arcanum` and adds that folder to your user
PATH. Then:

```powershell
$env:GATEWAY_DASHBOARD_PASSWORD = "choose-a-long-password"
$env:GATEWAY_DB_PATH = "$env:LOCALAPPDATA\Arcanum"
arcanum
```

If `arcanum` is not found, open a new terminal (PATH changes apply to new windows). Options
before running: `$env:ARCANUM_VERSION = "v1.1.0"` pins a release, `$env:ARCANUM_INSTALL_DIR`
changes the folder and `$env:ARCANUM_NO_PATH = "1"` leaves the PATH alone.

## npx (Node.js)

The quickest way if you have Node.js 18+ and **Git** installed (`npx` uses Git to fetch
straight from GitHub; without it you get `npm error enoent ... git`). No Docker:

```bash
npx github:rafaeldourado9/arcanum
```

On first run it downloads the binary for your system from the GitHub release, **checks its
SHA-256** against `checksums.txt` and keeps it in `~/.arcanum/`. The dashboard opens on
`http://localhost:3100`; sessions are stored in `~/.arcanum/data`. Set your keys like any
other install:

```bash
GATEWAY_API_KEY="$(openssl rand -hex 32)" GATEWAY_DASHBOARD_PASSWORD="a-long-password" npx github:rafaeldourado9/arcanum
```

To start a Docker Compose project instead, with keys already generated:

```bash
npx github:rafaeldourado9/arcanum init            # creates ./arcanum with compose.yaml and a private .env
cd arcanum && docker compose up -d
```

`ARCANUM_VERSION=v1.1.0` pins a release; `ARCANUM_HOME` changes where binaries and data go.

!!! warning "npx on Windows PowerShell"
    In PowerShell, `npx` is a `.ps1` script and Windows blocks scripts by default (*"running
    scripts is disabled on this system"*). Use **`npx.cmd`** instead of `npx`
    (`npx.cmd github:rafaeldourado9/arcanum`), run it from Command Prompt (cmd), or use the
    [Windows installer](#windows-powershell). To allow scripts for your user once:
    `Set-ExecutionPolicy -Scope CurrentUser RemoteSigned`.

## Docker

```bash
docker run -d --name arcanum -p 3100:3100 \
  -e GATEWAY_API_KEY="$(openssl rand -hex 32)" \
  -e GATEWAY_DASHBOARD_PASSWORD="choose-a-long-password" \
  -v arcanum-data:/data \
  ghcr.io/rafaeldourado9/arcanum:latest
```

The image is **distroless** (no shell, no package manager), runs as an unprivileged user,
works with a read-only root filesystem and has a built-in health check. It is published for
`linux/amd64` and `linux/arm64`, signed with Cosign and shipped with an SBOM.

Ready-made Compose files live in the repository's
[`deploy/`](https://github.com/rafaeldourado9/arcanum/tree/main/deploy) folder:

=== "This machine only"

    ```bash
    curl -fsSLO https://raw.githubusercontent.com/rafaeldourado9/arcanum/main/deploy/docker-compose.yml
    curl -fsSL https://raw.githubusercontent.com/rafaeldourado9/arcanum/main/.env.example -o .env
    # edit .env: set GATEWAY_API_KEY and GATEWAY_DASHBOARD_PASSWORD
    docker compose up -d
    ```

=== "Public server with HTTPS"

    ```bash
    curl -fsSLO https://raw.githubusercontent.com/rafaeldourado9/arcanum/main/deploy/docker-compose.caddy.yml
    curl -fsSL https://raw.githubusercontent.com/rafaeldourado9/arcanum/main/.env.example -o .env
    echo "ARCANUM_DOMAIN=wa.example.com" >> .env   # its DNS must point to this server
    # edit .env: set GATEWAY_API_KEY and GATEWAY_DASHBOARD_PASSWORD
    docker compose -f docker-compose.caddy.yml up -d
    ```

    Caddy obtains and renews the Let's Encrypt certificate. Arcanum itself is not
    published on any port.

??? note "Verify the image signature"

    ```bash
    cosign verify ghcr.io/rafaeldourado9/arcanum:latest \
      --certificate-identity-regexp 'https://github.com/rafaeldourado9/arcanum/' \
      --certificate-oidc-issuer https://token.actions.githubusercontent.com
    ```

## Install script (Linux, macOS)

```bash
curl -fsSL https://arcanumgateway.com.br/install.sh | sh
```

The script detects your OS and CPU, downloads the latest release, **checks its SHA-256**
against `checksums.txt` and installs `/usr/local/bin/arcanum`. Then:

```bash
GATEWAY_API_KEY="$(openssl rand -hex 32)" GATEWAY_DB_PATH="$HOME/.arcanum" arcanum
```

Without root the script uses `sudo` to write to `/usr/local/bin`. To install **without sudo**
into your own folder (created if missing), pass the variable on the `sh` side:

```bash
curl -fsSL https://arcanumgateway.com.br/install.sh | ARCANUM_BIN_DIR="$HOME/.local/bin" sh
```

To pin a version: `... | sh -s -- --version v1.1.0`. Prefer to read it first? It is
[short](https://arcanumgateway.com.br/install.sh).

Tested on Ubuntu 24.04 and Alpine (root and regular user) and on Windows 11 (PowerShell 5.1).

## systemd service

On a Linux server, let the script set everything up:

```bash
curl -fsSL https://arcanumgateway.com.br/install.sh | sh -s -- --systemd
```

It creates:

- the system user `arcanum` and `/var/lib/arcanum` (permission `700`) for the sessions;
- `/etc/arcanum/arcanum.env`, readable only by root, with a random API key and dashboard
  password (`sudo grep -E 'API_KEY|PASSWORD' /etc/arcanum/arcanum.env` shows them);
- the `arcanum` service, enabled on boot.

The unit is heavily sandboxed (no new privileges, read-only system, private `/tmp`, no
capabilities, restricted system calls). Day to day:

```bash
sudo systemctl status arcanum
sudo journalctl -u arcanum -f
sudo systemctl restart arcanum   # after editing /etc/arcanum/arcanum.env
```

## Binary download

Every [release](https://github.com/rafaeldourado9/arcanum/releases) has archives for
Linux, macOS and Windows on `amd64` and `arm64`, plus `checksums.txt` and a build
provenance attestation.

=== "Windows (PowerShell)"

    ```powershell
    # after extracting arcanum_<version>_windows_amd64.zip
    $bytes = New-Object byte[] 32
    [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    $env:GATEWAY_API_KEY = ($bytes | ForEach-Object { $_.ToString("x2") }) -join ""
    Write-Host "API key: $env:GATEWAY_API_KEY"
    $env:GATEWAY_DB_PATH = "$env:LOCALAPPDATA\Arcanum"
    .\arcanum.exe
    ```

=== "Linux / macOS"

    ```bash
    tar -xzf arcanum_*_linux_amd64.tar.gz
    sha256sum -c --ignore-missing checksums.txt
    GATEWAY_API_KEY="$(openssl rand -hex 32)" ./arcanum
    ```

`arcanum version` prints the version; `arcanum healthcheck` exits `0` when the gateway on
`GATEWAY_PORT` is healthy.

## Kubernetes

```bash
kubectl create namespace arcanum
kubectl -n arcanum create secret generic arcanum \
  --from-literal=GATEWAY_API_KEY="$(openssl rand -hex 32)" \
  --from-literal=GATEWAY_DASHBOARD_PASSWORD="$(openssl rand -hex 12)"
kubectl -n arcanum apply -f https://raw.githubusercontent.com/rafaeldourado9/arcanum/main/deploy/kubernetes.yaml
```

The manifest runs **one replica** with the `Recreate` strategy (a WhatsApp session must
never be opened by two pods), a persistent volume, a read-only root filesystem and no
capabilities. Add your own Ingress with TLS.

## From source

Requires Go 1.26+ and Node.js 22+ (for the dashboard). No C compiler needed.

```bash
git clone https://github.com/rafaeldourado9/arcanum.git && cd arcanum
(cd web && npm ci && npm run build)        # builds the dashboard into the binary
go build -o arcanum ./cmd/gateway
GATEWAY_API_KEY="$(openssl rand -hex 32)" ./arcanum
```

Skip the dashboard step and the binary still works: `/` explains how to build it.

## Upgrading from 1.0

Version 1.1 runs as a non-root user (UID 65532). A data volume created by 1.0 belongs to
root, so give it to the new user once:

```bash
docker run --rm -v arcanum-data:/data alpine chown -R 65532:65532 /data
```

Version 1.1 also requires the API key on every route: set `GATEWAY_API_KEY` and send it
from your application ([Authentication](authentication.md)).
