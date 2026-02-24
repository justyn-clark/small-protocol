#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const binaryPath = path.resolve(__dirname, "..", "vendor", "small");

if (!fs.existsSync(binaryPath)) {
  console.error("SMALL binary missing. Reinstall or run npm rebuild @small-protocol/small");
  process.exit(1);
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: "inherit"
});

if (result.error) {
  console.error(`Failed to launch SMALL: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status === null ? 1 : result.status);
