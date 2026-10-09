// Mood board cards: pictures and short notes that belong to a character, a place or a scene.
// No imports, so it can be tested alone.

export type Card = { id: number; order?: number; text?: string; image?: string };

export const MAX_TEXT = 600; // letters in the note of one card
export const MAX_CARDS = 60; // cards for one owner
export const MAX_FILE = 25 * 1024 * 1024; // bytes of one picture file, before it is made small

/** The cards in the order you added them. */
export const sortCards = <T extends Card>(list: T[]): T[] => [...list].sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.id - b.id);

/** The order number for a new card: after the last one. */
export const nextOrder = (list: Card[]): number => list.reduce((m, c) => Math.max(m, c.order ?? 0), 0) + 1;

/** A note, cut to a safe length. */
export const cleanText = (raw: unknown): string => String(raw ?? '').replace(/\r\n/g, '\n').slice(0, MAX_TEXT);

/** Whether a dropped or chosen file can go on a mood board. Gives a reason when it cannot. */
export function whyNot(file: { type: string; size: number; name?: string }): string {
  if (!String(file.type ?? '').startsWith('image/')) return `${file.name ?? 'That file'} is not a picture.`;
  if (file.size > MAX_FILE) return `${file.name ?? 'That picture'} is too large (more than 25 MB).`;
  return '';
}
