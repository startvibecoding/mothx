#!/usr/bin/env node

const { platform, arch } = require('os');
const fs = require('fs');
const path = require('path');

// Platform and architecture mapping
const PLATFORM_MAP = {
  'linux': 'linux',
  'darwin': 'darwin',
  'win32': 'windows'
};

const ARCH_MAP = {
  'x64': 'amd64',
  'arm64': 'arm64'
};

function getBinaryName() {
  const os = PLATFORM_MAP[platform()];
  const architecture = ARCH_MAP[arch()];
  
  if (!os || !architecture) {
    throw new Error(`Unsupported platform: ${platform()} ${arch()}`);
  }
  
  const ext = os === 'windows' ? '.exe' : '';
  return `vibecoding-${os}-${architecture}${ext}`;
}

function main() {
  try {
    const sourceName = getBinaryName();
    const isWindows = platform() === 'win32';
    const destName = isWindows ? 'vibecoding.exe' : 'vibecoding';
    
    const binDir = path.join(__dirname, 'bin');
    const sourcePath = path.join(binDir, sourceName);
    const destPath = path.join(binDir, destName);
    
    // Check if source binary exists
    if (!fs.existsSync(sourcePath)) {
      console.error(`Error: Binary not found for your platform: ${sourceName}`);
      console.error('Supported platforms: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64');
      process.exit(1);
    }
    
    // Copy binary to the expected name (vibecoding or vibecoding.exe)
    fs.copyFileSync(sourcePath, destPath);
    
    // Make binary executable on Unix
    if (!isWindows) {
      fs.chmodSync(destPath, '755');
    }
    
    console.log(`VibeCoding installed successfully for ${platform()}/${arch()}`);
  } catch (error) {
    console.error('Installation error:', error.message);
    process.exit(1);
  }
}

main();
