import { db, type Item } from './db.svelte';

/** "2026-10-03" for a date, in local time. */
export const dayKey = (d: Date = new Date()): string =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

const daysAgo = (n: number): Date => {
  const d = new Date();
  d.setHours(12, 0, 0, 0); // noon: a daylight saving change cannot skip a day
  d.setDate(d.getDate() - n);
  return d;
};

let making: Promise<Item> | null = null;

async function todayRow(): Promise<Item> {
  const key = dayKey();
  const found = db.lists.wordday.find((d) => d.date === key);
  if (found) return found;
  making ??= db.save('wordday', { date: key, words: 0 }).finally(() => { making = null; });
  return making;
}

/** Add to today's count (a negative number when words were cut). Called while you type. */
export async function addWords(delta: number) {
  if (!delta) return;
  const row = await todayRow();
  row.words = (row.words ?? 0) + delta;
  db.queue('wordday', row);
}

/** Words written on one day. Cutting more than you wrote counts as 0, not less. */
export const wordsOn = (key: string): number =>
  Math.max(0, db.lists.wordday.find((d) => d.date === key)?.words ?? 0);

export const wordsToday = () => wordsOn(dayKey());

/** The daily goal in words. 0 means no goal. */
export function dailyGoal(): number {
  const n = Number(db.lists.setting.find((s) => s.key === 'dailyGoal')?.value);
  return Number.isFinite(n) && n > 0 ? n : 0;
}

export async function setDailyGoal(n: number) {
  const row = db.lists.setting.find((s) => s.key === 'dailyGoal');
  await db.save('setting', { ...(row ?? {}), key: 'dailyGoal', value: Math.max(0, Math.floor(n) || 0) });
}

/** The last `n` days, oldest first. */
export function lastDays(n: number): { key: string; date: Date; words: number }[] {
  return Array.from({ length: n }, (_, i) => {
    const date = daysAgo(n - 1 - i);
    return { key: dayKey(date), date, words: wordsOn(dayKey(date)) };
  });
}

/**
 * Days in a row that reached the goal (with no goal set: days with any writing).
 * Today does not break the streak while it is still going: it counts only once it reaches the goal.
 */
export function streak(): { current: number; best: number } {
  const need = dailyGoal() || 1;
  const hit = (key: string) => wordsOn(key) >= need;
  let current = 0;
  for (let i = hit(dayKey()) ? 0 : 1; hit(dayKey(daysAgo(i))); i++) current++;

  const keys = db.lists.wordday.map((d) => d.date as string).filter(hit).sort();
  let best = 0, run = 0, prev = '';
  for (const k of keys) {
    run = prev && dayKey(new Date(new Date(prev + 'T12:00:00').getTime() + 864e5)) === k ? run + 1 : 1;
    best = Math.max(best, run);
    prev = k;
  }
  return { current, best: Math.max(best, current) };
}
