/**
 * Repeated word check, the plain part (no app data in here).
 * Three things: a word used twice close together, sentences that start the same way,
 * and the words a text uses most.
 */

// Everyday words that are fine to repeat.
const STOP = new Set((
  'about above after again against all also always among and another any anyone anything are around away back because been before behind being below between both but came can cannot come could did does doing done down during each either else even ever every everyone everything few for from gave get gets getting give goes going gone got had has have having her here hers herself him himself his how however into its itself just keep kept know knew let like made make many may maybe might more most much must myself never next none nor not nothing now off often once one only onto other others our ours out over own perhaps put quite rather said say says see seem seemed seems she should since some someone something still such take taken than that their theirs them then there these they thing things this those though thought through thus till too took toward towards under until upon very want wanted was way well went were what whatever when where whether which while who whom whose why will with within without would yet you your yours yourself ' +
  'asked looked turned began started then than ' +
  "don't didn't can't won't wasn't isn't aren't couldn't wouldn't shouldn't hadn't hasn't haven't i'm i've i'll i'd he's she's it's that's there's what's let's we're we've we'll they're they've they'll you're you've you'll"
).split(/\s+/));

const WORD = /\p{L}+(?:['’]\p{L}+)?/gu;
const norm = (w: string) => w.toLowerCase().replace(/’/g, "'");

type Tok = { w: string; at: number; end: number };

function tokens(text: string): Tok[] {
  const out: Tok[] = [];
  for (const m of text.matchAll(WORD)) out.push({ w: norm(m[0]), at: m.index!, end: m.index! + m[0].length });
  return out;
}

const WINDOW = 20; // words
const MIN_LEN = 4;

/** Text around a place, with the repeated parts marked ⟦like this⟧. */
function around(text: string, from: number, to: number, marks: [number, number][]): string {
  const a = Math.max(0, from - 20), b = Math.min(text.length, to + 20);
  let s = '', pos = a;
  for (const [m0, m1] of [...marks].sort((x, y) => x[0] - y[0])) {
    if (m0 < pos || m1 > b) continue;
    s += text.slice(pos, m0) + '⟦' + text.slice(m0, m1) + '⟧';
    pos = m1;
  }
  s += text.slice(pos, b);
  return (a > 0 ? '…' : '') + s.replace(/\s+/g, ' ') + (b < text.length ? '…' : '');
}

export type Close = { word: string; times: number; clusters: number; snippet: string };

/** Words (4+ letters, not everyday words, not names) used again within a few words. */
export function closeRepeats(text: string, skip: Set<string>): Close[] {
  const toks = tokens(text);
  const last = new Map<string, number>();
  const found = new Map<string, { idx: number[]; first: number }>();
  const clusterEnd = new Map<string, number>(); // token index where the open cluster of a word last grew
  const clusters = new Map<string, number>();
  toks.forEach((t, i) => {
    if (t.w.length < MIN_LEN || STOP.has(t.w) || skip.has(t.w)) return;
    const prev = last.get(t.w);
    last.set(t.w, i);
    if (prev === undefined || i - prev > WINDOW) return;
    const f = found.get(t.w) ?? { idx: [prev], first: prev };
    if (!found.has(t.w)) found.set(t.w, f);
    if (f.idx[f.idx.length - 1] !== prev) f.idx.push(prev);
    f.idx.push(i);
    if (clusterEnd.get(t.w) === undefined || prev !== clusterEnd.get(t.w)) clusters.set(t.w, (clusters.get(t.w) ?? 0) + 1);
    clusterEnd.set(t.w, i);
  });
  const out: Close[] = [];
  for (const [word, f] of found) {
    const idx = [...new Set(f.idx)];
    const a = toks[f.first], b = toks[Math.min(idx[1], toks.length - 1)];
    out.push({ word, times: idx.length, clusters: clusters.get(word) ?? 1, snippet: around(text, a.at, b.end, [[a.at, a.end], [b.at, b.end]]) });
  }
  return out.sort((x, y) => y.times - x.times || x.word.localeCompare(y.word));
}

export type Start = { word: string; run: number; at: number; snippet: string };

/** The same first word on 3 or more sentences in a row. */
export function sameStarts(text: string, minRun = 3): Start[] {
  const sentences: { word: string; at: number; end: number }[] = [];
  const re = /[^.!?…\n]+(?:[.!?…]+["'”’)\]]*|$)/g;
  for (const m of text.matchAll(re)) {
    const first = /\p{L}+(?:['’]\p{L}+)?/u.exec(m[0]);
    if (!first) continue;
    sentences.push({ word: norm(first[0]), at: m.index! + first.index, end: m.index! + m[0].length });
  }
  const out: Start[] = [];
  for (let i = 0; i < sentences.length;) {
    let j = i + 1;
    while (j < sentences.length && sentences[j].word === sentences[i].word) j++;
    if (j - i >= minRun) {
      const marks = sentences.slice(i, Math.min(j, i + 3)).map((s): [number, number] => [s.at, s.at + s.word.length]);
      out.push({ word: sentences[i].word, run: j - i, at: sentences[i].at, snippet: around(text, sentences[i].at, sentences[Math.min(j, i + 3) - 1].at + 12, marks) });
    }
    i = j;
  }
  return out;
}

export type Top = { word: string; count: number; per1000: number };

/** The words used most, over all the texts. Everyday words and names are left out. */
export function topWords(texts: string[], skip: Set<string>, limit = 20): { words: Top[]; total: number } {
  const counts = new Map<string, number>();
  let total = 0;
  for (const t of texts) {
    for (const k of tokens(t)) {
      total++;
      if (k.w.length < 5 || STOP.has(k.w) || skip.has(k.w)) continue;
      counts.set(k.w, (counts.get(k.w) ?? 0) + 1);
    }
  }
  const words = [...counts]
    .filter(([, c]) => c >= 5)
    .map(([word, count]) => ({ word, count, per1000: total ? Math.round((count / total) * 10000) / 10 : 0 }))
    .sort((a, b) => b.count - a.count || a.word.localeCompare(b.word))
    .slice(0, limit);
  return { words, total };
}
