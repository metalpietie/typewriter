/**
 * The finish date plan, the plain part (no app data in here).
 * You give a goal in words, a finish date, the words you have now, and how much you write on a normal day.
 * It says how many words to write each day to finish on time.
 */

const DAY = 864e5;

/** A date such as "2026-12-31", at noon in local time (so a daylight saving change cannot move the day). Null if it is not a date. */
export function parseDay(s: unknown): Date | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(String(s ?? '').trim());
  if (!m) return null;
  const d = new Date(+m[1], +m[2] - 1, +m[3], 12);
  return d.getFullYear() === +m[1] && d.getMonth() === +m[2] - 1 && d.getDate() === +m[3] ? d : null;
}

const noon = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate(), 12);
const daysBetween = (from: Date, to: Date) => Math.round((to.getTime() - from.getTime()) / DAY);
const addDays = (d: Date, n: number) => { const x = noon(d); x.setDate(x.getDate() + n); return x; };

export type Plan =
  | { state: 'nodate' } // no finish date
  | { state: 'nogoal'; days: number } // a date, but no word goal
  | { state: 'done'; goal: number } // the goal is reached
  | { state: 'late'; left: number; daysLate: number } // the date has passed and words are left
  | {
      state: 'open';
      left: number; // words still to write
      days: number; // days left, counting today and the finish day
      perDay: number; // words to write each day
      verdict: 'on' | 'behind' | 'idle'; // compared with how much you write on a normal day
      projected: Date | null; // the day you would finish at your normal pace
    };

/**
 * total: words written in the book. goal: the word goal. finishBy: "YYYY-MM-DD".
 * pace: words you write on a normal day (for example the average of the last 7 days).
 * Today counts as a day to write, and so does the finish day.
 */
export function planFor(o: { total: number; goal: number; finishBy: unknown; now: Date; pace: number }): Plan {
  const end = parseDay(o.finishBy);
  if (!end) return { state: 'nodate' };
  const today = noon(o.now);
  const days = daysBetween(today, end) + 1;
  if (!(o.goal > 0)) return { state: 'nogoal', days };

  const left = Math.max(0, Math.round(o.goal - o.total));
  if (left === 0) return { state: 'done', goal: o.goal };
  if (days <= 0) return { state: 'late', left, daysLate: 1 - days };

  const perDay = Math.ceil(left / days);
  const pace = Math.max(0, o.pace);
  return {
    state: 'open', left, days, perDay,
    verdict: pace >= perDay ? 'on' : pace > 0 ? 'behind' : 'idle',
    projected: pace > 0 ? addDays(today, Math.ceil(left / pace) - 1) : null,
  };
}
