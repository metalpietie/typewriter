import { Store } from '../../bindings/typewriter/internal/app';
import { sortCards, nextOrder, cleanText } from './mood-core';

// The mood board cards of one character, place or scene. They are read when needed, and not kept in the big lists,
// because a picture can be large. The rules are in mood-core.ts.

export type Mood = { id: number; owner: number; text: string; image: string; order: number; at: string };

export async function listMood(owner: number): Promise<Mood[]> {
  const docs = (await Store.ListBy('mood', 'owner', owner)) ?? [];
  return sortCards(docs.map((d) => ({ image: '', text: '', order: 0, at: '', ...JSON.parse(d.data), id: d.id }) as Mood));
}

/** Add a card to the end. A card holds a picture, a note, or both. */
export async function addMood(owner: number, existing: Mood[], fields: { image?: string; text?: string }): Promise<Mood> {
  const data = { owner, image: fields.image ?? '', text: cleanText(fields.text), order: nextOrder(existing), at: new Date().toISOString() };
  const saved = await Store.Save({ id: 0, kind: 'mood', data: JSON.stringify(data), updatedAt: '' });
  return { ...data, id: saved.id };
}

export async function saveMood(m: Mood) {
  const { id, ...data } = m;
  await Store.Save({ id, kind: 'mood', data: JSON.stringify({ ...data, text: cleanText(data.text) }), updatedAt: '' });
}

export async function deleteMood(id: number) {
  await Store.Delete('mood', id);
}
