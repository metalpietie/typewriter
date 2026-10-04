import { db, plainText, type Item } from './db.svelte';
import { scenesOf, sceneLabel } from './scenes';
import { findTypos, type Typo } from './namespell-core';
import { replaceIn } from './findreplace';
import { addVersion } from './versions';

export type Place = { scene: Item; label: string; snippet: string; times: number };
export type NameIssue = Typo & {
  key: string; // stays the same while the problem stays the same (used for "skip")
  places: Place[];
  reason: string;
};

const wordRe = (w: string) => new RegExp(`(?<![\\p{L}\\p{N}_])${w.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(?![\\p{L}\\p{N}_])`, 'gu');

/** Every name you have given a character, place, faction or item. */
function worldNames(): string[] {
  const out: string[] = [];
  for (const c of db.lists.character) {
    out.push(String(c.name ?? ''));
    for (const a of String(c.aliases ?? '').split(/[,;\n]/)) out.push(a);
  }
  for (const k of ['location', 'faction', 'item'] as const) for (const i of db.lists[k]) out.push(String(i.name ?? ''));
  return out.filter((n) => n.trim());
}

/** Scenes in reading order. */
function allScenes(): Item[] {
  const out: Item[] = [];
  for (const b of [...db.lists.book].sort((a, b2) => a.order - b2.order)) {
    for (const c of db.lists.chapter.filter((c) => c.book === b.id).sort((a, b2) => a.order - b2.order)) out.push(...scenesOf(c.id));
  }
  return out;
}

export function findNameIssues(): NameIssue[] {
  const scenes = allScenes();
  const texts = scenes.map((s) => plainText(s.body ?? ''));
  const typos = findTypos(texts, worldNames());
  return typos.map((t) => {
    const re = wordRe(t.word);
    const places: Place[] = [];
    scenes.forEach((s, i) => {
      const text = texts[i];
      const hits = [...text.matchAll(re)];
      if (!hits.length) return;
      const at = hits[0].index!;
      places.push({
        scene: s, label: sceneLabel(s), times: hits.length,
        snippet: (at > 36 ? '…' : '') + text.slice(Math.max(0, at - 36), at) + '⟦' + t.word + '⟧' + text.slice(at + t.word.length, at + t.word.length + 36),
      });
    });
    return {
      ...t,
      key: `n|${t.word}|${t.suggest}`,
      places,
      reason: t.from === 'name'
        ? `looks like a name in your world data: “${t.suggest}”`
        : `looks like a typo of “${t.suggest}”, which you write more often`,
    };
  });
}

/** Change the word to the suggested spelling in every scene. The old text goes to History. */
export async function fixEverywhere(issue: NameIssue): Promise<number> {
  db.flush(); // save what is being typed first
  const re = wordRe(issue.word);
  let count = 0;
  for (const p of issue.places) {
    const scene = db.get('scene', p.scene.id);
    if (!scene) continue;
    const out = replaceIn(scene.body ?? '', re, issue.suggest);
    if (!out.count) continue;
    await addVersion(scene.id, scene.body ?? '', `Before changing “${issue.word}” to “${issue.suggest}”`);
    await db.save('scene', { ...scene, body: out.html });
    count += out.count;
  }
  return count;
}
