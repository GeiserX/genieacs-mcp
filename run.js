#!/usr/bin/env node
"use strict";

const { execFileSync } = require("child_process");
const { ensureBinary } = require("./install");

async function main() {
  // npm 12, pnpm and Bun skip install scripts by default, so the binary may
  // not be there yet: fetch and verify it now.
  const bin = await ensureBinary();
  const env = Object.assign({}, process.env, { TRANSPORT: "stdio" });
  try {
    execFileSync(bin, [], { env, stdio: "inherit" });
  } catch (err) {
    process.exit(err.status || 1);
  }
}

main().catch((err) => {
  process.stderr.write(`genieacs-mcp: ${err.message}\n`);
  process.exit(1);
});
