import { db, plainText, type Kind } from './db.svelte';
import { sceneLabel } from './scenes';
import { backlinks } from './backlinks';
import { impossible, ageAt } from './ages';
import { timeText } from './calendar';

/** One thing that looks unfinished. */
export type Gap = {
  key: string; // stays the same while the problem stays the same
  group: string; // the heading it sits under
  open: { kind: Kind; id: number }; // what to open to fix it
  name: string;
  problem: string;
};

export const GROUPS = ['Characters', 'Places', 'Scenes', 'Factions', 'Items', 'Events', 'Lore notes'] as const;

const blank = (v: unknown) => v === null || v === undefined || String(v).trim() === '';

export function findGaps(): Gap[] {
  const out: Gap[] = [];
  const add = (group: string, kind: Kind, id: number, code: string, name: string, problem: string) =>
    out.push({ key: `x|${kind}|${id}|${code}`, group, open: { kind, id }, name, problem });

  const related = new Set<number>();
  for (const r of db.lists.relationship) { related.add(r.a); related.add(r.b); }

  for (const c of db.lists.character) {
    const name = c.name || 'Unnamed';
    if (blank(c.age) && blank(c.born)) add('Characters', 'character', c.id, 'age', name, 'has no age');
    if (blank(c.description)) add('Characters', 'character', c.id, 'desc', name, 'has no description');
    if (blank(c.backstory) && blank(c.goals)) add('Characters', 'character', c.id, 'story', name, 'has no backstory and no goals');
    if (!related.has(c.id)) add('Characters', 'character', c.id, 'rel', name, 'has no relationships');
    const seen = db.lists.scene.some((s) => s.pov === c.id || (s.characters ?? []).includes(c.id)) || backlinks('character', c.id).length > 0;
    if (!seen) add('Characters', 'character', c.id, 'scene', name, 'is in no scene');
  }

  for (const l of db.lists.location) {
    const name = l.name || 'Unnamed';
    if (!l.map) add('Places', 'location', l.id, 'map', name, 'is on no map');
    if (blank(l.description)) add('Places', 'location', l.id, 'desc', name, 'has no description');
  }

  for (const s of db.lists.scene) {
    const name = sceneLabel(s);
    if (!s.pov) add('Scenes', 'scene', s.id, 'pov', name, 'has no point of view');
    if (!s.location) add('Scenes', 'scene', s.id, 'loc', name, 'has no place');
    if (blank(s.time)) add('Scenes', 'scene', s.id, 'time', name, 'has no timeline number');
    if (blank(s.summary)) add('Scenes', 'scene', s.id, 'sum', name, 'has no summary');
    if (blank(plainText(s.body ?? ''))) add('Scenes', 'scene', s.id, 'body', name, 'has no text yet');
  }

  // a person in a scene or event dated before their birth or after their death
  const when = (c: any, t: unknown) => (ageAt(c, t)?.state === 'unborn' ? `was not born yet (born ${timeText(c.born)})` : `had already died (died ${timeText(c.died)})`);
  for (const s of db.lists.scene) {
    const cast = new Set<number>([...(s.pov ? [s.pov] : []), ...(s.characters ?? [])]);
    for (const id of cast) {
      const c = db.get('character', id);
      if (c && impossible(c, s.time)) add('Scenes', 'scene', s.id, 'age' + id, sceneLabel(s), `has ${c.name || 'someone'}, who ${when(c, s.time)} on ${timeText(s.time)}`);
    }
  }
  for (const e of db.lists.event) {
    for (const id of e.characters ?? []) {
      const c = db.get('character', id);
      if (c && impossible(c, e.time)) add('Events', 'event', e.id, 'age' + id, e.title || 'Untitled', `has ${c.name || 'someone'}, who ${when(c, e.time)} on ${timeText(e.time)}`);
    }
  }

  const members = new Set<number>(db.lists.membership.map((m) => m.faction));
  for (const f of db.lists.faction) {
    const name = f.name || 'Unnamed';
    if (!members.has(f.id)) add('Factions', 'faction', f.id, 'members', name, 'has no members');
    if (!f.hq) add('Factions', 'faction', f.id, 'hq', name, 'has no headquarters');
    if (blank(f.description)) add('Factions', 'faction', f.id, 'desc', name, 'has no description');
  }

  const logged = new Set<number>(db.lists.itemlog.map((l) => l.item));
  for (const i of db.lists.item) {
    const name = i.name || 'Unnamed';
    if (blank(i.description)) add('Items', 'item', i.id, 'desc', name, 'has no description');
    if (!logged.has(i.id)) add('Items', 'item', i.id, 'log', name, 'has no journey steps');
  }

  for (const e of db.lists.event) {
    const name = e.title || 'Untitled';
    if (blank(e.time)) add('Events', 'event', e.id, 'time', name, 'has no timeline number');
    if (!e.location) add('Events', 'event', e.id, 'loc', name, 'has no place');
  }

  for (const n of db.lists.note) {
    if (blank(n.body)) add('Lore notes', 'note', n.id, 'body', n.title || 'Untitled', 'is empty');
  }

  return out;
}

/** The gaps you have not hidden. */
export function openGaps(all: Gap[]): Gap[] {
  const ignored = new Set(db.lists.plotignore.map((i) => i.key));
  return all.filter((g) => !ignored.has(g.key));
}
