<script lang="ts">
  import { onMount, onDestroy, tick } from 'svelte';
  import { db, type Item } from '../db.svelte';
  import { Exporter } from '../../../bindings/typewriter';
  import SaveIndicator from '../SaveIndicator.svelte';
  import ConfirmButton from '../ConfirmButton.svelte';
  import {
    type MapDoc, type Layer, type LayerType, type Pt,
    newDoc, uid, renderLayer, renderDoc, docToDataUrl, paintStroke, DabWalker, dabSpacing,
    smoothPoints, simplify, hitItem, moveItem, labelBox, iconThumb,
    TERRAINS, STYLES, LINE_TYPES, ICONS, FONTS, LAYER_NAMES,
  } from './draw';

  let { map, onclose }: { map: Item; onclose: () => void } = $props();

  const TOOLS = [
    ['select', '↖', 'Select and move (V)'],
    ['pan', '✋', 'Pan (H, or hold Space)'],
    ['land', '🏝', 'Land (L)'],
    ['brush', '🖌', 'Terrain brush (B)'],
    ['erase', '⌫', 'Eraser (E)'],
    ['line', '〰', 'Rivers, roads and borders (R)'],
    ['stamp', '⛰', 'Stamps (S)'],
    ['label', 'T', 'Text label (T)'],
  ] as const;
  type Tool = (typeof TOOLS)[number][0];
  const NEEDS: Partial<Record<Tool, LayerType>> = { land: 'land', brush: 'paint', erase: 'paint', line: 'lines', stamp: 'icons', label: 'labels' };
  const SHORTCUT: Record<string, Tool> = { v: 'select', h: 'pan', l: 'land', b: 'brush', e: 'erase', r: 'line', s: 'stamp', t: 'label' };

  // ----- the drawing itself. It is plain data, so a counter tells the screen when it changed. -----
  const existing = db.lists.mapdoc.find((d) => d.map === map.id);
  let docId = existing?.id ?? 0;
  let doc: MapDoc = existing?.doc
    ? ($state.snapshot(existing.doc) as MapDoc)
    : newDoc(map.w || 2400, map.h || 1500, map.style || 'classic');
  let rev = $state(0);
  const bump = () => rev++;

  let tool = $state<Tool>('select');
  let activeId = $state(doc.layers[0].id);
  let sel = $state<{ layer: string; idx: number } | null>(null);
  let notice = $state('');
  let noticeTimer: ReturnType<typeof setTimeout>;
  let exportScale = $state(1);
  let exportMsg = $state('');

  // tool options
  let terrain = $state('grass');
  let color = $state(TERRAINS[0].color);
  const unit = doc.w / 2400; // sizes start in proportion to the map
  let brushSize = $state(Math.round(90 * unit));
  let hardness = $state(0.45);
  let opacity = $state(0.9);
  let lineType = $state('river');
  let lineWidth = $state(Math.round(LINE_TYPES.river.width * 1.5 * unit));
  let pathMode = $state<'free' | 'click'>('free');
  let icon = $state('mountain');
  let iconSize = $state(Math.round(90 * unit));
  let scatter = $state(false);
  let fontIdx = $state(0);
  let labelSize = $state(Math.round(50 * unit));
  let labelColor = $state('#3b2f25');
  let halo = $state(true);

  const layersView = $derived.by(() => {
    rev;
    return doc.layers
      .map((l) => ({ id: l.id, name: l.name, type: l.type, visible: l.visible, locked: l.locked, count: l.items.length }))
      .reverse();
  });
  const selView = $derived.by(() => {
    rev;
    const it = sel ? doc.layers.find((l) => l.id === sel!.layer)?.items[sel.idx] : null;
    return it ? { ...it } : null;
  });
  const style = $derived.by(() => { rev; return { water: doc.water, land: doc.land, ink: doc.ink }; });

  const iconThumbs: Record<string, string> = $state({});
  const ghosts = new Map<string, HTMLImageElement>();
  let newType = $state<LayerType>('paint');

  function say(msg: string) {
    notice = msg;
    clearTimeout(noticeTimer);
    noticeTimer = setTimeout(() => (notice = ''), 3500);
  }

  // ----- history -----
  const snap = () => JSON.stringify({ water: doc.water, land: doc.land, ink: doc.ink, layers: doc.layers });
  let history: string[] = [snap()];
  let hi = $state(0);
  let commitTimer: ReturnType<typeof setTimeout>;

  function commit() {
    clearTimeout(commitTimer);
    history.length = hi + 1;
    history.push(snap());
    if (history.length > 60) history.shift();
    hi = history.length - 1;
    persist();
    bump();
  }
  /** For sliders and text boxes, so one drag is one undo step. */
  function commitSoon() {
    clearTimeout(commitTimer);
    commitTimer = setTimeout(commit, 450);
    persist();
  }
  function restore(i: number) {
    const s = JSON.parse(history[i]);
    doc.water = s.water; doc.land = s.land; doc.ink = s.ink; doc.layers = s.layers;
    caches.clear();
    sel = null;
    if (!doc.layers.some((l) => l.id === activeId)) activeId = doc.layers[0]?.id ?? '';
    bump(); requestDraw(); persist();
  }
  function undo() { clearTimeout(commitTimer); if (hi > 0) restore(--hi); }
  function redo() { if (hi < history.length - 1) restore(++hi); }

  // ----- saving -----
  let imageTimer: ReturnType<typeof setTimeout>;
  let imageDirty = false;

  function persist() {
    if (!docId) return;
    db.queue('mapdoc', { id: docId, kind: 'mapdoc', updatedAt: '', map: map.id, doc } as Item);
    imageDirty = true;
    clearTimeout(imageTimer);
    imageTimer = setTimeout(saveImage, 4000);
  }

  /** The Maps screen shows a picture of the drawing, so pins and paths can sit on top of it. */
  async function saveImage() {
    clearTimeout(imageTimer);
    if (!imageDirty) return;
    imageDirty = false;
    const cur = db.get('map', map.id);
    if (!cur) return;
    const image = docToDataUrl(doc);
    await db.save('map', { ...$state.snapshot(cur), image, w: doc.w, h: doc.h, bg: doc.water, drawn: true });
  }

  // ----- canvas, view and layer pictures -----
  let canvas = $state<HTMLCanvasElement>();
  let cw = $state(800);
  let ch = $state(600);
  let view = { tx: 0, ty: 0, k: 1 };
  let raf = 0;
  let hover: Pt | null = null;
  let spaceDown = false;
  let userMovedView = false; // until you zoom or pan, keep the whole map in view when the window resizes

  const caches = new Map<string, { c: HTMLCanvasElement; dirty: boolean }>();

  function layerCanvas(layer: Layer) {
    let e = caches.get(layer.id);
    if (!e) {
      const c = document.createElement('canvas');
      c.width = doc.w; c.height = doc.h;
      e = { c, dirty: true };
      caches.set(layer.id, e);
    }
    if (e.dirty) {
      const ctx = e.c.getContext('2d')!;
      ctx.clearRect(0, 0, doc.w, doc.h);
      renderLayer(ctx, layer, doc);
      e.dirty = false;
    }
    return e.c;
  }
  const dirty = (id: string) => { const e = caches.get(id); if (e) e.dirty = true; requestDraw(); };
  const dirtyAll = () => { for (const e of caches.values()) e.dirty = true; requestDraw(); };

  function requestDraw() { if (!raf) raf = requestAnimationFrame(draw); }

  function fit() {
    const k = Math.min(cw / doc.w, ch / doc.h) * 0.92;
    view = { k, tx: (cw - doc.w * k) / 2, ty: (ch - doc.h * k) / 2 };
    userMovedView = false;
    requestDraw();
  }

  $effect(() => {
    if (!canvas || cw < 10 || ch < 10) return;
    const dpr = window.devicePixelRatio || 1;
    canvas.width = Math.round(cw * dpr);
    canvas.height = Math.round(ch * dpr);
    if (!userMovedView) fit(); else requestDraw();
  });

  // paths being drawn right now
  let freePts: Pt[] | null = null;
  let clickPts = $state<Pt[] | null>(null);

  function draw() {
    raf = 0;
    if (!canvas) return;
    const ctx = canvas.getContext('2d')!;
    const dpr = window.devicePixelRatio || 1;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, cw, ch);
    ctx.save();
    ctx.translate(view.tx, view.ty);
    ctx.scale(view.k, view.k);
    ctx.imageSmoothingQuality = 'high';
    ctx.shadowColor = 'rgba(0,0,0,.45)'; ctx.shadowBlur = 24 / view.k;
    ctx.fillStyle = doc.water;
    ctx.fillRect(0, 0, doc.w, doc.h);
    ctx.shadowColor = 'transparent';
    for (const layer of doc.layers) {
      if (layer.visible && (layer.items.length || caches.has(layer.id))) ctx.drawImage(layerCanvas(layer), 0, 0);
    }
    drawPreview(ctx);
    drawSelection(ctx);
    ctx.restore();
  }

  function trace(ctx: CanvasRenderingContext2D, poly: Pt[], closed = false) {
    ctx.beginPath();
    poly.forEach((p, i) => (i ? ctx.lineTo(p[0], p[1]) : ctx.moveTo(p[0], p[1])));
    if (closed) ctx.closePath();
  }

  function drawPreview(ctx: CanvasRenderingContext2D) {
    const k = view.k;
    const pts = freePts ?? clickPts;
    if (pts && pts.length) {
      const shown = clickPts && hover ? [...pts, hover] : pts;
      ctx.save();
      ctx.lineJoin = 'round'; ctx.lineCap = 'round';
      if (tool === 'land') {
        if (shown.length > 2) { trace(ctx, smoothPoints(shown, true), true); ctx.fillStyle = 'rgba(255,255,255,.25)'; ctx.fill(); }
        ctx.strokeStyle = doc.ink;
      } else ctx.strokeStyle = LINE_TYPES[lineType].color;
      ctx.lineWidth = 2.5 / k;
      trace(ctx, smoothPoints(shown, tool === 'land' && shown.length > 2));
      ctx.stroke();
      if (clickPts) {
        ctx.fillStyle = '#fff';
        for (const p of pts) { ctx.beginPath(); ctx.arc(p[0], p[1], 4 / k, 0, Math.PI * 2); ctx.fill(); }
      }
      ctx.restore();
    }
    if (hover && !drag) {
      ctx.save();
      ctx.lineWidth = 1.2 / k;
      if (tool === 'brush' || tool === 'erase') {
        ctx.strokeStyle = 'rgba(255,255,255,.9)';
        ctx.beginPath(); ctx.arc(hover[0], hover[1], brushSize / 2, 0, Math.PI * 2); ctx.stroke();
        ctx.strokeStyle = 'rgba(0,0,0,.6)';
        ctx.beginPath(); ctx.arc(hover[0], hover[1], brushSize / 2 + 1 / k, 0, Math.PI * 2); ctx.stroke();
      } else if (tool === 'stamp') {
        ctx.globalAlpha = 0.5;
        ctx.translate(hover[0], hover[1]);
        ctx.scale(iconSize / 44, iconSize / 44);
        let img = ghosts.get(icon);
        if (!img) { img = new Image(); img.src = iconThumbs[icon]; ghosts.set(icon, img); }
        if (img.complete) ctx.drawImage(img, -22, -22, 44, 44);
      }
      ctx.restore();
    }
  }

  function selectedItem() {
    if (!sel) return null;
    const layer = doc.layers.find((l) => l.id === sel!.layer);
    const item = layer?.items[sel.idx];
    return layer && item ? { layer, item } : null;
  }

  function drawSelection(ctx: CanvasRenderingContext2D) {
    const s = selectedItem();
    if (!s) return;
    const { layer, item } = s;
    const k = view.k;
    ctx.save();
    ctx.lineWidth = 1.5 / k;
    ctx.strokeStyle = '#2cd5e8';
    ctx.setLineDash([6 / k, 4 / k]);
    if (layer.type === 'icons') {
      ctx.strokeRect(item.x - item.size / 2, item.y - item.size / 2, item.size, item.size);
    } else if (layer.type === 'labels') {
      const { w, h } = labelBox(item);
      ctx.translate(item.x, item.y);
      ctx.rotate(((item.rot ?? 0) * Math.PI) / 180);
      ctx.strokeRect(-w / 2 - 4, -h * 0.65, w + 8, h * 1.3);
    } else if (item.pts) {
      trace(ctx, smoothPoints(item.pts, layer.type === 'land'), layer.type === 'land');
      ctx.stroke();
      ctx.setLineDash([]);
      ctx.fillStyle = '#fff';
      for (const p of item.pts as Pt[]) { ctx.beginPath(); ctx.rect(p[0] - 4 / k, p[1] - 4 / k, 8 / k, 8 / k); ctx.fill(); ctx.stroke(); }
    }
    ctx.restore();
  }

  // ----- layers -----
  /** Find the layer a tool should draw on. Make one if the map has none of that type. */
  function layerFor(type: LayerType): Layer | null {
    const ok = (l: Layer) => l.type === type && l.visible && !l.locked;
    const active = doc.layers.find((l) => l.id === activeId);
    const found = active && ok(active) ? active : doc.layers.find(ok);
    if (found) { activeId = found.id; return found; }
    if (doc.layers.some((l) => l.type === type)) {
      say(`The ${LAYER_NAMES[type].toLowerCase()} layer is hidden or locked.`);
      return null;
    }
    const layer: Layer = { id: uid(), name: LAYER_NAMES[type], type, visible: true, locked: false, items: [] };
    doc.layers.push(layer);
    activeId = layer.id;
    return layer;
  }

  function addLayer(type: LayerType) {
    const n = doc.layers.filter((l) => l.type === type).length + 1;
    const layer: Layer = { id: uid(), name: n > 1 ? `${LAYER_NAMES[type]} ${n}` : LAYER_NAMES[type], type, visible: true, locked: false, items: [] };
    doc.layers.push(layer);
    activeId = layer.id;
    commit();
  }
  function toggleLayer(id: string, field: 'visible' | 'locked') {
    const l = doc.layers.find((x) => x.id === id);
    if (!l) return;
    l[field] = !l[field];
    if (field === 'locked' && l.locked && sel?.layer === id) sel = null;
    if (field === 'visible' && !l.visible && sel?.layer === id) sel = null;
    requestDraw(); commit();
  }
  function renameLayer(id: string, name: string) {
    const l = doc.layers.find((x) => x.id === id);
    if (l) { l.name = name; commitSoon(); bump(); }
  }
  function moveLayer(id: string, dir: -1 | 1) {
    const i = doc.layers.findIndex((l) => l.id === id);
    const j = i + dir;
    if (i < 0 || j < 0 || j >= doc.layers.length) return;
    [doc.layers[i], doc.layers[j]] = [doc.layers[j], doc.layers[i]];
    requestDraw(); commit();
  }
  function deleteLayer(id: string) {
    if (doc.layers.length <= 1) { say('A map needs at least one layer.'); return; }
    doc.layers = doc.layers.filter((l) => l.id !== id);
    caches.delete(id);
    if (activeId === id) activeId = doc.layers[doc.layers.length - 1].id;
    if (sel?.layer === id) sel = null;
    requestDraw(); commit();
  }

  // ----- pointer handling -----
  type Drag =
    | { t: 'pan'; sx: number; sy: number; tx: number; ty: number }
    | { t: 'stroke'; layer: Layer; item: any; walker: DabWalker; ctx: CanvasRenderingContext2D }
    | { t: 'path' }
    | { t: 'move'; layer: Layer; item: any; last: Pt; moved: boolean }
    | { t: 'vertex'; layer: Layer; item: any; i: number }
    | { t: 'scatter'; layer: Layer; last: Pt };
  let drag: Drag | null = null;

  function toDoc(e: { clientX: number; clientY: number }): Pt {
    const r = canvas!.getBoundingClientRect();
    return [(e.clientX - r.left - view.tx) / view.k, (e.clientY - r.top - view.ty) / view.k];
  }

  function chooseTool(t: Tool) {
    if (clickPts) clickPts = null;
    freePts = null;
    tool = t;
    const need = NEEDS[t];
    if (need) {
      const active = doc.layers.find((l) => l.id === activeId);
      if (!active || active.type !== need) {
        const l = doc.layers.find((x) => x.type === need && x.visible && !x.locked);
        if (l) activeId = l.id;
      }
    }
    requestDraw();
  }

  function pointerDown(e: PointerEvent) {
    canvas!.setPointerCapture(e.pointerId);
    if (e.button === 1 || e.button === 2 || spaceDown || tool === 'pan') {
      userMovedView = true;
      drag = { t: 'pan', sx: e.clientX, sy: e.clientY, tx: view.tx, ty: view.ty };
      return;
    }
    if (e.button !== 0) return;
    const p = toDoc(e);
    switch (tool) {
      case 'select': startSelect(p); break;
      case 'brush': case 'erase': startStroke(p); break;
      case 'land': case 'line':
        if (pathMode === 'free') { freePts = [p]; drag = { t: 'path' }; }
        else clickAdd(p);
        break;
      case 'stamp': startStamp(p); break;
      case 'label': placeLabel(p); break;
    }
    requestDraw();
  }

  function pointerMove(e: PointerEvent) {
    const p = toDoc(e);
    hover = p;
    if (!drag) { if (tool !== 'select' && tool !== 'pan') requestDraw(); return; }
    switch (drag.t) {
      case 'pan':
        view.tx = drag.tx + e.clientX - drag.sx; view.ty = drag.ty + e.clientY - drag.sy;
        break;
      case 'stroke': {
        const last = drag.item.pts[drag.item.pts.length - 1];
        if (Math.hypot(p[0] - last[0], p[1] - last[1]) < 1.5) return;
        drag.item.pts.push(p);
        paintStroke(drag.ctx, drag.item, drag.walker, [p]);
        break;
      }
      case 'path': {
        const last = freePts![freePts!.length - 1];
        if (Math.hypot(p[0] - last[0], p[1] - last[1]) >= Math.max(2, 3 / view.k)) freePts!.push(p);
        break;
      }
      case 'move': {
        const dx = p[0] - drag.last[0], dy = p[1] - drag.last[1];
        if (dx || dy) { moveItem(drag.item, dx, dy); drag.last = p; drag.moved = true; dirty(drag.layer.id); }
        break;
      }
      case 'vertex':
        drag.item.pts[drag.i] = p;
        dirty(drag.layer.id);
        break;
      case 'scatter':
        if (Math.hypot(p[0] - drag.last[0], p[1] - drag.last[1]) >= iconSize * 0.7) { dropIcon(drag.layer, p, true); drag.last = p; dirty(drag.layer.id); }
        break;
    }
    requestDraw();
  }

  function pointerUp() {
    const d = drag;
    drag = null;
    if (!d) return;
    switch (d.t) {
      case 'stroke': commit(); break;
      case 'path': finishPath(freePts!); freePts = null; break;
      case 'move': if (d.moved) commit(); break;
      case 'vertex': case 'scatter': commit(); break;
    }
    requestDraw();
  }

  function wheel(e: WheelEvent) {
    e.preventDefault();
    userMovedView = true;
    const r = canvas!.getBoundingClientRect();
    const wx = (e.clientX - r.left - view.tx) / view.k, wy = (e.clientY - r.top - view.ty) / view.k;
    view.k = Math.min(8, Math.max(0.05, view.k * Math.exp(-e.deltaY * 0.0015)));
    view.tx = e.clientX - r.left - wx * view.k; view.ty = e.clientY - r.top - wy * view.k;
    requestDraw();
  }

  // ----- tools -----
  function startStroke(p: Pt) {
    const layer = layerFor('paint');
    if (!layer) return;
    const cached = layerCanvas(layer); // draw what is already there first
    const item = tool === 'erase'
      ? { k: 'erase', size: brushSize, hardness, pts: [p] }
      : { k: 'paint', color, size: brushSize, hardness, opacity, pts: [p] };
    layer.items.push(item);
    const ctx = cached.getContext('2d')!;
    const walker = new DabWalker(dabSpacing(brushSize));
    paintStroke(ctx, item, walker, [p]);
    drag = { t: 'stroke', layer, item, walker, ctx };
  }

  function finishPath(pts: Pt[]) {
    const isLand = tool === 'land';
    const clean = simplify(pts, Math.max(1.5, 2 / view.k));
    if (clean.length < (isLand ? 3 : 2)) return;
    const layer = layerFor(isLand ? 'land' : 'lines');
    if (!layer) return;
    layer.items.push(isLand ? { k: 'land', pts: clean } : { k: 'line', type: lineType, width: lineWidth, pts: clean });
    dirty(layer.id);
    commit();
  }

  function clickAdd(p: Pt) {
    if (!clickPts) { clickPts = [p]; return; }
    if (tool === 'land' && clickPts.length >= 3 && Math.hypot(p[0] - clickPts[0][0], p[1] - clickPts[0][1]) < 12 / view.k) {
      finishClick();
      return;
    }
    clickPts.push(p);
  }
  function finishClick() {
    const pts = clickPts;
    clickPts = null;
    if (pts) finishPath(pts);
    requestDraw();
  }

  function dropIcon(layer: Layer, p: Pt, jitter: boolean) {
    const r = () => Math.random() - 0.5;
    layer.items.push({
      k: 'icon', icon,
      x: p[0] + (jitter ? r() * iconSize * 0.6 : 0), y: p[1] + (jitter ? r() * iconSize * 0.6 : 0),
      size: Math.round(iconSize * (jitter ? 0.8 + Math.random() * 0.4 : 1)), flip: jitter ? Math.random() < 0.5 : false,
    });
  }
  function startStamp(p: Pt) {
    const layer = layerFor('icons');
    if (!layer) return;
    dropIcon(layer, p, false);
    dirty(layer.id);
    if (scatter) drag = { t: 'scatter', layer, last: p };
    else commit();
  }

  async function placeLabel(p: Pt) {
    const layer = layerFor('labels');
    if (!layer) return;
    layer.items.push({ k: 'label', text: 'New label', x: p[0], y: p[1], size: labelSize, font: FONTS[fontIdx].css, rot: 0, color: labelColor, halo, spacing: 2 });
    sel = { layer: layer.id, idx: layer.items.length - 1 };
    chooseTool('select');
    dirty(layer.id);
    commit();
    await tick();
    labelInput?.focus();
    labelInput?.select();
  }
  let labelInput = $state<HTMLInputElement>();

  function startSelect(p: Pt) {
    const tol = 8 / view.k;
    const cur = selectedItem();
    if (cur && cur.item.pts && !cur.layer.locked) {
      const i = (cur.item.pts as Pt[]).findIndex((q) => Math.hypot(q[0] - p[0], q[1] - p[1]) < 9 / view.k);
      if (i >= 0) { drag = { t: 'vertex', layer: cur.layer, item: cur.item, i }; return; }
    }
    for (let li = doc.layers.length - 1; li >= 0; li--) {
      const layer = doc.layers[li];
      if (!layer.visible || layer.locked) continue;
      for (let ii = layer.items.length - 1; ii >= 0; ii--) {
        const item = layer.items[ii];
        if (layer.type !== 'paint' && hitItem(layer, item, p, tol)) {
          sel = { layer: layer.id, idx: ii };
          activeId = layer.id;
          drag = { t: 'move', layer, item, last: p, moved: false };
          bump();
          return;
        }
      }
    }
    sel = null;
    bump();
  }

  function deleteSelected() {
    const s = selectedItem();
    if (!s) return;
    s.layer.items.splice(sel!.idx, 1);
    sel = null;
    dirty(s.layer.id);
    commit();
  }
  function duplicateSelected() {
    const s = selectedItem();
    if (!s) return;
    const copy = structuredClone(s.item);
    moveItem(copy, 24, 24);
    s.layer.items.push(copy);
    sel = { layer: s.layer.id, idx: s.layer.items.length - 1 };
    dirty(s.layer.id);
    commit();
  }

  /** Change one field of the selected item. */
  function setProp(field: string, value: any) {
    const s = selectedItem();
    if (!s) return;
    s.item[field] = value;
    dirty(s.layer.id);
    commitSoon();
    bump();
  }
  function setStyle(field: 'water' | 'land' | 'ink', value: string) {
    doc[field] = value;
    dirtyAll();
    commitSoon();
    bump();
  }

  // ----- keyboard -----
  function key(e: KeyboardEvent) {
    const typing = (e.target as HTMLElement | null)?.closest?.('input, textarea, select');
    const mod = e.ctrlKey || e.metaKey;
    const k = e.key.toLowerCase();
    if (mod && !typing) {
      if (k === 'z' && !e.shiftKey) { e.preventDefault(); e.stopImmediatePropagation(); undo(); }
      else if (k === 'y' || (k === 'z' && e.shiftKey)) { e.preventDefault(); e.stopImmediatePropagation(); redo(); }
      else if (k === 'd') { e.preventDefault(); duplicateSelected(); }
      else if (k === '0') { e.preventDefault(); fit(); }
      return;
    }
    if (typing || mod) return;
    if (e.key === ' ') { spaceDown = true; e.preventDefault(); return; }
    if (SHORTCUT[k]) { chooseTool(SHORTCUT[k]); return; }
    if (e.key === 'Delete' || e.key === 'Backspace') { deleteSelected(); }
    else if (e.key === 'Enter') { if (clickPts) finishClick(); }
    else if (e.key === 'Escape') { clickPts = null; freePts = null; sel = null; bump(); requestDraw(); }
    else if (e.key === '[' || e.key === ']') {
      const f = e.key === ']' ? 1.15 : 1 / 1.15;
      if (tool === 'stamp') iconSize = Math.max(8, Math.min(300, Math.round(iconSize * f)));
      else brushSize = Math.max(4, Math.min(400, Math.round(brushSize * f)));
      requestDraw();
    }
  }
  const keyUp = (e: KeyboardEvent) => { if (e.key === ' ') spaceDown = false; };

  onMount(async () => {
    window.addEventListener('keydown', key, true);
    window.addEventListener('keyup', keyUp, true);
    for (const n of ICONS) iconThumbs[n] = iconThumb(n, '#4a3a2a');
    if (!docId) docId = (await db.save('mapdoc', { map: map.id, doc })).id;
    fit();
  });

  onDestroy(() => {
    window.removeEventListener('keydown', key, true);
    window.removeEventListener('keyup', keyUp, true);
    cancelAnimationFrame(raf);
    clearTimeout(commitTimer);
    if (docId && history.length > 1) db.queue('mapdoc', { id: docId, kind: 'mapdoc', updatedAt: '', map: map.id, doc } as Item);
    db.flush();
    saveImage().catch(() => {});
  });

  async function exportPng() {
    try {
      const url = renderDoc(doc, exportScale).toDataURL('image/png');
      const path = await Exporter.SaveImage(map.name || 'map', url);
      exportMsg = path ? `Saved: ${path}` : '';
    } catch (e: any) {
      exportMsg = e?.message ?? String(e);
    }
  }

  const cursor = $derived(spaceDown || tool === 'pan' ? 'grab' : tool === 'select' ? 'default' : 'crosshair');
</script>

<div class="editor">
  <header>
    <button class="primary" onclick={onclose}>← Done</button>
    <strong>{map.name}</strong>
    <button disabled={hi === 0} onclick={undo} title="Undo (Ctrl+Z)">↶ Undo</button>
    <button disabled={hi >= history.length - 1} onclick={redo} title="Redo (Ctrl+Y)">↷ Redo</button>
    <button onclick={fit} title="Fit to window (Ctrl+0)">Fit</button>
    <span class="grow"></span>
    {#if notice}<span class="notice">{notice}</span>{/if}
    <SaveIndicator />
    <select bind:value={exportScale} title="Export size" style="width:auto">
      <option value={1}>PNG 1×</option><option value={2}>PNG 2×</option>
    </select>
    <button onclick={exportPng}>Export PNG…</button>
  </header>
  {#if exportMsg}<div class="exportmsg">{exportMsg}</div>{/if}

  <aside class="left">
    <div class="tools">
      {#each TOOLS as [id, glyph, title]}
        <button class:active={tool === id} {title} onclick={() => chooseTool(id)}>{glyph}</button>
      {/each}
    </div>

    <div class="opts">
      {#if tool === 'select'}
        <p class="muted">Click something to select it. Drag to move it. Drag a square to reshape a line or coast. Press Delete to remove it.</p>
      {:else if tool === 'pan'}
        <p class="muted">Drag to move the view. Scroll to zoom. You can also hold Space with any tool.</p>
      {:else if tool === 'land'}
        <h3>Land</h3>
        <p class="muted">Draw around an island or continent. The shape closes by itself.</p>
        {@render modeToggle()}
      {:else if tool === 'brush' || tool === 'erase'}
        <h3>{tool === 'brush' ? 'Terrain brush' : 'Eraser'}</h3>
        {#if tool === 'brush'}
          <div class="swatches">
            {#each TERRAINS as t}
              <button class="swatch" class:on={terrain === t.key} style:background={t.color} title={t.name}
                onclick={() => { terrain = t.key; color = t.color; }}></button>
            {/each}
          </div>
          <label>Custom color</label>
          <input type="color" bind:value={color} oninput={() => (terrain = '')} />
        {/if}
        <label>Size · {brushSize}  <span class="muted">( [ and ] )</span></label>
        <input type="range" min="4" max="400" bind:value={brushSize} />
        <label>Softness · {Math.round((1 - hardness) * 100)}%</label>
        <input type="range" min="0" max="0.95" step="0.05" value={1 - hardness} oninput={(e) => (hardness = 1 - +e.currentTarget.value)} />
        {#if tool === 'brush'}
          <label>Strength · {Math.round(opacity * 100)}%</label>
          <input type="range" min="0.1" max="1" step="0.05" bind:value={opacity} />
        {/if}
      {:else if tool === 'line'}
        <h3>Lines</h3>
        <div class="seg">
          {#each Object.entries(LINE_TYPES) as [id, t]}
            <button class:active={lineType === id} onclick={() => { lineType = id; lineWidth = Math.round(t.width * 1.5 * unit); }}>{t.name}</button>
          {/each}
        </div>
        <label>Width · {lineWidth}</label>
        <input type="range" min="1" max="40" bind:value={lineWidth} />
        {@render modeToggle()}
      {:else if tool === 'stamp'}
        <h3>Stamps</h3>
        <div class="icons">
          {#each ICONS as n}
            <button class:on={icon === n} title={n} onclick={() => (icon = n)}><img src={iconThumbs[n]} alt={n} /></button>
          {/each}
        </div>
        <label>Size · {iconSize}  <span class="muted">( [ and ] )</span></label>
        <input type="range" min="8" max="300" bind:value={iconSize} />
        <label class="check"><input type="checkbox" bind:checked={scatter} /> Scatter: drag to drop many</label>
      {:else if tool === 'label'}
        <h3>Text label</h3>
        <p class="muted">Click the map to place a label. Then type its text on the right.</p>
        <label>Font</label>
        <select bind:value={fontIdx}>{#each FONTS as f, i}<option value={i}>{f.name}</option>{/each}</select>
        <label>Size · {labelSize}</label>
        <input type="range" min="10" max="200" bind:value={labelSize} />
        <label>Color</label>
        <input type="color" bind:value={labelColor} />
        <label class="check"><input type="checkbox" bind:checked={halo} /> Light outline</label>
      {/if}
    </div>
  </aside>

  <div class="stage" bind:clientWidth={cw} bind:clientHeight={ch}>
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <canvas
      bind:this={canvas}
      style:cursor={cursor}
      style:width="{cw}px" style:height="{ch}px"
      onpointerdown={pointerDown}
      onpointermove={pointerMove}
      onpointerup={pointerUp}
      onpointerleave={() => { hover = null; requestDraw(); }}
      ondblclick={() => { if (clickPts) { clickPts.pop(); finishClick(); } }}
      oncontextmenu={(e) => e.preventDefault()}
      onwheel={wheel}
    ></canvas>
    {#if clickPts}<div class="hint">Click to add points. Double-click or press Enter to finish. Esc cancels.</div>{/if}
  </div>

  <aside class="right">
    <h3>Layers</h3>
    <div class="layers">
      {#each layersView as l (l.id)}
        <div class="layer" class:on={l.id === activeId}>
          <button class="small" title={l.visible ? 'Hide' : 'Show'} onclick={() => toggleLayer(l.id, 'visible')}>{l.visible ? '👁' : '—'}</button>
          <button class="small" title={l.locked ? 'Unlock' : 'Lock'} onclick={() => toggleLayer(l.id, 'locked')}>{l.locked ? '🔒' : '🔓'}</button>
          <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
          <span class="name" onclick={() => (activeId = l.id)}>
            <input value={l.name} oninput={(e) => renameLayer(l.id, e.currentTarget.value)} onfocus={() => (activeId = l.id)} />
            <small class="muted">{LAYER_NAMES[l.type]} · {l.count}</small>
          </span>
          <button class="small" title="Move up" onclick={() => moveLayer(l.id, 1)}>↑</button>
          <button class="small" title="Move down" onclick={() => moveLayer(l.id, -1)}>↓</button>
          <ConfirmButton label="✕" onconfirm={() => deleteLayer(l.id)} />
        </div>
      {/each}
    </div>
    <div class="row" style="margin-top:.4rem">
      <select bind:value={newType} style="flex:1">
        {#each Object.entries(LAYER_NAMES) as [t, n]}<option value={t}>{n}</option>{/each}
      </select>
      <button class="small" onclick={() => addLayer(newType)}>+ Layer</button>
    </div>

    {#if selView}
      <h3 style="margin-top:1.2rem">Selected {selView.k}</h3>
      {#if selView.k === 'label'}
        <label>Text</label>
        <input bind:this={labelInput} value={selView.text} oninput={(e) => setProp('text', e.currentTarget.value)} />
        <label>Font</label>
        <select value={selView.font} onchange={(e) => setProp('font', e.currentTarget.value)}>
          {#each FONTS as f}<option value={f.css}>{f.name}</option>{/each}
        </select>
        <label>Size · {selView.size}</label>
        <input type="range" min="10" max="240" value={selView.size} oninput={(e) => setProp('size', +e.currentTarget.value)} />
        <label>Rotation · {selView.rot}°</label>
        <input type="range" min="-90" max="90" value={selView.rot} oninput={(e) => setProp('rot', +e.currentTarget.value)} />
        <label>Letter spacing · {selView.spacing}</label>
        <input type="range" min="0" max="30" value={selView.spacing} oninput={(e) => setProp('spacing', +e.currentTarget.value)} />
        <label>Color</label>
        <input type="color" value={selView.color} oninput={(e) => setProp('color', e.currentTarget.value)} />
        <label class="check"><input type="checkbox" checked={selView.halo} onchange={(e) => setProp('halo', e.currentTarget.checked)} /> Light outline</label>
      {:else if selView.k === 'icon'}
        <label>Picture</label>
        <select value={selView.icon} onchange={(e) => setProp('icon', e.currentTarget.value)}>{#each ICONS as n}<option>{n}</option>{/each}</select>
        <label>Size · {selView.size}</label>
        <input type="range" min="8" max="400" value={selView.size} oninput={(e) => setProp('size', +e.currentTarget.value)} />
        <label class="check"><input type="checkbox" checked={selView.flip} onchange={(e) => setProp('flip', e.currentTarget.checked)} /> Flip</label>
      {:else if selView.k === 'line'}
        <label>Type</label>
        <select value={selView.type} onchange={(e) => setProp('type', e.currentTarget.value)}>
          {#each Object.entries(LINE_TYPES) as [id, t]}<option value={id}>{t.name}</option>{/each}
        </select>
        <label>Width · {selView.width}</label>
        <input type="range" min="1" max="40" value={selView.width} oninput={(e) => setProp('width', +e.currentTarget.value)} />
      {:else if selView.k === 'land'}
        <p class="muted">Drag the squares to reshape the coast. Land color comes from Map style below.</p>
      {/if}
      <div class="row" style="margin-top:.6rem">
        <button class="small" onclick={duplicateSelected}>Duplicate</button>
        <button class="small danger" onclick={deleteSelected}>Delete</button>
      </div>
    {/if}

    <h3 style="margin-top:1.2rem">Map style</h3>
    <div class="seg">
      {#each Object.entries(STYLES) as [id, s]}
        <button onclick={() => { setStyle('water', s.water); setStyle('land', s.land); setStyle('ink', s.ink); }}>{s.name}</button>
      {/each}
    </div>
    <div class="row three">
      <div><label>Water</label><input type="color" value={style.water} oninput={(e) => setStyle('water', e.currentTarget.value)} /></div>
      <div><label>Land</label><input type="color" value={style.land} oninput={(e) => setStyle('land', e.currentTarget.value)} /></div>
      <div><label>Ink</label><input type="color" value={style.ink} oninput={(e) => setStyle('ink', e.currentTarget.value)} /></div>
    </div>
    <p class="muted" style="font-size:.78rem;margin-top:.8rem">
      {doc.w} × {doc.h}. Changes save by themselves. Pins and character paths go on this map in the Maps screen.
    </p>
  </aside>
</div>

{#snippet modeToggle()}
  <label>Drawing style</label>
  <div class="seg">
    <button class:active={pathMode === 'free'} onclick={() => (pathMode = 'free')}>Freehand</button>
    <button class:active={pathMode === 'click'} onclick={() => (pathMode = 'click')}>Click points</button>
  </div>
{/snippet}

<style>
  .editor { display: grid; grid-template-columns: 210px minmax(0, 1fr) 270px; grid-template-rows: auto auto minmax(0, 1fr); height: 100%; min-height: 0; }
  header { grid-column: 1 / -1; display: flex; align-items: center; gap: .5rem; padding: .5rem .8rem; background: var(--panel); border-bottom: 1px solid var(--line); }
  .grow { flex: 1; }
  .notice { color: var(--accent); font-size: .85rem; }
  .exportmsg { grid-column: 1 / -1; padding: .3rem .8rem; font-size: .8rem; color: var(--ok); background: var(--panel); border-bottom: 1px solid var(--line); }
  .editor > :global(.stage), .stage { grid-row: 3; }
  aside { grid-row: 3; background: var(--panel); padding: .8rem; overflow: auto; min-height: 0; }
  .left { border-right: 1px solid var(--line); }
  .right { border-left: 1px solid var(--line); }
  .tools { display: grid; grid-template-columns: repeat(4, 1fr); gap: .3rem; }
  .tools button { padding: .4rem 0; font-size: 1.05rem; }
  .opts { margin-top: .8rem; }
  .stage { position: relative; min-width: 0; min-height: 0; overflow: hidden; background: var(--stage); user-select: none; -webkit-user-select: none; }
  canvas { position: absolute; inset: 0; touch-action: none; display: block; }
  .hint { position: absolute; bottom: .8rem; left: 50%; transform: translateX(-50%); background: var(--panel); border: 1px solid var(--line); border-radius: 8px; padding: .3rem .8rem; font-size: .8rem; pointer-events: none; }
  .swatches { display: grid; grid-template-columns: repeat(5, 1fr); gap: .3rem; }
  .swatch { aspect-ratio: 1; padding: 0; border-radius: 50%; border: 2px solid var(--line); }
  .swatch.on { border-color: var(--text); }
  .seg { display: flex; flex-wrap: wrap; gap: .25rem; }
  .seg button { padding: .2rem .5rem; font-size: .8rem; }
  .icons { display: grid; grid-template-columns: repeat(4, 1fr); gap: .25rem; }
  .icons button { padding: .15rem; background: #e9dfc6; }
  .icons button.on { outline: 2px solid var(--accent); }
  .icons img { width: 100%; display: block; }
  .check { display: flex; gap: .4rem; align-items: center; color: var(--text); }
  .check input { width: auto; }
  input[type='range'] { padding: 0; }
  .layers { display: flex; flex-direction: column; gap: .25rem; }
  .layer { display: flex; align-items: center; gap: .2rem; padding: .2rem; border: 1px solid transparent; border-radius: 6px; }
  .layer.on { background: var(--panel-2); border-color: var(--line); }
  .layer .name { flex: 1; min-width: 0; display: flex; flex-direction: column; }
  .layer .name input { padding: .1rem .3rem; font-size: .85rem; background: transparent; border-color: transparent; }
  .layer .name input:focus { background: var(--bg); }
  .layer small { padding-left: .3rem; font-size: .7rem; }
  .row.three { display: grid; grid-template-columns: repeat(3, 1fr); gap: .4rem; }
  h3 { margin-top: 0; }
</style>
