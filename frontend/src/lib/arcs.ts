import { db, CHARACTER_COLORS, type Item } from './db.svelte';
import { orderedScenes, scenesOf, sceneLabel } from './scenes';

// A story arc is one plot thread. An "arcscene" item says that one scene belongs to one arc,
// and (optionally) what the scene does for it: start it, turn it, or end it.

export type Beat = '' | 'start' | 'turn' | 'end';
export const BEATS: [Beat, string][] = [['', 'touches'], ['start', 'starts'], ['turn', 'turns'], ['end', 'ends']];
export const BEAT_LETTER: Record<Beat, string> = { '': '', start: 'S', turn: 'T', end: 'E' };

/** This many chapters in a row with no scene of the arc is called a gap. */
export const GAP_CHAPTERS = 3;

export const arcsSorted = (): Item[] => [...db.lists.arc].sort((a, b) => (a.order ?? a.id) - (b.order ?? b.id));

export const linkFor = (arcId: number, sceneId: number) => db.lists.arcscene.find((l) => l.arc === arcId && l.scene === sceneId);
export const arcsOfScene = (sceneId: number): Item[] =>
  arcsSorted().filter((a) => linkFor(a.id, sceneId));

/** The links of one arc, with their scenes, in reading order. Links to missing scenes are left out. */
export function arcLinks(arcId: number): { link: Item; scene: Item }[] {
  const rank = new Map(orderedScenes().map((s, i) => [s.id, i]));
  return db.lists.arcscene
    .filter((l) => l.arc === arcId && rank.has(l.scene))
    .map((l) => ({ link: l, scene: db.get('scene', l.scene)! }))
    .sort((a, b) => rank.get(a.scene.id)! - rank.get(b.scene.id)!);
}

export async function addArc(): Promise<Item> {
  const list = arcsSorted();
  const last = list[list.length - 1];
  return db.save('arc', {
    name: 'New arc', notes: '', color: CHARACTER_COLORS[list.length % CHARACTER_COLORS.length],
    order: (last?.order ?? last?.id ?? 0) + 1,
  });
}

export async function tagScene(arcId: number, sceneId: number, beat: Beat = '') {
  if (linkFor(arcId, sceneId)) return;
  await db.save('arcscene', { arc: arcId, scene: sceneId, beat });
}
export async function untagScene(arcId: number, sceneId: number) {
  const l = linkFor(arcId, sceneId);
  if (l) await db.remove('arcscene', l.id, false);
}
export function setBeat(link: Item, beat: Beat) {
  link.beat = beat;
  db.queue('arcscene', link);
}

export type ArcWarning = { text: string; scene?: number };

/** What looks wrong with one arc. These are hints, not errors. */
export function arcWarnings(arcId: number): ArcWarning[] {
  const links = arcLinks(arcId);
  if (!links.length) return [{ text: 'No scene is tagged with this arc yet.' }];
  const out: ArcWarning[] = [];
  const at = (b: Beat) => links.filter((x) => (x.link.beat ?? '') === b);
  const starts = at('start'), ends = at('end');

  if (!starts.length) out.push({ text: 'No scene is marked as the start.' });
  else if (starts.length > 1) out.push({ text: `${starts.length} scenes are marked as the start.`, scene: starts[1].scene.id });
  if (!ends.length) out.push({ text: 'No scene is marked as the end. The thread is left open.' });
  else if (ends.length > 1) out.push({ text: `${ends.length} scenes are marked as the end.`, scene: ends[1].scene.id });

  const first = links[0], last = links[links.length - 1];
  if (starts.length && starts[0] !== first) out.push({ text: `The start is not the first scene. ${sceneLabel(first.scene)} comes before it.`, scene: first.scene.id });
  if (ends.length && ends[ends.length - 1] !== last) out.push({ text: `Scenes come after the end. ${sceneLabel(last.scene)} is later.`, scene: last.scene.id });

  // chapters with no scene of the arc, between its first and last chapter
  for (const b of db.lists.book) {
    const chapters = db.lists.chapter.filter((c) => c.book === b.id).sort((x, y) => x.order - y.order);
    const used = chapters.map((c) => scenesOf(c.id).some((s) => linkFor(arcId, s.id)));
    const from = used.indexOf(true), to = used.lastIndexOf(true);
    if (from < 0) continue;
    let run = 0;
    for (let i = from; i <= to + 1; i++) {
      if (i <= to && !used[i]) { run++; continue; }
      if (run >= GAP_CHAPTERS) {
        const a = chapters[i - run], z = chapters[i - 1];
        const where = db.lists.book.length > 1 ? ` in ${b.title || 'the book'}` : '';
        out.push({ text: `Nothing for ${run} chapters in a row: ch. ${a.order} to ${z.order}${where}.` });
      }
      run = 0;
    }
  }
  return out;
}
