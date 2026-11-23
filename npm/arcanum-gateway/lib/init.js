"use strict";
// `arcanum init [pasta]`: cria um projeto pronto para docker compose, com chave de API,
// senha do painel e segredo de webhook aleatorios num .env que so o dono le.

const crypto = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");

const COMPOSE = `# Arcanum - created by "npx github:rafaeldourado9/arcanum init"
# Start:   docker compose up -d
# Open:    http://localhost:3100 (dashboard password in .env)
services:
  arcanum:
    image: ghcr.io/rafaeldourado9/arcanum:latest
    restart: unless-stopped
    env_file: .env
    ports:
      - "127.0.0.1:3100:3100"
    volumes:
      - arcanum-data:/data
    read_only: true
    tmpfs:
      - /tmp
    security_opt:
      - no-new-privileges:true
    cap_drop: [ALL]

volumes:
  arcanum-data:
`;

const random = (bytes) => crypto.randomBytes(bytes).toString("hex");

function env() {
  return `# Arcanum configuration. Reference: https://arcanum.rafaeldourado.tech/en/configuration/
# Keep this file private: it controls your WhatsApp numbers.
GATEWAY_API_KEY=${random(32)}
GATEWAY_DASHBOARD_PASSWORD=${random(12)}
GATEWAY_META_APP_SECRET=${random(32)}
GATEWAY_WEBHOOK_FORMAT=simple
# Where incoming messages go ({instance} becomes the session name):
# GATEWAY_WEBHOOK_FORWARD_URL=https://your-app.example.com/webhooks/whatsapp/{instance}
`;
}

const README = `# Arcanum

1. \`docker compose up -d\`
2. Open http://localhost:3100 and sign in with GATEWAY_DASHBOARD_PASSWORD from \`.env\`.
3. Sessions > New session, scan the QR code with WhatsApp.

Your app calls the API with the header \`X-Api-Key: <GATEWAY_API_KEY>\` and verifies webhooks
with GATEWAY_META_APP_SECRET.

- Guide: https://arcanum.rafaeldourado.tech/en/
- Tutorials: https://arcanum.rafaeldourado.tech/en/tutorials/
- Before exposing it to the internet: https://arcanum.rafaeldourado.tech/en/deployment/
`;

function init(target = "arcanum", log = console.log) {
  const dir = path.resolve(target);
  if (fs.existsSync(path.join(dir, ".env")) || fs.existsSync(path.join(dir, "compose.yaml"))) {
    throw new Error(`${dir} already has an Arcanum project (compose.yaml or .env); choose another folder`);
  }
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(path.join(dir, "compose.yaml"), COMPOSE);
  fs.writeFileSync(path.join(dir, ".env"), env(), { mode: 0o600 });
  fs.writeFileSync(path.join(dir, ".gitignore"), ".env\n");
  fs.writeFileSync(path.join(dir, "README.md"), README);

  const rel = path.relative(process.cwd(), dir) || ".";
  log(`\nCreated an Arcanum project in ${rel}/\n`);
  log("  compose.yaml   the gateway, hardened, on http://localhost:3100");
  log("  .env           random API key, dashboard password and webhook secret (private)\n");
  log("Next steps:");
  log(`  cd ${rel}`);
  log("  docker compose up -d");
  log("  open http://localhost:3100   (password: GATEWAY_DASHBOARD_PASSWORD in .env)\n");
  log("No Docker? Run it directly: npx github:rafaeldourado9/arcanum");
  log("Guide: https://arcanum.rafaeldourado.tech/en/\n");
  return dir;
}

module.exports = { init };
