// The bookshelf: the rules that need no screen. No imports, so it can be tested alone.

/** A made-up cover: the same title and seed always give the same look. A new seed gives another look. */
export type ShelfSpec = {
  template: 'classic' | 'modern' | 'band';
  title: string; subtitle: string; author: string;
  bg: string; fg: string; accent: string; dim: number;
};

// background, text and accent colours that look good together
export const PALETTES: [string, string, string][] = [
  ['#1f2a44', '#f4efe6', '#c9a45c'], ['#3b1f2b', '#f6ece4', '#e0a458'], ['#12343b', '#f1f5f2', '#e8b04b'],
  ['#2d3a2e', '#f3f1e4', '#c7b26b'], ['#2b2250', '#f0eefb', '#f2a65a'], ['#5a2a27', '#fbefe4', '#e9c46a'],
  ['#102a43', '#f0f4f8', '#52c7b8'], ['#f1e8d8', '#2a2118', '#a4452c'], ['#e9eef2', '#14213d', '#e07a5f'],
  ['#232323', '#f5f5f5', '#d4af37'], ['#4a1942', '#f8eef6', '#f4a261'], ['#0b3d2e', '#eef6f1', '#f0c987'],
];
const TEMPLATES: ShelfSpec['template'][] = ['classic', 'modern', 'band'];

/** A whole number from a text (FNV-1a). */
export function hash(text: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < text.length; i++) { h ^= text.charCodeAt(i); h = Math.imul(h, 0x01000193); }
  return h >>> 0;
}

export function shelfSpec(title: string, seed = 0): ShelfSpec {
  const name = title.trim() || 'Untitled';
  const h = hash(name.toLowerCase() + '#' + seed);
  const [bg, fg, accent] = PALETTES[h % PALETTES.length];
  return { template: TEMPLATES[(h >>> 8) % TEMPLATES.length], title: name, subtitle: '', author: '', bg, fg, accent, dim: 0 };
}

/** The list without the moving item, and with it put back at an index. The index counts the list without the moving item. */
export function insertAt<T>(list: T[], moving: T, index: number): T[] {
  const rest = list.filter((x) => x !== moving);
  const at = Math.max(0, Math.min(rest.length, index));
  return [...rest.slice(0, at), moving, ...rest.slice(at)];
}

export type Box = { x: number; y: number; w: number; h: number };

/**
 * Where a dragged book would land among the books of a shelf: the number of books that stay before it.
 * boxes are the other books of the shelf in reading order (left to right, row by row). The point is the pointer.
 */
export function dropIndex(boxes: Box[], px: number, py: number): number {
  for (let i = 0; i < boxes.length; i++) {
    const b = boxes[i];
    const after = py < b.y + b.h && (py < b.y || px < b.x + b.w / 2);
    if (after) return i;
  }
  return boxes.length;
}

/** The index of the shelf that the pointer is over. Between shelves, the nearest one in height. -1 when there are none. */
export function shelfAt(shelves: Box[], px: number, py: number): number {
  let best = -1, bestD = Infinity;
  for (let i = 0; i < shelves.length; i++) {
    const s = shelves[i];
    const dy = py < s.y ? s.y - py : py > s.y + s.h ? py - (s.y + s.h) : 0;
    const dx = px < s.x ? s.x - px : px > s.x + s.w ? px - (s.x + s.w) : 0;
    const d = dy + dx;
    if (d < bestD) { best = i; bestD = d; }
  }
  return best;
}

/** Has the pointer moved far enough from where it went down to start a drag? (A short move is still a click.) */
export const dragStarted = (dx: number, dy: number, threshold = 6): boolean => Math.hypot(dx, dy) >= threshold;

/** How many pixels to scroll while dragging near the top or bottom edge of the view. Negative is up. */
export function edgeScroll(py: number, top: number, bottom: number, zone = 70, max = 18): number {
  if (py < top + zone) return -Math.round(max * Math.min(1, (top + zone - py) / zone));
  if (py > bottom - zone) return Math.round(max * Math.min(1, (py - (bottom - zone)) / zone));
  return 0;
}

/** "12 chapters · 34,500 words" (small numbers are said as they are). */
export function bookLine(chapters: number, words: number, chapterWord = 'chapter'): string {
  const c = `${chapters} ${chapterWord}${chapters === 1 ? '' : 's'}`;
  return `${c} · ${words.toLocaleString('en-US')} words`;
}
