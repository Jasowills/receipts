#!/usr/bin/env node
'use strict';
// bin entry: forward argv, stdio, and the exit code to the vendored binary.
const { spawnSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const vendor = path.join(__dirname, '..', 'vendor');
const binary = path.join(vendor, process.platform === 'win32' ? 'receipts.exe' : 'receipts');
if (!fs.existsSync(binary)) {
  console.error('receipts: vendored binary missing; reinstall the package');
  process.exit(1);
}
const child = spawnSync(binary, process.argv.slice(2), { stdio: 'inherit' });
process.exit(child.status === null ? 1 : child.status);
