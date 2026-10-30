// Where the cursor was in each text (a scene, a Notes page), so the editor opens there again and you can type at once.
// A choice of this computer, like the editor look: it is kept in the browser storage, not in the project.

const KEY = 'typewriter.cursor';
const MAX = 500; // texts remembered; the ones not opened for the longest time go first

function load(): Record<string, number> {
  try {
    const v = JSON.parse(localStorage.getItem(KEY) ?? '{}');
    return v && typeof v === 'object' ? v : {};
  } catch { return {}; }
}

/** The place of the cursor in a text the last time, or null when it was never opened. place is like "scene:12". */
export function savedCursor(place: string): number | null {
  const n = load()[place];
  return typeof n === 'number' && Number.isFinite(n) && n >= 0 ? n : null;
}

/** Remember the place of the cursor in a text. */
export function rememberCursor(place: string, pos: number) {
  const all = load();
  delete all[place]; // the newest goes last, so the oldest is dropped first
  all[place] = pos;
  const keys = Object.keys(all);
  for (const k of keys.slice(0, Math.max(0, keys.length - MAX))) delete all[k];
  try { localStorage.setItem(KEY, JSON.stringify(all)); } catch { /* it lasts until the app closes */ }
}
