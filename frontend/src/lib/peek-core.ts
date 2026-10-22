// What the card shows when you point at an @ link in the text. No imports, so it can be tested alone.

export type PeekKind = 'character' | 'location' | 'note' | 'faction' | 'item' | 'term';
export type Peek = { title: string; sub: string; text: string; color: string };

const WORD: Record<PeekKind, string> = { character: 'Character', location: 'Place', note: 'Lore', faction: 'Faction', item: 'Item', term: 'Glossary' };
export const MAX_TEXT = 180;

/** Plain text of a bit of HTML (a lore note body), with the spaces made tidy. */
export function plain(html: string): string {
  return String(html ?? '')
    .replace(/<\/(p|h[1-6]|li|blockquote)>|<br\s*\/?>/gi, ' ').replace(/<[^>]+>/g, '')
    .replace(/&nbsp;/g, ' ').replace(/&amp;/g, '&').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"').replace(/&#39;|&apos;/g, "'")
    .replace(/\s+/g, ' ').trim();
}

/** Text cut at a word, with … when it was longer. */
export function clip(text: string, max = MAX_TEXT): string {
  if (text.length <= max) return text;
  const cut = text.slice(0, max);
  const sp = cut.lastIndexOf(' ');
  return (sp > max * 0.6 ? cut.slice(0, sp) : cut).replace(/[\s,.;:–—-]+$/, '') + '…';
}

/** The card of one item. A missing item (deleted) gives null. */
export function peekOf(kind: PeekKind, it: Record<string, any> | null | undefined): Peek | null {
  if (!it) return null;
  const title = String((kind === 'note' ? it.title : kind === 'term' ? it.word : it.name) ?? '').trim() || 'Untitled';
  const kindWord = String((kind === 'note' ? it.category : kind === 'character' ? it.role : it.type) ?? '').trim();
  const sub = [WORD[kind], kindWord].filter(Boolean).join(' · ');
  const raw = kind === 'note' ? plain(it.body) : String((kind === 'term' ? it.meaning : it.description) ?? '').replace(/\s+/g, ' ').trim();
  const color = kind === 'character' || kind === 'faction' ? String(it.color ?? '') : '';
  return { title, sub, text: clip(raw), color };
}
