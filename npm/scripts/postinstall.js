#!/usr/bin/env node

// Skip postinstall output in CI or when suppressed
if (process.env.CI || process.env.npm_config_yes || process.env.VIBECODING_SKIP_POSTINSTALL) {
  process.exit(0);
}

const os = require('os');
const path = require('path');

const chalk = (s) => `\x1b[36m${s}\x1b[0m`;  // cyan
const bold  = (s) => `\x1b[1m${s}\x1b[0m`;
const dim   = (s) => `\x1b[2m${s}\x1b[0m`;

function configPath() {
  if (process.platform === 'win32') {
    const appData = process.env.APPDATA || path.join(os.homedir(), 'AppData', 'Roaming');
    return path.join(appData, 'vibecoding', 'settings.json');
  }
  return path.join(os.homedir(), '.vibecoding', 'settings.json');
}

console.log();
console.log(`  ${bold('🚀 VibeCoding')} ${dim('— Terminal AI Coding Assistant')}`);
console.log();
console.log(`  ${bold('Quick Start')}`);
console.log();
console.log(`    vibecoding                        ${dim('Start interactive mode')}`);
console.log(`    vibecoding -P "hello world in Go" ${dim('One-shot mode')}`);
console.log();
console.log(`  ${bold('Setup')}`);
console.log();
console.log(`    In TUI, type ${chalk('/auth')} to add API keys and switch providers`);
console.log();
console.log(`  ${bold('Site')}    ${chalk('https://startvibecoding.work/')}`);
console.log(`  ${bold('Src')}    ${chalk('https://github.com/startvibecoding/vibecoding')}`);
console.log(`  ${bold('Config')}  ${dim(configPath())}`);
console.log();
