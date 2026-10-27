#!/usr/bin/env node
"use strict";

require("./install")
  .ensureBinary()
  .catch((err) => {
    process.stderr.write(`Failed to install genieacs-mcp: ${err.message}\n`);
    process.exit(1);
  });
