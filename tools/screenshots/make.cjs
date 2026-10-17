// Takes the tutorial screenshots: node tools/screenshots/make.cjs [name ...]
// It starts the app in a plain browser with a demo book (stub.ts), and uses Microsoft Edge or Chrome without a window to take each picture.
// The pictures go to docs/images. Run it again after a screen changes.
const { spawn, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const http = require('node:http');
const path = require('node:path');

const root = path.resolve(__dirname, '../..');
const out = path.join(root, 'docs', 'images');
const SIZE = '1400,850';
const ALL = ['bookshelf', 'dashboard', 'write-book', 'write-text', 'characters', 'places', 'maps', 'link', 'scene-panel', 'quick-actions', 'comments', 'notes', 'board', 'statistics', 'checks', 'backup', 'export', 'sources'];
const os = require('node:os');
const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'typewriter-shots-')); // a fresh profile, so a running browser does not take the job
const names = process.argv.slice(2).length ? process.argv.slice(2) : ALL;

const browser = [
  'C:/Program Files/Google/Chrome/Application/chrome.exe', '/usr/bin/google-chrome', '/usr/bin/chromium',
  'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe', 'C:/Program Files/Microsoft/Edge/Application/msedge.exe',
].find((p) => fs.existsSync(p));
if (!browser) { console.error('No Edge or Chrome found.'); process.exit(1); }

const up = () => new Promise((res) => http.get('http://127.0.0.1:5199/', (r) => { r.resume(); res(true); }).on('error', () => res(false)));
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

(async () => {
  fs.mkdirSync(out, { recursive: true });
  const vite = spawn(process.execPath, [path.join(root, 'frontend/node_modules/vite/bin/vite.js'), '--config', path.join(__dirname, 'vite.config.mjs')], { cwd: path.join(root, 'frontend'), stdio: 'ignore' });
  try {
    for (let i = 0; i < 60 && !(await up()); i++) await sleep(500);
    const base = 'http://127.0.0.1:5199/@fs/' + path.join(__dirname, 'shots.html').replace(/\\/g, '/');
    for (const n of names) {
      const file = path.join(out, n + '.png');
      const r = spawnSync(browser, ['--headless=new', '--disable-gpu', '--hide-scrollbars', '--window-size=' + SIZE, '--virtual-time-budget=20000', '--screenshot=' + file, base + '?s=' + n], { stdio: 'pipe', timeout: 120000, windowsHide: true });
      console.log(fs.existsSync(file) ? 'ok   ' + n : 'FAIL ' + n, r.status, r.error ? r.error.message : '');
    }
  } finally { vite.kill(); try { fs.rmSync(profile, { recursive: true, force: true }); } catch { /* Edge may still hold a file. The system cleans the temp folder later. */ } }
})();
