import { Store } from '../../bindings/typewriter/internal/app';
import { drawCover } from './cover';
import { shelfSpec } from './shelf-core';

// The pictures on the bookshelf. They are small, and kept until the app closes, so the shelf opens fast.

const made = new Map<string, string>();

/** A cover made from the title: the same title and seed always give the same picture. */
export function generatedCover(title: string, seed: number): string {
  const key = `${seed}#${title}`;
  const hit = made.get(key);
  if (hit) return hit;
  const c = document.createElement('canvas');
  drawCover(c, shelfSpec(title, seed), null, 0.2); // 320 by 480
  const url = c.toDataURL('image/jpeg', 0.88);
  if (made.size > 300) made.clear();
  made.set(key, url);
  return url;
}

const own = new Map<string, Promise<string>>();

/** The cover that the writer made or chose for a book. Empty when there is none. */
export function savedCover(project: string, book: number): Promise<string> {
  const key = project + '/' + book;
  let p = own.get(key);
  if (!p) { p = Store.BookCover(project, book).catch(() => ''); own.set(key, p); }
  return p;
}

/** Forget the saved covers, for example after a book moved. */
export const forgetCovers = () => own.clear();
