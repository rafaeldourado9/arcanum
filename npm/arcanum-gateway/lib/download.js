"use strict";
// Baixa o binario do Arcanum da release do GitHub, confere o SHA-256 contra o
// checksums.txt publicado na mesma release e guarda em ~/.arcanum/bin/<versao>/.
// Sem dependencias: so modulos do Node.

const crypto = require("node:crypto");
const fs = require("node:fs");
const http = require("node:http");
const https = require("node:https");
const os = require("node:os");
const path = require("node:path");

const REPO = "rafaeldourado9/arcanum";

function platform() {
  const osName = { linux: "linux", darwin: "darwin", win32: "windows" }[process.platform];
  const arch = { x64: "amd64", arm64: "arm64" }[process.arch];
  if (!osName || !arch) {
    throw new Error(`Arcanum has no prebuilt binary for ${process.platform}/${process.arch}. Use Docker: https://arcanumgateway.com.br/en/installation/`);
  }
  return { osName, arch, ext: osName === "windows" ? ".exe" : "" };
}

function get(url, redirects = 5) {
  return new Promise((resolve, reject) => {
    const client = url.startsWith("http://") ? http : https; // http so para espelho/teste via ARCANUM_RELEASE_BASE
    const req = client.get(url, { headers: { "User-Agent": "arcanum-gateway-npm", Accept: "*/*" } }, (res) => {
      if ([301, 302, 303, 307, 308].includes(res.statusCode) && res.headers.location && redirects > 0) {
        res.resume();
        resolve(get(new URL(res.headers.location, url).toString(), redirects - 1));
        return;
      }
      if (res.statusCode !== 200) {
        res.resume();
        reject(new Error(`GET ${url} -> HTTP ${res.statusCode}`));
        return;
      }
      const chunks = [];
      res.on("data", (c) => chunks.push(c));
      res.on("end", () => resolve(Buffer.concat(chunks)));
      res.on("error", reject);
    });
    req.setTimeout(60000, () => req.destroy(new Error(`timeout downloading ${url}`)));
    req.on("error", reject);
  });
}

async function latestVersion() {
  const body = await get(`https://api.github.com/repos/${REPO}/releases/latest`);
  const tag = JSON.parse(body.toString("utf8")).tag_name;
  if (!tag) throw new Error("could not find the latest Arcanum release");
  return tag;
}

/**
 * Garante o binario e devolve o caminho. `version`: "latest" ou "vX.Y.Z".
 * ARCANUM_RELEASE_BASE troca a origem (espelho interno ou teste).
 */
async function ensureBinary(version, log = () => {}) {
  const { osName, arch, ext } = platform();
  const tag = version === "latest" ? await latestVersion() : version.startsWith("v") ? version : `v${version}`;
  const dir = path.join(process.env.ARCANUM_HOME || path.join(os.homedir(), ".arcanum"), "bin", tag);
  const bin = path.join(dir, `arcanum${ext}`);
  if (fs.existsSync(bin)) return { bin, tag };

  const num = tag.replace(/^v/, "");
  const file = `arcanum_${num}_${osName}_${arch}${ext}`;
  const base = (process.env.ARCANUM_RELEASE_BASE || `https://github.com/${REPO}/releases/download/${tag}`).replace(/\/$/, "");

  log(`Downloading Arcanum ${tag} (${osName}/${arch})...`);
  const [data, sums] = await Promise.all([get(`${base}/${file}`), get(`${base}/checksums.txt`)]);

  const line = sums.toString("utf8").split(/\r?\n/).find((l) => l.trim().endsWith(` ${file}`) || l.trim().endsWith(`*${file}`));
  if (!line) throw new Error(`${file} is not listed in checksums.txt`);
  const expected = line.trim().split(/\s+/)[0].toLowerCase();
  const actual = crypto.createHash("sha256").update(data).digest("hex");
  if (expected !== actual) throw new Error(`checksum mismatch for ${file}: expected ${expected}, got ${actual}`);
  log("Checksum OK.");

  fs.mkdirSync(dir, { recursive: true });
  const tmp = `${bin}.${process.pid}.tmp`;
  fs.writeFileSync(tmp, data, { mode: 0o755 });
  fs.renameSync(tmp, bin);
  return { bin, tag };
}

module.exports = { ensureBinary, platform };
