// Revision passes. No imports, so it can be tested alone.
// A pass is a round of revision for one book: plot, dialogue, line edit. A scene is done in a pass when you tick it.
// The scene keeps the ids of the passes it is done in, in `passes`.

export type PassLike = { id: number; name?: string; order?: number };
export type SceneLike = { id?: number; passes?: number[] | null };

/** The passes people use most. */
export const USUAL_PASSES = ['Plot', 'Characters', 'Dialogue', 'Line edit', 'Proofread'];

/** Passes in the order you set. */
export const sorted = <T extends PassLike>(list: T[]): T[] => [...list].sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.id - b.id);

/** The passes this scene is done in. An id of a pass that is gone is left out. */
export function doneIn(scene: SceneLike, passes: PassLike[]): number[] {
  const have = new Set(passes.map((p) => p.id));
  return [...new Set(scene.passes ?? [])].filter((id) => have.has(id));
}

export const isDone = (scene: SceneLike, passId: number): boolean => (scene.passes ?? []).includes(passId);

/** The list with the pass ticked, or unticked when it was ticked. */
export function toggled(list: number[] | null | undefined, passId: number): number[] {
  const cur = list ?? [];
  return cur.includes(passId) ? cur.filter((id) => id !== passId) : [...cur, passId];
}

/** How many of these scenes are done in one pass. */
export function progress(scenes: SceneLike[], passId: number): { done: number; total: number } {
  return { done: scenes.filter((s) => isDone(s, passId)).length, total: scenes.length };
}

/** How many of the book's passes this scene is done in. */
export function summary(scene: SceneLike, passes: PassLike[]): { done: number; total: number } {
  return { done: doneIn(scene, passes).length, total: passes.length };
}

/** The scenes that still need a pass. */
export const todo = <T extends SceneLike>(scenes: T[], passId: number): T[] => scenes.filter((s) => !isDone(s, passId));

/** The order numbers after moving one pass up (-1) or down (1). The list is returned in the new order. */
export function moved<T extends PassLike>(list: T[], id: number, dir: -1 | 1): T[] {
  const a = sorted(list);
  const i = a.findIndex((p) => p.id === id);
  const j = i + dir;
  if (i < 0 || j < 0 || j >= a.length) return a;
  [a[i], a[j]] = [a[j], a[i]];
  return a;
}

/** A clean name: one line, no extra spaces, at most 40 letters. */
export const cleanName = (raw: string): string => String(raw ?? '').replace(/\s+/g, ' ').trim().slice(0, 40).trim();
