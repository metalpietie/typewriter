import { db, type Item } from './db.svelte';
import { chapterWords } from './scenes';
import { lastDays } from './progress';
import { planFor, type Plan } from './finish-core';

export * from './finish-core';

/** Words written in one book: its chapters added up. */
export function bookWords(bookId: number): number {
  return db.lists.chapter.filter((c) => c.book === bookId).reduce((n, c) => n + chapterWords(c), 0);
}

/**
 * Words you write on a normal day: the average of the 7 days before today.
 * Today is left out, because it is not over yet. This counts all your writing, not one book.
 */
export function normalPace(): number {
  const days = lastDays(8).slice(0, 7);
  return Math.round(days.reduce((n, d) => n + d.words, 0) / 7);
}

export function planOf(book: Item): Plan {
  return planFor({ total: bookWords(book.id), goal: Number(book.goal) || 0, finishBy: book.finishBy, now: new Date(), pace: normalPace() });
}

/** "31 December 2026", in the language of the computer. */
export const dayText = (d: Date): string => d.toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' });
