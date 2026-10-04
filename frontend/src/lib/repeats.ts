import { db, plainText, type Item } from './db.svelte';
import { scenesOf, sceneLabel } from './scenes';
import { nameWords } from './namespell-core';
import { worldNames } from './namespell';
import { closeRepeats, sameStarts, topWords, type Top } from './repeats-core';

export type CloseRow = { key: string; scene: Item; label: string; word: string; times: number; clusters: number; snippet: string };
export type StartRow = { key: string; scene: Item; label: string; word: string; run: number; snippet: string };
export type Repeats = { close: CloseRow[]; starts: StartRow[]; top: Top[]; words: number; scenes: number };

/** Look for repeats in one book, or in all books (scope 'all'). Names from your world data are ignored. */
export function findRepeats(scope: number | 'all'): Repeats {
  const skip = new Set<string>(nameWords(worldNames()).keys());
  const scenes: Item[] = [];
  for (const b of [...db.lists.book].sort((a, b2) => a.order - b2.order)) {
    if (scope !== 'all' && b.id !== scope) continue;
    for (const c of db.lists.chapter.filter((c) => c.book === b.id).sort((a, b2) => a.order - b2.order)) scenes.push(...scenesOf(c.id));
  }
  const texts = scenes.map((s) => plainText(s.body ?? ''));
  const close: CloseRow[] = [], starts: StartRow[] = [];
  scenes.forEach((scene, i) => {
    const label = sceneLabel(scene);
    for (const c of closeRepeats(texts[i], skip)) close.push({ key: `r|c|${scene.id}|${c.word}`, scene, label, ...c });
    for (const s of sameStarts(texts[i])) starts.push({ key: `r|s|${scene.id}|${s.word}|${s.run}`, scene, label, word: s.word, run: s.run, snippet: s.snippet });
  });
  const { words, total } = topWords(texts, skip);
  return { close, starts, top: words, words: total, scenes: scenes.length };
}
