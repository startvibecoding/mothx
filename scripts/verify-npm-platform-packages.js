#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');

const repoRoot = path.resolve(__dirname, '..');
const pkgPath = process.argv[2]
  ? path.resolve(process.argv[2])
  : path.join(repoRoot, 'npm', 'mothx', 'package.json');
const registry = (process.env.NPM_REGISTRY || process.env.npm_config_registry || 'https://registry.npmjs.org').replace(/\/+$/, '');

function readPackageJSON(file) {
  return JSON.parse(fs.readFileSync(file, 'utf8'));
}

function checkPackage(name, version) {
  const url = `${registry}/${encodeURIComponent(name)}/${encodeURIComponent(version)}`;
  return new Promise((resolve) => {
    https.get(url, (res) => {
      res.resume();
      resolve({ name, version, ok: res.statusCode === 200, status: res.statusCode });
    }).on('error', (err) => {
      resolve({ name, version, ok: false, error: err.message });
    });
  });
}

async function main() {
  const pkg = readPackageJSON(pkgPath);
  const deps = pkg.optionalDependencies || {};
  const entries = Object.entries(deps).sort(([a], [b]) => a.localeCompare(b));
  if (entries.length === 0) {
    throw new Error(`${pkgPath} has no optionalDependencies`);
  }

  const results = await Promise.all(entries.map(([name, version]) => checkPackage(name, version)));
  const missing = results.filter((result) => !result.ok);
  if (missing.length > 0) {
    console.error(`Missing npm platform packages required by ${pkg.name}@${pkg.version}:`);
    for (const result of missing) {
      const reason = result.error || `HTTP ${result.status}`;
      console.error(`  - ${result.name}@${result.version}: ${reason}`);
    }
    process.exit(1);
  }

  console.log(`Verified ${results.length} platform packages for ${pkg.name}@${pkg.version}`);
}

main().catch((err) => {
  console.error(err.message || err);
  process.exit(1);
});
