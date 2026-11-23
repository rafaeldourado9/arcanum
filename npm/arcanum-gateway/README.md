# arcanum-gateway

Run [Arcanum](https://arcanum.rafaeldourado.tech/en/), the open-source, self-hosted WhatsApp
gateway, with one command. No Docker needed.

```bash
npx arcanum-gateway
```

Open **http://localhost:3100** for the dashboard (the temporary API key is printed on start).
Set your own keys with environment variables:

```bash
GATEWAY_API_KEY="$(openssl rand -hex 32)" GATEWAY_DASHBOARD_PASSWORD="a-long-password" npx arcanum-gateway
```

## Create a Docker Compose project

```bash
npm create arcanum        # or: npx arcanum-gateway init my-gateway
cd arcanum && docker compose up -d
```

Creates `compose.yaml` and a private `.env` with a random API key, dashboard password and
webhook secret.

## How it works

The package has no dependencies. On first run it downloads the Arcanum binary for your
system (Linux, macOS, Windows; x64 or arm64) from the
[GitHub release](https://github.com/rafaeldourado9/arcanum/releases), **checks its SHA-256**
against the published `checksums.txt` and keeps it in `~/.arcanum/bin/<version>/`. Sessions
are stored in `~/.arcanum/data` unless `GATEWAY_DB_PATH` is set.

| Variable | |
|---|---|
| `ARCANUM_VERSION` | Run a specific release, e.g. `v1.1.0`. |
| `ARCANUM_HOME` | Where binaries and data go (default `~/.arcanum`). |

Guide, API and tutorials: https://arcanum.rafaeldourado.tech/en/ · MIT
