// Scene tags. No imports, so it can be tested alone.

export type Tagged = { id?: number; tags?: string[] | null };

export const MAX_TAG = 30;

/** "  Dream  Scene " becomes "dream scene". Empty text gives "". */
export function cleanTag(raw: string): string {
  return String(raw ?? '').replace(/[,#]/g, ' ').replace(/\s+/g, ' ').trim().toLowerCase().slice(0, MAX_TAG).trim();
}

/** The tags of one scene, clean and without repeats. */
export function tagsOf(s: Tagged): string[] {
  const out: string[] = [];
  for (const t of s.tags ?? []) { const c = cleanTag(t); if (c && !out.includes(c)) out.push(c); }
  return out;
}

/** The list with one tag added. The same list comes back when the tag is empty or already there. */
export function withTag(list: string[], raw: string): string[] {
  const c = cleanTag(raw);
  return !c || list.includes(c) ? list : [...list, c];
}

export const withoutTag = (list: string[], tag: string): string[] => list.filter((t) => t !== tag);

/** Every tag in use and how many scenes have it. The most used come first. */
export function tagCounts(scenes: Tagged[]): [string, number][] {
  const n = new Map<string, number>();
  for (const s of scenes) for (const t of tagsOf(s)) n.set(t, (n.get(t) ?? 0) + 1);
  return [...n].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
}

/** An empty tag means "all scenes". */
export const hasTag = (s: Tagged, tag: string): boolean => !tag || tagsOf(s).includes(tag);
