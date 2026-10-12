// A simple cover made from a template: the rules for the layout. No imports, so it can be tested alone.
// The drawing itself (canvas) is in cover.ts.

export type Template = 'classic' | 'modern' | 'band';

export const TEMPLATES: [Template, string][] = [
  ['classic', 'Classic: a frame, the title in the middle'],
  ['modern', 'Modern: big title on the left'],
  ['band', 'Band: a coloured band across the cover'],
];

export type Spec = {
  template: Template;
  title: string;
  subtitle: string;
  author: string;
  bg: string;     // background colour, such as #1f2a44
  fg: string;     // text colour
  accent: string; // frame, bar or band colour
  dim: number;    // how much a background picture is darkened, from 0 to 0.8
};

/** The size of a cover in pixels. The ratio 2:3 is the usual one for a book. */
export const COVER_W = 1600;
export const COVER_H = 2400;

export const DEFAULT_SPEC: Spec = { template: 'classic', title: '', subtitle: '', author: '', bg: '#1f2a44', fg: '#f4efe6', accent: '#c9a45c', dim: 0.35 };

const HEX = /^#[0-9a-fA-F]{6}$/;
const colour = (v: unknown, fallback: string): string => (typeof v === 'string' && HEX.test(v) ? v.toLowerCase() : fallback);
const line = (v: unknown, max: number): string => String(v ?? '').replace(/\s+/g, ' ').trim().slice(0, max);

/** Whatever was saved, made safe: a template that exists, colours that are colours, short single-line texts. */
export function cleanSpec(raw: unknown, base: Partial<Spec> = {}): Spec {
  const r = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>;
  const d = { ...DEFAULT_SPEC, ...base };
  const dim = Number(r.dim);
  return {
    template: TEMPLATES.some(([k]) => k === r.template) ? (r.template as Template) : d.template,
    title: line(r.title ?? d.title, 120), subtitle: line(r.subtitle ?? d.subtitle, 120), author: line(r.author ?? d.author, 80),
    bg: colour(r.bg, d.bg), fg: colour(r.fg, d.fg), accent: colour(r.accent, d.accent),
    dim: Number.isFinite(dim) ? Math.min(0.8, Math.max(0, dim)) : d.dim,
  };
}

/** Break a text into lines that are not wider than maxWidth. A word that is wider than that stays on a line of its own. */
export function wrapLines(text: string, maxWidth: number, measure: (s: string) => number): string[] {
  const lines: string[] = [];
  let cur = '';
  for (const word of text.split(' ').filter(Boolean)) {
    const next = cur ? cur + ' ' + word : word;
    if (cur && measure(next) > maxWidth) { lines.push(cur); cur = word; } else cur = next;
  }
  if (cur) lines.push(cur);
  return lines;
}

export type Fit = { size: number; lines: string[] };

/**
 * The biggest type size (from max down to min) at which the text fits a box, and its lines.
 * measure(text, size) is the width of the text at that size. At the smallest size the text is cut to the lines that fit.
 */
export function fitText(text: string, boxW: number, boxH: number, measure: (s: string, size: number) => number, opts: { max?: number; min?: number; lineHeight?: number } = {}): Fit {
  const max = opts.max ?? 220, min = opts.min ?? 56, lh = opts.lineHeight ?? 1.12;
  const t = text.trim();
  if (!t) return { size: min, lines: [] };
  for (let size = max; size >= min; size -= 4) {
    const lines = wrapLines(t, boxW, (s) => measure(s, size));
    if (lines.length * size * lh <= boxH && lines.every((l) => measure(l, size) <= boxW)) return { size, lines };
  }
  const lines = wrapLines(t, boxW, (s) => measure(s, min));
  return { size: min, lines: lines.slice(0, Math.max(1, Math.floor(boxH / (min * lh)))) };
}

/** Black or white, whichever reads better on this background colour. */
export function textOn(bg: string): string {
  if (!HEX.test(bg)) return '#ffffff';
  const [r, g, b] = [1, 3, 5].map((i) => parseInt(bg.slice(i, i + 2), 16) / 255).map((c) => (c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)));
  return 0.2126 * r + 0.7152 * g + 0.0722 * b > 0.18 ? '#111111' : '#ffffff';
}

/** The size that a picture is drawn at to fill a box (cover fit): it is cut at the edges, not stretched. */
export function coverFit(imgW: number, imgH: number, boxW: number, boxH: number): { w: number; h: number; x: number; y: number } {
  if (imgW <= 0 || imgH <= 0) return { w: boxW, h: boxH, x: 0, y: 0 };
  const k = Math.max(boxW / imgW, boxH / imgH);
  const w = imgW * k, h = imgH * k;
  return { w, h, x: (boxW - w) / 2, y: (boxH - h) / 2 };
}

/** The size that a picture is made at for a cover: the long side at most 2400, the ratio kept. */
export function scaledSize(w: number, h: number, longSide = 2400): { w: number; h: number } {
  if (w <= 0 || h <= 0) return { w: 1, h: 1 };
  const k = Math.min(1, longSide / Math.max(w, h));
  return { w: Math.max(1, Math.round(w * k)), h: Math.max(1, Math.round(h * k)) };
}
