// Data model, drawing and hit testing for the map drawing tool.
// A map is a list of layers. Each layer holds items of one type.

export type Pt = [number, number];
export type LayerType = 'land' | 'paint' | 'lines' | 'icons' | 'labels';

export interface Layer {
  id: string;
  name: string;
  type: LayerType;
  visible: boolean;
  locked: boolean;
  items: any[];
}

export interface MapDoc {
  v: 1;
  w: number;
  h: number;
  water: string;
  land: string;
  ink: string;
  layers: Layer[];
}

export const uid = () => Math.random().toString(36).slice(2, 9);

export const LAYER_NAMES: Record<LayerType, string> = {
  land: 'Land', paint: 'Terrain', lines: 'Lines', icons: 'Icons', labels: 'Labels',
};

export const TERRAINS = [
  { key: 'grass', name: 'Grassland', color: '#86b35c' },
  { key: 'forest', name: 'Forest', color: '#487f4a' },
  { key: 'hills', name: 'Hills', color: '#b2a565' },
  { key: 'rock', name: 'Mountains', color: '#8d857c' },
  { key: 'desert', name: 'Desert', color: '#e3cb8d' },
  { key: 'snow', name: 'Snow', color: '#f1f5f7' },
  { key: 'swamp', name: 'Swamp', color: '#617f66' },
  { key: 'sand', name: 'Beach', color: '#ecdba6' },
  { key: 'lake', name: 'Water', color: '#9fc3d1' },
];

export const STYLES: Record<string, { name: string; water: string; land: string; ink: string }> = {
  classic: { name: 'Classic', water: '#a8cbd8', land: '#eadfbf', ink: '#4a3a2a' },
  parchment: { name: 'Parchment', water: '#d8c79b', land: '#f1e5c5', ink: '#5a4430' },
  dark: { name: 'Dark', water: '#1f3a4d', land: '#3f4d41', ink: '#e8dfc8' },
};

export const LINE_TYPES: Record<string, { name: string; color: string; width: number; dash: number[] }> = {
  river: { name: 'River', color: '#4a8fb8', width: 6, dash: [] },
  road: { name: 'Road', color: '#8a6a43', width: 3, dash: [10, 6] },
  trail: { name: 'Trail', color: '#8a6a43', width: 2, dash: [2, 5] },
  border: { name: 'Border', color: '#a8423c', width: 3, dash: [14, 5, 3, 5] },
};

export const ICONS = ['mountain', 'hills', 'pine', 'oak', 'castle', 'city', 'village', 'port', 'tower', 'cave'];

export const FONTS = [
  { name: 'Serif', css: 'Georgia, "Times New Roman", serif' },
  { name: 'Old style', css: 'Palatino, "Palatino Linotype", "Book Antiqua", serif' },
  { name: 'Sans', css: '"Segoe UI", Helvetica, Arial, sans-serif' },
  { name: 'Script', css: '"Segoe Script", "Brush Script MT", cursive' },
];

export function newDoc(w: number, h: number, style = 'classic'): MapDoc {
  const s = STYLES[style] ?? STYLES.classic;
  const layer = (type: LayerType): Layer => ({ id: uid(), name: LAYER_NAMES[type], type, visible: true, locked: false, items: [] });
  return {
    v: 1, w, h, water: s.water, land: s.land, ink: s.ink,
    layers: [layer('land'), layer('paint'), layer('lines'), layer('icons'), layer('labels')],
  };
}

// ---------- colors ----------

function rgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '');
  const f = h.length === 3 ? h.split('').map((c) => c + c).join('') : h;
  return [parseInt(f.slice(0, 2), 16) || 0, parseInt(f.slice(2, 4), 16) || 0, parseInt(f.slice(4, 6), 16) || 0];
}
const rgba = (hex: string, a: number) => { const [r, g, b] = rgb(hex); return `rgba(${r},${g},${b},${a})`; };
export function mix(a: string, b: string, t: number) {
  const x = rgb(a), y = rgb(b);
  return `rgb(${Math.round(x[0] + (y[0] - x[0]) * t)},${Math.round(x[1] + (y[1] - x[1]) * t)},${Math.round(x[2] + (y[2] - x[2]) * t)})`;
}

// ---------- geometry ----------

/** Catmull-Rom curve through the points, as a dense list of points. */
export function smoothPoints(pts: Pt[], closed: boolean, steps = 8): Pt[] {
  const n = pts.length;
  if (n < 3) return pts.slice();
  const at = (i: number): Pt => (closed ? pts[(i + n) % n] : pts[Math.max(0, Math.min(n - 1, i))]);
  const out: Pt[] = [];
  const segs = closed ? n : n - 1;
  for (let i = 0; i < segs; i++) {
    const p0 = at(i - 1), p1 = at(i), p2 = at(i + 1), p3 = at(i + 2);
    for (let s = 0; s < steps; s++) {
      const t = s / steps, t2 = t * t, t3 = t2 * t;
      out.push([
        0.5 * (2 * p1[0] + (-p0[0] + p2[0]) * t + (2 * p0[0] - 5 * p1[0] + 4 * p2[0] - p3[0]) * t2 + (-p0[0] + 3 * p1[0] - 3 * p2[0] + p3[0]) * t3),
        0.5 * (2 * p1[1] + (-p0[1] + p2[1]) * t + (2 * p0[1] - 5 * p1[1] + 4 * p2[1] - p3[1]) * t2 + (-p0[1] + 3 * p1[1] - 3 * p2[1] + p3[1]) * t3),
      ]);
    }
  }
  if (!closed) out.push(pts[n - 1]);
  return out;
}

function segDist(p: Pt, a: Pt, b: Pt) {
  const dx = b[0] - a[0], dy = b[1] - a[1];
  const l2 = dx * dx + dy * dy;
  const t = l2 ? Math.max(0, Math.min(1, ((p[0] - a[0]) * dx + (p[1] - a[1]) * dy) / l2)) : 0;
  return Math.hypot(p[0] - (a[0] + t * dx), p[1] - (a[1] + t * dy));
}

/** Ramer-Douglas-Peucker: drop points that barely change the line. */
export function simplify(pts: Pt[], eps: number): Pt[] {
  if (pts.length < 3) return pts;
  const keep = new Array(pts.length).fill(false);
  keep[0] = keep[pts.length - 1] = true;
  const stack: [number, number][] = [[0, pts.length - 1]];
  while (stack.length) {
    const [a, b] = stack.pop()!;
    let worst = 0, at = -1;
    for (let i = a + 1; i < b; i++) {
      const d = segDist(pts[i], pts[a], pts[b]);
      if (d > worst) { worst = d; at = i; }
    }
    if (at >= 0 && worst > eps) { keep[at] = true; stack.push([a, at], [at, b]); }
  }
  return pts.filter((_, i) => keep[i]);
}

function polyDist(p: Pt, poly: Pt[], closed: boolean) {
  let best = Infinity;
  const n = poly.length;
  for (let i = 0; i < (closed ? n : n - 1); i++) best = Math.min(best, segDist(p, poly[i], poly[(i + 1) % n]));
  return best;
}

function inside(p: Pt, poly: Pt[]) {
  let c = false;
  for (let i = 0, j = poly.length - 1; i < poly.length; j = i++) {
    const [xi, yi] = poly[i], [xj, yj] = poly[j];
    if (yi > p[1] !== yj > p[1] && p[0] < ((xj - xi) * (p[1] - yi)) / (yj - yi) + xi) c = !c;
  }
  return c;
}

// ---------- brush ----------

const stamps = new Map<string, HTMLCanvasElement>();

/** A soft round dab, cached so strokes with thousands of dabs stay fast. */
function stamp(color: string, size: number, hardness: number) {
  size = Math.max(2, Math.round(size));
  const key = `${color}|${size}|${hardness}`;
  let c = stamps.get(key);
  if (!c) {
    c = document.createElement('canvas');
    c.width = c.height = size;
    const x = c.getContext('2d')!;
    const r = size / 2;
    const g = x.createRadialGradient(r, r, 0, r, r, r);
    g.addColorStop(0, rgba(color, 1));
    g.addColorStop(Math.min(0.98, Math.max(0, hardness)), rgba(color, 1));
    g.addColorStop(1, rgba(color, 0));
    x.fillStyle = g;
    x.fillRect(0, 0, size, size);
    if (stamps.size > 200) stamps.clear();
    stamps.set(key, c);
  }
  return c;
}

/** Walks along a stroke and says where to put the next dab. Gives the same result in one go or piece by piece. */
export class DabWalker {
  private last: Pt | null = null;
  private carry = 0;
  constructor(private spacing: number) {}
  walk(p: Pt, emit: (x: number, y: number) => void) {
    if (!this.last) { this.last = p; emit(p[0], p[1]); return; }
    const [x0, y0] = this.last;
    const dx = p[0] - x0, dy = p[1] - y0, d = Math.hypot(dx, dy);
    if (d === 0) return;
    let pos = this.spacing - this.carry;
    while (pos <= d) { emit(x0 + (dx * pos) / d, y0 + (dy * pos) / d); pos += this.spacing; }
    this.carry = d - (pos - this.spacing);
    this.last = p;
  }
}

export const dabSpacing = (size: number) => Math.max(1, size * 0.14);

export function paintStroke(ctx: CanvasRenderingContext2D, it: any, walker: DabWalker, pts: Pt[]) {
  const erase = it.k === 'erase';
  const s = stamp(erase ? '#000000' : it.color, it.size, it.hardness);
  ctx.save();
  ctx.globalAlpha = erase ? 1 : it.opacity;
  ctx.globalCompositeOperation = erase ? 'destination-out' : 'source-over';
  for (const p of pts) walker.walk(p, (x, y) => ctx.drawImage(s, x - s.width / 2, y - s.height / 2));
  ctx.restore();
}

// ---------- icons ----------

function house(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, fill: string) {
  ctx.fillStyle = fill;
  ctx.beginPath(); ctx.rect(x, y, w, h); ctx.fill(); ctx.stroke();
  ctx.beginPath(); ctx.moveTo(x - 2, y); ctx.lineTo(x + w / 2, y - h * 0.7); ctx.lineTo(x + w + 2, y); ctx.closePath();
  ctx.fillStyle = 'rgba(160,90,60,.85)'; ctx.fill(); ctx.stroke();
}

function crenels(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, fill: string) {
  ctx.fillStyle = fill;
  for (let cx = x; cx < x + w - 1; cx += 4) { ctx.beginPath(); ctx.rect(cx, y - 3, 2.5, 3.5); ctx.fill(); ctx.stroke(); }
}

/** Draws one icon in a box from -20 to 20. The caller moves and scales the canvas first. */
function iconShape(ctx: CanvasRenderingContext2D, name: string, ink: string) {
  ctx.lineWidth = 1.6; ctx.lineJoin = 'round'; ctx.lineCap = 'round'; ctx.strokeStyle = ink;
  const path = (fill: string, ...cmds: any[]) => {
    ctx.beginPath();
    for (const [op, ...a] of cmds) (ctx as any)[op](...a);
    ctx.closePath(); ctx.fillStyle = fill; ctx.fill(); ctx.stroke();
  };
  switch (name) {
    case 'mountain':
      path('#d3c9b3', ['moveTo', -19, 14], ['lineTo', -5, -12], ['lineTo', 2, -3], ['lineTo', 9, -15], ['lineTo', 20, 14]);
      path('rgba(0,0,0,.2)', ['moveTo', 9, -15], ['lineTo', 20, 14], ['lineTo', 9, 14], ['lineTo', 6, -3]);
      path('#f6f8fa', ['moveTo', -5, -12], ['lineTo', -9, -5], ['lineTo', -6, -6], ['lineTo', -4, -3], ['lineTo', -1, -6]);
      break;
    case 'hills':
      path('#c9bd88', ['moveTo', -19, 13], ['quadraticCurveTo', -10, -9, 1, 13]);
      path('#bfb27c', ['moveTo', -3, 13], ['quadraticCurveTo', 8, -13, 20, 13]);
      break;
    case 'pine':
      ctx.fillStyle = '#6b4c32'; ctx.fillRect(-1.6, 12, 3.2, 7);
      for (let i = 0; i < 3; i++) {
        const y = -19 + i * 8, hw = 7 + i * 3;
        path('#5f8c55', ['moveTo', 0, y], ['lineTo', hw, y + 13], ['lineTo', -hw, y + 13]);
      }
      break;
    case 'oak':
      ctx.fillStyle = '#6b4c32'; ctx.fillRect(-2, 5, 4, 14);
      for (const [x, y, r] of [[-8, 3, 8], [8, 3, 8], [0, -5, 11]]) {
        ctx.beginPath(); ctx.arc(x, y, r, 0, Math.PI * 2); ctx.fillStyle = '#7fae63'; ctx.fill(); ctx.stroke();
      }
      break;
    case 'castle':
      path('#d9cfb6', ['rect', -12, -3, 24, 20]);
      path('#d9cfb6', ['rect', -17, -11, 8, 28]);
      path('#d9cfb6', ['rect', 9, -11, 8, 28]);
      crenels(ctx, -17, -11, 8, '#d9cfb6'); crenels(ctx, 9, -11, 8, '#d9cfb6'); crenels(ctx, -12, -3, 24, '#d9cfb6');
      path(ink, ['moveTo', -3, 17], ['lineTo', -3, 8], ['quadraticCurveTo', 0, 4, 3, 8], ['lineTo', 3, 17]);
      break;
    case 'city':
      house(ctx, -8, -4, 14, 12, '#e3d8bf'); // back
      house(ctx, -17, 4, 13, 12, '#e8dec6');
      house(ctx, 3, 3, 15, 14, '#dfd3b8');
      break;
    case 'village':
      house(ctx, -10, -1, 20, 16, '#e5dac1');
      path(ink, ['rect', -2.5, 8, 5, 7]);
      break;
    case 'port':
      ctx.beginPath(); ctx.arc(0, -12, 3.5, 0, Math.PI * 2); ctx.stroke();
      ctx.beginPath(); ctx.moveTo(0, -8.5); ctx.lineTo(0, 14); ctx.moveTo(-6, -3); ctx.lineTo(6, -3); ctx.stroke();
      ctx.beginPath(); ctx.moveTo(-13, 4); ctx.quadraticCurveTo(-10, 16, 0, 14); ctx.quadraticCurveTo(10, 16, 13, 4); ctx.stroke();
      break;
    case 'tower':
      path('#d9cfb6', ['rect', -6, -12, 12, 31]);
      crenels(ctx, -6, -12, 12, '#d9cfb6');
      path(ink, ['rect', -2, 12, 4, 7]);
      path(ink, ['rect', -1, -4, 2, 5]);
      break;
    case 'cave':
      path('#bdb198', ['moveTo', -19, 15], ['quadraticCurveTo', -14, -13, 0, -13], ['quadraticCurveTo', 14, -13, 19, 15]);
      path(ink, ['moveTo', -7, 15], ['quadraticCurveTo', -7, 3, 0, 3], ['quadraticCurveTo', 7, 3, 7, 15]);
      break;
  }
}

function drawIcon(ctx: CanvasRenderingContext2D, it: any, doc: MapDoc) {
  ctx.save();
  ctx.translate(it.x, it.y);
  ctx.scale((it.size / 40) * (it.flip ? -1 : 1), it.size / 40);
  iconShape(ctx, it.icon, doc.ink);
  ctx.restore();
}

/** A small picture of an icon, for the palette. */
export function iconThumb(name: string, ink = '#4a3a2a', px = 44): string {
  const c = document.createElement('canvas');
  c.width = c.height = px;
  const x = c.getContext('2d')!;
  x.translate(px / 2, px / 2);
  x.scale(px / 44, px / 44);
  iconShape(x, name, ink);
  return c.toDataURL();
}

// ---------- labels ----------

let measureCtx: CanvasRenderingContext2D | null = null;

function labelFont(it: any) { return `italic ${it.size}px ${it.font}`; }

export function labelBox(it: any): { w: number; h: number } {
  measureCtx ??= document.createElement('canvas').getContext('2d')!;
  measureCtx.font = labelFont(it);
  return { w: measureCtx.measureText(it.text || ' ').width + (it.spacing ?? 0) * (it.text?.length ?? 1), h: it.size };
}

function drawLabel(ctx: CanvasRenderingContext2D, it: any) {
  ctx.save();
  ctx.translate(it.x, it.y);
  ctx.rotate(((it.rot ?? 0) * Math.PI) / 180);
  ctx.font = labelFont(it);
  (ctx as any).letterSpacing = `${it.spacing ?? 0}px`;
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  if (it.halo) {
    ctx.lineJoin = 'round';
    ctx.lineWidth = Math.max(3, it.size * 0.28);
    ctx.strokeStyle = 'rgba(255,255,255,.65)';
    ctx.strokeText(it.text, 0, 0);
  }
  ctx.fillStyle = it.color;
  ctx.fillText(it.text, 0, 0);
  ctx.restore();
}

// ---------- lines and land ----------

function strokePoly(ctx: CanvasRenderingContext2D, poly: Pt[], closed: boolean) {
  ctx.beginPath();
  poly.forEach((p, i) => (i ? ctx.lineTo(p[0], p[1]) : ctx.moveTo(p[0], p[1])));
  if (closed) ctx.closePath();
}

function drawLine(ctx: CanvasRenderingContext2D, it: any) {
  const style = LINE_TYPES[it.type] ?? LINE_TYPES.road;
  const poly = smoothPoints(it.pts, false);
  ctx.save();
  ctx.lineCap = 'round'; ctx.lineJoin = 'round';
  ctx.strokeStyle = it.color ?? style.color;
  if (it.type === 'river') {
    // Thin at the source, wide at the mouth.
    for (let i = 1; i < poly.length; i++) {
      ctx.lineWidth = it.width * (0.35 + 0.65 * (i / poly.length));
      ctx.beginPath(); ctx.moveTo(poly[i - 1][0], poly[i - 1][1]); ctx.lineTo(poly[i][0], poly[i][1]); ctx.stroke();
    }
  } else {
    ctx.lineWidth = it.width;
    ctx.setLineDash(style.dash.map((d) => d * Math.max(1, it.width / 3)));
    strokePoly(ctx, poly, false);
    ctx.stroke();
  }
  ctx.restore();
}

function drawLandHalo(ctx: CanvasRenderingContext2D, it: any, doc: MapDoc) {
  const poly = smoothPoints(it.pts, true);
  const shallow = mix(doc.water, '#ffffff', 0.45);
  ctx.save();
  ctx.lineJoin = 'round';
  for (const [w, a] of [[26, 0.22], [15, 0.3], [7, 0.4]]) {
    ctx.lineWidth = w; ctx.globalAlpha = a; ctx.strokeStyle = shallow;
    strokePoly(ctx, poly, true); ctx.stroke();
  }
  ctx.restore();
}

function drawLand(ctx: CanvasRenderingContext2D, it: any, doc: MapDoc) {
  const poly = smoothPoints(it.pts, true);
  ctx.save();
  ctx.lineJoin = 'round';
  strokePoly(ctx, poly, true);
  ctx.fillStyle = doc.land; ctx.fill();
  ctx.lineWidth = 2.2; ctx.strokeStyle = doc.ink; ctx.stroke();
  ctx.restore();
}

// ---------- layers ----------

export function renderLayer(ctx: CanvasRenderingContext2D, layer: Layer, doc: MapDoc) {
  switch (layer.type) {
    case 'land':
      for (const it of layer.items) drawLandHalo(ctx, it, doc);
      for (const it of layer.items) drawLand(ctx, it, doc);
      break;
    case 'paint':
      for (const it of layer.items) {
        const walker = new DabWalker(dabSpacing(it.size));
        paintStroke(ctx, it, walker, it.pts);
      }
      break;
    case 'lines':
      for (const it of layer.items) drawLine(ctx, it);
      break;
    case 'icons':
      for (const it of [...layer.items].sort((a, b) => a.y - b.y)) drawIcon(ctx, it, doc);
      break;
    case 'labels':
      for (const it of layer.items) drawLabel(ctx, it);
      break;
  }
}

/** Draws the whole map to one canvas, for saving or exporting. */
export function renderDoc(doc: MapDoc, scale = 1): HTMLCanvasElement {
  const out = document.createElement('canvas');
  out.width = Math.round(doc.w * scale);
  out.height = Math.round(doc.h * scale);
  const octx = out.getContext('2d')!;
  octx.fillStyle = doc.water;
  octx.fillRect(0, 0, out.width, out.height);
  for (const layer of doc.layers) {
    if (!layer.visible || !layer.items.length) continue;
    const c = document.createElement('canvas');
    c.width = out.width; c.height = out.height;
    const ctx = c.getContext('2d')!;
    ctx.scale(scale, scale);
    renderLayer(ctx, layer, doc);
    octx.drawImage(c, 0, 0);
  }
  return out;
}

export function docToDataUrl(doc: MapDoc, mime = 'image/webp', scale = 1) {
  return renderDoc(doc, scale).toDataURL(mime, 0.92);
}

// ---------- picking and moving ----------

export function hitItem(layer: Layer, it: any, p: Pt, tol: number): boolean {
  switch (layer.type) {
    case 'icons': return Math.hypot(p[0] - it.x, p[1] - it.y) <= it.size * 0.55;
    case 'labels': {
      const { w, h } = labelBox(it);
      const a = -((it.rot ?? 0) * Math.PI) / 180;
      const dx = p[0] - it.x, dy = p[1] - it.y;
      const rx = dx * Math.cos(a) - dy * Math.sin(a), ry = dx * Math.sin(a) + dy * Math.cos(a);
      return Math.abs(rx) <= w / 2 + 4 && Math.abs(ry) <= h * 0.65;
    }
    case 'lines': return polyDist(p, smoothPoints(it.pts, false), false) <= tol + it.width / 2;
    case 'land': {
      const poly = smoothPoints(it.pts, true);
      return inside(p, poly) || polyDist(p, poly, true) <= tol;
    }
    default: return false;
  }
}

export function moveItem(it: any, dx: number, dy: number) {
  if (it.pts && it.k !== 'paint' && it.k !== 'erase') it.pts = it.pts.map((q: Pt) => [q[0] + dx, q[1] + dy]);
  else if ('x' in it) { it.x += dx; it.y += dy; }
}
