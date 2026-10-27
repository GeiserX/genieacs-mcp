"use strict";

// Downloads the genieacs-mcp release binary for this platform and checks it
// against checksums.txt, which the release workflow puts inside the npm
// package. The checksum therefore comes from the package npm verified (with
// its provenance), not from the GitHub release the binary is fetched from.
// Used by postinstall.js and, when install scripts were skipped (npm 12 skips
// them by default), by run.js on first start.

const { execFileSync } = require("child_process");
const crypto = require("crypto");
const fs = require("fs");
const https = require("https");
const path = require("path");

const VERSION = require("./package.json").version;
const REPO = "GeiserX/genieacs-mcp";
const BIN_NAME = process.platform === "win32" ? "genieacs-mcp.exe" : "genieacs-mcp";
const BIN_DIR = path.join(__dirname, "bin");
const BIN_PATH = path.join(BIN_DIR, BIN_NAME);
const CHECKSUMS_PATH = path.join(__dirname, "checksums.txt");
const MAX_REDIRECTS = 5;
const MAX_DOWNLOAD_BYTES = 200 * 1024 * 1024;
// A connection that sends nothing for this long is dropped, so a stalled
// download fails with a message instead of hanging the MCP client on start.
const DOWNLOAD_IDLE_MS = 30 * 1000;

function log(msg) {
  // stderr only: run.js shares stdout with the MCP client.
  process.stderr.write(`genieacs-mcp: ${msg}\n`);
}

function assetName(platform = process.platform, arch = process.arch) {
  const p = { darwin: "darwin", linux: "linux", win32: "windows" }[platform];
  const a = { x64: "amd64", arm64: "arm64" }[arch];
  if (!p || !a) {
    throw new Error(`unsupported platform: ${platform}-${arch}`);
  }
  return `genieacs-mcp_${VERSION}_${p}_${a}.${p === "windows" ? "zip" : "tar.gz"}`;
}

// parseChecksums reads goreleaser's checksums.txt ("<sha256>  <file>" lines).
function parseChecksums(text) {
  const out = new Map();
  for (const line of text.split(/\r?\n/)) {
    const m = line.trim().match(/^([0-9a-f]{64})\s+\*?(\S+)$/i);
    if (m) out.set(m[2], m[1].toLowerCase());
  }
  return out;
}

function expectedSha256(asset, checksumsPath = CHECKSUMS_PATH) {
  if (!fs.existsSync(checksumsPath)) {
    throw new Error("checksums.txt is missing from the package; refusing to install a binary that cannot be verified");
  }
  const sum = parseChecksums(fs.readFileSync(checksumsPath, "utf8")).get(asset);
  if (!sum) {
    throw new Error(`checksums.txt has no entry for ${asset}; refusing to install it`);
  }
  return sum;
}

// checkRedirect allows only https hops to GitHub and its asset hosts.
function checkRedirect(location, base) {
  let u;
  try {
    u = new URL(location, base);
  } catch {
    throw new Error("download redirected to an invalid URL");
  }
  const host = u.hostname.toLowerCase();
  const allowed = host === "github.com" || host.endsWith(".githubusercontent.com");
  if (u.protocol !== "https:" || !allowed) {
    throw new Error(`refusing download redirect to ${u.protocol}//${host}`);
  }
  return u.toString();
}

// download streams url to dest, following at most MAX_REDIRECTS https
// redirects, and resolves with the sha256 of what it wrote. It fails when the
// connection is idle for idleMs.
function download(url, dest, redirects = 0, idleMs = DOWNLOAD_IDLE_MS) {
  return new Promise((resolve, reject) => {
    const req = https.get(url, { headers: { "User-Agent": "genieacs-mcp-installer" } }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume();
        if (redirects >= MAX_REDIRECTS) {
          return reject(new Error("too many download redirects"));
        }
        let next;
        try {
          next = checkRedirect(res.headers.location, url);
        } catch (err) {
          return reject(err);
        }
        return download(next, dest, redirects + 1, idleMs).then(resolve, reject);
      }
      if (res.statusCode !== 200) {
        res.resume();
        return reject(new Error(`download failed: HTTP ${res.statusCode}`));
      }
      const hash = crypto.createHash("sha256");
      const out = fs.createWriteStream(dest, { mode: 0o600 });
      let size = 0;
      res.on("data", (chunk) => {
        size += chunk.length;
        if (size > MAX_DOWNLOAD_BYTES) {
          res.destroy(new Error("download is larger than expected; refusing it"));
          return;
        }
        hash.update(chunk);
      });
      res.on("error", (err) => {
        out.destroy();
        reject(err);
      });
      res.pipe(out);
      out.on("error", reject);
      out.on("finish", () => resolve(hash.digest("hex")));
    });
    req.setTimeout(idleMs, () => {
      req.destroy(new Error(`download stalled: no data for ${idleMs / 1000}s`));
    });
    req.on("error", reject);
  });
}

function extract(archive, asset, destDir) {
  // tar reads both formats: bsdtar ships with Windows 10 and later.
  const args = asset.endsWith(".zip") ? ["-xf", archive] : ["-xzf", archive];
  execFileSync("tar", [...args, "-C", destDir, BIN_NAME], { stdio: "ignore" });
}

async function ensureBinary() {
  if (fs.existsSync(BIN_PATH)) {
    return BIN_PATH;
  }
  const asset = assetName();
  const want = expectedSha256(asset);
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${asset}`;

  fs.mkdirSync(BIN_DIR, { recursive: true });
  const work = fs.mkdtempSync(path.join(BIN_DIR, ".download-"));
  try {
    const archive = path.join(work, asset);
    log(`downloading v${VERSION} for ${process.platform}-${process.arch}`);
    const got = await download(url, archive);
    if (got !== want) {
      throw new Error(`checksum mismatch for ${asset}: expected ${want}, got ${got}; refusing to install it`);
    }
    extract(archive, asset, work);
    const extracted = path.join(work, BIN_NAME);
    if (process.platform !== "win32") {
      fs.chmodSync(extracted, 0o755);
    }
    fs.renameSync(extracted, BIN_PATH);
    log(`installed ${BIN_PATH}`);
    return BIN_PATH;
  } finally {
    fs.rmSync(work, { recursive: true, force: true });
  }
}

module.exports = { BIN_PATH, assetName, parseChecksums, expectedSha256, checkRedirect, download, ensureBinary };
