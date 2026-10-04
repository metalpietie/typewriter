import { db, type Item } from './db.svelte';
import { orderedScenes, sceneLabel } from './scenes';
import { timeText } from './calendar';

// A character has a birth date (`born`) and maybe a death date (`died`). Both are timeline numbers,
// the same numbers that scenes and events use. In your own calendar a date is the year plus a part
// for the day, so the age in whole years is the plain difference of the two numbers.

const EPS = 1e-6;
const num = (v: unknown): number | null => (v === null || v === undefined || v === '' || !Number.isFinite(Number(v)) ? null : Number(v));

export type Age = {
  state: 'unborn' | 'alive' | 'dead';
  years: number | null; // age in whole years; for a dead person, the age at death
  text: string; // for example "23", "not born yet", or "dead, died aged 40"
};

/** How old a character is at a moment of story time. Null when we cannot tell. */
export function ageAt(c: Item | undefined, time: unknown): Age | null {
  const born = num(c?.born), t = num(time);
  if (born === null || t === null) return null;
  if (t < born - EPS) return { state: 'unborn', years: null, text: 'not born yet' };
  const died = num(c?.died);
  if (died !== null && t > died + EPS) {
    const at = Math.max(0, Math.floor(died - born + EPS));
    return { state: 'dead', years: at, text: `dead, died aged ${at}` };
  }
  const years = Math.max(0, Math.floor(t - born + EPS));
  return { state: 'alive', years, text: String(years) };
}

/** A short note for a list: "age 23", "not born yet", "dead". Empty when we cannot tell. */
export function shortAge(c: Item | undefined, time: unknown): string {
  const a = ageAt(c, time);
  if (!a) return '';
  return a.state === 'alive' ? `age ${a.years}` : a.state === 'unborn' ? 'not born yet' : 'dead';
}

/** True for an age that cannot be in the story: not born yet, or already dead. */
export const impossible = (c: Item | undefined, time: unknown) => {
  const a = ageAt(c, time);
  return !!a && a.state !== 'alive';
};

export type Moment = { key: string; time: number; when: string; what: string; open: { kind: 'scene' | 'event'; id: number }; age: Age };

/** The scenes and events a character is in that have a date, in time order, with the age at each. */
export function momentsOf(characterId: number): Moment[] {
  const c = db.get('character', characterId);
  if (!c || num(c.born) === null) return [];
  const out: Moment[] = [];
  for (const s of orderedScenes()) {
    const t = num(s.time);
    if (t === null || !(s.pov === characterId || (s.characters ?? []).includes(characterId))) continue;
    const age = ageAt(c, t);
    if (age) out.push({ key: 's' + s.id, time: t, when: timeText(t), what: sceneLabel(s), open: { kind: 'scene', id: s.id }, age });
  }
  for (const e of db.lists.event) {
    const t = num(e.time);
    if (t === null || !(e.characters ?? []).includes(characterId)) continue;
    const age = ageAt(c, t);
    if (age) out.push({ key: 'e' + e.id, time: t, when: timeText(t), what: e.title || 'Untitled event', open: { kind: 'event', id: e.id }, age });
  }
  return out.sort((a, b) => a.time - b.time);
}
