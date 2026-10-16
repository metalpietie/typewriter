// Vim text objects (iw, aw, i", a(, is ...) on the text of one paragraph. No imports, so it can be tested alone.
// A range is [from, to): from is the first character, to is the first character after it. Both are offsets in the text.

export type Range = [number, number];

const SPACE = /\s/;
const WORD = /[\p{L}\p{N}_]/u;

/** 0: white space. 1: a letter, a digit or _ (or any other non-space for a big WORD). 2: any other mark. */
const cls = (ch: string, big: boolean): number => (SPACE.test(ch) ? 0 : big ? 1 : WORD.test(ch) ? 1 : 2);

/** The run of equal characters around i, as [start, end, class]. */
function run(text: string, i: number, big: boolean): [number, number, number] {
  const c = cls(text[i], big);
  let s = i, e = i + 1;
  while (s > 0 && cls(text[s - 1], big) === c) s--;
  while (e < text.length && cls(text[e], big) === c) e++;
  return [s, e, c];
}

function word(text: string, pos: number, around: boolean, big: boolean, count: number): Range | null {
  if (!text.length) return null;
  pos = Math.min(pos, text.length - 1);
  const [s0, e0, c0] = run(text, pos, big);
  if (!around) {
    let e = e0;
    for (let k = 1; k < count && e < text.length; k++) e = run(text, e, big)[1];
    return [s0, e];
  }
  let s = s0, e = e0;
  if (c0 === 0) { // on white space: the space and the word after it
    for (let k = 0; k < count && e < text.length; k++) {
      e = run(text, e, big)[1];           // the word
      if (k < count - 1 && e < text.length) e = run(text, e, big)[1]; // the space before the next word
    }
    return [s, e];
  }
  let trailing = false;
  for (let k = 0; k < count; k++) {
    if (k > 0) { if (e >= text.length) break; e = run(text, e, big)[1]; } // the next word
    trailing = e < text.length && cls(text[e], big) === 0;
    if (trailing) e = run(text, e, big)[1];
  }
  if (!trailing && s > 0 && cls(text[s - 1], big) === 0) s = run(text, s - 1, big)[0]; // no space after: take the space before
  return [s, e];
}

const QUOTES = new Set(['"', "'", '`']);

/** The pair of quotes around pos, or the first pair after it. Quotes are paired from the start of the paragraph. */
function quoted(text: string, pos: number, q: string, around: boolean): Range | null {
  const marks: number[] = [];
  for (let i = 0; i < text.length; i++) {
    if (text[i] !== q || text[i - 1] === '\\') continue;
    // an apostrophe inside a word (it's, don't) is not a quote
    if (q === "'" && i > 0 && i + 1 < text.length && WORD.test(text[i - 1]) && WORD.test(text[i + 1])) continue;
    marks.push(i);
  }
  let pair: Range | null = null;
  for (let k = 0; k + 1 < marks.length; k += 2) {
    const [s, e] = [marks[k], marks[k + 1]];
    if (pos >= s && pos <= e) { pair = [s, e]; break; }
    if (s > pos && !pair) pair = [s, e];
  }
  if (!pair) return null;
  const [s, e] = pair;
  if (!around) return [s + 1, e];
  return withSpace(text, s, e + 1);
}

/** a" and aw take the white space after the object, or before it when there is none after. */
function withSpace(text: string, s: number, e: number): Range {
  if (e < text.length && SPACE.test(text[e])) { while (e < text.length && SPACE.test(text[e])) e++; return [s, e]; }
  while (s > 0 && SPACE.test(text[s - 1])) s--;
  return [s, e];
}

const OPEN: Record<string, [string, string]> = {
  '(': ['(', ')'], ')': ['(', ')'], b: ['(', ')'],
  '{': ['{', '}'], '}': ['{', '}'], B: ['{', '}'],
  '[': ['[', ']'], ']': ['[', ']'],
  '<': ['<', '>'], '>': ['<', '>'],
};

/** The count-th pair of brackets that has pos inside it (or on it), counted from the inside. */
function bracketed(text: string, pos: number, open: string, close: string, around: boolean, count: number): Range | null {
  const pairs: Range[] = [];
  const stack: number[] = [];
  for (let i = 0; i < text.length; i++) {
    if (text[i] === open) stack.push(i);
    else if (text[i] === close && stack.length) pairs.push([stack.pop()!, i]);
  }
  const around_ = pairs.filter(([s, e]) => s <= pos && pos <= e).sort((a, b) => b[0] - a[0]);
  const hit = around_[count - 1];
  if (!hit) return null;
  return around ? [hit[0], hit[1] + 1] : [hit[0] + 1, hit[1]];
}

/** The sentences of a paragraph, each as [start, end of text, end with the space after it). */
function sentences(text: string): [number, number, number][] {
  const out: [number, number, number][] = [];
  const re = /\S[^]*?(?:[.!?]+["')\]”’]*(?=\s|$)|$)\s*/g;
  let m: RegExpExecArray | null;
  while ((m = re.exec(text)) && m[0].length) {
    const full = m[0];
    out.push([m.index, m.index + full.trimEnd().length, m.index + full.length]);
  }
  return out;
}

function sentence(text: string, pos: number, around: boolean): Range | null {
  const list = sentences(text);
  for (let k = 0; k < list.length; k++) {
    const [s, e, f] = list[k];
    if (pos >= s && pos < e) {
      if (!around) return [s, e];
      if (f > e) return [s, f];                               // the space after it
      return k > 0 ? [list[k - 1][1], e] : [s, e];            // the last one: the space before it
    }
    if (pos >= e && pos < f) { // in the space between two sentences
      if (!around) return [e, f];
      return k + 1 < list.length ? [e, list[k + 1][1]] : [e, f];
    }
  }
  return null;
}

/**
 * The range of a text object. kind is "i" (inner) or "a" (around). obj is the key after it: w W s p " ' ` ( ) b { } B [ ] < >.
 * pos is the cursor, as an offset in the text of the paragraph.
 */
export function textObject(text: string, pos: number, kind: 'i' | 'a', obj: string, count = 1): Range | null {
  const around = kind === 'a';
  pos = Math.max(0, pos);
  if (obj === 'w') return word(text, pos, around, false, count);
  if (obj === 'W') return word(text, pos, around, true, count);
  if (obj === 's') return sentence(text, Math.min(pos, Math.max(0, text.length - 1)), around);
  if (obj === 'p') return text.length ? [0, text.length] : null; // a paragraph is a block here
  if (QUOTES.has(obj)) {
    const r = quoted(text, pos, obj, around);
    if (r || obj !== '"') return r;
    // typographic quotes: “ and ”
    const s = text.lastIndexOf('“', pos), e = text.indexOf('”', Math.max(pos, s + 1));
    if (s >= 0 && e > s && !text.slice(s, pos).includes('”')) return around ? withSpace(text, s, e + 1) : [s + 1, e];
    return null;
  }
  const pair = OPEN[obj];
  if (pair) return bracketed(text, pos, pair[0], pair[1], around, count);
  return null;
}
