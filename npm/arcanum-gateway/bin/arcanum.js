#!/usr/bin/env node
"use strict";
// npx github:rafaeldourado9/arcanum            -> baixa (uma vez) e roda o gateway com o painel
// npx github:rafaeldourado9/arcanum init [dir] -> cria um projeto docker compose com chaves geradas
// npx github:rafaeldourado9/arcanum version    -> versao do binario
// Qualquer outro argumento vai direto para o binario. Configuracao: variaveis GATEWAY_*.

const { spawn } = require("node:child_process");
const os = require("node:os");
const path = require("node:path");
const { ensureBinary } = require("../lib/download");
const { init } = require("../lib/init");

const pkg = require("../package.json");

function help() {
  console.log(`Arcanum ${pkg.version}: self-hosted WhatsApp gateway

Usage:
  npx github:rafaeldourado9/arcanum                 run the gateway (dashboard on http://localhost:3100)
  npx github:rafaeldourado9/arcanum init [folder]   create a Docker Compose project with generated keys
  npx github:rafaeldourado9/arcanum version         print the gateway version
  npx github:rafaeldourado9/arcanum help            this help

Configure with GATEWAY_* environment variables, for example:
  GATEWAY_API_KEY=... GATEWAY_DASHBOARD_PASSWORD=... npx github:rafaeldourado9/arcanum

Without GATEWAY_API_KEY a temporary key is printed on start. Sessions are stored in
~/.arcanum/data unless GATEWAY_DB_PATH is set.

Guide: https://arcanumgateway.com.br/en/`);
}

async function main() {
  const [cmd, ...rest] = process.argv.slice(2);

  if (cmd === "help" || cmd === "--help" || cmd === "-h") return help();
  if (cmd === "init") {
    init(rest[0] || "arcanum");
    return;
  }

  // A versao do pacote npm acompanha a release do gateway; "0.0.0-dev" (repositorio) usa a ultima.
  const version = process.env.ARCANUM_VERSION || (pkg.version.startsWith("0.0.0") ? "latest" : `v${pkg.version}`);
  const { bin } = await ensureBinary(version, (m) => console.error(m));

  const env = { ...process.env };
  if (!env.GATEWAY_DB_PATH) env.GATEWAY_DB_PATH = path.join(os.homedir(), ".arcanum", "data");

  const child = spawn(bin, cmd ? [cmd, ...rest] : [], { stdio: "inherit", env });
  for (const sig of ["SIGINT", "SIGTERM"]) process.on(sig, () => child.kill(sig));
  child.on("exit", (code, signal) => process.exit(signal ? 1 : code ?? 0));
  child.on("error", (err) => {
    console.error(`could not start ${bin}: ${err.message}`);
    process.exit(1);
  });
}

main().catch((err) => {
  console.error(`arcanum: ${err.message}`);
  process.exit(1);
});
