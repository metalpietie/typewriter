import { db, type Item } from './db.svelte';
import { sceneLabel } from './scenes';

/** One place where someone (or something) is at one moment of story time. */
export type Presence = {
  source: 'scene' | 'event' | 'path' | 'item';
  id: number; // the scene, event, path step, or item the row opens
  text: string;
  place: Where;
  holder?: number | null; // items only: who has it
};
type Where = { loc: number | null; map: number | null; x: number | null; y: number | null; label: string };

export type Conflict = {
  key: string;
  kind: 'character' | 'item';
  subject: number;
  name: string;
  time: number;
  when: string;
  rows: Presence[];
};

export type Stats = {
  scenes: number; scenesNoTime: number; scenesNoPlace: number; // scenes checked, and scenes that could not be
  events: number; steps: number; itemSteps: number;
};

const NEAR = 0.04; // two map points closer than this (of the map width) are the same spot

/** Is place `up` the same as, or above, place `down`? A street is inside its city. */
function inside(down: number, up: number): boolean {
  const seen = new Set<number>();
  for (let id: number | null = down; id && !seen.has(id); id = db.get('location', id)?.parent ?? null) {
    if (id === up) return true;
    seen.add(id);
  }
  return false;
}

function whereOf(locId: number | null | undefined, map?: number | null, x?: number | null, y?: number | null): Where | null {
  const loc = locId ? db.get('location', locId) : undefined;
  if (loc) return { loc: loc.id, map: loc.map ?? null, x: loc.x ?? null, y: loc.y ?? null, label: loc.name };
  if (map && x != null && y != null) return { loc: null, map, x, y, label: 'a spot on “' + db.name('map', map) + '”' };
  return null;
}

/** Can one person be in both places at once? Places inside each other are fine. */
function compatible(a: Where, b: Where): boolean {
  if (a.loc && b.loc) return a.loc === b.loc || inside(a.loc, b.loc) || inside(b.loc, a.loc);
  if (a.map && b.map && a.x != null && b.x != null && a.y != null && b.y != null) {
    return a.map === b.map && Math.hypot(a.x - b.x, a.y - b.y) < NEAR;
  }
  return true; // not enough to tell
}

const sameTime = (a: number, b: number) => Math.abs(a - b) < 1e-9;
const num = (v: unknown): number | null => (v === null || v === undefined || v === '' || !Number.isFinite(Number(v)) ? null : Number(v));

type Slot = { time: number; when: string; rows: Presence[] };

/** Group rows that share a moment in time. */
function addTo(map: Map<string, Slot>, subject: string, time: number, when: string, row: Presence) {
  const key = subject + '|' + time;
  const slot = map.get(key) ?? { time, when, rows: [] };
  if (!slot.when && when) slot.when = when;
  slot.rows.push(row);
  map.set(key, slot);
}

/** The rows of a slot that clash with at least one other row. */
function clashing(rows: Presence[], differs: (a: Presence, b: Presence) => boolean): Presence[] {
  return rows.filter((a) => rows.some((b) => b !== a && differs(a, b)));
}

export function findConflicts(): { conflicts: Conflict[]; stats: Stats } {
  const stats: Stats = { scenes: 0, scenesNoTime: 0, scenesNoPlace: 0, events: 0, steps: 0, itemSteps: 0 };
  const chars = new Map<string, Slot>();
  const items = new Map<string, Slot>();
  const known = (id: number | null | undefined) => !!id && !!db.get('character', id);

  for (const s of db.lists.scene) {
    const time = num(s.time);
    const place = whereOf(s.location);
    if (time === null) stats.scenesNoTime++;
    else if (!place) stats.scenesNoPlace++;
    if (time === null || !place) continue;
    stats.scenes++;
    const row: Presence = { source: 'scene', id: s.id, text: sceneLabel(s), place };
    for (const c of new Set<number>([...(s.pov ? [s.pov] : []), ...(s.characters ?? [])])) {
      if (known(c)) addTo(chars, 'c' + c, time, s.when ?? '', row);
    }
  }

  for (const e of db.lists.event) {
    const time = num(e.time);
    const place = whereOf(e.location);
    if (time === null || !place) continue;
    stats.events++;
    const row: Presence = { source: 'event', id: e.id, text: `Event “${e.title}”`, place };
    for (const c of e.characters ?? []) if (known(c)) addTo(chars, 'c' + c, time, e.when ?? '', row);
  }

  for (const p of db.lists.pathpoint) {
    const time = num(p.time);
    const place = whereOf(p.location, p.map, p.x, p.y);
    if (time === null || !place || !known(p.character)) continue;
    stats.steps++;
    addTo(chars, 'c' + p.character, time, p.when ?? '', { source: 'path', id: p.id, text: `Path step “${p.label || 'step'}”`, place });
  }

  // an item cannot be in two hands, or two places, at the same time
  for (const l of db.lists.itemlog) {
    const scene = db.get('scene', l.scene);
    const time = num(scene?.time);
    const item = db.get('item', l.item);
    if (!scene || time === null || !item) continue;
    stats.itemSteps++;
    const holder = db.get('character', l.character);
    const place = whereOf(l.location) ?? { loc: null, map: null, x: null, y: null, label: '' };
    const text = `${holder ? holder.name + ' has it' : 'Nobody has it'} (${sceneLabel(scene)})`;
    addTo(items, 'i' + item.id, time, scene.when ?? '', { source: 'item', id: item.id, text, place, holder: holder?.id ?? null });
  }

  const out: Conflict[] = [];
  const done = (kind: Conflict['kind'], subject: number, name: string, slot: Slot, rows: Presence[]) => {
    if (rows.length < 2) return;
    const ids = rows.map((r) => r.source[0] + r.id + (r.source === 'item' ? ':' + (r.holder ?? '') : '')).sort().join(',');
    out.push({ key: `${kind[0]}${subject}|${slot.time}|${ids}`, kind, subject, name, time: slot.time, when: slot.when, rows });
  };

  for (const [k, slot] of chars) {
    const id = +k.split('|')[0].slice(1);
    done('character', id, db.name('character', id), slot, clashing(slot.rows, (a, b) => !compatible(a.place, b.place)));
  }
  for (const [k, slot] of items) {
    const id = +k.split('|')[0].slice(1);
    const holder = (r: Presence) => r.holder ?? null;
    done('item', id, db.name('item', id), slot, clashing(slot.rows, (a, b) =>
      (holder(a) !== null && holder(b) !== null && holder(a) !== holder(b)) || !compatible(a.place, b.place)));
  }
  return { conflicts: out.sort((a, b) => a.time - b.time), stats };
}

/** The conflicts you have not dismissed. */
export function openConflicts(all: Conflict[]): Conflict[] {
  const ignored = new Set(db.lists.plotignore.map((i: Item) => i.key));
  return all.filter((c) => !ignored.has(c.key));
}
