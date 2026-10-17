// How the writing page looks. No imports, so it can be tested alone.

export type Look = { font: string; size: string; spacing: string; width: string; typewriter: boolean; focusLine: boolean; vim: boolean };
type Choice = readonly [id: string, label: string, css: string];

export const FONTS: Choice[] = [
  ['serif', 'Serif', "Georgia, 'Times New Roman', serif"],
  ['book', 'Book', "'Palatino Linotype', 'Book Antiqua', Palatino, Georgia, serif"],
  ['sans', 'Sans', "system-ui, 'Segoe UI', Roboto, Arial, sans-serif"],
  ['mono', 'Typewriter', "'Courier New', Courier, monospace"],
  ['code', 'Code', "'JetBrainsMono NF', Consolas, monospace"],
];
export const SIZES: Choice[] = [['small', 'Small', '0.95rem'], ['normal', 'Normal', '1.1rem'], ['large', 'Large', '1.3rem'], ['huge', 'Huge', '1.5rem']];
export const SPACINGS: Choice[] = [['tight', 'Tight', '1.5'], ['normal', 'Normal', '1.8'], ['loose', 'Loose', '2.1']];
// "auto" is how it always was: the whole page, and a narrow column in focus mode
export const WIDTHS: Choice[] = [['auto', 'Auto', 'none'], ['narrow', 'Narrow', '34rem'], ['medium', 'Medium', '44rem'], ['wide', 'Wide', '56rem']];

export const DEFAULT_LOOK: Look = { font: 'serif', size: 'normal', spacing: 'normal', width: 'auto', typewriter: false, focusLine: false, vim: false };

const pick = (list: Choice[], id: unknown, fallback: string) => (list.some((c) => c[0] === id) ? String(id) : fallback);

/** Saved text from storage, made safe. A wrong or missing value becomes the normal one. */
export function cleanLook(raw: unknown): Look {
  const r = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>;
  const d = DEFAULT_LOOK;
  return {
    font: pick(FONTS, r.font, d.font),
    size: pick(SIZES, r.size, d.size),
    spacing: pick(SPACINGS, r.spacing, d.spacing),
    width: pick(WIDTHS, r.width, d.width),
    typewriter: r.typewriter === true,
    focusLine: r.focusLine === true,
    vim: r.vim === true,
  };
}

const css = (list: Choice[], id: string) => (list.find((c) => c[0] === id) ?? list[0])[2];

/** CSS variables for the editor, as one style text. */
export function lookVars(l: Look): string {
  return `--ed-font:${css(FONTS, l.font)};--ed-size:${css(SIZES, l.size)};--ed-lh:${css(SPACINGS, l.spacing)};--ed-width:${css(WIDTHS, l.width)}`;
}

export const isDefaultLook = (l: Look): boolean => JSON.stringify(l) === JSON.stringify(DEFAULT_LOOK);
