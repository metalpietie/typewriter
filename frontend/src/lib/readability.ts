import type { Item } from './db.svelte';
import { scenesOf } from './scenes';
import { countsOf, addCounts, htmlToText, NO_COUNTS, type Counts } from './readability-core';

// Words, sentences and syllables of scenes and chapters. The rules are in readability-core.ts.
// Counting is slow for a long book, so the answer is kept for each text until the text changes.

const MAX_CACHE = 3000;
const cache = new Map<string, Counts>();

function countsOfBody(body: string): Counts {
  const hit = cache.get(body);
  if (hit) return hit;
  const c = countsOf(body.trimStart().startsWith('<') ? htmlToText(body) : body);
  if (cache.size >= MAX_CACHE) cache.clear();
  cache.set(body, c);
  return c;
}

/** The counts of one chapter: its scenes added up. An older chapter without scenes uses its own text. */
export function chapterCounts(c: Item): Counts {
  const sc = scenesOf(c.id);
  const bodies = sc.length ? sc.map((s) => String(s.body ?? '')) : [String(c.body ?? '')];
  return bodies.reduce((sum, b) => addCounts(sum, countsOfBody(b)), NO_COUNTS);
}
