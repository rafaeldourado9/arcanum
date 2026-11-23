# Arcanum installer for Windows (PowerShell 5.1+).
#
#   irm https://arcanum.rafaeldourado.tech/install.ps1 | iex
#
# Works even when running scripts is disabled (the installer runs in memory).
# What it does:
#   1. Downloads arcanum.exe for your CPU from the latest GitHub release.
#   2. Checks its SHA-256 against the published checksums.txt.
#   3. Installs it to %LOCALAPPDATA%\Programs\Arcanum and adds that folder to your user PATH.
# Nothing needs administrator rights. Options (set before running):
#   $env:ARCANUM_VERSION = "v1.1.0"        pin a release
#   $env:ARCANUM_INSTALL_DIR = "C:\tools"   install somewhere else
#   $env:ARCANUM_NO_PATH = "1"              do not touch the user PATH

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"   # Invoke-WebRequest is much faster without the progress bar
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$repo = "rafaeldourado9/arcanum"

function Fail($msg) { Write-Host "error: $msg" -ForegroundColor Red; throw $msg }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    default { Fail "unsupported CPU: $($env:PROCESSOR_ARCHITECTURE)" }
}

$tag = $env:ARCANUM_VERSION
if (-not $tag) {
    $tag = (Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -Headers @{ "User-Agent" = "arcanum-installer" }).tag_name
    if (-not $tag) { Fail "could not find the latest release" }
}
$num = $tag.TrimStart("v")
$file = "arcanum_${num}_windows_${arch}.exe"
$base = "https://github.com/$repo/releases/download/$tag"

$tmp = Join-Path ([IO.Path]::GetTempPath()) ("arcanum-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading Arcanum $tag (windows/$arch)..."
    Invoke-WebRequest -Uri "$base/$file" -OutFile (Join-Path $tmp $file) -UseBasicParsing
    Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile (Join-Path $tmp "checksums.txt") -UseBasicParsing

    $line = Get-Content (Join-Path $tmp "checksums.txt") | Where-Object { $_ -match ("\s\*?" + [Regex]::Escape($file) + "$") } | Select-Object -First 1
    if (-not $line) { Fail "$file is not listed in checksums.txt" }
    $expected = ($line -split "\s+")[0].ToLower()
    $actual = (Get-FileHash -Algorithm SHA256 -Path (Join-Path $tmp $file)).Hash.ToLower()
    if ($expected -ne $actual) { Fail "checksum mismatch for $file (expected $expected, got $actual)" }
    Write-Host "Checksum OK."

    $dest = $env:ARCANUM_INSTALL_DIR
    if (-not $dest) { $dest = Join-Path $env:LOCALAPPDATA "Programs\Arcanum" }
    New-Item -ItemType Directory -Force -Path $dest | Out-Null
    Copy-Item -Force (Join-Path $tmp $file) (Join-Path $dest "arcanum.exe")
}
finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

if (-not $env:ARCANUM_NO_PATH) {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not $userPath) { $userPath = "" }
    if (($userPath -split ";") -notcontains $dest) {
        [Environment]::SetEnvironmentVariable("Path", ($userPath.TrimEnd(";") + ";" + $dest).TrimStart(";"), "User")
    }
    $env:Path = "$env:Path;$dest"
}

$version = & (Join-Path $dest "arcanum.exe") version
Write-Host ""
Write-Host "Installed $version to $dest" -ForegroundColor Green
Write-Host ""
Write-Host "Start it (dashboard on http://localhost:3100):"
Write-Host '  $env:GATEWAY_DASHBOARD_PASSWORD = "choose-a-long-password"'
Write-Host '  $env:GATEWAY_DB_PATH = "$env:LOCALAPPDATA\Arcanum"'
Write-Host "  arcanum"
Write-Host ""
Write-Host "Open a new terminal if 'arcanum' is not found. Guide: https://arcanum.rafaeldourado.tech"
