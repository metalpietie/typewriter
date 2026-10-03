/** Pure calendar maths. The timeline number of a date is its year plus a fraction for the day. */
export type Month = { name: string; days: number };
export type Calendar = {
  months: Month[];
  weekdays: string[]; // optional
  weekdayShift: number; // which weekday is the first day of year 0
  yearNames: string[]; // optional: the names repeat, such as the animal years
  prefix: string; // before the year number, such as "Year"
  suffix: string; // after the year number, such as "AE"
};
export type Parts = { year: number; month: number | null; day: number | null; index: number | null };

export const EMPTY: Calendar = { months: [], weekdays: [], weekdayShift: 0, yearNames: [], prefix: 'Year', suffix: '' };

const len = (m: Month) => Math.max(1, Math.floor(Number(m.days)) || 1);
export const monthLength = len;
export const daysInYear = (c: Calendar) => c.months.reduce((a, m) => a + len(m), 0);
export const usable = (c: Calendar | null | undefined): c is Calendar => !!c && c.months.length > 0;
const round6 = (n: number) => Math.round(n * 1e6) / 1e6;

/**
 * The timeline number of a date. It points at the middle of the day, so a day
 * is never the same number as "the whole year" (which is the plain year number).
 */
export function toTime(c: Calendar, year: number, month: number, day: number): number {
  let index = 0;
  for (let i = 0; i < month; i++) index += len(c.months[i]);
  index += Math.min(Math.max(1, day), len(c.months[month])) - 1;
  return round6(year + (index + 0.5) / daysInYear(c));
}

/** The date of a timeline number. A plain year number has no month or day. */
export function fromTime(c: Calendar, t: number): Parts {
  const year = Math.floor(t + 1e-9);
  const frac = t - year;
  if (frac < 1e-7) return { year, month: null, day: null, index: null };
  const dpy = daysInYear(c);
  const index = Math.min(dpy - 1, Math.floor(frac * dpy));
  let left = index;
  for (let m = 0; m < c.months.length; m++) {
    if (left < len(c.months[m])) return { year, month: m, day: left + 1, index };
    left -= len(c.months[m]);
  }
  return { year, month: null, day: null, index: null };
}

export function weekdayOf(c: Calendar, p: Parts): string | null {
  const n = c.weekdays.length;
  if (!n || p.index === null) return null;
  const abs = p.year * daysInYear(c) + p.index + Math.floor(Number(c.weekdayShift) || 0);
  return c.weekdays[((abs % n) + n) % n];
}

export function yearName(c: Calendar, year: number): string | null {
  const n = c.yearNames.length;
  return n ? c.yearNames[((year % n) + n) % n] : null;
}

/** "Tuesday, 14 Frostmoon, Year 312 AE (Dragon)" */
export function format(c: Calendar, t: number): string {
  const p = fromTime(c, t);
  const era = [c.prefix, String(p.year) + (c.suffix ? ' ' + c.suffix : '')].filter((x) => x && x.trim()).join(' ');
  const name = yearName(c, p.year);
  const tail = era + (name ? ` (${name})` : '');
  if (p.month === null) return tail;
  const wd = weekdayOf(c, p);
  return `${wd ? wd + ', ' : ''}${p.day} ${c.months[p.month].name}, ${tail}`;
}

/** A plain starting point the writer can change. */
export function standard(): Calendar {
  const m = [['January', 31], ['February', 28], ['March', 31], ['April', 30], ['May', 31], ['June', 30], ['July', 31], ['August', 31], ['September', 30], ['October', 31], ['November', 30], ['December', 31]];
  return {
    months: m.map(([name, days]) => ({ name: name as string, days: days as number })),
    weekdays: ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'],
    weekdayShift: 0, yearNames: [], prefix: 'Year', suffix: '',
  };
}
