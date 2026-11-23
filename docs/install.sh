#!/bin/sh
# Arcanum installer for Linux and macOS.
#
#   curl -fsSL https://arcanum.rafaeldourado.tech/install.sh | sh
#   curl -fsSL https://arcanum.rafaeldourado.tech/install.sh | sh -s -- --systemd
#
# What it does:
#   1. Detects your OS and CPU, downloads the latest release from GitHub and checks its
#      SHA-256 against the published checksums.txt.
#   2. Installs the binary to /usr/local/bin/arcanum (or $ARCANUM_BIN_DIR).
#   3. With --systemd (Linux): creates the "arcanum" system user, /var/lib/arcanum for the
#      sessions, /etc/arcanum/arcanum.env with a random API key and dashboard password
#      (readable only by root), and starts the service.
#
# Options: --version vX.Y.Z   --systemd   --help
set -eu

REPO="rafaeldourado9/arcanum"
VERSION="${ARCANUM_VERSION:-latest}"
BIN_DIR="${ARCANUM_BIN_DIR:-/usr/local/bin}"
SYSTEMD=0

say() { printf '%s\n' "$*"; }
fail() { printf 'error: %s\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --systemd) SYSTEMD=1; shift ;;
    -h|--help) sed -n '2,17p' "$0" 2>/dev/null || true; exit 0 ;;
    *) fail "unknown option: $1" ;;
  esac
done

need() { command -v "$1" >/dev/null 2>&1 || fail "'$1' is required"; }
need curl
need tar
need uname

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  linux|darwin) ;;
  *) fail "unsupported OS: $os (use Docker, or download the Windows .zip from GitHub Releases)" ;;
esac
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) fail "unsupported CPU: $arch" ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
  [ -n "$VERSION" ] || fail "could not find the latest release"
fi
num="${VERSION#v}"
file="arcanum_${num}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$VERSION"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

say "Downloading Arcanum $VERSION ($os/$arch)..."
curl -fsSL -o "$tmp/$file" "$base/$file" || fail "download failed: $base/$file"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt" || fail "could not download checksums.txt"

expected="$(grep " $file\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
[ -n "$expected" ] || fail "$file is not listed in checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$file" | cut -d' ' -f1)"
else
  actual="$(shasum -a 256 "$tmp/$file" | cut -d' ' -f1)"
fi
[ "$expected" = "$actual" ] || fail "checksum mismatch for $file (expected $expected, got $actual)"
say "Checksum OK."

tar -xzf "$tmp/$file" -C "$tmp"

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
  if [ -w "$BIN_DIR" ]; then SUDO=""; elif command -v sudo >/dev/null 2>&1; then SUDO="sudo"; else fail "run as root or set ARCANUM_BIN_DIR to a writable folder"; fi
fi
$SUDO install -m 0755 "$tmp/arcanum" "$BIN_DIR/arcanum"
say "Installed $BIN_DIR/arcanum ($("$BIN_DIR/arcanum" version))"

if [ "$SYSTEMD" -eq 0 ]; then
  say ""
  say "Start it:"
  say "  GATEWAY_API_KEY=\"\$(openssl rand -hex 32)\" GATEWAY_DB_PATH=\$HOME/.arcanum arcanum"
  say "Then open http://localhost:3100 (dashboard) and http://localhost:3100/docs (API)."
  say "Run as a service: re-run with --systemd. Guide: https://arcanum.rafaeldourado.tech/en/"
  exit 0
fi

[ "$os" = "linux" ] || fail "--systemd is only for Linux"
command -v systemctl >/dev/null 2>&1 || fail "systemd not found"

if ! id arcanum >/dev/null 2>&1; then
  $SUDO useradd --system --home-dir /var/lib/arcanum --shell /usr/sbin/nologin arcanum
fi
$SUDO install -d -m 0700 -o arcanum -g arcanum /var/lib/arcanum
$SUDO install -d -m 0755 /etc/arcanum

if [ ! -f /etc/arcanum/arcanum.env ]; then
  rand() { od -An -N32 -tx1 /dev/urandom | tr -d ' \n'; }
  key="$(rand)"
  pass="$(rand | cut -c1-24)"
  $SUDO sh -c "umask 077; cat > /etc/arcanum/arcanum.env" <<EOF
# Arcanum configuration. Reference: https://arcanum.rafaeldourado.tech/en/configuration/
GATEWAY_API_KEY=$key
GATEWAY_DASHBOARD_PASSWORD=$pass
GATEWAY_DB_PATH=/var/lib/arcanum
GATEWAY_LOG_FORMAT=json
# GATEWAY_WEBHOOK_FORWARD_URL=https://your-app.example.com/webhooks/whatsapp/{instance}
# GATEWAY_META_APP_SECRET=
EOF
  created=1
else
  created=0
fi

$SUDO install -m 0644 "$tmp/deploy/systemd/arcanum.service" /etc/systemd/system/arcanum.service
$SUDO systemctl daemon-reload
$SUDO systemctl enable --now arcanum

say ""
say "Arcanum is running as a systemd service (systemctl status arcanum)."
say "Dashboard: http://<this-server>:3100  (put it behind HTTPS before exposing it)"
if [ "$created" -eq 1 ]; then
  say "Dashboard password and API key are in /etc/arcanum/arcanum.env (root only):"
  say "  sudo grep -E 'API_KEY|PASSWORD' /etc/arcanum/arcanum.env"
fi
