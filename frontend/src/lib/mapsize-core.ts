// The size of a new map: a shape (aspect ratio) and a size. No imports, so it can be tested alone.

export const SHAPES: [string, string][] = [
  ['1:1', 'Square (1:1)'], ['4:3', 'Landscape (4:3)'], ['3:2', 'Landscape (3:2)'], ['16:9', 'Wide (16:9)'],
  ['3:4', 'Portrait (3:4)'], ['2:3', 'Portrait (2:3)'], ['custom', 'Custom…'],
];
/** The long side of the map in pixels. */
export const SIZES: [string, string, number][] = [['small', 'Small', 1000], ['medium', 'Medium', 2000], ['large', 'Large', 3000]];

export const MIN_SIDE = 200, MAX_SIDE = 8000;
export const DEFAULT_SHAPE = '1:1', DEFAULT_SIZE = 'medium';

const side = (v: unknown): number => {
  const n = Math.round(Number(v));
  return Number.isFinite(n) ? Math.min(MAX_SIDE, Math.max(MIN_SIDE, n)) : MIN_SIDE;
};

/** The width and height of a new map. The long side is the chosen size, and the other side follows the shape. Custom uses your own numbers. */
export function mapDims(shape: string, size: string, custom: { w: unknown; h: unknown } = { w: 2000, h: 2000 }): { w: number; h: number } {
  if (shape === 'custom') return { w: side(custom.w), h: side(custom.h) };
  const m = /^(\d+):(\d+)$/.exec(SHAPES.some(([k]) => k === shape) ? shape : DEFAULT_SHAPE)!;
  const rw = +m[1], rh = +m[2];
  const long = (SIZES.find(([k]) => k === size) ?? SIZES.find(([k]) => k === DEFAULT_SIZE)!)[2];
  return rw >= rh ? { w: long, h: Math.round((long * rh) / rw) } : { w: Math.round((long * rw) / rh), h: long };
}
