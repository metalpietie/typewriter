import { db, type Item } from './db.svelte';
import { sorted, cleanName, toggled, moved, USUAL_PASSES } from './passes-core';

// Revision passes: the helpers that read and change the data. The rules are in passes-core.ts.

/** The passes of one book, in the order you set. */
export const passesOf = (bookId: number | null | undefined): Item[] => sorted(db.lists.pass.filter((p) => p.book === bookId));

/** Every scene of one book, in reading order. */
export function scenesInBook(bookId: number | null | undefined): Item[] {
  const chapters = db.lists.chapter.filter((c) => c.book === bookId).sort((a, b) => a.order - b.order);
  return chapters.flatMap((c) => db.lists.scene.filter((s) => s.chapter === c.id).sort((a, b) => a.order - b.order));
}

/** Add a pass at the end. Nothing happens for an empty name, or a name this book has. */
export async function addPass(bookId: number, raw: string): Promise<boolean> {
  const name = cleanName(raw);
  const have = passesOf(bookId);
  if (!name || have.some((p) => String(p.name).toLowerCase() === name.toLowerCase())) return false;
  await db.save('pass', { book: bookId, name, order: have.length ? Math.max(...have.map((p) => p.order)) + 1 : 1 });
  return true;
}

export async function addUsualPasses(bookId: number) {
  for (const name of USUAL_PASSES) await addPass(bookId, name);
}

export function renamePass(p: Item, raw: string): string {
  const name = cleanName(raw);
  if (!name || name === p.name) return p.name;
  p.name = name;
  db.queue('pass', p);
  return name;
}

/** Move a pass up or down, and write the new order. */
export async function movePass(bookId: number, id: number, dir: -1 | 1) {
  const list = moved(passesOf(bookId), id, dir);
  for (const [i, p] of list.entries()) if (p.order !== i + 1) { p.order = i + 1; await db.save('pass', p); }
}

/** Delete a pass. Its ticks come off the scenes first, because a new pass could get the same number. */
export async function removePass(p: Item) {
  for (const s of db.lists.scene) {
    if ((s.passes ?? []).includes(p.id)) { s.passes = s.passes.filter((id: number) => id !== p.id); await db.save('scene', s); }
  }
  await db.remove('pass', p.id, false);
}

/** Tick or untick a pass for one scene. */
export function tick(scene: Item, passId: number) {
  scene.passes = toggled(scene.passes, passId);
  db.queue('scene', scene);
}
