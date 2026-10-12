// Motion: how much the screen animates. No imports, so it can be tested alone.
// A level is a set of numbers: how long things take, how far they slide, how big the ripple is.

export type Level = 'off' | 'calm' | 'smooth' | 'lively';
export const LEVELS: Level[] = ['off', 'calm', 'smooth', 'lively'];
export const LABEL: Record<Level, string> = { off: 'Off', calm: 'Calm', smooth: 'Smooth', lively: 'Lively' };

export type Params = {
  dur: number;         // milliseconds for a small change, such as a menu opening
  ease: string;        // a CSS cubic-bezier
  travel: number;      // pixels that a page or a pop-up slides
  stagger: number;     // milliseconds between the items of a page that arrive one by one
  rippleMs: number;    // how long a button ripple lasts
  rippleScale: number; // how far a ripple grows, compared with the button size
  rippleAlpha: number; // how strong the ripple starts
};

export const PARAMS: Record<Level, Params> = {
  off:    { dur: 0,   ease: 'linear',                       travel: 0,  stagger: 0,  rippleMs: 0,   rippleScale: 0,   rippleAlpha: 0 },
  calm:   { dur: 140, ease: 'cubic-bezier(.2,0,0,1)',       travel: 4,  stagger: 0,  rippleMs: 450, rippleScale: 1.6, rippleAlpha: .14 },
  smooth: { dur: 200, ease: 'cubic-bezier(.2,.8,.2,1)',     travel: 10, stagger: 28, rippleMs: 600, rippleScale: 2.2, rippleAlpha: .18 },
  lively: { dur: 320, ease: 'cubic-bezier(.34,1.56,.64,1)', travel: 18, stagger: 55, rippleMs: 800, rippleScale: 2.8, rippleAlpha: .24 },
};

/** A saved choice, made safe. Anything that is not a level gives the fallback. */
export const cleanLevel = (raw: unknown, fallback: Level): Level => (LEVELS.includes(raw as Level) ? (raw as Level) : fallback);

/** The level to start with: your saved choice. With no choice, motion is off when the system asks for less motion. */
export const startLevel = (saved: unknown, systemWantsLess: boolean): Level =>
  LEVELS.includes(saved as Level) ? (saved as Level) : systemWantsLess ? 'off' : 'smooth';

/** The next level in the list, going around. */
export const nextLevel = (l: Level): Level => LEVELS[(LEVELS.indexOf(l) + 1) % LEVELS.length];

/** The CSS variables for a level, ready to put on the page. */
export function cssVars(p: Params): Record<string, string> {
  return {
    '--m-dur': `${p.dur}ms`, '--m-ease': p.ease, '--m-travel': `${p.travel}px`, '--m-stagger': `${p.stagger}ms`,
  };
}

/** Parse "cubic-bezier(a,b,c,d)" into a function from time (0 to 1) to progress. Linear when it does not parse. */
export function bezier(text: string): (t: number) => number {
  const m = /cubic-bezier\(\s*([-\d.]+)\s*,\s*([-\d.]+)\s*,\s*([-\d.]+)\s*,\s*([-\d.]+)\s*\)/.exec(text);
  if (!m) return (t) => t;
  const [x1, y1, x2, y2] = [+m[1], +m[2], +m[3], +m[4]];
  const cx = 3 * x1, bx = 3 * (x2 - x1) - cx, ax = 1 - cx - bx;
  const cy = 3 * y1, by = 3 * (y2 - y1) - cy, ay = 1 - cy - by;
  const X = (s: number) => ((ax * s + bx) * s + cx) * s;
  const Y = (s: number) => ((ay * s + by) * s + cy) * s;
  return (t) => {
    if (t <= 0) return 0;
    if (t >= 1) return 1;
    let lo = 0, hi = 1, s = t;
    for (let i = 0; i < 24; i++) { // find the curve position for this time
      const x = X(s);
      if (Math.abs(x - t) < 1e-5) break;
      if (x < t) lo = s; else hi = s;
      s = (lo + hi) / 2;
    }
    return Y(s);
  };
}

/** A smooth slowing-down curve, for counting numbers. */
export const easeOutCubic = (t: number): number => 1 - Math.pow(1 - Math.min(1, Math.max(0, t)), 3);

/** The number to show at time t (0 to 1) when counting from one number to another. Always a whole number. */
export function countAt(from: number, to: number, t: number): number {
  if (t >= 1) return Math.round(to);
  return Math.round(from + (to - from) * easeOutCubic(t));
}
