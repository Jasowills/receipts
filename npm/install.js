'use strict';
// Postinstall: download the pinned Receipts release for this platform,
// verify its SHA-256 checksum against the release manifest, cache it
// under vendor/, and never fetch "latest". No elevated privileges needed.
const crypto = require('crypto');
const fs = require('fs');
const https = require('https');
const os = require('os');
const path = require('path');

const OWNER = 'OWNER';
const REPO = 'receipts';

function assetFor(platform, arch, version) {
  const table = {
    'darwin:arm64': `receipts_${version}_Darwin_arm64.tar.gz`,
    'darwin:x64': `receipts_${version}_Darwin_x86_64.tar.gz`,
    'linux:x64': `receipts_${version}_Linux_x86_64.tar.gz`,
    'linux:arm64': `receipts_${version}_Linux_arm64.tar.gz`,
  };
  const asset = table[`${platform}:${arch}`];
  if (!asset) {
    throw new Error(`receipts: unsupported platform ${platform}-${arch}`);
  }
  return asset;
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    https.get(url, (res) => {
      if (res.statusCode !== 200) {
        reject(new Error(`receipts: download status ${res.statusCode} for ${url}`));
        return;
      }
      const out = fs.createWriteStream(dest);
      res.pipe(out);
      out.on('finish', () => out.close(resolve));
      out.on('error', reject);
    }).on('error', reject);
  });
}

function sha256File(file) {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash('sha256');
    fs.createReadStream(file)
      .on('data', (chunk) => hash.update(chunk))
      .on('end', () => resolve(hash.digest('hex')))
      .on('error', reject);
  });
}

async function install() {
  const pkg = JSON.parse(fs.readFileSync(path.join(__dirname, 'package.json'), 'utf8'));
  const version = pkg.version;
  const asset = assetFor(os.platform(), os.arch(), version);
  const base = `https://github.com/${OWNER}/${REPO}/releases/download/v${version}`;
  const vendor = path.join(__dirname, 'vendor');
  fs.mkdirSync(vendor, { recursive: true });
  const tgz = path.join(vendor, asset);
  const sums = path.join(vendor, 'checksums.txt');
  await download(`${base}/${asset}`, tgz);
  await download(`${base}/checksums.txt`, sums);
  const want = fs.readFileSync(sums, 'utf8')
    .split('\n')
    .map((line) => line.trim().split(/\s+/))
    .find(([, name]) => name === asset);
  if (!want) {
    throw new Error(`receipts: ${asset} missing from checksum manifest`);
  }
  const got = await sha256File(tgz);
  if (got !== want[0]) {
    throw new Error(`receipts: checksum mismatch for ${asset}`);
  }
  const { execFileSync } = require('child_process');
  execFileSync('tar', ['-xzf', tgz, '-C', vendor]);
  console.log(`receipts: installed ${asset} (checksum ok)`);
}

if (require.main === module) {
  install().catch((err) => {
    console.error(err.message);
    process.exit(1);
  });
}

module.exports = { assetFor, sha256File };
