// The pages and style choices of a manuscript export. No imports, so it can be tested alone.

export type ExportPages = {
  copyright: string; dedication: string; thanks: string; about: string;
  sceneBreak: 'stars' | 'hash' | 'dashes' | 'blank' | 'heading'; // 'heading': the title of each scene is printed as a sub-heading (a non-fiction book)
  chapterStyle: 'full' | 'title' | 'number' | 'plain';
  citationStyle: 'authordate' | 'numbered' | 'notes';
  bibliography: boolean; // print the list of the sources that were cited, after the last chapter
  indexPage: boolean; // print an index of the terms that are marked in the text, at the end
  exerciseList: boolean; // print all the exercise boxes together, after the last chapter
  takeaways: boolean; // print the key takeaways at the end of each chapter that has some
};

export const SCENE_BREAKS: [ExportPages['sceneBreak'], string][] = [
  ['stars', '* * *  (three stars)'], ['hash', '#  (one hash mark)'], ['dashes', '— — —  (three dashes)'], ['blank', 'An empty line'], ['heading', 'A heading: the title of each section'],
];
export const CHAPTER_STYLES: [ExportPages['chapterStyle'], string][] = [
  ['full', 'Chapter 3: The title'], ['title', 'The title'], ['number', 'Chapter 3'], ['plain', '3. The title'],
];

export const CITATION_STYLES: [ExportPages['citationStyle'], string][] = [
  ['authordate', 'Author and year, for example (Smith, 2020, p. 45)'], ['numbered', 'Numbers, for example [1]'], ['notes', 'A footnote for each citation'],
];

export const DEFAULT_PAGES: ExportPages = { copyright: '', dedication: '', thanks: '', about: '', sceneBreak: 'stars', chapterStyle: 'full', citationStyle: 'authordate', bibliography: false, indexPage: false, exerciseList: false, takeaways: false };

const MAX_TEXT = 5000; // letters for each page

const text = (v: unknown): string => (typeof v === 'string' ? v.replace(/\r\n/g, '\n').slice(0, MAX_TEXT) : '');
const pick = <T extends string>(v: unknown, allowed: readonly [T, string][], fallback: T): T =>
  allowed.some(([k]) => k === v) ? (v as T) : fallback;

/** Whatever was saved, made safe: unknown choices go back to the usual one, and a page that is not text is empty. */
export function cleanPages(raw: unknown, nonfiction = false): ExportPages {
  const r = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>;
  return {
    copyright: text(r.copyright), dedication: text(r.dedication), thanks: text(r.thanks), about: text(r.about),
    // a non-fiction book starts with section headings and plain chapter titles
    sceneBreak: pick(r.sceneBreak, SCENE_BREAKS, nonfiction ? 'heading' : 'stars'),
    chapterStyle: pick(r.chapterStyle, CHAPTER_STYLES, nonfiction ? 'title' : 'full'),
    citationStyle: pick(r.citationStyle, CITATION_STYLES, 'authordate'),
    bibliography: r.bibliography === true,
    indexPage: r.indexPage === true,
    exerciseList: r.exerciseList === true,
    takeaways: r.takeaways === true,
  };
}

/** A line such as "© 2026 Name. All rights reserved." for the copyright box, to start from. */
export function copyrightStarter(author: string, year = new Date().getFullYear()): string {
  const name = author.trim();
  return name ? `© ${year} ${name}\nAll rights reserved.` : '';
}
