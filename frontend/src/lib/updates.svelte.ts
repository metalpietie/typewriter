import { Browser, Events } from '@wailsio/runtime';
import { Updates } from '../../bindings/typewriter';
import type { UpdateInfo } from '../../bindings/typewriter/models.js';
import { db } from './db.svelte';

// New versions from GitHub: the check, the download, and the restart. The work is done in Go (updates.go).
// The choices (check at start, a skipped version) belong to this computer, like the colour theme.

const AUTO = 'typewriter-update-check', SKIP = 'typewriter-update-skip';
const read = (key: string): string | null => { try { return localStorage.getItem(key); } catch { return null; } };
const write = (key: string, value: string) => { try { localStorage.setItem(key, value); } catch { /* the choice lasts until the app closes */ } };

export type Phase = 'idle' | 'checking' | 'none' | 'found' | 'downloading' | 'ready' | 'restarting' | 'error';

export const updates = $state<{
  phase: Phase;
  open: boolean;           // the update window is on screen
  info: UpdateInfo | null; // what the last check found
  error: string;
  written: number; total: number; // the download, in bytes
  auto: boolean;           // check when the app starts, and once a day
  later: boolean;          // the new version starts when the window is closed
}>({ phase: 'idle', open: false, info: null, error: '', written: 0, total: 0, auto: read(AUTO) !== 'off', later: false });

export function setAuto(on: boolean) {
  updates.auto = on;
  write(AUTO, on ? 'on' : 'off');
}

const busy = () => updates.phase === 'checking' || updates.phase === 'downloading' || updates.phase === 'restarting';
const messageOf = (e: any) => String(e?.message ?? e).replace(/^.*?Error: /, '');

/**
 * Ask GitHub for a newer version. quiet is the check at start: it opens the window only when there is a version
 * that was not skipped, and it says nothing when GitHub cannot be reached.
 */
export async function checkForUpdates(quiet = false) {
  if (busy() || updates.phase === 'ready') { if (!quiet) updates.open = true; return; }
  updates.phase = 'checking';
  updates.error = '';
  if (!quiet) updates.open = true;
  try {
    const info = await Updates.CheckForUpdate();
    updates.info = info;
    if (!info.available) { updates.phase = 'none'; return; }
    if (quiet && read(SKIP) === info.version) { updates.phase = 'idle'; return; }
    updates.phase = 'found';
    updates.open = true;
  } catch (e) {
    updates.error = messageOf(e);
    updates.phase = 'error';
    if (quiet) updates.phase = 'idle';
  }
}

export function skipThisVersion() {
  if (updates.info?.version) write(SKIP, updates.info.version);
  closeWindow();
}

export function closeWindow() {
  if (updates.phase === 'downloading' || updates.phase === 'restarting') return; // it goes on: the window cannot go away now
  updates.open = false;
  if (updates.phase !== 'ready') updates.phase = 'idle';
}

/** Download and check the new version. Nothing changes until the restart. */
export async function install() {
  updates.phase = 'downloading';
  updates.error = '';
  updates.written = 0;
  updates.total = 0;
  const off = Events.On('wails:updater:download-progress', (e: any) => {
    const p = Array.isArray(e?.data) ? e.data[0] : e?.data;
    if (p) { updates.written = p.written ?? 0; updates.total = p.total ?? 0; }
  });
  try {
    await Updates.InstallUpdate();
    updates.phase = 'ready';
  } catch (e) {
    updates.error = messageOf(e);
    updates.phase = 'error';
  } finally {
    off();
  }
}

/** Save everything, make a copy of the data (in Go), then close and start the new version. */
export async function restartNow() {
  updates.phase = 'restarting';
  updates.error = '';
  try {
    await db.settle();
    await Updates.RestartToUpdate();
  } catch (e) {
    updates.error = messageOf(e);
    updates.phase = 'ready';
  }
}

/** Start the new version when the window is closed, not now. */
export async function restartLater() {
  try {
    await Updates.RestartOnClose();
    updates.later = true;
    updates.open = false;
  } catch (e) {
    updates.error = messageOf(e);
  }
}

export function openReleasePage() {
  const page = updates.info?.page || 'https://github.com/metalpietie/typewriter/releases/latest';
  Browser.OpenURL(page).catch(() => {});
}

let started = false;
/** The check at start (a little later, so the app opens fast) and once a day while the app stays open. */
export function startAutoCheck() {
  if (started) return;
  started = true;
  const quietCheck = () => { if (updates.auto && (updates.phase === 'idle' || updates.phase === 'none' || updates.phase === 'error')) checkForUpdates(true); };
  setTimeout(quietCheck, 8000);
  setInterval(quietCheck, 24 * 60 * 60 * 1000);
}
