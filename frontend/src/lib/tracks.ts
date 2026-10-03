import { db } from './db.svelte';
import { sceneLabel } from './scenes';

/** One moment where a character is at a spot on a map. */
export type Stop = {
  time: number;
  map: number;
  x: number; // 0 to 1, across the map
  y: number;
  loc: number | null; // the place it belongs to, if any
  label: string; // for example the name of the path step
  source: 'scene' | 'event' | 'path';
  id: number; // the scene, event or path step
  text: string;
};

const num = (v: unknown): number | null => (v === null || v === undefined || v === '' || !Number.isFinite(Number(v)) ? null : Number(v));

/** A place on a map, or null when it has no spot. */
function spotOf(locId: number | null | undefined): { map: number; x: number; y: number; loc: number } | null {
  const l = locId ? db.get('location', locId) : undefined;
  return l && l.map && l.x != null && l.y != null ? { map: l.map, x: l.x, y: l.y, loc: l.id } : null;
}

/**
 * Where each character is over story time: path steps, scenes and events that have a
 * timeline number and a spot on a map. Each list is sorted by time.
 */
export function tracks(): Map<number, Stop[]> {
  const out = new Map<number, Stop[]>();
  const add = (char: number | null | undefined, stop: Stop) => {
    if (!char || !db.get('character', char)) return;
    const list = out.get(char) ?? [];
    list.push(stop);
    out.set(char, list);
  };

  for (const s of db.lists.scene) {
    const time = num(s.time), spot = spotOf(s.location);
    if (time === null || !spot) continue;
    const stop: Stop = { time, ...spot, label: '', source: 'scene', id: s.id, text: sceneLabel(s) };
    for (const c of new Set<number>([...(s.pov ? [s.pov] : []), ...(s.characters ?? [])])) add(c, stop);
  }
  for (const e of db.lists.event) {
    const time = num(e.time), spot = spotOf(e.location);
    if (time === null || !spot) continue;
    const stop: Stop = { time, ...spot, label: '', source: 'event', id: e.id, text: `Event “${e.title}”` };
    for (const c of e.characters ?? []) add(c, stop);
  }
  for (const p of db.lists.pathpoint) {
    const time = num(p.time);
    if (time === null || !p.map || p.x == null || p.y == null) continue;
    add(p.character, { time, map: p.map, x: p.x, y: p.y, loc: p.location ?? null, label: p.label ?? '', source: 'path', id: p.id, text: `Path step “${p.label || 'step'}”` });
  }
  for (const list of out.values()) list.sort((a, b) => a.time - b.time);
  return out;
}

export type Where = { x: number; y: number; state: 'at' | 'moving' | 'last'; stop: Stop };

/** The last stop at or before time t, and the next one. */
function around(stops: Stop[], t: number): { p?: Stop; q?: Stop } {
  let i = -1;
  while (i + 1 < stops.length && stops[i + 1].time <= t + 1e-9) i++;
  return { p: stops[i], q: stops[i + 1] };
}

/**
 * Where a character is on one map at time t. Between two stops on the same map the character
 * is on the straight line between them. Before the first stop it is nowhere. After the last
 * stop it stays there (state "last").
 */
export function whereAt(stops: Stop[], t: number, map: number): Where | null {
  const { p, q } = around(stops, t);
  if (!p || p.map !== map) return null;
  if (!q) return { x: p.x, y: p.y, state: 'last', stop: p };
  if (q.map !== map) return { x: p.x, y: p.y, state: 'at', stop: p };
  const f = (t - p.time) / (q.time - p.time);
  const moving = f > 0 && (p.x !== q.x || p.y !== q.y);
  return { x: p.x + (q.x - p.x) * f, y: p.y + (q.y - p.y) * f, state: moving ? 'moving' : 'at', stop: p };
}

export const stopName = (s: Stop): string =>
  s.loc ? db.name('location', s.loc) : s.label || `a spot on “${db.name('map', s.map)}”`;

/** A sentence for the "who is where" list, and the map the character is on. */
export function statusAt(stops: Stop[], t: number): { text: string; map: number } | null {
  const { p, q } = around(stops, t);
  if (!p) return stops.length ? { text: 'has not appeared yet', map: stops[0].map } : null;
  if (!q) return { text: `last seen at ${stopName(p)}`, map: p.map };
  if (q.map === p.map && (p.x !== q.x || p.y !== q.y) && t > p.time) return { text: `is travelling from ${stopName(p)} to ${stopName(q)}`, map: p.map };
  return { text: `is at ${stopName(p)}`, map: p.map };
}

/** The earliest and latest time of the stops on one map. */
export function rangeOn(all: Map<number, Stop[]>, map: number, chars: number[]): { min: number; max: number } | null {
  let min = Infinity, max = -Infinity;
  for (const c of chars) for (const s of all.get(c) ?? []) {
    if (s.map !== map) continue;
    min = Math.min(min, s.time);
    max = Math.max(max, s.time);
  }
  return min <= max ? { min, max } : null;
}
