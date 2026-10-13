// Reading time and reading level of English text. No imports, so it can be tested alone.
// The level is the Flesch reading ease and the Flesch-Kincaid grade. Both use only the length of
// the sentences and the length of the words, so they are a rough guide, not a judge.

export type Counts = { words: number; sentences: number; syllables: number };

export const DEFAULT_WPM = 230; // words a minute, for a reader of fiction
export const WPM_LIMITS: [number, number] = [80, 600];
/** A chapter shorter than this is not judged. A few sentences say too little. */
export const MIN_WORDS = 300;
/** A chapter this many school grades above or below its book is marked. */
export const FLAG_GRADES = 2;

export const NO_COUNTS: Counts = { words: 0, sentences: 0, syllables: 0 };

/** The number of syllables of one word, by a simple rule. It is right for most English words. */
export function countSyllables(raw: string): number {
  let w = String(raw ?? '').toLowerCase().replace(/[^a-z]/g, '');
  if (!w) return 0;
  if (w.length <= 3) return 1;
  w = w.replace(/(?:[^laeiouy]es|ed|[^laeiouy]e)$/, '').replace(/^y/, '');
  const groups = w.match(/[aeiouy]+/g);
  return Math.max(1, groups ? groups.length : 1);
}

const WORD = /[\p{L}\p{N}][\p{L}\p{N}'’-]*/gu;

const HAS_WORD = /[\p{L}\p{N}]/u; // not global: a global pattern remembers where it stopped

/** Count the words, sentences and syllables of a text. A new line ends a sentence. */
export function countsOf(text: string): Counts {
  const t = String(text ?? '');
  const words = t.match(WORD) ?? [];
  if (!words.length) return { ...NO_COUNTS };
  let syllables = 0;
  for (const w of words) syllables += countSyllables(w);
  let sentences = 0;
  for (const part of t.split(/[.!?…]+["”’')\]]*(?:\s+|$)|\n+/)) if (HAS_WORD.test(part)) sentences++;
  return { words: words.length, sentences: Math.max(1, sentences), syllables };
}

export const addCounts = (a: Counts, b: Counts): Counts => ({
  words: a.words + b.words, sentences: a.sentences + b.sentences, syllables: a.syllables + b.syllables,
});

/** Flesch reading ease, from 0 (very hard) to 100 (very easy). Null when there is too little text. */
export function ease(c: Counts): number | null {
  if (c.words < MIN_WORDS || !c.sentences) return null;
  const v = 206.835 - 1.015 * (c.words / c.sentences) - 84.6 * (c.syllables / c.words);
  return Math.max(0, Math.min(100, v));
}

/** Flesch-Kincaid grade: the school grade that can read the text. Null when there is too little text. */
export function grade(c: Counts): number | null {
  if (c.words < MIN_WORDS || !c.sentences) return null;
  return Math.min(20, Math.max(0, 0.39 * (c.words / c.sentences) + 11.8 * (c.syllables / c.words) - 15.59));
}

/** The words for a reading ease score. */
export function easeLabel(e: number | null): string {
  if (e === null) return 'too short to judge';
  if (e >= 90) return 'very easy';
  if (e >= 80) return 'easy';
  if (e >= 70) return 'fairly easy';
  if (e >= 60) return 'plain';
  if (e >= 50) return 'fairly hard';
  if (e >= 30) return 'hard';
  return 'very hard';
}

/** How long the words take to read: "45 sec", "12 min", "1 h 5 min". */
export function readingTime(words: number, wpm = DEFAULT_WPM): string {
  const speed = Math.min(WPM_LIMITS[1], Math.max(WPM_LIMITS[0], Number(wpm) || DEFAULT_WPM));
  const minutes = words / speed;
  if (words <= 0) return '0 min';
  if (minutes < 1) return `${Math.max(1, Math.round(minutes * 60))} sec`;
  const m = Math.round(minutes);
  return m < 60 ? `${m} min` : `${Math.floor(m / 60)} h${m % 60 ? ` ${m % 60} min` : ''}`;
}

/** A reading speed from the box: whole words a minute inside the limits. */
export function cleanWpm(raw: unknown): number {
  const n = Math.floor(Number(raw));
  return Number.isFinite(n) && n > 0 ? Math.min(WPM_LIMITS[1], Math.max(WPM_LIMITS[0], n)) : DEFAULT_WPM;
}

export type Mark = 'harder' | 'easier' | '';

/** Which chapters are much harder or easier than the rest of their book. The average counts each chapter by its words. */
export function marks(chapters: Counts[]): Mark[] {
  let weight = 0, sum = 0;
  const grades = chapters.map((c) => grade(c));
  grades.forEach((g, i) => { if (g !== null) { weight += chapters[i].words; sum += g * chapters[i].words; } });
  if (!weight) return chapters.map(() => '');
  const mean = sum / weight;
  return grades.map((g) => (g === null ? '' : g - mean >= FLAG_GRADES ? 'harder' : mean - g >= FLAG_GRADES ? 'easier' : ''));
}

/** Text made from the HTML of a scene. A paragraph, heading, list item or line break ends a sentence. */
export function htmlToText(html: string): string {
  return String(html ?? '')
    .replace(/<\/(p|h[1-6]|li|blockquote)>|<br\s*\/?>|<hr\s*\/?>/gi, '\n')
    .replace(/<[^>]*>/g, '')
    .replace(/&nbsp;/g, ' ').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"').replace(/&#39;|&apos;/g, "'").replace(/&amp;/g, '&');
}

/** The most a chapter may be above the target grade before it is marked. */
export const TARGET_MARGIN = 2;

/** The target grade a writer chose: a whole number from 1 to 20, or 0 for no target. */
export function cleanTarget(raw: unknown): number {
  const n = Math.round(Number(raw));
  return Number.isFinite(n) && n >= 1 ? Math.min(20, n) : 0;
}

/** Is this text harder to read than the target allows? A chapter that is too short to judge is never over. */
export function overTarget(c: Counts, target: number): boolean {
  const g = grade(c);
  return target > 0 && g !== null && g > target + TARGET_MARGIN;
}
