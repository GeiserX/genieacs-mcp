"use strict";

const test = require("node:test");
const assert = require("node:assert");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { assetName, parseChecksums, expectedSha256, checkRedirect } = require("./install");

const SUM = "a".repeat(64);

test("parseChecksums reads goreleaser lines", () => {
  const m = parseChecksums(`${SUM}  genieacs-mcp_1.0.0_linux_amd64.tar.gz\n${"b".repeat(64)} *other.zip\nnot a line\n`);
  assert.strictEqual(m.get("genieacs-mcp_1.0.0_linux_amd64.tar.gz"), SUM);
  assert.strictEqual(m.get("other.zip"), "b".repeat(64));
  assert.strictEqual(m.size, 2);
});

test("expectedSha256 refuses when checksums.txt is missing or has no entry", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "gmcp-"));
  const file = path.join(dir, "checksums.txt");
  assert.throws(() => expectedSha256("x.tar.gz", file), /missing/);
  fs.writeFileSync(file, `${SUM}  y.tar.gz\n`);
  assert.throws(() => expectedSha256("x.tar.gz", file), /no entry/);
  assert.strictEqual(expectedSha256("y.tar.gz", file), SUM);
  fs.rmSync(dir, { recursive: true });
});

test("checkRedirect allows only https to GitHub hosts", () => {
  const base = "https://github.com/GeiserX/genieacs-mcp/releases/download/v1/x.tar.gz";
  assert.ok(checkRedirect("https://release-assets.githubusercontent.com/a?b=c", base));
  assert.ok(checkRedirect("https://objects.githubusercontent.com/a", base));
  assert.throws(() => checkRedirect("http://objects.githubusercontent.com/a", base), /refusing/);
  assert.throws(() => checkRedirect("https://evil.example/a", base), /refusing/);
  assert.throws(() => checkRedirect("https://githubusercontent.com.evil.example/a", base), /refusing/);
});

test("assetName maps platforms to release assets", () => {
  assert.match(assetName("linux", "x64"), /_linux_amd64\.tar\.gz$/);
  assert.match(assetName("win32", "arm64"), /_windows_arm64\.zip$/);
  assert.throws(() => assetName("aix", "ppc64"), /unsupported/);
});
