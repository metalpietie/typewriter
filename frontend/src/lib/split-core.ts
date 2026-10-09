// Rules for splitting a scene in two and merging two scenes. No imports, so it can be tested alone.

const EMPTY_P = '<p(?:\\s[^>]*)?>(?:\\s|<br\\s*/?>|&nbsp;)*</p>';
const LEADING = new RegExp('^(?:\\s*' + EMPTY_P + ')+', 'i');
const TRAILING = new RegExp('(?:' + EMPTY_P + '\\s*)+$', 'i');

/** The text of some HTML, without the tags. */
const textOf = (html: string) => String(html ?? '').replace(/<[^>]*>/g, ' ').replace(/&nbsp;/g, ' ').trim();

/** True when the HTML holds any words. */
export const hasText = (html: string): boolean => textOf(html) !== '';

/** The HTML without empty paragraphs at its start and its end. A cut at the end of a paragraph leaves one. */
export const trimEmptyParas = (html: string): string =>
  String(html ?? '').replace(LEADING, '').replace(TRAILING, '').trim()
    .replace(/^(<p(?:\s[^>]*)?>)\s+/i, '$1') // a cut inside a sentence leaves a space at the edge
    .replace(/\s+(<\/p>)$/i, '$1');

/** The ids of the comments that mark text in this HTML. */
export function commentIds(html: string): number[] {
  const ids = new Set<number>();
  for (const m of String(html ?? '').matchAll(/data-comment="(\d+)"/g)) ids.add(Number(m[1]));
  return [...ids];
}

/** Two bodies as one. Text saved in the old plain form is made into paragraphs first. */
export function joinBodies(a: string, b: string, toHtml: (s: string) => string): string {
  const x = toHtml(a ?? ''), y = toHtml(b ?? '');
  if (!hasText(x)) return y;
  if (!hasText(y)) return x;
  return x + y;
}

/** The two lists as one, without repeats, in order. */
export const union = <T,>(a: T[] | null | undefined, b: T[] | null | undefined): T[] => [...new Set([...(a ?? []), ...(b ?? [])])];

/** What both lists have. A pass is done for the new scene only when it was done for both. */
export const both = <T,>(a: T[] | null | undefined, b: T[] | null | undefined): T[] => (a ?? []).filter((x) => (b ?? []).includes(x));

/** Two short texts as one line each, with a space between. An empty one is left out. */
export const joinText = (a: string | null | undefined, b: string | null | undefined): string =>
  [String(a ?? '').trim(), String(b ?? '').trim()].filter(Boolean).join(' ');

/** The order numbers 1, 2, 3 … for a list that may have gaps or repeats, keeping the order it has. */
export const renumbered = <T extends { order: number }>(list: T[]): T[] =>
  [...list].sort((a, b) => a.order - b.order).map((s, i) => ({ ...s, order: i + 1 }));
