// Callout boxes, captions and ideas: the lists of kinds. No imports, so it can be tested alone.
// The export code (boxes.go) has the same kinds, with the colours of the printed boxes.

export const BOXES: [string, string][] = [
  ['idea', 'Key idea'], ['story', 'Story'], ['example', 'Example'], ['exercise', 'Exercise'], ['warning', 'Warning'], ['quote', 'Quote'],
];
export const CAPTIONS: [string, string][] = [['figure', 'Figure'], ['table', 'Table'], ['chart', 'Chart']];
export const IDEA_KINDS: [string, string][] = [['idea', 'Idea'], ['quote', 'Quote'], ['anecdote', 'Anecdote'], ['statistic', 'Statistic']];

/** The saved fields of an idea, made safe. */
export function cleanIdea(raw: Record<string, unknown>): { form: string; text: string; source: number | null; loc: string; tags: string; notes: string } {
  const src = Number(raw.source);
  return {
    form: IDEA_KINDS.some(([k]) => k === raw.form) ? String(raw.form) : 'idea',
    text: String(raw.text ?? '').slice(0, 5000),
    source: Number.isInteger(src) && src > 0 ? src : null,
    loc: String(raw.loc ?? '').replace(/\s+/g, ' ').trim().slice(0, 60),
    tags: String(raw.tags ?? '').slice(0, 200),
    notes: String(raw.notes ?? '').slice(0, 5000),
  };
}

/** The html for the end of a chapter: a heading, and an empty exercise box to fill in. */
export const EXERCISE_TEMPLATE = '<h2>Exercises</h2><div data-callout="exercise"><p></p></div>';
