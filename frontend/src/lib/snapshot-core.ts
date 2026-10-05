// Compare a book snapshot with the book as it is now. Only diff-core is imported, so it can be tested alone.
import { countChanges, diffWords, tokens } from './diff-core';

/** One scene, with its text as plain text (paragraphs split by a line break). */
export type Item = { id: number; chapter: number; title: string; order: number; text: string };
export type Chap = { id: number; title: string; order: number };

export type State = 'same' | 'changed' | 'new' | 'gone';

/** What happened to one scene since the snapshot. */
export type Row = {
  id: number;
  title: string;
  chapter: string;     // the chapter now. For a removed scene, the chapter it had
  moved: boolean;      // it sits in another chapter than at the snapshot
  state: State;        // same text, changed text, written after the snapshot, or removed since
  wordsThen: number;
  wordsNow: number;
  added: number;       // words added since the snapshot
  cut: number;         // words cut since the snapshot
};

const norm = (t: string) => tokens(t).join(' ');
const words = (t: string) => tokens(t).filter((w) => w !== '\n').length;

/** The scenes of a snapshot against the scenes now, in reading order. */
export function compare(thenScenes: Item[], thenChaps: Chap[], nowScenes: Item[], nowChaps: Chap[]): Row[] {
  const thenById = new Map(thenScenes.map((s) => [s.id, s]));
  const nowById = new Map(nowScenes.map((s) => [s.id, s]));
  const chapNow = new Map(nowChaps.map((c) => [c.id, c]));
  const chapThen = new Map(thenChaps.map((c) => [c.id, c]));
  const keyed: { key: [number, number, number]; row: Row }[] = [];

  for (const s of nowScenes) {
    const old = thenById.get(s.id);
    const c = chapNow.get(s.chapter);
    const base = { id: s.id, title: s.title, chapter: c?.title ?? '', wordsNow: words(s.text) };
    const key: [number, number, number] = [c?.order ?? 1e9, s.order, s.id];
    if (!old) {
      keyed.push({ key, row: { ...base, moved: false, state: 'new', wordsThen: 0, added: base.wordsNow, cut: 0 } });
      continue;
    }
    const same = norm(old.text) === norm(s.text);
    const n = same ? { added: 0, cut: 0 } : countChanges(diffWords(old.text, s.text));
    keyed.push({ key, row: { ...base, moved: old.chapter !== s.chapter, state: same ? 'same' : 'changed', wordsThen: words(old.text), ...n } });
  }
  for (const old of thenScenes) {
    if (nowById.has(old.id)) continue;
    const c = chapThen.get(old.chapter);
    const w = words(old.text);
    keyed.push({
      key: [c?.order ?? 1e9, old.order, old.id],
      row: { id: old.id, title: old.title, chapter: c?.title ?? '', moved: false, state: 'gone', wordsThen: w, wordsNow: 0, added: 0, cut: w },
    });
  }
  return keyed.sort((a, b) => a.key[0] - b.key[0] || a.key[1] - b.key[1] || a.key[2] - b.key[2]).map((k) => k.row);
}

export type Totals = { changed: number; added: number; gone: number; moved: number; wordsAdded: number; wordsCut: number; wordsThen: number; wordsNow: number };

export function totals(rows: Row[]): Totals {
  const t: Totals = { changed: 0, added: 0, gone: 0, moved: 0, wordsAdded: 0, wordsCut: 0, wordsThen: 0, wordsNow: 0 };
  for (const r of rows) {
    if (r.state === 'changed') t.changed++;
    else if (r.state === 'new') t.added++;
    else if (r.state === 'gone') t.gone++;
    if (r.moved) t.moved++;
    t.wordsAdded += r.added; t.wordsCut += r.cut; t.wordsThen += r.wordsThen; t.wordsNow += r.wordsNow;
  }
  return t;
}

/** A scene that is not exactly as it was: changed, new, removed, or moved. */
export const isDifferent = (r: Row): boolean => r.state !== 'same' || r.moved;
