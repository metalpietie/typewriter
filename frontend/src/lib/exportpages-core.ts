// The pages and style choices of a manuscript export. No imports, so it can be tested alone.

export type ExportPages = {
  copyright: string; dedication: string; thanks: string; about: string;
  sceneBreak: 'stars' | 'hash' | 'dashes' | 'blank';
  chapterStyle: 'full' | 'title' | 'number' | 'plain';
};

export const SCENE_BREAKS: [ExportPages['sceneBreak'], string][] = [
  ['stars', '* * *  (three stars)'], ['hash', '#  (one hash mark)'], ['dashes', '— — —  (three dashes)'], ['blank', 'An empty line'],
];
export const CHAPTER_STYLES: [ExportPages['chapterStyle'], string][] = [
  ['full', 'Chapter 3: The title'], ['title', 'The title'], ['number', 'Chapter 3'], ['plain', '3. The title'],
];

export const DEFAULT_PAGES: ExportPages = { copyright: '', dedication: '', thanks: '', about: '', sceneBreak: 'stars', chapterStyle: 'full' };

const MAX_TEXT = 5000; // letters for each page

const text = (v: unknown): string => (typeof v === 'string' ? v.replace(/\r\n/g, '\n').slice(0, MAX_TEXT) : '');
const pick = <T extends string>(v: unknown, allowed: readonly [T, string][], fallback: T): T =>
  allowed.some(([k]) => k === v) ? (v as T) : fallback;

/** Whatever was saved, made safe: unknown choices go back to the usual one, and a page that is not text is empty. */
export function cleanPages(raw: unknown): ExportPages {
  const r = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>;
  return {
    copyright: text(r.copyright), dedication: text(r.dedication), thanks: text(r.thanks), about: text(r.about),
    sceneBreak: pick(r.sceneBreak, SCENE_BREAKS, 'stars'),
    chapterStyle: pick(r.chapterStyle, CHAPTER_STYLES, 'full'),
  };
}

/** A line such as "© 2026 Name. All rights reserved." for the copyright box, to start from. */
export function copyrightStarter(author: string, year = new Date().getFullYear()): string {
  const name = author.trim();
  return name ? `© ${year} ${name}\nAll rights reserved.` : '';
}
