#!/usr/bin/env node
import { createInterface } from 'node:readline';

const rl = createInterface({ input: process.stdin });
let buffer = '';

rl.on('line', async (line) => {
  buffer += line;
  let req;
  try {
    req = JSON.parse(buffer);
    buffer = '';
  } catch {
    return;
  }
  const timeout = req.timeout_ms ?? 30000;
  const resp = { id: req.id, ok: false };
  try {
    if (req.command === 'navigate') {
      if (!req.url) throw new Error('url required');
      const { chromium } = await import('playwright');
      const browser = await chromium.launch({
        headless: req.options?.headless !== false,
        executablePath: req.options?.browser_path ? undefined : undefined,
      });
      const page = await browser.newPage();
      await page.goto(req.url, { timeout, waitUntil: 'domcontentloaded' });
      resp.ok = true;
      resp.title = await page.title();
      resp.text = (await page.textContent('body'))?.slice(0, 2000) ?? '';
      await browser.close();
    } else {
      resp.error = `unknown command ${req.command}`;
    }
  } catch (err) {
    const msg = String(err?.message ?? err);
    resp.error = msg;
    if (/timeout/i.test(msg)) resp.blocker = 'timeout';
    else if (/403|429|captcha|paywall/i.test(msg)) resp.blocker = 'access';
  }
  process.stdout.write(JSON.stringify(resp) + '\n');
});