#!/usr/bin/env node

// Since npm installs the correct platform package via optionalDependencies,
// this script just finds the installed platform binary and links it to bin/.

const { platform, arch } = require('os');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

function isMusl() {
  try {
    const output = execSync('ldd --version 2>&1', { encoding: 'utf8', timeout: 3000 });
    return output.includes('musl');
  } catch {
    // ldd not found or error, check for musl library
    try {
      return fs.readdirSync('/lib').some(f => f.startsWith('ld-musl'));
    } catch {
      return false;
    }
  }
}

function getPlatformKey() {
  const p = platform();
  const a = arch();
  if (p === 'linux' && isMusl()) {
    return `linux-musl-${a}`;
  }
  return `${p}-${a}`;
}

const PLATFORM_PACKAGES = {
  'linux-x64':        'vibecoding-installer-linux-x64',
  'linux-arm64':      'vibecoding-installer-linux-arm64',
  'linux-musl-x64':   'vibecoding-installer-linux-musl-x64',
  'darwin-x64':       'vibecoding-installer-darwin-x64',
  'darwin-arm64':     'vibecoding-installer-darwin-arm64',
  'win32-x64':        'vibecoding-installer-win32-x64',
  'win32-arm64':      'vibecoding-installer-win32-arm64',
};

function main() {
  const key = getPlatformKey();
  const pkgName = PLATFORM_PACKAGES[key];

  if (!pkgName) {
    console.error(`Error: Unsupported platform: ${key}`);
    console.error(`Supported: ${Object.keys(PLATFORM_PACKAGES).join(', ')}`);
    process.exit(1);
  }

  // Find the platform package in node_modules
  let platformPkgDir;
  try {
    platformPkgDir = path.dirname(require.resolve(pkgName + '/package.json'));
  } catch {
    console.error(`Error: Platform package '${pkgName}' not installed.`);
    console.error('Your platform may not be supported, or the optional dependency was skipped.');
    process.exit(1);
  }

  const isWindows = platform() === 'win32';
  const srcName = isWindows ? 'vibecoding.exe' : 'vibecoding';
  const destName = isWindows ? 'vibecoding.exe' : 'vibecoding';

  const srcPath = path.join(platformPkgDir, 'bin', srcName);
  const destPath = path.join(__dirname, 'bin', destName);

  if (!fs.existsSync(srcPath)) {
    console.error(`Error: Binary not found at ${srcPath}`);
    process.exit(1);
  }

  // Ensure bin directory exists
  const binDir = path.join(__dirname, 'bin');
  fs.mkdirSync(binDir, { recursive: true });

  // Copy binary
  fs.copyFileSync(srcPath, destPath);

  if (!isWindows) {
    fs.chmodSync(destPath, '755');
  }

  console.log(`VibeCoding installed successfully (${key})`);
  console.log('');
  console.log('  Install directory: ' + destPath);

  // Config directory
  const homeDir = require('os').homedir();
  const configDir = isWindows
    ? path.join(process.env.APPDATA || path.join(homeDir, 'AppData', 'Roaming'), 'vibecoding')
    : path.join(homeDir, '.vibecoding');
  console.log('  Config directory : ' + configDir);
  console.log('    - Settings file: ' + path.join(configDir, 'settings.json'));
  console.log('');

  if (!isWindows) {
    console.log('  If "vibecoding" command is not found, add to your PATH:');
    console.log('');
    console.log('    # Bash:');
    console.log(`    export PATH="${path.dirname(destPath)}:$PATH"`);
    console.log('');
    console.log('    # Zsh:');
    console.log(`    export PATH="${path.dirname(destPath)}:$PATH"`);
    console.log('');
    console.log('    # Fish:');
    console.log(`    set -gx PATH ${path.dirname(destPath)} $PATH`);
  } else {
    console.log('  If "vibecoding" command is not found, add to your PATH:');
    console.log('');
    console.log('    # PowerShell (current session):');
    console.log(`    $env:Path += ";${path.dirname(destPath)}"`);
    console.log('');
    console.log('    # PowerShell (permanent):');
    console.log(`    [Environment]::SetEnvironmentVariable('Path', $env:Path + ';${path.dirname(destPath)}', 'User')`);
    console.log('');
    console.log('    # CMD (permanent):');
    console.log(`    setx Path "%Path%;${path.dirname(destPath)}"`);
  }
  console.log('');
  console.log('  Or run directly: npx vibecoding');
}

main();
