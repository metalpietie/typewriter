<script lang="ts">
  import { forceSimulation, forceLink, forceManyBody, forceCenter, forceCollide } from 'd3-force';
  import { untrack } from 'svelte';
  import { db, REL_TYPES } from '../lib/db.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';

  let selRel = $state<number | null>(null);
  let focus = $state<number | null>(null);
  let hidden = $state<string[]>([]);
  let newA = $state<number | null>(null);
  let newB = $state<number | null>(null);
  let newType = $state('friend');

  let width = $state(800);
  let height = $state(600);
  let view = $state({ tx: 0, ty: 0, k: 1 });
  let svg = $state<SVGSVGElement>();

  const known = new Map<number, { x: number; y: number }>();
  let pos = $state<Record<number, { x: number; y: number }>>({});
  let sim: any = null;
  let nodes: any[] = [];

  const rel = $derived(db.get('relationship', selRel));
  const visible = $derived(
    db.lists.relationship.filter(
      (r) => !hidden.includes(r.type) && db.get('character', r.a) && db.get('character', r.b),
    ),
  );

  const NODE_H = 34;
  const nodeW = (c: any) => Math.max(80, (c?.name?.length ?? 4) * 7.5 + 30);
  let layoutRun = $state(0);

  // Build the layout when people or links change. Saved positions stay fixed, so the graph is calm and stays put.
  $effect(() => {
    layoutRun;
    const ids = db.lists.character.map((c) => c.id);
    const links = visible.map((r) => ({ source: r.a as number, target: r.b as number }));
    untrack(() => {
      sim?.stop();
      nodes = ids.map((id) => {
        const c = db.get('character', id);
        const at = known.get(id) ?? (c?.gx != null ? { x: c.gx, y: c.gy } : null);
        if (c?.gx != null) return { id, x: c.gx, y: c.gy, fx: c.gx, fy: c.gy };
        return { id, ...(at ?? {}) };
      });
      sim = forceSimulation(nodes)
        .force('link', forceLink(links).id((n: any) => n.id).distance(190).strength(0.25))
        .force('charge', forceManyBody().strength(-420))
        .force('center', forceCenter(0, 0).strength(0.05))
        .force('collide', forceCollide(62))
        .alphaDecay(0.08)
        .velocityDecay(0.65)
        .on('tick', () => {
          const next: Record<number, { x: number; y: number }> = {};
          for (const n of nodes) { next[n.id] = { x: n.x, y: n.y }; known.set(n.id, next[n.id]); }
          pos = next;
        })
        .on('end', () => {
          // Save the settled spot of every new node so it never moves again.
          for (const n of nodes) {
            const c = db.get('character', n.id);
            if (c && c.gx == null && n.x != null) db.save('character', { ...c, gx: Math.round(n.x), gy: Math.round(n.y) });
          }
        });
    });
    return () => sim?.stop();
  });

  async function relayout() {
    known.clear();
    for (const c of db.lists.character) await db.save('character', { ...c, gx: null, gy: null });
    layoutRun++;
  }

  $effect(() => { view.tx = width / 2; view.ty = height / 2; });

  // Links leave and enter the sides of the boxes with horizontal handles, like a node editor.
  // Links between the same two people leave from slightly different heights.
  const edges = $derived.by(() => {
    const seen = new Map<string, number>();
    const total = new Map<string, number>();
    const key = (r: any) => [r.a, r.b].sort().join('-');
    for (const r of visible) total.set(key(r), (total.get(key(r)) ?? 0) + 1);
    return visible.flatMap((r) => {
      const p1 = pos[r.a], p2 = pos[r.b];
      if (!p1 || !p2) return [];
      const k = key(r);
      const i = seen.get(k) ?? 0; seen.set(k, i + 1);
      const n = total.get(k)!;
      const spread = (i - (n - 1) / 2) * 9;
      const dir = p2.x >= p1.x ? 1 : -1; // which way the link leaves person A
      const sx = p1.x + dir * nodeW(db.get('character', r.a)) / 2, sy = p1.y + spread;
      const ex = p2.x - dir * nodeW(db.get('character', r.b)) / 2, ey = p2.y + spread;
      const reach = Math.max(70, Math.abs(ex - sx) * 0.55) + Math.abs(i - (n - 1) / 2) * 25;
      const c1x = sx + dir * reach, c2x = ex - dir * reach;
      return [{
        r, d: `M${sx},${sy} C${c1x},${sy} ${c2x},${ey} ${ex},${ey}`,
        lx: (sx + 3 * c1x + 3 * c2x + ex) / 8, ly: (sy + ey) / 2,
      }];
    });
  });

  function toWorld(e: PointerEvent | WheelEvent) {
    const b = svg!.getBoundingClientRect();
    return { x: (e.clientX - b.left - view.tx) / view.k, y: (e.clientY - b.top - view.ty) / view.k };
  }

  let drag: null | { prev?: { fx: number | null; fy: number | null }; node: any | null; sx: number; sy: number; tx: number; ty: number; moved: boolean } = null;
  function down(e: PointerEvent) {
    e.preventDefault(); // stops text selection while dragging
    svg!.setPointerCapture(e.pointerId);
    const el = (e.target as Element).closest('[data-node]') as HTMLElement | null;
    const node = el ? nodes.find((n) => n.id === +el.dataset.node!) : null;
    drag = { node: node ?? null, sx: e.clientX, sy: e.clientY, tx: view.tx, ty: view.ty, moved: false };
    if (node) { drag.prev = { fx: node.fx ?? null, fy: node.fy ?? null }; node.fx = node.x; node.fy = node.y; }
  }
  function move(e: PointerEvent) {
    if (!drag) return;
    if (Math.hypot(e.clientX - drag.sx, e.clientY - drag.sy) > 3) drag.moved = true;
    if (drag.node && drag.moved) { const p = toWorld(e); drag.node.fx = drag.node.x = p.x; drag.node.fy = drag.node.y = p.y; pos = { ...pos, [drag.node.id]: { x: p.x, y: p.y } }; }
    else if (drag.moved) { view.tx = drag.tx + e.clientX - drag.sx; view.ty = drag.ty + e.clientY - drag.sy; }
  }
  function up() {
    if (!drag) return;
    if (drag.node) {
      const n = drag.node;
      if (!drag.moved) {
        n.fx = drag.prev?.fx ?? null; n.fy = drag.prev?.fy ?? null;
        focus = focus === n.id ? null : n.id;
      } else {
        // The node stays where you drop it, and the spot is saved.
        const c = db.get('character', n.id);
        if (c) db.save('character', { ...c, gx: Math.round(n.fx), gy: Math.round(n.fy) });
      }
    } else if (!drag.moved) focus = null;
    drag = null;
  }
  function wheel(e: WheelEvent) {
    const before = toWorld(e);
    view.k = Math.min(3, Math.max(0.2, view.k * (e.deltaY < 0 ? 1.1 : 1 / 1.1)));
    const b = svg!.getBoundingClientRect();
    view.tx = e.clientX - b.left - before.x * view.k;
    view.ty = e.clientY - b.top - before.y * view.k;
  }

  function connected(id: number) {
    if (!focus) return true;
    return id === focus || visible.some((r) => (r.a === focus && r.b === id) || (r.b === focus && r.a === id));
  }

  async function add() {
    if (!newA || !newB || newA === newB) return;
    const r = await db.save('relationship', { a: newA, b: newB, type: newType, description: '', since: '', directed: false });
    selRel = r.id;
  }
  function save() { if (rel) db.queue('relationship', rel); }
  async function del() { if (selRel) { await db.remove('relationship', selRel); selRel = null; } }
  function toggle(t: string) { hidden = hidden.includes(t) ? hidden.filter((x) => x !== t) : [...hidden, t]; }
</script>

<div class="split">
  <aside class="side">
    <h3>New link</h3>
    <select bind:value={newA}><option value={null}>Person A…</option>{#each db.lists.character as c}<option value={c.id}>{c.name}</option>{/each}</select>
    <div style="height:.4rem"></div>
    <select bind:value={newB}><option value={null}>Person B…</option>{#each db.lists.character as c}<option value={c.id}>{c.name}</option>{/each}</select>
    <div class="row" style="margin-top:.4rem">
      <select bind:value={newType}>{#each Object.keys(REL_TYPES) as t}<option>{t}</option>{/each}</select>
      <button class="primary" disabled={!newA || !newB || newA === newB} onclick={add}>Link</button>
    </div>

    <h3 style="margin-top:1.2rem">Show types</h3>
    {#each Object.entries(REL_TYPES) as [t, color]}
      <button class="small" style="margin:0 .25rem .25rem 0;border-color:{hidden.includes(t) ? 'var(--line)' : color};opacity:{hidden.includes(t) ? .4 : 1}"
        onclick={() => toggle(t)}>{t}</button>
    {/each}

    <h3 style="margin-top:1.2rem">All links ({db.lists.relationship.length})</h3>
    <div class="list">
      {#each db.lists.relationship as r (r.id)}
        <button class="item" class:on={r.id === selRel} onclick={() => (selRel = r.id)}>
          <span class="dot" style:background={REL_TYPES[r.type]}></span>
          {db.name('character', r.a)} – {db.name('character', r.b)}
        </button>
      {/each}
    </div>

    {#if rel}
      <div class="card">
        <div class="head"><strong>Edit link</strong><ConfirmButton onconfirm={del} /></div>
        <label>Person A</label>
        <select bind:value={rel.a} onchange={save}>{#each db.lists.character as c}<option value={c.id}>{c.name}</option>{/each}</select>
        <label>Person B</label>
        <select bind:value={rel.b} onchange={save}>{#each db.lists.character as c}<option value={c.id}>{c.name}</option>{/each}</select>
        <label>Type</label>
        <select bind:value={rel.type} onchange={save}>{#each Object.keys(REL_TYPES) as t}<option>{t}</option>{/each}</select>
        <label>Label on graph (e.g. “sister”, “teacher of”)</label>
        <input bind:value={rel.label} oninput={save} />
        <label>Since (story time or chapter)</label>
        <input bind:value={rel.since} oninput={save} />
        <label style="display:flex;gap:.4rem;align-items:center">
          <input type="checkbox" style="width:auto" bind:checked={rel.directed} onchange={save} /> One-way (A → B)
        </label>
        <label>Details</label>
        <textarea bind:value={rel.description} oninput={save}></textarea>
      </div>
    {/if}
  </aside>

  <div class="stage" bind:clientWidth={width} bind:clientHeight={height}>
    <div class="toolbar">
      <button onclick={() => { view = { tx: width / 2, ty: height / 2, k: 1 }; focus = null; }}>Reset view</button>
      <button onclick={relayout}>Re-arrange</button>
      {#if focus}<span class="chip">Focus: {db.name('character', focus)} (click empty space to clear)</span>{/if}
    </div>
    {#if db.lists.character.length === 0}
      <div class="empty" style="position:absolute;inset:3rem 1.5rem">Add characters first, then link them here.</div>
    {/if}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <svg bind:this={svg} onpointerdown={down} onpointermove={move} onpointerup={up} onwheel={wheel}>
      <defs>
        <marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
          <path d="M0,0 L10,5 L0,10 z" fill="context-stroke" />
        </marker>
      </defs>
      <g transform="translate({view.tx},{view.ty}) scale({view.k})">
        {#each edges as e (e.r.id)}
          {@const dim = focus && e.r.a !== focus && e.r.b !== focus}
          <g opacity={dim ? 0.12 : 1} onclick={() => (selRel = e.r.id)} role="presentation" style="cursor:pointer">
            <path d={e.d} fill="none" stroke="transparent" stroke-width="14" />
            <path d={e.d} fill="none" stroke={REL_TYPES[e.r.type]} stroke-width={e.r.id === selRel ? 4 : 2}
              marker-end={e.r.directed ? 'url(#arrow)' : undefined} />
            <text x={e.lx} y={e.ly} text-anchor="middle" font-size="11" fill={REL_TYPES[e.r.type]}
              paint-order="stroke" stroke="#0b0f14" stroke-width="3">{e.r.label || e.r.type}</text>
          </g>
        {/each}
        {#each db.lists.character as c (c.id)}
          {#if pos[c.id]}
            <g data-node={c.id} transform="translate({pos[c.id].x},{pos[c.id].y})" opacity={connected(c.id) ? 1 : 0.2} style="cursor:grab">
              <rect x={-nodeW(c) / 2} y={-NODE_H / 2} width={nodeW(c)} height={NODE_H} rx="12" fill={c.color}
                stroke={c.id === focus ? '#fff' : '#0b0f14'} stroke-width={c.id === focus ? 3 : 2} />
              <text y="4.5" text-anchor="middle" font-size="13" font-weight="600" fill="#0b0f14" pointer-events="none">{c.name}</text>
            </g>
          {/if}
        {/each}
      </g>
    </svg>
  </div>
</div>
