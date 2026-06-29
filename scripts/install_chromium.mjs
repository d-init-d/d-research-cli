#!/usr/bin/env node
import { mkdir } from 'node:fs/promises';
import path from 'node:path';

const target = process.argv[2];
if (!target) {
  console.error('usage: node install_chromium.mjs <targetDir>');
  process.exit(1);
}
await mkdir(target, { recursive: true });
const marker = path.join(target, '.installed');
await import('node:fs/promises').then((fs) => fs.writeFile(marker, new Date().toISOString(), 'utf8'));
console.log('chromium provision marker written to', marker);
console.log('Run `npx playwright install chromium` in release packaging for full browser binaries.');