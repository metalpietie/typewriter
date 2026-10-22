import { db, type Item } from './db.svelte';
import { scenesOf, sceneLabel } from './scenes';
import { namesOf, pattern } from './mentions';

export type LinkKind = 'character' | 'location' | 'note' | 'faction' | 'item' | 'term';

/** What an @ link can point at. */
export const LINK_KINDS: { kind: LinkKind; word: string }[] = [
  { kind: 'character', word: 'character' },
  { kind: 'location', word: 'place' },
  { kind: 'note', word: 'lore' },
  { kind: 'faction', word: 'faction' },
  { kind: 'item', word: 'item' },
  { kind: 'term', word: 'glossary' },
];

/** The name an item goes by in the text. */
export const labelOf = (kind: LinkKind, it: Item): string => String((kind === 'note' ? it.title : kind === 'term' ? it.word : it.name) ?? '').trim();

export type Back = { scene: Item; label: string; tagged: boolean; named: number };

/** Plain text of saved scene HTML. Fast and rough: good enough to find names. */
const strip = (html: string) =>
  html.replace(/<\/(p|h[1-6]|li|blockquote)>|<br\s*\/?>/gi, ' ').replace(/<[^>]+>/g, '')
    .replace(/&amp;/g, '&').replace(/&#39;|&apos;/g, "'").replace(/&quot;/g, '"').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&nbsp;/g, ' ');

/** Every scene that tags an item with @ or names it in the text, in reading order. */
export function backlinks(kind: LinkKind, id: number): Back[] {
  const it = db.get(kind, id);
  if (!it) return [];
  const names = kind === 'character' ? namesOf(it) : labelOf(kind, it).length >= 3 ? [{ text: labelOf(kind, it), ignoreCase: false }] : [];
  const res = names.map((n) => new RegExp(pattern(n.text, n.ignoreCase).source, n.ignoreCase ? 'giu' : 'gu'));
  const tag = `data-ref="${kind}:${id}"`;
  const out: Back[] = [];
  const books = [...db.lists.book].sort((a, b) => a.order - b.order);
  for (const b of books) {
    for (const c of db.lists.chapter.filter((x) => x.book === b.id).sort((x, y) => x.order - y.order)) {
      for (const s of scenesOf(c.id)) {
        const body = s.body ?? '';
        const tagged = body.includes(tag);
        let named = 0;
        if (res.length && body) {
          const text = strip(body);
          const spans: [number, number][] = [];
          for (const re of res) for (const m of text.matchAll(re)) spans.push([m.index!, m.index! + m[0].length]);
          // "Aldric Stone" also holds "Aldric": count a place in the text once
          spans.sort((x, y) => x[0] - y[0]);
          let end = -1;
          for (const [from, to] of spans) if (from >= end) { named++; end = to; }
        }
        if (tagged || named) out.push({ scene: s, label: `${b.title} · ${sceneLabel(s)}`, tagged, named });
      }
    }
  }
  return out;
}
