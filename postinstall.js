#!/usr/bin/env node
"use strict";

const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");
const https = require("https");
const http = require("http");

const VERSION = require("./package.json").version;
const REPO = "GeiserX/genieacs-mcp";
const BIN_NAME = process.platform === "win32" ? "genieacs-mcp.exe" : "genieacs-mcp";
const BIN_DIR = path.join(__dirname, "bin");
const BIN_PATH = path.join(BIN_DIR, BIN_NAME);

function getPlatformArch() {
  const platform = process.platform;
  const arch = process.arch;

  const platformMap = { darwin: "darwin", linux: "linux", win32: "windows" };
  const archMap = { x64: "amd64", arm64: "arm64" };

  const p = platformMap[platform];
  const a = archMap[arch];

  if (!p || !a) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }

  return { platform: p, arch: a };
}

function getAssetName() {
  const { platform, arch } = getPlatformArch();
  const ext = platform === "windows" ? "zip" : "tar.gz";
  return `genieacs-mcp_${VERSION}_${platform}_${arch}.${ext}`;
}

function downloadFile(url) {
  return new Promise((resolve, reject) => {
    const get = url.startsWith("https") ? https.get : http.get;
    get(url, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        return downloadFile(res.headers.location).then(resolve, reject);
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`Download failed: HTTP ${res.statusCode}`));
      }
      const chunks = [];
      res.on("data", (chunk) => chunks.push(chunk));
      res.on("end", () => resolve(Buffer.concat(chunks)));
      res.on("error", reject);
    }).on("error", reject);
  });
}

async function extract(buffer, assetName) {
  fs.mkdirSync(BIN_DIR, { recursive: true });

  if (assetName.endsWith(".zip")) {
    const tmpZip = path.join(BIN_DIR, "tmp.zip");
    fs.writeFileSync(tmpZip, buffer);
    execSync(`unzip -o "${tmpZip}" "${BIN_NAME}" -d "${BIN_DIR}"`, { stdio: "ignore" });
    fs.unlinkSync(tmpZip);
  } else {
    const tmpTar = path.join(BIN_DIR, "tmp.tar.gz");
    fs.writeFileSync(tmpTar, buffer);
    execSync(`tar -xzf "${tmpTar}" -C "${BIN_DIR}" "${BIN_NAME}"`, { stdio: "ignore" });
    fs.unlinkSync(tmpTar);
  }

  if (process.platform !== "win32") {
    fs.chmodSync(BIN_PATH, 0o755);
  }
}

async function main() {
  if (fs.existsSync(BIN_PATH)) {
    console.log("genieacs-mcp binary already exists, skipping download.");
    return;
  }

  const assetName = getAssetName();
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${assetName}`;

  console.log(`Downloading genieacs-mcp v${VERSION} for ${process.platform}-${process.arch}...`);
  const buffer = await downloadFile(url);

  console.log("Extracting binary...");
  await extract(buffer, assetName);

  console.log(`Installed genieacs-mcp to ${BIN_PATH}`);
}

main().catch((err) => {
  console.error(`Failed to install genieacs-mcp: ${err.message}`);
  process.exit(1);
});
