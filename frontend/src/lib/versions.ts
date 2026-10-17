import { Store } from '../../bindings/typewriter/internal/app';
import { wordCount } from './db.svelte';

/** One saved copy of a scene's text. */
export type Version = { id: number; scene: number; body: string; words: number; label: string; auto: boolean; at: string };

const AUTO_EVERY = 10 * 60 * 1000; // an automatic copy at most this often per scene
const AUTO_KEEP = 40; // automatic copies kept per scene; your own named copies stay

/** Every saved copy of one scene, newest first. */
export async function listVersions(sceneId: number): Promise<Version[]> {
  const docs = (await Store.ListBy('version', 'scene', sceneId)) ?? [];
  return docs.map((d) => ({ ...JSON.parse(d.data), id: d.id }));
}

export async function addVersion(sceneId: number, body: string, label = '', auto = false): Promise<Version> {
  const data = { scene: sceneId, body, words: wordCount(body), label, auto, at: new Date().toISOString() };
  const saved = await Store.Save({ id: 0, kind: 'version', data: JSON.stringify(data), updatedAt: '' });
  return { ...data, id: saved.id };
}

export async function deleteVersion(id: number) {
  await Store.Delete('version', id);
}

/** Keep the last AUTO_KEEP automatic copies of a scene. */
async function prune(sceneId: number) {
  const autos = (await listVersions(sceneId)).filter((v) => v.auto);
  for (const v of autos.slice(AUTO_KEEP)) await deleteVersion(v.id);
}

const lastAuto = new Map<number, number>(); // scene id → time of its last automatic copy

/**
 * Call this with the text as it was just before an edit.
 * It keeps a copy of that text, but only when the last copy is old enough.
 */
export async function maybeAutoVersion(sceneId: number, before: string) {
  if (!before.trim()) return;
  const now = Date.now();
  if (now - (lastAuto.get(sceneId) ?? 0) < AUTO_EVERY) return;
  lastAuto.set(sceneId, now); // set first, so quick typing does not start two copies
  try {
    const [newest] = await listVersions(sceneId);
    if (newest && newest.body === before) return;
    if (newest && now - Date.parse(newest.at) < AUTO_EVERY) { lastAuto.set(sceneId, Date.parse(newest.at)); return; }
    await addVersion(sceneId, before, '', true);
    await prune(sceneId);
  } catch { /* history is a bonus; never break writing */ }
}
