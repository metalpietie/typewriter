import { db, type Item } from './db.svelte';

/** The journey of one item: who had it, and where, in order. */
export function logsOf(itemId: number): Item[] {
  return db.lists.itemlog.filter((l) => l.item === itemId).sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.id - b.id);
}

/** Where an item is now: its last log entry. */
export function nowOf(itemId: number): { holder: Item | undefined; place: Item | undefined; log: Item } | null {
  const logs = logsOf(itemId);
  const log = logs[logs.length - 1];
  return log ? { holder: db.get('character', log.character), place: db.get('location', log.location), log } : null;
}

/** Items a character holds now. */
export function itemsHeldBy(characterId: number): Item[] {
  return db.lists.item.filter((it) => nowOf(it.id)?.holder?.id === characterId);
}

/** Factions a character belongs to, with their rank. */
export function membershipsOf(characterId: number): { faction: Item; rank: string }[] {
  return db.lists.membership
    .filter((m) => m.character === characterId)
    .flatMap((m) => { const f = db.get('faction', m.faction); return f ? [{ faction: f, rank: m.rank ?? '' }] : []; });
}
