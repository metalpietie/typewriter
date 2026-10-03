/** Region shapes: a border drawn on a map. Points are 0..1 across the map, like pins. */
export type Pt = { x: number; y: number };

/** Is the point inside the shape? (ray casting) */
export function inside(poly: Pt[], x: number, y: number): boolean {
  let hit = false;
  for (let i = 0, j = poly.length - 1; i < poly.length; j = i++) {
    const a = poly[i], b = poly[j];
    if (a.y > y !== b.y > y && x < ((b.x - a.x) * (y - a.y)) / (b.y - a.y) + a.x) hit = !hit;
  }
  return hit;
}

/** The middle of the shape, for its name label. */
export function center(poly: Pt[]): Pt {
  if (!poly.length) return { x: 0, y: 0 };
  let x = 0, y = 0;
  for (const p of poly) { x += p.x; y += p.y; }
  return { x: x / poly.length, y: y / poly.length };
}

/** Where on the outline (edge i runs from point i to point i+1) is the click closest? */
export function nearestEdge(poly: Pt[], x: number, y: number): number {
  let best = 0, bestD = Infinity;
  for (let i = 0; i < poly.length; i++) {
    const a = poly[i], b = poly[(i + 1) % poly.length];
    const dx = b.x - a.x, dy = b.y - a.y;
    const len = dx * dx + dy * dy || 1e-9;
    const t = Math.max(0, Math.min(1, ((x - a.x) * dx + (y - a.y) * dy) / len));
    const d = Math.hypot(x - (a.x + t * dx), y - (a.y + t * dy));
    if (d < bestD) { bestD = d; best = i; }
  }
  return best;
}
