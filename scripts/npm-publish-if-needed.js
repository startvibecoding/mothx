#!/usr/bin/env node

const childProcess = require('child_process');
const fs = require('fs');
const http = require('http');
const https = require('https');
const path = require('path');

function usage() {
  console.error('Usage: npm-publish-if-needed.js [--tag <tag>] [--registry <url>] <package-dir> [-- <npm publish args>]');
}

function parseArgs(argv) {
  const args = {
    tag: 'latest',
    registry: process.env.NPM_REGISTRY || process.env.npm_config_registry || 'https://registry.npmjs.org',
    packageDir: '',
    publishArgs: [],
  };

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === '--') {
      args.publishArgs = argv.slice(i + 1);
      break;
    }
    if (arg === '--tag') {
      i += 1;
      if (i >= argv.length) {
        throw new Error('--tag requires a value');
      }
      args.tag = argv[i];
      continue;
    }
    if (arg === '--registry') {
      i += 1;
      if (i >= argv.length) {
        throw new Error('--registry requires a value');
      }
      args.registry = argv[i];
      continue;
    }
    if (arg.startsWith('--')) {
      throw new Error(`Unknown option: ${arg}`);
    }
    if (args.packageDir) {
      throw new Error(`Unexpected extra package directory: ${arg}`);
    }
    args.packageDir = arg;
  }

  if (!args.packageDir) {
    args.packageDir = process.cwd();
  }
  args.packageDir = path.resolve(args.packageDir);
  args.registry = args.registry.replace(/\/+$/, '');
  return args;
}

function readPackageJSON(packageDir) {
  const pkgPath = path.join(packageDir, 'package.json');
  const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
  if (!pkg.name || !pkg.version) {
    throw new Error(`${pkgPath} must contain name and version`);
  }
  return pkg;
}

function packageVersionURL(registry, name, version) {
  return `${registry}/${encodeURIComponent(name)}/${encodeURIComponent(version)}`;
}

function requestStatus(url) {
  return new Promise((resolve, reject) => {
    const parsed = new URL(url);
    const client = parsed.protocol === 'http:' ? http : https;
    const req = client.get(parsed, {
      headers: {
        Accept: 'application/json',
        'User-Agent': 'mothx-release-script',
      },
    }, (res) => {
      res.resume();
      res.on('end', () => resolve(res.statusCode));
    });
    req.setTimeout(15000, () => {
      req.destroy(new Error(`Timed out checking ${url}`));
    });
    req.on('error', reject);
  });
}

async function packageVersionExists(registry, name, version) {
  const url = packageVersionURL(registry, name, version);
  const status = await requestStatus(url);
  if (status === 200) {
    return true;
  }
  if (status === 404) {
    return false;
  }
  throw new Error(`Failed to check ${name}@${version}: HTTP ${status}`);
}

function publishPackage(packageDir, tag, registry, extraArgs) {
  const npmCmd = process.env.NPM || 'npm';
  const args = ['publish', '--tag', tag, '--registry', registry, ...extraArgs];
  const result = childProcess.spawnSync(npmCmd, args, {
    cwd: packageDir,
    stdio: 'inherit',
    env: process.env,
  });

  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    process.exit(result.status);
  }
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const pkg = readPackageJSON(args.packageDir);
  const label = `${pkg.name}@${pkg.version}`;

  if (await packageVersionExists(args.registry, pkg.name, pkg.version)) {
    console.log(`  Skipping ${label}: already published`);
    return;
  }

  console.log(`  Publishing ${label} with tag ${args.tag}...`);
  publishPackage(args.packageDir, args.tag, args.registry, args.publishArgs);
}

main().catch((err) => {
  usage();
  console.error(err.message || err);
  process.exit(1);
});
