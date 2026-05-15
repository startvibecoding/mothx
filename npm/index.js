#!/usr/bin/env node

const { platform, arch } = require('os');
const fs = require('fs');
const path = require('path');

// Platform/arch mapping to npm package name
const PLATFORM_PACKAGES = {
  'linux-x64':    'vibecoding-installer-linux-x64',
  'linux-arm64':  'vibecoding-installer-linux-arm64',
  'darwin-x64':   'vibecoding-installer-darwin-x64',
  'darwin-arm64': 'vibecoding-installer-darwin-arm64',
  'win32-x64':    'vibecoding-installer-win32-x64',
  'win32-arm64':  'vibecoding-installer-win32-arm64',
};

const key = `${platform()}-${arch()}`;
const pkgName = PLATFORM_PACKAGES[key];

if (!pkgName) {
  throw new Error(
    `Unsupported platform: ${key}\n` +
    `Supported: ${Object.keys(PLATFORM_PACKAGES).join(', ')}`
  );
}

const isWindows = platform() === 'win32';
const binaryName = isWindows ? 'vibecoding.exe' : 'vibecoding';
const binPath = path.join(path.dirname(require.resolve(pkgName)), 'bin', binaryName);

module.exports = binPath;
