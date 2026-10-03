import type { Item } from './db.svelte';
import { calendar, daysInYear } from './calendar';

/** A way to travel and how far it goes in one day (in the map's unit). */
export type Mode = { name: string; perDay: number };

export const DEFAULT_MODES: Mode[] = [
  { name: 'On foot', perDay: 30 },
  { name: 'Horse', perDay: 60 },
  { name: 'Cart', perDay: 25 },
];

export type Scale = { dist: number; unit: string; modes: Mode[] };

/** The scale of a map, or null when the writer has not set it. */
export function scaleOf(map: Item | undefined | null): Scale | null {
  const dist = Number(map?.scaleDist);
  if (!map || !(dist > 0)) return null;
  const modes = ((map.modes ?? DEFAULT_MODES) as Mode[]).filter((m) => Number(m.perDay) > 0 && String(m.name ?? '').trim());
  return { dist, unit: String(map.scaleUnit || 'km').trim() || 'km', modes };
}

/** Straight-line distance between two spots (0 to 1 across the map), in the map's unit. */
export function distance(map: Item, a: { x: number; y: number }, b: { x: number; y: number }): number {
  const sc = scaleOf(map);
  const w = Number(map.w) || 1, h = Number(map.h) || 1;
  return sc ? (Math.hypot((a.x - b.x) * w, (a.y - b.y) * h) / w) * sc.dist : 0;
}

const round = (n: number) => (n >= 100 ? Math.round(n) : n >= 10 ? Math.round(n * 10) / 10 : Math.round(n * 100) / 100);

export const fmtDist = (d: number, unit: string) => `${round(d).toLocaleString()} ${unit}`;

export function fmtDays(d: number): string {
  if (d < 0.1) return 'under 0.1 day';
  const n = d < 10 ? Math.round(d * 10) / 10 : Math.round(d);
  return `${n.toLocaleString()} day${n === 1 ? '' : 's'}`;
}

/** Days of story time between two timeline numbers. A year is the length of your calendar year, or 365 days. */
export function storyDays(t1: number, t2: number): number {
  const c = calendar();
  return Math.abs(t2 - t1) * (c ? daysInYear(c) : 365);
}
