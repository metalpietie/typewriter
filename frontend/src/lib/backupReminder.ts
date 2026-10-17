import { Store } from '../../bindings/typewriter/internal/app';
import { db } from './db.svelte';

// The app reminds you to save a copy of your data outside the app (a USB drive, or a folder you copy by hand).
// "Last backup" is the last time you used the Backup button. The automatic daily copies stay inside the
// app's folder, so they do not count: if the computer breaks, they are gone too.

export const REMIND_CHOICES: [number, string][] = [[0, 'Never'], [3, 'After 3 days'], [7, 'After a week'], [14, 'After 2 weeks'], [30, 'After a month']];
const DEFAULT_DAYS = 7;
const MIN_WORDS = 100; // a project with almost no writing is not worth a reminder

/** After how many days the reminder shows. 0 means never. */
export function reminderDays(): number {
  const row = db.lists.setting.find((s) => s.key === 'backupDays');
  const n = Number(row?.value);
  return row && Number.isFinite(n) && n >= 0 ? n : DEFAULT_DAYS;
}

export async function setReminderDays(n: number) {
  const row = db.lists.setting.find((s) => s.key === 'backupDays');
  await db.save('setting', { ...(row ?? {}), key: 'backupDays', value: n });
}

/** Whole days since a time such as "2026-10-04T15:04:05Z". Null when there is no time. */
export function daysSince(iso: string, now: Date = new Date()): number | null {
  const t = Date.parse(iso);
  return Number.isFinite(t) ? Math.max(0, Math.floor((now.getTime() - t) / 864e5)) : null;
}

export const agoText = (days: number): string => (days <= 0 ? 'today' : days === 1 ? 'yesterday' : `${days} days ago`);

// "Remind me tomorrow" is kept for this day only.
const SNOOZE = 'typewriter-backup-snooze';
const today = () => new Date().toDateString();
export function snoozedToday(): boolean {
  try { return localStorage.getItem(SNOOZE) === today(); } catch { return false; }
}
export function snoozeToday() {
  try { localStorage.setItem(SNOOZE, today()); } catch { /* the reminder just comes back sooner */ }
}

/** Should the reminder show now? words: how much is written in this project. */
export function reminderDue(words: number): boolean {
  const days = reminderDays();
  if (!days || snoozedToday()) return false;
  if (!db.lastBackup) return words >= MIN_WORDS;
  return (daysSince(db.lastBackup) ?? 0) >= days;
}

/** Wait until every edit has reached the database, so the backup is complete. */
export async function settle() {
  db.flush();
  for (let i = 0; i < 40 && db.waiting + db.busy > 0; i++) await new Promise((r) => setTimeout(r, 50));
}

/** Ask where to save, then save a copy. Returns the path, or "" if you cancelled. */
export async function backupNow(): Promise<string> {
  await settle();
  const path = await Store.Backup();
  if (path) db.lastBackup = (await Store.LastBackup()) || new Date().toISOString();
  return path;
}
