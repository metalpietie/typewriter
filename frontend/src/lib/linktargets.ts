import { db } from './db.svelte';
import { LINK_KINDS, labelOf, type LinkKind } from './backlinks';
import { hasKind } from './projecttype.svelte';

/** Something an @ link can point at. */
export type Cand = { kind: LinkKind; id: number; label: string; word: string; also: string };

/** The characters, places, lore, factions and items that match what you typed. The best match is first. */
export function linkTargets(q: string, limit = 8): Cand[] {
  const query = q.trim().toLowerCase();
  const all: Cand[] = [];
  for (const { kind, word } of LINK_KINDS) {
    if (!hasKind(kind)) continue;
    for (const it of db.lists[kind]) {
      const label = labelOf(kind, it);
      if (label) all.push({ kind, id: it.id, label, word, also: String(it.aliases ?? '').toLowerCase() });
    }
  }
  if (!query) return all.slice(0, limit);
  const score = (c: Cand) => {
    const l = c.label.toLowerCase();
    if (l === query) return -0.5; // the exact name comes first
    if (l.startsWith(query)) return 0;
    if (l.split(/\s+/).some((w) => w.startsWith(query))) return 1;
    if (c.also.includes(query)) return 2;
    return l.includes(query) ? 3 : -1;
  };
  return all.map((c) => [score(c), c] as const).filter(([n]) => n !== -1).sort((a, b) => a[0] - b[0]).slice(0, limit).map(([, c]) => c);
}
