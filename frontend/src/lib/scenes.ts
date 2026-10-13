import { db, wordCount, type Item } from './db.svelte';
import { word } from './projecttype.svelte';

export const STATUSES = ['idea', 'draft', 'revised', 'final'];

/** The scenes of one chapter, in order. */
export function scenesOf(chapterId: number | null | undefined): Item[] {
  return db.lists.scene.filter((s) => s.chapter === chapterId).sort((a, b) => a.order - b.order);
}

export const sceneWords = (s: Item) => wordCount(s.body ?? '');

/** Words in a chapter: its scenes added up. A chapter without scenes uses its own text. */
export function chapterWords(c: Item): number {
  const sc = scenesOf(c.id);
  return sc.length ? sc.reduce((n, s) => n + sceneWords(s), 0) : wordCount(c.body ?? '');
}

/** Everyone who is in a chapter, as the point of view or in the cast of any scene. */
export function chapterCast(c: Item): number[] {
  const sc = scenesOf(c.id);
  const sources = sc.length ? sc : [c];
  const ids = new Set<number>();
  for (const s of sources) {
    if (s.pov) ids.add(s.pov);
    for (const id of s.characters ?? []) ids.add(id);
  }
  return [...ids].filter((id) => db.get('character', id));
}

export function chapterHasPov(c: Item, characterId: number): boolean {
  const sc = scenesOf(c.id);
  return (sc.length ? sc : [c]).some((s) => s.pov === characterId);
}

/** "Chapter 7, scene 2" style name, for lists. */
export function sceneLabel(s: Item): string {
  const c = db.get('chapter', s.chapter);
  const n = scenesOf(s.chapter).findIndex((x) => x.id === s.id) + 1;
  return `${c ? `ch. ${c.order}` : 'ch. ?'}, ${word('scene')} ${n}${s.title ? ': ' + s.title : ''}`;
}

export async function addScene(chapter: Item, fields: Record<string, unknown> = {}): Promise<Item> {
  const n = scenesOf(chapter.id).length + 1;
  return db.save('scene', {
    chapter: chapter.id, book: chapter.book, order: n, title: `${word('Scene')} ${n}`, body: '', summary: '',
    status: 'draft', tags: [], pov: null, location: null, characters: [], when: '', time: null, goal: null,
    ...fields,
  });
}

/**
 * Older chapters held their own text and cast. Move that into a first scene, once.
 * The text is copied into the scene before the chapter is cleared.
 */
export async function migrateChapters() {
  for (const c of [...db.lists.chapter]) {
    if (scenesOf(c.id).length) continue;
    const hasText = !!(c.body ?? '').trim();
    const hasInfo = !!(c.pov || c.location || (c.characters ?? []).length || c.when || c.time != null);
    if (!hasText && !hasInfo) continue;
    await addScene(c, {
      title: `${word('Scene')} 1`, body: c.body ?? '', status: c.status ?? 'draft', pov: c.pov ?? null,
      location: c.location ?? null, characters: [...(c.characters ?? [])], when: c.when ?? '', time: c.time ?? null,
    });
    await db.save('chapter', { ...c, body: '', pov: null, location: null, characters: [], when: '', time: null });
  }
}

/** Every scene of every book, in reading order. */
export function orderedScenes(): Item[] {
  const out: Item[] = [];
  for (const b of [...db.lists.book].sort((x, y) => x.order - y.order)) {
    for (const c of db.lists.chapter.filter((x) => x.book === b.id).sort((x, y) => x.order - y.order)) out.push(...scenesOf(c.id));
  }
  return out;
}
