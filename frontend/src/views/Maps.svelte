<script lang="ts">
  import { db, PLACE_TYPES } from '../lib/db.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import { scenesOf } from '../lib/scenes';
  import MapEditor from '../lib/mapdraw/MapEditor.svelte';

  let mapId = $state<number | null>(null);
  let mode = $state<'pan' | 'pin' | 'path'>('pan');
  let pathChar = $state<number | null>(null);
  let hiddenChars = $state<number[]>([]);
  let selPin = $state<number | null>(null);
  let selPoint = $state<number | null>(null);
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
  let drag: null | { kind: string; id: number; sx: number; sy: number; tx: number; ty: number; moved: boolean } = null;

  function down(e: PointerEvent) {
    e.preventDefault(); // stops text selection while dragging
    if (!map) return;
    svg!.setPointerCapture(e.pointerId);
    const el = (e.target as Element).closest('[data-kind]') as HTMLElement | null;
    drag = {
      kind: el?.dataset.kind ?? 'bg', id: +(el?.dataset.id ?? 0),
      sx: e.clientX, sy: e.clientY, tx: view.tx, ty: view.ty, moved: false,
    };
  }
  function move(e: PointerEvent) {
    if (!drag) return;
    if (!drag.moved && Math.hypot(e.clientX - drag.sx, e.clientY - drag.sy) < 4) return;
    drag.moved = true;
    if (drag.kind === 'bg') {
      view.tx = drag.tx + e.clientX - drag.sx; view.ty = drag.ty + e.clientY - drag.sy;
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
      return;
    }
    if (d.kind === 'pin') {
      const l = db.get('location', d.id)!;
      if (mode === 'path' && pathChar) await addPoint(l.x, l.y, l.id);
      else { selPin = l.id; selPoint = null; }
    } else if (d.kind === 'point') {
      selPoint = d.id; selPin = null;
    } else {
      const p = toWorld(e);
      const x = Math.min(1, Math.max(0, p.x / W)), y = Math.min(1, Math.max(0, p.y / H));
      if (p.x < 0 || p.y < 0 || p.x > W || p.y > H) return;
      if (mode === 'pin') await addPin(x, y);
      else if (mode === 'path' && pathChar) await addPoint(x, y, null);
      else { selPin = null; selPoint = null; }
    }
  }
  function wheel(e: WheelEvent) {
    const b = svg!.getBoundingClientRect();
    const wx = (e.clientX - b.left - view.tx) / view.k, wy = (e.clientY - b.top - view.ty) / view.k;
    view.k = Math.min(8, Math.max(0.05, view.k * (e.deltaY < 0 ? 1.15 : 1 / 1.15)));
    view.tx = e.clientX - b.left - wx * view.k; view.ty = e.clientY - b.top - wy * view.k;
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
      {/if}
      {#if !map.image && !map.drawn}
        <div class="row"><div><label>Width</label><input type="number" bind:value={map.w} oninput={saveMap} /></div>
          <div><label>Height</label><input type="number" bind:value={map.h} oninput={saveMap} /></div></div>
        <label>Background</label><input type="color" bind:value={map.bg} oninput={saveMap} />
      {/if}
      <label>Notes</label><textarea bind:value={map.notes} oninput={saveMap}></textarea>
      <div style="margin-top:.5rem"><ConfirmButton label="Delete map" onconfirm={delMap} /></div>

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
        <button onclick={fit}>Fit</button>
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
        </g>
      </svg>
    {:else}
      <div class="empty" style="padding:2rem">Add a map image, or a blank map, to begin.</div>
    {/if}
  </div>

  <aside class="side right">
    {#if map}
      {#if pin}
        <div class="head"><h3 style="margin:0">Place</h3><span class="row fit" style="gap:.2rem">
          <button class="small" onclick={unpin}>Unpin</button><ConfirmButton onconfirm={delPin} /></span></div>
        <label>Name</label><input bind:value={pin.name} oninput={savePin} />
        <label>Type</label>
        <select bind:value={pin.type} onchange={savePin}>{#each PLACE_TYPES as t}<option>{t}</option>{/each}</select>
        <label>Description</label><textarea bind:value={pin.description} oninput={savePin}></textarea>
      {:else if point}
        <div class="head"><h3 style="margin:0">Path step · {db.name('character', point.character)}</h3>
          <ConfirmButton onconfirm={() => delPoint(point.id)} /></div>
        <label>What happens here</label><input bind:value={point.label} oninput={savePoint} placeholder="e.g. Leaves the village" />
        <label>When (story time)</label><input bind:value={point.when} oninput={savePoint} />
        <label>Timeline position (number, e.g. year)</label><input type="number" step="any" bind:value={point.time} oninput={savePoint} />
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
        <p class="muted">Click a pin or a path step to edit it.</p>
      {/if}

      {#if pathChar}
        <h3 style="margin-top:1.4rem">{db.name('character', pathChar)}’s path on this map</h3>
        <div class="list">
          {#each myPoints as p, i (p.id)}
            <div class="row" style="gap:.2rem">
              <button class="item" class:on={p.id === selPoint} style="flex:1" onclick={() => { selPoint = p.id; selPin = null; }}>
                {i + 1}. {p.label || (p.location ? db.name('location', p.location) : 'Step')}
              </button>
              <button class="small" onclick={() => movePoint(p, -1)}>↑</button>
              <button class="small" onclick={() => movePoint(p, 1)}>↓</button>
            </div>
          {:else}<div class="muted">No steps yet.</div>{/each}
        </div>
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
