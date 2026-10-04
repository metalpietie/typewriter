// Compare two texts word by word. No imports, so it can be tested alone.
// A paragraph break is "\n" in the text. It counts as one word-like token, so a moved paragraph break shows up.

export type Kind = 'same' | 'add' | 'del';
/** A run of words that were kept, added or cut. text "\n" is a paragraph break. */
export type Piece = { kind: Kind; text: string };

const BREAK = '\n';
const MAX_EDITS = 1500; // a bigger rewrite is shown as "all cut, all new". Memory grows with the square of this.

export const tokens = (s: string): string[] => (s.match(/\n+|[^\s]+/g) ?? []).map((t) => (t[0] === '\n' ? BREAK : t));

type Op = '=' | '+' | '-';

/** Myers' shortest edit script. Returns null when the texts differ by more than max edits. */
function edits(a: string[], b: string[], max: number): Op[] | null {
  const n = a.length, m = b.length;
  if (!n) return b.map(() => '+' as Op);
  if (!m) return a.map(() => '-' as Op);
  const maxD = Math.min(n + m, max);
  const off = maxD + 1;
  const v = new Int32Array(2 * maxD + 3);
  const trace: Int32Array[] = [];
  for (let d = 0; d <= maxD; d++) {
    trace.push(v.slice()); // the state before round d
    for (let k = -d; k <= d; k += 2) {
      let x = k === -d || (k !== d && v[off + k - 1] < v[off + k + 1]) ? v[off + k + 1] : v[off + k - 1] + 1;
      let y = x - k;
      while (x < n && y < m && a[x] === b[y]) { x++; y++; }
      v[off + k] = x;
      if (x >= n && y >= m) return walkBack(trace, d, n, m, off);
    }
  }
  return null;
}

function walkBack(trace: Int32Array[], D: number, n: number, m: number, off: number): Op[] {
  const ops: Op[] = [];
  let x = n, y = m;
  for (let d = D; d > 0; d--) {
    const v = trace[d];
    const k = x - y;
    const prevK = k === -d || (k !== d && v[off + k - 1] < v[off + k + 1]) ? k + 1 : k - 1;
    const px = v[off + prevK], py = px - prevK;
    while (x > px && y > py) { ops.push('='); x--; y--; }
    ops.push(x === px ? '+' : '-');
    x = px; y = py;
  }
  while (x > 0 && y > 0) { ops.push('='); x--; y--; }
  return ops.reverse();
}

/** What changed from oldText to newText. Words only in newText are "add". Words only in oldText are "del". */
export function diffWords(oldText: string, newText: string): Piece[] {
  const a = tokens(oldText), b = tokens(newText);
  let head = 0;
  while (head < a.length && head < b.length && a[head] === b[head]) head++;
  let tail = 0;
  while (tail < a.length - head && tail < b.length - head && a[a.length - 1 - tail] === b[b.length - 1 - tail]) tail++;
  const ma = a.slice(head, a.length - tail), mb = b.slice(head, b.length - tail);

  const ops: Op[] = Array(head).fill('=');
  const mid = edits(ma, mb, MAX_EDITS);
  if (mid) ops.push(...mid);
  else { ops.push(...ma.map(() => '-' as Op), ...mb.map(() => '+' as Op)); }
  ops.push(...Array<Op>(tail).fill('='));

  // walk both lists with the script, and join words of the same kind
  const out: Piece[] = [];
  let i = 0, j = 0;
  const push = (kind: Kind, word: string) => {
    const last = out[out.length - 1];
    if (word === BREAK || !last || last.kind !== kind || last.text === BREAK) out.push({ kind, text: word });
    else last.text += ' ' + word;
  };
  for (const op of ops) {
    if (op === '=') { push('same', b[j]); i++; j++; }
    else if (op === '+') { push('add', b[j]); j++; }
    else { push('del', a[i]); i++; }
  }
  return out;
}

const words = (p: Piece) => (p.text === BREAK ? 0 : p.text.split(' ').length);

/** How many words were added and cut. */
export function countChanges(pieces: Piece[]): { added: number; cut: number } {
  let added = 0, cut = 0;
  for (const p of pieces) {
    if (p.kind === 'add') added += words(p);
    else if (p.kind === 'del') cut += words(p);
  }
  return { added, cut };
}

/** Pieces grouped in paragraphs. A paragraph break that was cut does not start a new paragraph. */
export function paragraphsOf(pieces: Piece[]): Piece[][] {
  const out: Piece[][] = [[]];
  for (const p of pieces) {
    if (p.text === BREAK) { if (p.kind !== 'del' && out[out.length - 1].length) out.push([]); }
    else out[out.length - 1].push(p);
  }
  return out.filter((x) => x.length);
}
