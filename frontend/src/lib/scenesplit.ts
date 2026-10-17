import { Store } from '../../bindings/typewriter/internal/app';
import { db, toHtml, type Item } from './db.svelte';
import { scenesOf } from './scenes';
import { addVersion } from './versions';
import { trimEmptyParas, hasText, commentIds, joinBodies, union, both, joinText } from './split-core';

// Split one scene in two at the cursor, or merge a scene with the one after it.
// The rules are in split-core.ts. This file reads and writes the data.

/** Split a scene. before and after are the HTML on each side of the cursor. Returns the new scene, or an error text. */
export async function splitScene(scene: Item, beforeHtml: string, afterHtml: string): Promise<Item | string> {
  const before = trimEmptyParas(beforeHtml), after = trimEmptyParas(afterHtml);
  if (!hasText(before) || !hasText(after)) return 'Put the cursor inside the text, with words before it and after it.';

  // keep the whole text as a copy first, so the split can be undone from History
  await addVersion(scene.id, scene.body ?? '', 'Before splitting');

  // the scenes after this one move down by one place
  for (const s of scenesOf(scene.chapter)) {
    if (s.order > scene.order) { s.order += 1; await db.save('scene', s); }
  }
  const next = await db.save('scene', {
    chapter: scene.chapter, book: scene.book, order: scene.order + 1,
    title: scene.title ? `${scene.title} (2)` : '', body: after, summary: '',
    status: scene.status ?? 'draft', tags: [...(scene.tags ?? [])], passes: [...(scene.passes ?? [])],
    pov: scene.pov ?? null, location: scene.location ?? null, characters: [...(scene.characters ?? [])],
    ignoreMentions: [...(scene.ignoreMentions ?? [])], when: scene.when ?? '', time: scene.time ?? null, goal: null,
  });
  scene.body = before;
  await db.save('scene', scene);

  // a comment on text that moved goes with the text
  const moved = new Set(commentIds(after));
  for (const c of db.lists.comment) {
    if (c.scene === scene.id && moved.has(c.id)) { c.scene = next.id; await db.save('comment', c); }
  }
  return next;
}

/** The scene after this one in the same chapter, if there is one. */
export const sceneAfter = (scene: Item | undefined): Item | undefined => {
  if (!scene) return undefined;
  const list = scenesOf(scene.chapter);
  return list[list.findIndex((s) => s.id === scene.id) + 1];
};

/** Merge a scene with the one after it. The text of the second goes after the text of the first. Returns an error text, or null. */
export async function mergeWithNext(scene: Item): Promise<string | null> {
  const other = sceneAfter(scene);
  if (!other) return 'There is no scene after this one in the chapter.';

  await addVersion(scene.id, scene.body ?? '', 'Before merging');

  scene.body = joinBodies(scene.body ?? '', other.body ?? '', toHtml);
  scene.characters = union(scene.characters, other.characters);
  scene.tags = union(scene.tags, other.tags);
  scene.ignoreMentions = union(scene.ignoreMentions, other.ignoreMentions);
  scene.passes = both(scene.passes, other.passes);
  scene.pov = scene.pov ?? other.pov ?? null;
  scene.location = scene.location ?? other.location ?? null;
  if (!scene.when && other.when) scene.when = other.when;
  if (scene.time == null && other.time != null) scene.time = other.time;
  scene.summary = joinText(scene.summary, other.summary);
  await db.save('scene', scene);

  // what pointed at the second scene now points at the first
  for (const c of db.lists.comment) if (c.scene === other.id) { c.scene = scene.id; await db.save('comment', c); }
  for (const l of db.lists.itemlog) if (l.scene === other.id) { l.scene = scene.id; await db.save('itemlog', l); }
  for (const l of db.lists.arcscene) {
    if (l.scene !== other.id) continue;
    if (db.lists.arcscene.some((x) => x.scene === scene.id && x.arc === l.arc)) continue; // the first scene is in this arc already
    l.scene = scene.id; await db.save('arcscene', l);
  }
  // the History of the second scene is kept, as part of the History of the merged scene
  const docs = (await Store.ListBy('version', 'scene', other.id)) ?? [];
  for (const d of docs) await Store.Save({ ...d, data: JSON.stringify({ ...JSON.parse(d.data), scene: scene.id }) });

  await db.remove('scene', other.id, false);
  // order numbers 1, 2, 3 … again
  const rest = scenesOf(scene.chapter);
  for (const [i, s] of rest.entries()) if (s.order !== i + 1) { s.order = i + 1; await db.save('scene', s); }
  return null;
}
