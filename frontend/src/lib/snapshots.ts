import { Store } from '../../bindings/typewriter';
import { db, wordCount } from './db.svelte';
import { scenesOf } from './scenes';
import { blockText } from './diff';
import type { Chap, Item } from './snapshot-core';

/** A scene as it was when the snapshot was taken. */
export type SnapScene = { id: number; chapter: number; title: string; order: number; status: string; body: string };
/** A copy of a whole book at one moment. */
export type Snapshot = { id: number; book: number; name: string; at: string; words: number; chapters: Chap[]; scenes: SnapScene[] };

/** The snapshots of one book, newest first. They are not kept in memory: they are read when you open the list. */
export async function listSnapshots(bookId: number): Promise<Snapshot[]> {
  const docs = (await Store.ListBy('snapshot', 'book', bookId)) ?? [];
  return docs.map((d) => ({ ...JSON.parse(d.data), id: d.id } as Snapshot)).sort((a, b) => b.at.localeCompare(a.at) || b.id - a.id);
}

const chaptersOf = (bookId: number) => db.lists.chapter.filter((c) => c.book === bookId).sort((a, b) => a.order - b.order);

export const defaultName = (): string => 'Snapshot ' + new Date().toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' });

/** Save a copy of the book as it is now. */
export async function takeSnapshot(bookId: number, name: string): Promise<Snapshot> {
  const chapters = chaptersOf(bookId);
  const scenes = chapters.flatMap((c) => scenesOf(c.id)).map((s) => ({
    id: s.id, chapter: s.chapter, title: s.title ?? '', order: s.order, status: s.status ?? '', body: s.body ?? '',
  }));
  const data = {
    book: bookId,
    name: name.trim() || defaultName(),
    at: new Date().toISOString(),
    words: scenes.reduce((n, s) => n + wordCount(s.body), 0),
    chapters: chapters.map((c) => ({ id: c.id, title: c.title ?? '', order: c.order })),
    scenes,
  };
  const saved = await Store.Save({ id: 0, kind: 'snapshot', data: JSON.stringify(data), updatedAt: '' });
  return { ...data, id: saved.id };
}

/** Give a snapshot another name. The copy of the book inside it stays as it was. */
export async function renameSnapshot(snap: Snapshot, name: string): Promise<string> {
  const clean = name.trim().slice(0, 80);
  if (!clean || clean === snap.name) return snap.name;
  const { id, ...data } = snap;
  await Store.Save({ id, kind: 'snapshot', data: JSON.stringify({ ...data, name: clean }), updatedAt: '' });
  return clean;
}

export async function deleteSnapshot(id: number) {
  await Store.Delete('snapshot', id);
}

/** The book as it is now, in the form the comparison needs. */
export function currentBook(bookId: number): { chapters: Chap[]; scenes: Item[] } {
  const chapters = chaptersOf(bookId);
  return {
    chapters: chapters.map((c) => ({ id: c.id, title: c.title ?? '', order: c.order })),
    scenes: chapters.flatMap((c) => scenesOf(c.id)).map((s) => ({ id: s.id, chapter: s.chapter, title: s.title ?? '', order: s.order, text: blockText(s.body ?? '') })),
  };
}

/** The scenes of a snapshot in the form the comparison needs. */
export const snapshotItems = (snap: Snapshot): Item[] => snap.scenes.map((s) => ({ id: s.id, chapter: s.chapter, title: s.title, order: s.order, text: blockText(s.body) }));
