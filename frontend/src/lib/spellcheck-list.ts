import { db, plainText } from './db.svelte';
import { allScenes, type Place } from './namespell';
import { sceneLabel } from './scenes';
import { correct } from './spell.svelte';
import { wordsOf } from './spell-core';

// The Spelling list of the Checks tab: every word of the book that the spell check marks, with where it is.

export type SpellIssue = { word: string; count: number; places: Place[] };

/** The marked words of the scenes of one book (or of all books), the most used first. */
export function findSpelling(book: number | 'all'): SpellIssue[] {
  const byWord = new Map<string, SpellIssue>();
  for (const s of allScenes()) {
    if (book !== 'all' && db.get('chapter', s.chapter)?.book !== book) continue;
    const text = plainText(s.body ?? '');
    for (const w of wordsOf(text)) {
      if (correct(w.word, text.slice(Math.max(0, w.start - 1), w.start), text.slice(w.end, w.end + 1))) continue;
      let issue = byWord.get(w.word);
      if (!issue) byWord.set(w.word, (issue = { word: w.word, count: 0, places: [] }));
      issue.count++;
      const place = issue.places.find((p) => p.scene.id === s.id);
      if (place) { place.times++; continue; }
      issue.places.push({
        scene: s, label: sceneLabel(s), times: 1,
        snippet: (w.start > 36 ? '…' : '') + text.slice(Math.max(0, w.start - 36), w.start) + '⟦' + w.word + '⟧' + text.slice(w.end, w.end + 36),
      });
    }
  }
  return [...byWord.values()].sort((a, b) => b.count - a.count || a.word.localeCompare(b.word));
}

