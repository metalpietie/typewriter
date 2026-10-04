import { db, type Item } from './db.svelte';
import { scenesOf, sceneLabel } from './scenes';
import { around, dialogueWords, hashOf, sentences, weakWords, type Group } from './style-core';

export type ChapterRow = { id: number; first: number | null; label: string; total: number; dialogue: number };
export type LongRow = { key: string; scene: Item; label: string; words: number; text: string };
export type WeakStat = { word: string; group: Group; count: number; per1000: number };
export type StyleReport = {
  scenes: number;
  dialogue: { dialogue: number; total: number };
  chapters: ChapterRow[];
  long: LongRow[];
  weak: WeakStat[];
};
export type Example = { scene: Item; label: string; snippet: string };

// Reading a scene's HTML is slow, so each scene is read once and kept until its text changes.
const cache = new Map<number, { src: string; paras: string[] }>();

/** The paragraphs of a scene as plain text. */
function paragraphsOf(scene: Item): string[] {
  const src = String(scene.body ?? '');
  const old = cache.get(scene.id);
  if (old && old.src === src) return old.paras;
  let paras: string[];
  if (!src.trimStart().startsWith('<')) paras = src.split(/\r?\n/);
  else {
    const spaced = src.replace(/<\/(p|h[1-6]|li|blockquote)>|<br\s*\/?>|<hr\s*\/?>/gi, '\n');
    paras = (new DOMParser().parseFromString(spaced, 'text/html').body.textContent ?? '').split('\n');
  }
  paras = paras.map((p) => p.trim()).filter(Boolean);
  cache.set(scene.id, { src, paras });
  return paras;
}

/** The chapters of one book, or of all books (scope 'all'), in reading order, with their scenes. */
function chaptersIn(scope: number | 'all'): { chapter: Item; scenes: Item[] }[] {
  const out: { chapter: Item; scenes: Item[] }[] = [];
  for (const b of [...db.lists.book].sort((a, b2) => a.order - b2.order)) {
    if (scope !== 'all' && b.id !== scope) continue;
    for (const c of db.lists.chapter.filter((c) => c.book === b.id).sort((a, b2) => a.order - b2.order)) out.push({ chapter: c, scenes: scenesOf(c.id) });
  }
  return out;
}

/** Dialogue per chapter, sentences longer than `minLong` words, and the weak words, for one book or all. */
export function findStyle(scope: number | 'all', minLong: number): StyleReport {
  const chapters: ChapterRow[] = [], long: LongRow[] = [];
  const counts = new Map<string, WeakStat>();
  const all = { dialogue: 0, total: 0 };
  let scenes = 0;

  for (const { chapter, scenes: list } of chaptersIn(scope)) {
    const chapterParas: string[] = [];
    for (const scene of list) {
      const paras = paragraphsOf(scene);
      if (!paras.length) continue;
      scenes++;
      chapterParas.push(...paras);
      const label = sceneLabel(scene);
      for (const s of sentences(paras)) {
        if (s.words >= minLong) long.push({ key: `y|l|${scene.id}|${hashOf(s.text)}`, scene, label, words: s.words, text: s.text });
      }
      for (const p of paras) {
        for (const w of weakWords(p)) {
          const k = w.group + '|' + w.word;
          const stat = counts.get(k) ?? { word: w.word, group: w.group, count: 0, per1000: 0 };
          stat.count++;
          counts.set(k, stat);
        }
      }
    }
    const d = dialogueWords(chapterParas);
    all.dialogue += d.dialogue;
    all.total += d.total;
    if (d.total) chapters.push({ id: chapter.id, first: list.find((s) => paragraphsOf(s).length)?.id ?? null, label: `${chapter.order}. ${chapter.title || 'Untitled'}`, total: d.total, dialogue: d.dialogue });
  }

  const weak = [...counts.values()].map((s) => ({ ...s, per1000: all.total ? Math.round((s.count / all.total) * 10000) / 10 : 0 }));
  weak.sort((a, b) => b.count - a.count || a.word.localeCompare(b.word));
  long.sort((a, b) => b.words - a.words);
  return { scenes, dialogue: all, chapters, long, weak };
}

/** Where one weak word is used. */
export function weakExamples(scope: number | 'all', word: string, limit = 30): Example[] {
  const out: Example[] = [];
  for (const { scenes } of chaptersIn(scope)) {
    for (const scene of scenes) {
      for (const p of paragraphsOf(scene)) {
        for (const w of weakWords(p)) {
          if (w.word === word && out.length < limit) out.push({ scene, label: sceneLabel(scene), snippet: around(p, w.at, w.end) });
        }
      }
    }
  }
  return out;
}

