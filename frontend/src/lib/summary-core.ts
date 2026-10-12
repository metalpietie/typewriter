// How long the outline and the synopsis will be, from the summaries you wrote. No imports, so it can be tested alone.

export type Summed = { summary?: string | null };
export type Chapter = Summed & { scenes: Summed[] };

/** The words in a plain text. */
export const countWords = (text: string | null | undefined): number => (String(text ?? '').match(/\S+/g) ?? []).length;

const clean = (s: string | null | undefined): string => String(s ?? '').trim();

/** The words of a synopsis: for each chapter, its own summary, or else the summaries of its scenes. */
export function synopsisWords(chapters: Chapter[]): number {
  let n = 0;
  for (const c of chapters) {
    const own = clean(c.summary);
    n += own ? countWords(own) : c.scenes.reduce((sum, s) => sum + countWords(clean(s.summary)), 0);
  }
  return n;
}

/** The words of the summaries in an outline: every chapter summary and every scene summary. */
export function outlineWords(chapters: Chapter[]): number {
  return chapters.reduce((sum, c) => sum + countWords(clean(c.summary)) + c.scenes.reduce((n, s) => n + countWords(clean(s.summary)), 0), 0);
}

/** The scenes that have no summary yet. */
export const missingSummaries = (chapters: Chapter[]): number => chapters.reduce((sum, c) => sum + c.scenes.filter((s) => !clean(s.summary)).length, 0);

/** About how many pages some words fill, at 500 words a page. At least 1 when there are words. */
export const pagesOf = (words: number): number => (words <= 0 ? 0 : Math.max(1, Math.round(words / 500)));
