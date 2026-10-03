import { db, type Item } from './db.svelte';
import { EMPTY, format, usable, type Calendar } from './calendarmath';

export * from './calendarmath';

/** The setting that holds the calendar. */
export const calendarRow = (): Item | undefined => db.lists.setting.find((s) => s.key === 'calendar');

/** The writer's calendar, or null when none is set up. */
export function calendar(): Calendar | null {
  const r = calendarRow();
  if (!r) return null;
  const c: Calendar = {
    months: r.months ?? [], weekdays: r.weekdays ?? [], weekdayShift: r.weekdayShift ?? 0,
    yearNames: r.yearNames ?? [], prefix: r.prefix ?? EMPTY.prefix, suffix: r.suffix ?? '',
  };
  return usable(c) ? c : null;
}

/** A timeline number as text: a date in your calendar, or the plain number. */
export function timeText(t: number | null | undefined): string {
  if (t === null || t === undefined || t === ('' as unknown)) return '';
  const c = calendar();
  return c ? format(c, Number(t)) : String(t);
}
