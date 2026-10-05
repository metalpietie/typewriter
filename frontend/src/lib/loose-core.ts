// Characters who are loose ends: in no scene, in one scene only, or gone quiet. No imports, so it can be tested alone.

export type Char = { id: number; name: string; died?: boolean; ignore?: boolean };
/** One scene. chapter is the place of its chapter in reading order (0, 1, 2 ...). ids: everyone who appears in it. */
export type ScenePresence = { chapter: number; ids: number[] };

export type Finding = {
  id: number;
  scenes: number;          // in how many scenes
  chapters: number[];      // the chapters they are in, in order, without repeats
  gapFrom?: number;        // the chapter before the quiet stretch
  gapTo?: number;          // the chapter after it. Not set when the stretch runs to the end of the book
  gap?: number;            // how many chapters are in the quiet stretch
};

export type Result = {
  none: Finding[];   // in no scene
  once: Finding[];   // in exactly one scene
  quiet: Finding[];  // in more scenes, but gone for a long stretch of chapters
  ignored: Finding[];// you said it is fine
};

/**
 * Sort the characters. total: the number of chapters in the book (or the books) you look at.
 * threshold: how many chapters in a row without them is "quiet".
 * A character who died is not quiet at the end of the book, because they cannot come back.
 */
export function analyze(chars: Char[], scenes: ScenePresence[], total: number, threshold: number): Result {
  const inScenes = new Map<number, number>();
  const inChapters = new Map<number, Set<number>>();
  for (const s of scenes) {
    for (const id of new Set(s.ids)) {
      inScenes.set(id, (inScenes.get(id) ?? 0) + 1);
      if (!inChapters.has(id)) inChapters.set(id, new Set());
      inChapters.get(id)!.add(s.chapter);
    }
  }
  const out: Result = { none: [], once: [], quiet: [], ignored: [] };
  const need = Math.max(1, Math.floor(threshold) || 1);

  for (const c of chars) {
    const n = inScenes.get(c.id) ?? 0;
    const chapters = [...(inChapters.get(c.id) ?? [])].sort((a, b) => a - b);
    const f: Finding = { id: c.id, scenes: n, chapters };
    let bucket: keyof Result | null = null;

    if (n === 0) bucket = 'none';
    else if (n === 1) bucket = 'once';
    else {
      // the longest stretch between two appearances, and the stretch after the last one
      let best = 0, from = 0, to = 0;
      for (let i = 1; i < chapters.length; i++) {
        const gap = chapters[i] - chapters[i - 1] - 1;
        if (gap > best) { best = gap; from = chapters[i - 1]; to = chapters[i]; }
      }
      const last = chapters[chapters.length - 1];
      const trailing = c.died ? 0 : Math.max(0, total - 1 - last);
      if (trailing > best && trailing >= need) { f.gap = trailing; f.gapFrom = last; bucket = 'quiet'; }
      else if (best >= need) { f.gap = best; f.gapFrom = from; f.gapTo = to; bucket = 'quiet'; }
    }
    if (!bucket) continue;
    (c.ignore ? out.ignored : out[bucket]).push(f);
  }
  out.quiet.sort((a, b) => (b.gap ?? 0) - (a.gap ?? 0));
  out.once.sort((a, b) => (a.chapters[0] ?? 0) - (b.chapters[0] ?? 0));
  return out;
}
