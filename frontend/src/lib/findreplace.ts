import { db, toHtml, type Item } from './db.svelte';
import { scenesOf, sceneLabel } from './scenes';

export type Options = { query: string; replacement: string; matchCase: boolean; wholeWord: boolean; allBooks: boolean; bookId: number | null };

const escape = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

/** The search as a regular expression, or null when there is nothing to look for. */
export function buildRegex(o: Pick<Options, 'query' | 'matchCase' | 'wholeWord'>): RegExp | null {
  if (!o.query) return null;
  let src = escape(o.query);
  if (o.wholeWord) src = `(?<![\\p{L}\\p{N}_])${src}(?![\\p{L}\\p{N}_])`;
  return new RegExp(src, o.matchCase ? 'gu' : 'giu');
}

function textNodes(root: Node): Text[] {
  const out: Text[] = [];
  const walk = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  for (let n = walk.nextNode(); n; n = walk.nextNode()) out.push(n as Text);
  return out;
}

export type Hit = { before: string; match: string; after: string };

/** Every match in a scene's text, with a little text around it. */
export function hitsIn(html: string, re: RegExp): Hit[] {
  const doc = new DOMParser().parseFromString(toHtml(html), 'text/html');
  const hits: Hit[] = [];
  for (const t of textNodes(doc.body)) {
    const s = t.data;
    for (const m of s.matchAll(re)) {
      hits.push({
        before: s.slice(Math.max(0, m.index! - 36), m.index!),
        match: m[0],
        after: s.slice(m.index! + m[0].length, m.index! + m[0].length + 36),
      });
    }
  }
  return hits;
}

/**
 * Replace in the text only, never in tags.
 * A match that runs across a style change (half bold, half plain) is not found.
 */
export function replaceIn(html: string, re: RegExp, replacement: string): { html: string; count: number } {
  const doc = new DOMParser().parseFromString(toHtml(html), 'text/html');
  let count = 0;
  for (const t of textNodes(doc.body)) {
    const next = t.data.replace(re, () => { count++; return replacement; });
    if (next !== t.data) t.data = next;
  }
  return { html: count ? doc.body.innerHTML : html, count };
}

export type SceneHits = { scene: Item; label: string; hits: Hit[] };

/** Scenes that hold the search text, in reading order. */
export function search(o: Options): SceneHits[] {
  const re = buildRegex(o);
  if (!re) return [];
  const out: SceneHits[] = [];
  const books = [...db.lists.book].sort((a, b) => a.order - b.order).filter((b) => o.allBooks || b.id === o.bookId);
  for (const b of books) {
    const chapters = db.lists.chapter.filter((c) => c.book === b.id).sort((a, b2) => a.order - b2.order);
    for (const c of chapters) {
      for (const s of scenesOf(c.id)) {
        const hits = hitsIn(s.body ?? '', re);
        if (hits.length) out.push({ scene: s, label: `${o.allBooks ? b.title + ' · ' : ''}${sceneLabel(s)}`, hits });
      }
    }
  }
  return out;
}
