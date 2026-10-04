/**
 * Name spelling check, the plain part (no app data in here).
 * It finds a capitalised word that is almost, but not exactly, a name you use.
 */

export type Typo = {
  word: string; // as written in the text
  suggest: string; // what it should probably be
  count: number; // how often the word is in the text
  from: 'name' | 'text'; // 'name': it looks like a name in your world data. 'text': it looks like a typo of a name you use often in the text.
};

const MIN_LEN = 4; // shorter words give too many false alarms
const TOKEN = /\p{L}+(?:-\p{L}+)*/gu;

/** Edit distance, where swapping two next letters counts as one change. Stops early past max. */
export function distance(a: string, b: string, max: number): number {
  if (Math.abs(a.length - b.length) > max) return max + 1;
  const prev2: number[] = [];
  let prev: number[] = [];
  for (let j = 0; j <= b.length; j++) prev[j] = j;
  for (let i = 1; i <= a.length; i++) {
    const cur: number[] = [i];
    let best = i;
    for (let j = 1; j <= b.length; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1;
      let v = Math.min(prev[j] + 1, cur[j - 1] + 1, prev[j - 1] + cost);
      if (i > 1 && j > 1 && a[i - 1] === b[j - 2] && a[i - 2] === b[j - 1]) v = Math.min(v, prev2[j - 2] + 1);
      cur[j] = v;
      if (v < best) best = v;
    }
    if (best > max) return max + 1;
    for (let j = 0; j <= b.length; j++) prev2[j] = prev[j];
    prev = cur;
  }
  return prev[b.length];
}

/** How many changes still count as "the same name": 1 for short names, 2 for long ones. */
const allowed = (len: number) => (len >= 8 ? 2 : 1);

const isCapital = (w: string) => {
  const f = w[0];
  return f !== f.toLowerCase() && f === f.toUpperCase() && w !== w.toUpperCase();
};

/** Words in a name, e.g. "House of Valdoran" gives Valdoran (short words are left out). */
export function nameWords(names: string[]): Map<string, string> {
  const out = new Map<string, string>(); // lower case -> the way you wrote it
  for (const n of names) {
    for (const w of n.match(TOKEN) ?? []) {
      if (w.length >= MIN_LEN && !out.has(w.toLowerCase())) out.set(w.toLowerCase(), w);
    }
  }
  return out;
}

export function findTypos(texts: string[], names: string[]): Typo[] {
  const known = nameWords(names);
  const counts = new Map<string, number>(); // capitalised words in the text
  const ordinary = new Set<string>(); // words that also appear in lower case: not names
  for (const t of texts) {
    for (const m of t.matchAll(TOKEN)) {
      const w = m[0];
      if (w.length < MIN_LEN) continue;
      if (isCapital(w)) counts.set(w, (counts.get(w) ?? 0) + 1);
      else if (w === w.toLowerCase()) ordinary.add(w);
    }
  }

  // known names by first letter, so each word is only compared with a few
  const byLetter = new Map<string, string[]>();
  for (const [low] of known) {
    const k = low[0];
    byLetter.set(k, [...(byLetter.get(k) ?? []), low]);
  }
  // frequent words of the text that are not names in your data: the other side of a pair
  const frequent = new Map<string, string[]>();
  for (const [w, n] of counts) {
    const low = w.toLowerCase();
    if (n >= 3 && !known.has(low) && !ordinary.has(low)) frequent.set(low[0], [...(frequent.get(low[0]) ?? []), w]);
  }

  const out: Typo[] = [];
  for (const [word, count] of counts) {
    const low = word.toLowerCase();
    if (known.has(low) || ordinary.has(low)) continue; // spelled right, or just a normal word at a sentence start

    let best: { w: string; d: number } | null = null;
    for (const cand of byLetter.get(low[0]) ?? []) {
      const max = allowed(Math.min(cand.length, low.length));
      const d = distance(low, cand, max);
      if (d <= max && (!best || d < best.d)) best = { w: known.get(cand)!, d };
    }
    if (best) { out.push({ word, suggest: best.w, count, from: 'name' }); continue; }

    if (count > 2) continue; // a word used often is probably a real name that is not in your data yet
    let other: { w: string; n: number; d: number } | null = null;
    for (const cand of frequent.get(low[0]) ?? []) {
      const n = counts.get(cand)!;
      if (cand === word || n < count * 3) continue;
      const max = allowed(Math.min(cand.length, word.length));
      const d = distance(low, cand.toLowerCase(), max);
      if (d <= max && (!other || d < other.d || (d === other.d && n > other.n))) other = { w: cand, n, d };
    }
    if (other) out.push({ word, suggest: other.w, count, from: 'text' });
  }
  return out.sort((a, b) => b.count - a.count || a.word.localeCompare(b.word));
}
