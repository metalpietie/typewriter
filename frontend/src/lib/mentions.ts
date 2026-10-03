import { db, plainText, type Item } from './db.svelte';

const cache = new Map<string, RegExp>();

/** A name as a whole-word pattern. "Aldric" matches "Aldric's", not "Aldrich". */
function pattern(name: string, ignoreCase: boolean): RegExp {
  const key = (ignoreCase ? 'i:' : 's:') + name;
  let re = cache.get(key);
  if (!re) {
    const src = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    re = new RegExp(`(?<![\\p{L}\\p{N}_])${src}(?![\\p{L}\\p{N}_])`, ignoreCase ? 'iu' : 'u');
    cache.set(key, re);
  }
  return re;
}

/**
 * The names a writer may use for a character in the text:
 * the full name, the first name (when no other character shares it), and each "also called" name.
 */
function namesOf(c: Item): { text: string; ignoreCase: boolean }[] {
  const out: { text: string; ignoreCase: boolean }[] = [];
  const full = String(c.name ?? '').trim();
  if (full.length >= 2) out.push({ text: full, ignoreCase: false });
  const first = full.split(/\s+/)[0];
  if (first && first !== full && first.length >= 3) {
    const shared = db.lists.character.some((o) => o.id !== c.id && String(o.name ?? '').trim().split(/\s+/)[0] === first);
    if (!shared) out.push({ text: first, ignoreCase: false });
  }
  for (const a of String(c.aliases ?? '').split(/[,;\n]/)) {
    const t = a.trim();
    if (t.length >= 3) out.push({ text: t, ignoreCase: true });
  }
  return out;
}

/** Characters whose name appears in the text of a scene. */
export function mentionedIn(scene: Item): Item[] {
  const text = plainText(scene.body ?? '');
  if (!text.trim()) return [];
  return db.lists.character.filter((c) => namesOf(c).some((n) => pattern(n.text, n.ignoreCase).test(text)));
}

/** Named in the text, but not in the cast and not marked "not here". */
export function missingCast(scene: Item): Item[] {
  const inCast = new Set<number>([...(scene.pov ? [scene.pov] : []), ...(scene.characters ?? []), ...(scene.ignoreMentions ?? [])]);
  return mentionedIn(scene).filter((c) => !inCast.has(c.id));
}
