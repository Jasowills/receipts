'use strict';
// Checksum mechanism test: node crypto must agree with the system
// checksum tool on identical bytes.
const assert = require('assert');
const crypto = require('crypto');
const fs = require('fs');
const os = require('os');
const path = require('path');
const { execFileSync } = require('child_process');
const { sha256File } = require('../install.js');

(async () => {
  const file = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'rcpt-')), 'fixture.bin');
  fs.writeFileSync(file, Buffer.alloc(65536, 7));
  const fromNode = await sha256File(file);
  const fromSystem = execFileSync('shasum', ['-a', '256', file], { encoding: 'utf8' }).split(/\s+/)[0];
  assert.strictEqual(fromNode, fromSystem);
  console.log('checksum ok');
})().catch((err) => {
  console.error(err.message);
  process.exit(1);
});
