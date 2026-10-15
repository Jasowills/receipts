'use strict';
// Architecture mapping test: every supported platform pair must resolve
// to the GoReleaser asset name, and unknown pairs must throw.
const assert = require('assert');
const { assetFor } = require('../install.js');

assert.strictEqual(assetFor('darwin', 'arm64', '0.1.0'), 'receipts_0.1.0_Darwin_arm64.tar.gz');
assert.strictEqual(assetFor('darwin', 'x64', '0.1.0'), 'receipts_0.1.0_Darwin_x86_64.tar.gz');
assert.strictEqual(assetFor('linux', 'x64', '0.1.0'), 'receipts_0.1.0_Linux_x86_64.tar.gz');
assert.strictEqual(assetFor('linux', 'arm64', '0.1.0'), 'receipts_0.1.0_Linux_arm64.tar.gz');
assert.throws(() => assetFor('win32', 'x64', '0.1.0'), /unsupported platform/);
assert.throws(() => assetFor('darwin', 'ppc', '0.1.0'), /unsupported platform/);
console.log('mapping ok');
