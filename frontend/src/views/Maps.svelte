<script lang="ts">
  import { db, PLACE_TYPES, CHARACTER_COLORS } from '../lib/db.svelte';
  import { center, inside, nearestEdge, type Pt } from '../lib/regions';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import { nav } from '../lib/nav.svelte';
  import DateField from '../lib/DateField.svelte';
  import { scenesOf } from '../lib/scenes';
  import MapEditor from '../lib/mapdraw/MapEditor.svelte';
  import { calendar, timeText } from '../lib/calendar';
  import { DEFAULT_MODES, distance, fmtDays, fmtDist, scaleOf, storyDays } from '../lib/travel';
  import { rangeOn, statusAt, tracks, whereAt } from '../lib/tracks';

  let mapId = $state<number | null>(null);
  let mode = $state<'pan' | 'pin' | 'path' | 'measure' | 'region'>('pan');
  let pathChar = $state<number | null>(null);
  let hiddenChars = $state<number[]>([]);
  let selPin = $state<number | null>(null);
  let selPoint = $state<number | null>(null);
  let selRegion = $state<number | null>(null);
  let selCorner = $state<number | null>(null);
  let draft = $state<Pt[]>([]); // corners of a region being drawn
  let showRegions = $state(true);
  // Open a path step when the Checks tab sends you here.
  $effect(() => {
    const p = nav.pending;
    if (p?.kind !== 'pathpoint') return;
    nav.pending = null;
    const step = db.get('pathpoint', p.id);
    if (step) { mapId = step.map; selPoint = step.id; selPin = null; }
  });
  let placing = $state<number | null>(null); // existing place waiting for a click on the map
  let width = $state(800);
  let height = $state(600);
  let view = $state({ tx: 0, ty: 0, k: 1 });
  let svg = $state<SVGSVGElement>();

  const map = $derived(db.get('map', mapId));
  const W = $derived(map?.w ?? 1600);
  const H = $derived(map?.h ?? 1000);
  const pins = $derived(db.lists.location.filter((l) => l.map === mapId && l.x != null));
  const unplaced = $derived(db.lists.location.filter((l) => !l.map));
  const pin = $derived(db.get('location', selPin));
  const point = $derived(db.get('pathpoint', selPoint));
  const regions = $derived(db.lists.region.filter((r) => r.map === mapId));
  const region = $derived(db.get('region', selRegion));
  const placesIn = (r: any) => pins.filter((l) => inside(r.points, l.x, l.y));
  const regionsOf = (l: any) => regions.filter((r) => inside(r.points, l.x, l.y));
  $effect(() => { mapId; draft = []; selRegion = null; selCorner = null; });
  const paths = $derived(
    db.lists.character
      .filter((c) => !hiddenChars.includes(c.id))
      .map((c) => ({
        c,
        pts: db.lists.pathpoint.filter((p) => p.character === c.id && p.map === mapId).sort((a, b) => a.order - b.order),
      }))
      .filter((p) => p.pts.length > 0),
  );
  const myPoints = $derived(
    db.lists.pathpoint.filter((p) => p.character === pathChar && p.map === mapId).sort((a, b) => a.order - b.order),
  );
  const chapters = $derived(
    [...db.lists.chapter].sort((a, b) => (a.book - b.book) || (a.order - b.order)),
  );

  // ---- travel time ----
  const scale = $derived(scaleOf(map));
  let modeIdx = $state(0);
  const mode0 = $derived(scale?.modes[Math.min(modeIdx, (scale?.modes.length ?? 1) - 1)]);
  type Spot = { x: number; y: number; label: string };
  let ruler = $state<{ a?: Spot; b?: Spot }>({});
  const rulerDist = $derived(map && scale && ruler.a && ruler.b ? distance(map, ruler.a, ruler.b) : null);

  function legText(a: any, b: any): string {
    if (!map || !scale) return '';
    const d = distance(map, a, b);
    let t = fmtDist(d, scale.unit);
    if (mode0) t += ' · ' + fmtDays(d / mode0.perDay) + ' ' + mode0.name.toLowerCase();
    if (a.time != null && b.time != null && a.time !== '' && b.time !== '') t += ' · story gives ' + fmtDays(storyDays(+a.time, +b.time));
    return t;
  }
  const pathTotal = $derived.by(() => {
    if (!map || !scale || myPoints.length < 2) return null;
    let d = 0;
    for (let i = 0; i + 1 < myPoints.length; i++) d += distance(map, myPoints[i] as any, myPoints[i + 1] as any);
    return d;
  });
  // the default ways to travel become real rows, so the writer can see and change them
  $effect(() => {
    if (map && scale && !map.modes) { map.modes = DEFAULT_MODES.map((m) => ({ ...m })); saveMap(); }
  });
  function addMode() { if (map) { map.modes = [...(map.modes ?? []), { name: 'New way', perDay: 30 }]; saveMap(); } }
  function dropMode(i: number) { if (map) { map.modes = map.modes.filter((_: unknown, k: number) => k !== i); saveMap(); } }

  // ---- who is where: a slider over story time ----
  const allTracks = $derived(tracks());
  const shownChars = $derived(db.lists.character.filter((c) => !hiddenChars.includes(c.id)));
  const range = $derived(mapId ? rangeOn(allTracks, mapId, shownChars.map((c) => c.id)) : null);
  let T = $state<number | null>(null);
  let playing = $state(false);
  let playSpeed = $state(1);
  let showLast = $state(true);
  const Tc = $derived(range ? Math.min(range.max, Math.max(range.min, T ?? range.min)) : 0);
  $effect(() => { if (!range) { playing = false; T = null; } });
  $effect(() => {
    if (!playing || !range) return;
    const r = range;
    if ((T ?? r.min) >= r.max) T = r.min;
    const step = (r.max - r.min) / (600 / playSpeed); // about 30 seconds from end to end at speed 1
    const id = setInterval(() => {
      T = Math.min(r.max, (T ?? r.min) + step);
      if (T >= r.max) playing = false;
    }, 50);
    return () => clearInterval(id);
  });
  const markers = $derived(
    range && mapId
      ? shownChars.flatMap((c) => {
          const stops = allTracks.get(c.id) ?? [];
          const w = whereAt(stops, Tc, mapId!);
          if (!w || (w.state === 'last' && !showLast)) return [];
          const trail = stops.filter((s) => s.map === mapId && s.time <= Tc + 1e-9).map((s) => ({ x: s.x, y: s.y }));
          if (w.state === 'moving') trail.push({ x: w.x, y: w.y });
          return [{ c, w, trail }];
        })
      : [],
  );
  const whoList = $derived(
    range ? shownChars.flatMap((c) => { const st = statusAt(allTracks.get(c.id) ?? [], Tc); return st ? [{ c, ...st }] : []; }) : [],
  );
  const timeLabel = (t: number) => (calendar() ? timeText(t) : String(Math.round(t * 1000) / 1000));

  function fit() {
    const k = Math.min(width / W, height / H) * 0.95;
    view = { k, tx: (width - W * k) / 2, ty: (height - H * k) / 2 };
  }
  $effect(() => { mapId; width; height; fit(); });

  // ---- maps ----
  function readFile(file: File): Promise<string> {
    return new Promise((res, rej) => {
      const r = new FileReader();
      r.onload = () => res(r.result as string);
      r.onerror = () => rej(r.error);
      r.readAsDataURL(file);
    });
  }
  async function upload(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    const image = await readFile(file);
    const img = new Image();
    await new Promise((r) => { img.onload = r; img.src = image; });
    const m = await db.save('map', { name: file.name.replace(/\.[^.]+$/, ''), image, w: img.naturalWidth, h: img.naturalHeight, bg: '#2a3a2f', notes: '' });
    mapId = m.id; input.value = '';
  }
  let editing = $state(false);
  let drawSize = $state('2400x1500');
  async function drawNew() {
    const [w, h] = drawSize.split('x').map(Number);
    const m = await db.save('map', { name: 'New drawn map', image: '', w, h, bg: '#a8cbd8', notes: '', drawn: true, style: 'classic' });
    mapId = m.id; selPin = selPoint = null; editing = true;
  }
  // draw on top of an uploaded picture: the picture is kept as it was, under the drawing
  async function drawOnImage() {
    if (!map) return;
    await db.save('map', { ...$state.snapshot(map), original: map.original || map.image, drawn: true });
    selPin = selPoint = null; editing = true;
  }
  async function restoreOriginal() {
    if (!map?.original) return;
    const row = db.lists.mapdoc.find((d) => d.map === map.id);
    await db.save('map', { ...$state.snapshot(map), image: map.original, original: null, drawn: false });
    if (row) await db.remove('mapdoc', row.id, false);
  }
  async function blank() {
    const m = await db.save('map', { name: 'New map', image: '', w: 1600, h: 1000, bg: '#2a3a2f', notes: '' });
    mapId = m.id;
  }
  function saveMap() { if (map) db.queue('map', map); }
  async function delMap() {
    if (!mapId) return;
    await db.remove('map', mapId); mapId = null;
  }

  // ---- pointer handling: one handler for pan, drag and click ----
  function toWorld(e: PointerEvent) {
    const b = svg!.getBoundingClientRect();
    return { x: (e.clientX - b.left - view.tx) / view.k, y: (e.clientY - b.top - view.ty) / view.k };
  }
  let drag: null | { kind: string; id: number; idx: number; sx: number; sy: number; tx: number; ty: number; moved: boolean } = null;

  function down(e: PointerEvent) {
    e.preventDefault(); // stops text selection while dragging
    if (!map) return;
    svg!.setPointerCapture(e.pointerId);
    const el = (e.target as Element).closest('[data-kind]') as HTMLElement | null;
    let kind = el?.dataset.kind ?? 'bg';
    // a region only reacts in Move mode; in other modes a click goes through to the map
    if (kind === 'region' && mode !== 'pan') kind = 'bg';
    if ((kind === 'corner' || kind === 'edge') && mode !== 'pan') kind = 'bg';
    drag = {
      kind, id: +(el?.dataset.id ?? 0), idx: +(el?.dataset.idx ?? 0),
      sx: e.clientX, sy: e.clientY, tx: view.tx, ty: view.ty, moved: false,
    };
  }
  function move(e: PointerEvent) {
    if (!drag) return;
    if (!drag.moved && Math.hypot(e.clientX - drag.sx, e.clientY - drag.sy) < 4) return;
    drag.moved = true;
    if (drag.kind === 'bg' || drag.kind === 'region' || drag.kind === 'edge') {
      view.tx = drag.tx + e.clientX - drag.sx; view.ty = drag.ty + e.clientY - drag.sy;
      return;
    }
    if (drag.kind === 'corner') {
      const r = db.get('region', drag.id);
      if (!r) return;
      const p = toWorld(e);
      r.points[drag.idx] = { x: Math.min(1, Math.max(0, p.x / W)), y: Math.min(1, Math.max(0, p.y / H)) };
      return;
    }
    const item = drag.kind === 'pin' ? db.get('location', drag.id) : db.get('pathpoint', drag.id);
    if (!item) return;
    const p = toWorld(e);
    item.x = Math.min(1, Math.max(0, p.x / W)); item.y = Math.min(1, Math.max(0, p.y / H));
  }
  async function up(e: PointerEvent) {
    if (!drag) return;
    const d = drag; drag = null;
    if (d.moved) {
      if (d.kind === 'pin') { const l = db.get('location', d.id); if (l) await db.save('location', l); }
      if (d.kind === 'point') { const p = db.get('pathpoint', d.id); if (p) await db.save('pathpoint', p); }
      if (d.kind === 'corner') { const r = db.get('region', d.id); if (r) await db.save('region', r); }
      return;
    }
    if (mode === 'measure') {
      let spot: Spot | null = null;
      if (d.kind === 'pin') { const l = db.get('location', d.id); if (l) spot = { x: l.x, y: l.y, label: l.name }; }
      else if (d.kind === 'bg') {
        const p = toWorld(e);
        if (p.x >= 0 && p.y >= 0 && p.x <= W && p.y <= H) spot = { x: p.x / W, y: p.y / H, label: '' };
      }
      if (spot) ruler = !ruler.a || ruler.b ? { a: spot } : { a: ruler.a, b: spot };
      return;
    }
    if (mode === 'region') {
      if (d.kind === 'pin') return;
      const p = toWorld(e);
      if (p.x < 0 || p.y < 0 || p.x > W || p.y > H) return;
      const pt = { x: p.x / W, y: p.y / H };
      // a click near the first corner closes the shape
      if (draft.length >= 3 && Math.hypot((pt.x - draft[0].x) * W, (pt.y - draft[0].y) * H) * view.k < 12) await finishRegion();
      else draft = [...draft, pt];
      return;
    }
    if (d.kind === 'region') { selRegion = d.id; selCorner = null; selPin = selPoint = null; return; }
    if (d.kind === 'corner') { selCorner = d.idx; return; }
    if (d.kind === 'edge') {
      const r = db.get('region', d.id);
      if (!r) return;
      const p = toWorld(e);
      const at = nearestEdge(r.points, p.x / W, p.y / H) + 1;
      r.points = [...r.points.slice(0, at), { x: p.x / W, y: p.y / H }, ...r.points.slice(at)];
      selCorner = at;
      await db.save('region', r);
      return;
    }
    if (d.kind === 'pin') {
      const l = db.get('location', d.id)!;
      if (mode === 'path' && pathChar) await addPoint(l.x, l.y, l.id);
      else { selPin = l.id; selPoint = null; selRegion = null; }
    } else if (d.kind === 'point') {
      selPoint = d.id; selPin = null;
    } else {
      const p = toWorld(e);
      const x = Math.min(1, Math.max(0, p.x / W)), y = Math.min(1, Math.max(0, p.y / H));
      if (p.x < 0 || p.y < 0 || p.x > W || p.y > H) return;
      if (mode === 'pin') await addPin(x, y);
      else if (mode === 'path' && pathChar) await addPoint(x, y, null);
      else { selPin = null; selPoint = null; selRegion = null; }
    }
  }
  function wheel(e: WheelEvent) {
    const b = svg!.getBoundingClientRect();
    const wx = (e.clientX - b.left - view.tx) / view.k, wy = (e.clientY - b.top - view.ty) / view.k;
    view.k = Math.min(8, Math.max(0.05, view.k * (e.deltaY < 0 ? 1.15 : 1 / 1.15)));
    view.tx = e.clientX - b.left - wx * view.k; view.ty = e.clientY - b.top - wy * view.k;
  }

  // ---- regions ----
  async function finishRegion() {
    if (draft.length < 3) return;
    const r = await db.save('region', {
      map: mapId, name: 'New region', description: '', points: draft,
      color: CHARACTER_COLORS[db.lists.region.length % CHARACTER_COLORS.length],
    });
    draft = []; selRegion = r.id; selCorner = null; selPin = selPoint = null; mode = 'pan';
  }
  function saveRegion() { if (region) db.queue('region', region); }
  async function dropCorner() {
    if (!region || selCorner === null || region.points.length <= 3) return;
    region.points = region.points.filter((_: unknown, i: number) => i !== selCorner);
    selCorner = null;
    await db.save('region', region);
  }
  async function delRegion() { if (selRegion) { await db.remove('region', selRegion); selRegion = null; } }
  function keys(e: KeyboardEvent) {
    const t = e.target as HTMLElement;
    if (/^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName)) return;
    if (mode === 'region' && e.key === 'Enter') finishRegion();
    else if (mode === 'region' && e.key === 'Escape') { draft = []; mode = 'pan'; }
    else if (mode === 'region' && e.key === 'Backspace') draft = draft.slice(0, -1);
  }

  // ---- places ----
  async function addPin(x: number, y: number) {
    if (placing) {
      const l = db.get('location', placing)!;
      await db.save('location', { ...l, map: mapId, x, y });
      selPin = placing; placing = null; mode = 'pan';
    } else {
      const l = await db.save('location', { name: 'New place', type: 'city', description: '', parent: null, tags: '', map: mapId, x, y });
      selPin = l.id; selPoint = null;
    }
  }
  function savePin() { if (pin) db.queue('location', pin); }
  async function unpin() {
    if (pin) { await db.save('location', { ...pin, map: null, x: null, y: null }); selPin = null; }
  }
  async function delPin() { if (selPin) { await db.remove('location', selPin); selPin = null; } }

  // ---- paths ----
  async function addPoint(x: number, y: number, location: number | null) {
    const order = Math.max(0, ...db.lists.pathpoint.filter((p) => p.character === pathChar).map((p) => p.order)) + 1;
    const p = await db.save('pathpoint', { character: pathChar, map: mapId, x, y, order, label: '', chapter: null, location, when: '' });
    selPoint = p.id; selPin = null;
  }
  function savePoint() { if (point) db.queue('pathpoint', point); }
  async function movePoint(p: any, dir: -1 | 1) {
    const other = myPoints[myPoints.indexOf(p) + dir];
    if (!other) return;
    const a = p.order; p.order = other.order; other.order = a;
    await db.save('pathpoint', p); await db.save('pathpoint', other);
  }
  async function delPoint(id: number) { await db.remove('pathpoint', id); if (selPoint === id) selPoint = null; }
  function startPath(id: number) { pathChar = id; mode = 'path'; hiddenChars = hiddenChars.filter((h) => h !== id); }
  function toggleChar(id: number) {
    hiddenChars = hiddenChars.includes(id) ? hiddenChars.filter((h) => h !== id) : [...hiddenChars, id];
  }
  function polyline(pts: any[]) { return pts.map((p) => `${p.x * W},${p.y * H}`).join(' '); }
</script>

<svelte:window onkeydown={keys} />

{#if editing && map}
  <MapEditor {map} onclose={() => (editing = false)} />
{:else}
<div class="split wide">
  <aside class="side">
    <label style="margin-top:0">Map</label>
    <select bind:value={mapId} onchange={() => (selPin = selPoint = null)}>
      <option value={null}>— choose a map —</option>
      {#each db.lists.map as m}<option value={m.id}>{m.name}</option>{/each}
    </select>
    <div class="row" style="margin-top:.5rem">
      <label class="fit" style="margin:0"><span class="small primary" style="display:inline-block;padding:.35rem .75rem;border-radius:6px;background:var(--accent);color:var(--on-accent);font-weight:600;cursor:pointer">+ Image</span>
        <input type="file" accept="image/*" onchange={upload} style="display:none" /></label>
      <button onclick={blank}>+ Blank</button>
    </div>
    <div class="row" style="margin-top:.4rem">
      <button class="primary" onclick={drawNew}>✎ Draw a map</button>
      <select bind:value={drawSize} style="flex:0 0 6.5rem" title="Size of the new map">
        <option value="1600x1000">Small</option><option value="2400x1500">Medium</option><option value="3200x2000">Large</option>
      </select>
    </div>

    {#if map}
      <label>Name</label><input bind:value={map.name} oninput={saveMap} />
      {#if map.drawn}
        <button class="primary" style="margin-top:.6rem;width:100%" onclick={() => (editing = true)}>✎ Edit drawing</button>
        {#if map.original}
          <div style="margin-top:.4rem"><ConfirmButton label="Remove drawing, keep picture" onconfirm={restoreOriginal} /></div>
        {/if}
      {:else if map.image}
        <button class="primary" style="margin-top:.6rem;width:100%" onclick={drawOnImage}>✎ Draw on this picture</button>
      {/if}
      {#if !map.image && !map.drawn}
        <div class="row"><div><label>Width</label><input type="number" bind:value={map.w} oninput={saveMap} /></div>
          <div><label>Height</label><input type="number" bind:value={map.h} oninput={saveMap} /></div></div>
        <label>Background</label><input type="color" bind:value={map.bg} oninput={saveMap} />
      {/if}
      <label>Notes</label><textarea bind:value={map.notes} oninput={saveMap}></textarea>
      <div style="margin-top:.5rem"><ConfirmButton label="Delete map" onconfirm={delMap} /></div>

      <h3 style="margin-top:1.2rem">Scale and travel</h3>
      <label for="scaledist">The whole map is this wide</label>
      <div class="row">
        <input id="scaledist" type="number" min="0" step="any" bind:value={map.scaleDist} oninput={saveMap} placeholder="e.g. 500" />
        <input list="units" bind:value={map.scaleUnit} oninput={saveMap} placeholder="km" style="flex:0 0 5.5rem" aria-label="Unit" />
      </div>
      <datalist id="units"><option value="km"></option><option value="miles"></option><option value="leagues"></option></datalist>
      {#if scale}
        <label>Ways to travel (how far in one day)</label>
        {#each map.modes ?? [] as m, i (i)}
          <div class="row" style="margin-bottom:.2rem">
            <input bind:value={m.name} oninput={saveMap} aria-label="Name" />
            <input type="number" min="0" step="any" bind:value={m.perDay} oninput={saveMap} style="flex:0 0 4.5rem" aria-label="Distance per day" />
            <button class="small danger" title="Remove" onclick={() => dropMode(i)}>✕</button>
          </div>
        {/each}
        <button class="small" onclick={addMode}>+ Way to travel</button>
        <p class="muted" style="font-size:.8rem;margin:.4rem 0 0">Use the 📏 Measure tool to read a distance. Distances are straight lines.</p>
      {:else}
        <p class="muted" style="font-size:.8rem;margin:.4rem 0 0">Type how wide the map is, to measure distances and travel time.</p>
      {/if}

      <h3 style="margin-top:1.2rem">Character paths</h3>
      <p class="muted" style="font-size:.8rem;margin:0 0 .4rem">“Draw” picks a person and switches to path mode. Click the map to add steps. Click a place pin to snap to it.</p>
      {#each db.lists.character as c (c.id)}
        <div class="row" style="margin-bottom:.2rem">
          <input type="checkbox" style="width:auto;flex:0" checked={!hiddenChars.includes(c.id)} onchange={() => toggleChar(c.id)} />
          <span class="dot fit" style:background={c.color}></span>
          <span>{c.name}</span>
          <button class="small fit" class:active={pathChar === c.id && mode === 'path'} onclick={() => startPath(c.id)}>Draw</button>
        </div>
      {:else}<div class="muted">Add characters first.</div>{/each}
    {/if}
  </aside>

  <div class="stage" bind:clientWidth={width} bind:clientHeight={height}>
    {#if map}
      <div class="toolbar">
        <button class:active={mode === 'pan'} onclick={() => { mode = 'pan'; placing = null; }}>✋ Move</button>
        <button class:active={mode === 'pin'} onclick={() => (mode = 'pin')}>📍 Add place</button>
        <button class:active={mode === 'path'} disabled={!pathChar} onclick={() => (mode = 'path')}>
          〰 Path{pathChar ? ': ' + db.name('character', pathChar) : ''}
        </button>
        <button class:active={mode === 'measure'} onclick={() => { mode = 'measure'; placing = null; ruler = {}; }}>📏 Measure</button>
        <button class:active={mode === 'region'} onclick={() => { mode = 'region'; placing = null; draft = []; selRegion = null; }}>⬠ Region</button>
        <button onclick={fit}>Fit</button>
        {#if regions.length}<label class="chip" style="margin:0;display:flex;align-items:center;gap:.3rem"><input type="checkbox" style="width:auto" bind:checked={showRegions} />Regions</label>{/if}
        {#if mode === 'region'}
          <span class="chip">{draft.length < 3 ? 'Click the map to place corners. ' + draft.length + ' so far.' : draft.length + ' corners. Click the first one, or press Finish.'}</span>
          <button class="small primary" disabled={draft.length < 3} onclick={finishRegion}>Finish</button>
          <button class="small" disabled={!draft.length} onclick={() => (draft = draft.slice(0, -1))}>Undo corner</button>
          <button class="small" onclick={() => { draft = []; mode = 'pan'; }}>Cancel</button>
        {/if}
        {#if mode === 'measure'}
          <span class="chip">
            {#if !scale}Set the map width in the left side bar first.
            {:else if !ruler.a}Click a point or a pin.
            {:else if !ruler.b}Now click the second point.
            {:else if rulerDist !== null}{fmtDist(rulerDist, scale.unit)}{#each scale.modes as m} · {m.name} {fmtDays(rulerDist / m.perDay)}{/each}{/if}
          </span>
        {/if}
        {#if placing}<span class="chip">Click the map to place “{db.name('location', placing)}”</span>{/if}
      </div>
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <svg bind:this={svg} onpointerdown={down} onpointermove={move} onpointerup={up} onwheel={wheel}
        style:cursor={mode === 'pan' ? 'grab' : 'crosshair'}>
        <defs>
          <marker id="arr" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="4" markerHeight="4" orient="auto">
            <path d="M0,0 L10,5 L0,10 z" fill="context-stroke" />
          </marker>
        </defs>
        <g transform="translate({view.tx},{view.ty}) scale({view.k})">
          {#if map.image}
            <image href={map.image} width={W} height={H}  style="-webkit-user-drag:none" />
          {:else}
            <rect width={W} height={H} fill={map.bg} />
          {/if}
          <rect width={W} height={H} fill="none" stroke="#000" stroke-opacity=".5" stroke-width={2 / view.k} />

          {#if showRegions}
            {#each regions as r (r.id)}
              {@const on = r.id === selRegion}
              <polygon data-kind="region" data-id={r.id} points={polyline(r.points)} fill={r.color} fill-opacity={on ? 0.32 : 0.2}
                stroke={r.color} stroke-width={(on ? 3.5 : 2.5) / view.k} stroke-linejoin="round" style="cursor:pointer" />
              {#if on}
                <polygon data-kind="edge" data-id={r.id} points={polyline(r.points)} fill="none" stroke="transparent" stroke-width={14 / view.k} stroke-linejoin="round" style="cursor:copy" />
              {/if}
              {#if r.name}
                <text x={center(r.points).x * W} y={center(r.points).y * H} text-anchor="middle" font-size={16 / view.k} font-weight="700" fill="#fff"
                  paint-order="stroke" stroke="#000" stroke-width={4 / view.k} letter-spacing={1 / view.k} pointer-events="none">{r.name.toUpperCase()}</text>
              {/if}
            {/each}
          {/if}

          {#each paths as { c, pts } (c.id)}
            <polyline points={polyline(pts)} fill="none" stroke={c.color} stroke-width={3.5 / view.k}
              stroke-linejoin="round" stroke-dasharray="{10 / view.k} {6 / view.k}"
              marker-mid="url(#arr)" marker-end="url(#arr)" opacity=".9" />
          {/each}

          {#each pins as l (l.id)}
            <g data-kind="pin" data-id={l.id} transform="translate({l.x * W},{l.y * H}) scale({1 / view.k})" style="cursor:pointer">
              <circle r="9" fill="#0b0f14" stroke={l.id === selPin ? '#fff' : '#2cd5e8'} stroke-width="3" />
              <circle r="3.5" fill="#2cd5e8" />
              <text y="-15" text-anchor="middle" font-size="13" font-weight="600" fill="#fff" paint-order="stroke"
                stroke="#000" stroke-width="3.5" pointer-events="none">{l.name}</text>
            </g>
          {/each}

          {#each paths as { c, pts } (c.id)}
            {#each pts as p, i (p.id)}
              <g data-kind="point" data-id={p.id} transform="translate({p.x * W},{p.y * H}) scale({1 / view.k})" style="cursor:pointer">
                <circle r={p.id === selPoint ? 11 : 9} fill={c.color} stroke={p.id === selPoint ? '#fff' : '#0b0f14'} stroke-width="2.5" />
                <text y="4" text-anchor="middle" font-size="11" font-weight="700" fill="#0b0f14" pointer-events="none">{i + 1}</text>
                {#if p.label && p.id === selPoint}
                  <text y="26" text-anchor="middle" font-size="12" fill="#fff" paint-order="stroke" stroke="#000"
                    stroke-width="3.5" pointer-events="none">{p.label}</text>
                {/if}
              </g>
            {/each}
          {/each}
          {#if region && showRegions && mode === 'pan'}
            {#each region.points as p, i (i)}
              <circle data-kind="corner" data-id={region.id} data-idx={i} cx={p.x * W} cy={p.y * H} r={(i === selCorner ? 8 : 6) / view.k}
                fill={i === selCorner ? '#fff' : region.color} stroke="#000" stroke-width={2 / view.k} style="cursor:move" />
            {/each}
          {/if}
          {#if draft.length}
            <polyline points={polyline(draft)} fill="none" stroke="#fff" stroke-width={3 / view.k} stroke-dasharray="{8 / view.k} {5 / view.k}" />
            {#each draft as p, i}
              <circle cx={p.x * W} cy={p.y * H} r={(i === 0 ? 8 : 5) / view.k} fill={i === 0 ? '#2cd5e8' : '#fff'} stroke="#000" stroke-width={2 / view.k} />
            {/each}
          {/if}
          {#if ruler.a}
            <circle cx={ruler.a.x * W} cy={ruler.a.y * H} r={6 / view.k} fill="#fff" stroke="#000" stroke-width={2 / view.k} />
            {#if ruler.b}
              <line x1={ruler.a.x * W} y1={ruler.a.y * H} x2={ruler.b.x * W} y2={ruler.b.y * H} stroke="#fff" stroke-width={3 / view.k} stroke-dasharray="{8 / view.k} {5 / view.k}" />
              <circle cx={ruler.b.x * W} cy={ruler.b.y * H} r={6 / view.k} fill="#fff" stroke="#000" stroke-width={2 / view.k} />
            {/if}
          {/if}

          {#each markers as { c, w, trail } (c.id)}
            {#if trail.length > 1}
              <polyline points={polyline(trail)} fill="none" stroke={c.color} stroke-width={6 / view.k} stroke-linecap="round" stroke-linejoin="round" opacity=".85" />
            {/if}
            <g transform="translate({w.x * W},{w.y * H}) scale({1 / view.k})" opacity={w.state === 'last' ? 0.5 : 1} pointer-events="none">
              <circle r="12" fill={c.color} stroke="#fff" stroke-width="3" />
              <text y="4.5" text-anchor="middle" font-size="12" font-weight="700" fill="#0b0f14">{(c.name || '?').charAt(0).toUpperCase()}</text>
              <text y="28" text-anchor="middle" font-size="13" font-weight="600" fill="#fff" paint-order="stroke" stroke="#000" stroke-width="3.5">{c.name}</text>
            </g>
          {/each}
        </g>
      </svg>
      {#if range}
        <div class="timebar">
          <button class="small" onclick={() => (playing = !playing)} title={playing ? 'Pause' : 'Play'}>{playing ? '⏸' : '▶'}</button>
          <input type="range" min={range.min} max={range.max} step={(range.max - range.min) / 1000 || 1} value={Tc}
            oninput={(e) => { playing = false; T = +e.currentTarget.value; }} aria-label="Story time" />
          <strong class="when">{timeLabel(Tc)}</strong>
          <select bind:value={playSpeed} title="Play speed" style="width:auto">
            <option value={0.5}>Slow</option><option value={1}>Normal</option><option value={2}>Fast</option><option value={4}>Very fast</option>
          </select>
          <label class="last"><input type="checkbox" bind:checked={showLast} />Show last seen</label>
        </div>
      {/if}
    {:else}
      <div class="empty" style="padding:2rem">Add a map image, or a blank map, to begin.</div>
    {/if}
  </div>

  <aside class="side right">
    {#if map}
      {#if range}
        <h3 style="margin:0 0 .4rem">Who is where · {timeLabel(Tc)}</h3>
        {#each whoList as w (w.c.id)}
          <button class="who" onclick={() => { if (w.map !== mapId) { mapId = w.map; selPin = selPoint = null; } }} title={w.map !== mapId ? 'Show this map' : ''}>
            <span class="dot" style:background={w.c.color}></span>
            <span><b>{w.c.name}</b> {w.text}{#if w.map !== mapId} <span class="muted">(on “{db.name('map', w.map)}”)</span>{/if}</span>
          </button>
        {/each}
        <hr style="border:0;border-top:1px solid var(--line);margin:1rem 0" />
      {/if}
      {#if pin}
        <div class="head"><h3 style="margin:0">Place</h3><span class="row fit" style="gap:.2rem">
          <button class="small" onclick={unpin}>Unpin</button><ConfirmButton onconfirm={delPin} /></span></div>
        <label>Name</label><input bind:value={pin.name} oninput={savePin} />
        <label>Type</label>
        <select bind:value={pin.type} onchange={savePin}>{#each PLACE_TYPES as t}<option>{t}</option>{/each}</select>
        <label>Description</label><textarea bind:value={pin.description} oninput={savePin}></textarea>
        {#if regionsOf(pin).length}<p class="muted" style="margin:.5rem 0 0">In: {#each regionsOf(pin) as r, i}{i ? ', ' : ''}<button class="small" onclick={() => { selRegion = r.id; selPin = null; }}>{r.name || 'Region'}</button>{/each}</p>{/if}
      {:else if region}
        <div class="head"><h3 style="margin:0">Region</h3><ConfirmButton onconfirm={delRegion} /></div>
        <label>Name</label><input bind:value={region.name} oninput={saveRegion} placeholder="e.g. Kingdom of Ash" />
        <label>Colour</label><input type="color" bind:value={region.color} oninput={saveRegion} />
        <label>Description</label><textarea bind:value={region.description} oninput={saveRegion}></textarea>
        <p class="muted" style="font-size:.8rem;margin:.5rem 0">Drag a corner to move it. Click the outline to add a corner. Click a corner, then press Remove corner.</p>
        <button class="small" disabled={selCorner === null || region.points.length <= 3} onclick={dropCorner}>Remove corner</button>
        <h3 style="margin-top:1rem">Places inside · {placesIn(region).length}</h3>
        {#each placesIn(region) as l (l.id)}
          <button class="who" onclick={() => { selPin = l.id; selRegion = null; }}><span>{l.name} <span class="muted">({l.type})</span></span></button>
        {:else}<div class="muted">No pins inside.</div>{/each}
      {:else if point}
        <div class="head"><h3 style="margin:0">Path step · {db.name('character', point.character)}</h3>
          <ConfirmButton onconfirm={() => delPoint(point.id)} /></div>
        <label>What happens here</label><input bind:value={point.label} oninput={savePoint} placeholder="e.g. Leaves the village" />
        <DateField bind:time={point.time} bind:when={point.when} onchange={savePoint} placeholder="" />
        <label>Chapter</label>
        <select bind:value={point.chapter} onchange={() => { point.scene = null; savePoint(); }}>
          <option value={null}>— none —</option>
          {#each chapters as c}<option value={c.id}>{db.name('book', c.book, 'title')} · {c.order}. {c.title}</option>{/each}
        </select>
        {#if point.chapter && scenesOf(point.chapter).length}
          <label>Scene</label>
          <select bind:value={point.scene} onchange={savePoint}>
            <option value={null}>— none —</option>
            {#each scenesOf(point.chapter) as s, i}<option value={s.id}>{i + 1}. {s.title || 'Untitled'}</option>{/each}
          </select>
        {/if}
        <label>Place</label>
        <select bind:value={point.location} onchange={savePoint}>
          <option value={null}>— none —</option>
          {#each db.lists.location as l}<option value={l.id}>{l.name}</option>{/each}
        </select>
      {:else}
        <p class="muted">Click a pin, a path step or a region to edit it.</p>
      {/if}

      {#if regions.length && !region}
        <h3 style="margin-top:1.4rem">Regions on this map</h3>
        {#each regions as r (r.id)}
          <button class="who" onclick={() => { selRegion = r.id; selPin = selPoint = null; }}>
            <span class="dot" style:background={r.color}></span><span>{r.name || 'Region'} <span class="muted">· {placesIn(r).length} place{placesIn(r).length === 1 ? '' : 's'}</span></span>
          </button>
        {/each}
      {/if}

      {#if pathChar}
        <h3 style="margin-top:1.4rem">{db.name('character', pathChar)}’s path on this map</h3>
        {#if scale && scale.modes.length}
          <label>Travel by</label>
          <select bind:value={modeIdx}>{#each scale.modes as m, i}<option value={i}>{m.name}</option>{/each}</select>
        {/if}
        <div class="list">
          {#each myPoints as p, i (p.id)}
            <div class="row" style="gap:.2rem">
              <button class="item" class:on={p.id === selPoint} style="flex:1" onclick={() => { selPoint = p.id; selPin = null; }}>
                {i + 1}. {p.label || (p.location ? db.name('location', p.location) : 'Step')}
              </button>
              <button class="small" onclick={() => movePoint(p, -1)}>↑</button>
              <button class="small" onclick={() => movePoint(p, 1)}>↓</button>
            </div>
            {#if scale && i < myPoints.length - 1}<div class="muted leg">↓ {legText(p, myPoints[i + 1])}</div>{/if}
          {:else}<div class="muted">No steps yet.</div>{/each}
        </div>
        {#if pathTotal !== null && scale}<p class="muted" style="margin:.4rem 0 0">Whole path: {fmtDist(pathTotal, scale.unit)}{#if mode0} · {fmtDays(pathTotal / mode0.perDay)} {mode0.name.toLowerCase()}{/if}</p>{/if}
      {/if}

      {#if unplaced.length}
        <h3 style="margin-top:1.4rem">Places not on a map</h3>
        <select onchange={(e) => { const v = +(e.target as HTMLSelectElement).value; if (v) { placing = v; mode = 'pin'; } (e.target as HTMLSelectElement).value = ''; }}>
          <option value="">Place an existing one…</option>
          {#each unplaced as l}<option value={l.id}>{l.name}</option>{/each}
        </select>
      {/if}
    {/if}
  </aside>
</div>
{/if}

<style>
  .timebar { position: absolute; left: .6rem; right: .6rem; bottom: .6rem; display: flex; align-items: center; gap: .6rem; padding: .4rem .6rem; background: var(--panel); border: 1px solid var(--line); border-radius: 8px; z-index: 2; }
  .timebar input[type='range'] { flex: 1; min-width: 6rem; padding: 0; border: 0; background: none; }
  .timebar .when { white-space: nowrap; min-width: 6rem; }
  .timebar .last { display: flex; align-items: center; gap: .3rem; margin: 0; white-space: nowrap; }
  .timebar .last input { width: auto; }
  .who { display: flex; gap: .5rem; align-items: flex-start; width: 100%; text-align: left; background: none; border: 1px solid transparent; padding: .25rem .3rem; }
  .who:hover { background: var(--panel-2); border-color: var(--line); }
  .who .dot { margin-top: .3rem; }
  .leg { font-size: .75rem; margin: 0 0 .2rem .6rem; }
</style>
