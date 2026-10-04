/**
 * Glossary spelling check, the plain part (no app data in here).
 * The name check only looks at words with a capital letter. Made-up words are often lower case
 * ("ketha", "sylven"), so this finds a lower case word that is almost, but not exactly, a glossary word.
 */
import { distance, type Typo } from './namespell-core';

export type TermInput = { word: string; accepted: string };

const MIN_LEN = 5; // shorter words give too many false alarms
const TOKEN = /\p{L}+(?:-\p{L}+)*/gu;
const ONE_WORD = /^\p{L}+(?:-\p{L}+)*$/u;
const ENDINGS = ['s', 'es', 'd', 'ed', 'er', 'ers', 'ing', 'ly', 'y'];

const allowed = (len: number) => (len >= 8 ? 2 : 1);
const isLower = (w: string) => w === w.toLowerCase() && w !== w.toUpperCase();

/** "ketha" and "kethas" are one word with an ending, not a typo. */
function isEnding(a: string, b: string): boolean {
  const [short, long] = a.length <= b.length ? [a, b] : [b, a];
  return long.startsWith(short) && ENDINGS.includes(long.slice(short.length));
}

/** Capital words are the job of the name check. This only handles glossary words in lower case. */
export const isLowerTerm = (word: string) => ONE_WORD.test(word) && word.length >= MIN_LEN && isLower(word);

/**
 * texts: the plain text of each scene.
 * terms: the glossary. otherKnown: any other word that is spelled right (names of people and places).
 */
export function findTermVariants(texts: string[], terms: TermInput[], otherKnown: string[] = []): Typo[] {
  const mine = terms.map((t) => t.word.trim()).filter(isLowerTerm);
  if (!mine.length) return [];

  // every spelling that is fine: the glossary words, their accepted spellings, and the other names
  const fine = new Set<string>();
  for (const t of terms) {
    for (const w of [t.word, ...t.accepted.split(/[,;\n]/)]) for (const x of w.match(TOKEN) ?? []) fine.add(x.toLowerCase());
  }
  for (const n of otherKnown) for (const x of n.match(TOKEN) ?? []) fine.add(x.toLowerCase());

  const counts = new Map<string, number>(); // lower case words of the text
  for (const t of texts) {
    for (const m of t.matchAll(TOKEN)) {
      const w = m[0];
      if (w.length >= MIN_LEN && isLower(w)) counts.set(w, (counts.get(w) ?? 0) + 1);
    }
  }

  const byLetter = new Map<string, string[]>();
  for (const w of mine) byLetter.set(w[0], [...(byLetter.get(w[0]) ?? []), w]);

  const out: Typo[] = [];
  for (const [word, count] of counts) {
    if (fine.has(word)) continue;
    let best: { w: string; d: number } | null = null;
    for (const cand of byLetter.get(word[0]) ?? []) {
      const max = allowed(Math.min(cand.length, word.length));
      const d = distance(word, cand, max);
      if (d > max || isEnding(word, cand)) continue;
      // a word used a lot is probably a real word. A rare one is probably a slip.
      const inText = counts.get(cand) ?? 0;
      if (count > 2 && inText < count * 3) continue;
      if (!best || d < best.d) best = { w: cand, d };
    }
    if (best) out.push({ word, suggest: best.w, count, from: 'glossary' });
  }
  return out.sort((a, b) => b.count - a.count || a.word.localeCompare(b.word));
}
