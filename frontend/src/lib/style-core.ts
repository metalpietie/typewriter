/**
 * Style check, the plain part (no app data in here).
 * It works on paragraphs of plain text. It measures how much is dialogue, finds very long sentences,
 * and finds weak words: filter words (felt, saw), empty words (very, really), and words ending in -ly.
 */

const WORD = /\p{L}[\p{L}'’-]*/gu;
export const wordCount = (s: string): number => (s.match(WORD) ?? []).length;

// ---------- dialogue ----------

/**
 * Words inside quotation marks, and all words. A quote that is not closed ends with its paragraph.
 * “curly” quotes open and close. Straight quotes switch on and off.
 */
export function dialogueWords(paragraphs: string[]): { dialogue: number; total: number } {
  let dialogue = 0, total = 0;
  for (const p of paragraphs) {
    let inside = false, seg = '';
    const flush = () => { const n = wordCount(seg); total += n; if (inside) dialogue += n; seg = ''; };
    for (const ch of p) {
      if (ch === '“') { flush(); inside = true; }
      else if (ch === '”') { flush(); inside = false; }
      else if (ch === '"') { flush(); inside = !inside; }
      else seg += ch;
    }
    flush();
  }
  return { dialogue, total };
}

// ---------- sentences ----------

// A full stop after these does not end a sentence.
const ABBREVIATIONS = new Set(['mr', 'mrs', 'ms', 'dr', 'st', 'prof', 'sr', 'jr', 'vs', 'etc', 'mt', 'capt', 'col', 'gen', 'lt', 'sgt', 'no', 'fig']);

export type Sentence = { text: string; words: number };

/** The sentences of some paragraphs. A sentence ends at . ! ? or … when a capital letter, a quote, or nothing follows. */
export function sentences(paragraphs: string[]): Sentence[] {
  const out: Sentence[] = [];
  const end = /[.!?…]+["”’')\]]*(?=\s|$)/g;
  for (const p of paragraphs) {
    let start = 0;
    for (const m of p.matchAll(end)) {
      const at = m.index! + m[0].length;
      const before = p.slice(start, m.index!);
      const last = (before.match(/\p{L}+$/u) ?? [''])[0];
      const lone = m[0] === '.' && (ABBREVIATIONS.has(last.toLowerCase()) || (last.length === 1 && last === last.toUpperCase()));
      const next = p.slice(at).trimStart()[0] ?? '';
      const continues = next !== '' && next === next.toLowerCase() && next !== next.toUpperCase();
      if (lone || continues) continue;
      push(out, p.slice(start, at));
      start = at;
    }
    if (p.slice(start).trim()) push(out, p.slice(start));
  }
  return out;
}

function push(out: Sentence[], raw: string) {
  const text = raw.trim();
  const words = wordCount(text);
  if (words) out.push({ text, words });
}

// ---------- weak words ----------

export type Group = 'filter' | 'empty' | 'ly';
export const GROUPS: { id: Group; title: string; help: string }[] = [
  { id: 'filter', title: 'Filter words', help: 'They put a step between the reader and the scene: “She saw the door open” instead of “The door opened”.' },
  { id: 'empty', title: 'Empty words', help: 'They add little. Often the sentence is just as good without them.' },
  { id: 'ly', title: 'Words that end in -ly', help: 'Many are adverbs. A strong verb often does the job better. Some are fine.' },
];

const FILTER = new Set('felt saw heard noticed realized realised wondered seemed watched knew thought'.split(' '));
const EMPTY = new Set('very really just quite rather actually basically suddenly literally somewhat'.split(' '));
// ends in -ly, but is not an adverb you could cut
const LY_OK = new Set((
  'only family early lonely lovely friendly ugly belly silly holy july italy reply supply apply rely fly butterfly assembly anomaly bully jelly rally ally tally folly ' +
  'daily weekly monthly yearly deadly elderly orderly costly likely unlikely jolly hilly curly wobbly lively lonely ghostly kindly motherly fatherly brotherly sisterly ' +
  'worldly heavenly earthly cowardly scholarly timely manly womanly queenly kingly princely holly dolly molly polly sally billy willy lily emily kelly shelly ' +
  'hillbilly oatmeal-ly bodily chilly goodly godly gnarly burly surly curly swirly twirly pearly early'
).split(/\s+/));

export type Weak = { word: string; group: Group; at: number; end: number };

/** Every weak word in a text, with where it is. */
export function weakWords(text: string): Weak[] {
  const out: Weak[] = [];
  for (const m of text.matchAll(WORD)) {
    const word = m[0].toLowerCase().replace(/’/g, "'");
    let group: Group | null = null;
    if (FILTER.has(word)) group = 'filter';
    else if (EMPTY.has(word)) group = 'empty';
    else if (word.length >= 5 && word.endsWith('ly') && !LY_OK.has(word)) group = 'ly';
    if (group) out.push({ word, group, at: m.index!, end: m.index! + m[0].length });
  }
  return out;
}

/** Text around a place, with the part marked ⟦like this⟧. */
export function around(text: string, from: number, to: number): string {
  const a = Math.max(0, from - 40), b = Math.min(text.length, to + 40);
  return (a > 0 ? '…' : '') + (text.slice(a, from) + '⟦' + text.slice(from, to) + '⟧' + text.slice(to, b)).replace(/\s+/g, ' ') + (b < text.length ? '…' : '');
}

/** A short number for a sentence, so a skipped sentence is still skipped after a small edit elsewhere. */
export function hashOf(s: string): string {
  let h = 2166136261;
  for (const ch of s.toLowerCase().replace(/\s+/g, ' ').trim()) { h ^= ch.codePointAt(0)!; h = Math.imul(h, 16777619); }
  return (h >>> 0).toString(36);
}
