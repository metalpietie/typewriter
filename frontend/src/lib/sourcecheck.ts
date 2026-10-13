import { db } from './db.svelte';
import { orderedScenes, sceneLabel } from './scenes';
import { deadCites, neverCited, unsourced } from './sourcecheck-core';

/** The problems with sources in the whole book: claims that need a source, sources never cited, citations of a source that is gone. */
export function sourceProblems() {
  const pieces = orderedScenes().map((s) => ({ id: s.id, label: sceneLabel(s), body: String(s.body ?? '') }));
  const ids = db.lists.source.map((s) => s.id);
  return { claims: unsourced(pieces), unused: neverCited(ids, pieces), dead: deadCites(ids, pieces) };
}
