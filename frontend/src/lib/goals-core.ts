// Word goals at three levels: the book, a chapter and a scene. No imports, so it can be tested alone.

export type Scope = 'book' | 'chapter' | 'scene';
export type Level = { scope: Scope; words: number; goal: number };
export type Bar = { scopes: Scope[]; words: number; goal: number };

/**
 * The progress bars to show. Give the levels from the widest to the narrowest (book, chapter, scene).
 * A level without a goal has no bar. Levels with the same goal share one bar.
 * A shared bar counts the words of its widest level.
 */
export function mergeBars(levels: Level[]): Bar[] {
  const bars: Bar[] = [];
  for (const l of levels) {
    if (!(l.goal > 0)) continue;
    const same = bars.find((b) => b.goal === l.goal);
    if (same) same.scopes.push(l.scope);
    else bars.push({ scopes: [l.scope], words: l.words, goal: l.goal });
  }
  return bars;
}

/** What the single bar of a book on the shelf counts: the book goal, else the chapter goals together, else the scene goals together. */
export type ShelfGoal = { kind: 'book' | 'chapter' | 'scene' | ''; goal: number; words: number };

export type Part = { goal: number; words: number };

export function shelfGoal(bookGoal: number, chapters: Part[], scenes: Part[], bookWords: number): ShelfGoal {
  if (bookGoal > 0) return { kind: 'book', goal: bookGoal, words: bookWords };
  const sum = (list: Part[]): Part => list.filter((p) => p.goal > 0).reduce((a, p) => ({ goal: a.goal + p.goal, words: a.words + p.words }), { goal: 0, words: 0 });
  const c = sum(chapters);
  if (c.goal > 0) return { kind: 'chapter', ...c };
  const s = sum(scenes);
  if (s.goal > 0) return { kind: 'scene', ...s };
  return { kind: '', goal: 0, words: 0 };
}
