<script lang="ts">
  import Select from '../lib/Select.svelte';
  import { db } from '../lib/db.svelte';
  import { openItem } from '../lib/nav.svelte';
  import { layoutFamily, setFamRole, FAM_ROLES, NODE_H, type FamRole } from '../lib/familytree';

  let focusId = $state<number | null>(null); // show only the family of this person
  let selId = $state<number | null>(null);
  let newA = $state<number | null>(null);
  let newB = $state<number | null>(null);
  let newRole = $state<FamRole>('parent');

  let width = $state(800);
  let height = $state(600);
  let view = $state({ tx: 0, ty: 0, k: 1 });
  let svg = $state<SVGSVGElement>();

  const family = $derived(
    db.lists.relationship.filter((r) => r.type === 'family' && db.get('character', r.a) && db.get('character', r.b)),
  );
  const withRole = $derived(family.filter((r) => r.fam));
  const needRole = $derived(family.filter((r) => !r.fam));

  /** Everyone who is connected to the chosen person through family links. */
  const ids = $derived.by(() => {
    const all = new Set<number>();
    for (const r of withRole) { all.add(r.a); all.add(r.b); }
    if (!focusId) return [...all].sort((a, b) => a - b);
    const seen = new Set<number>(all.has(focusId) ? [focusId] : []);
    for (let grew = true; grew; ) {
      grew = false;
      for (const r of withRole) {
        if (seen.has(r.a) !== seen.has(r.b)) { seen.add(r.a); seen.add(r.b); grew = true; }
      }
    }
    return [...seen].sort((a, b) => a - b);
  });

  const nodeW = (id: number) => Math.max(96, (db.get('character', id)?.name?.length ?? 4) * 7.6 + 28);
  const layout = $derived(
    layoutFamily(ids, nodeW, {
      parents: withRole.filter((r) => r.fam === 'parent').map((r) => [r.a, r.b] as [number, number]),
      partners: withRole.filter((r) => r.fam === 'partner').map((r) => [r.a, r.b] as [number, number]),
      siblings: withRole.filter((r) => r.fam === 'sibling').map((r) => [r.a, r.b] as [number, number]),
    }),
  );

  function fit() {
    const w = layout.maxX - layout.minX + 80, h = layout.maxY + 100;
    const k = Math.min(1.2, Math.max(0.2, Math.min(width / w, height / h)));
    view = { k, tx: (width - (layout.maxX - layout.minX) * k) / 2 - layout.minX * k, ty: 50 * k };
  }
  // fit again when the people on screen change
  let fitted = '';
  $effect(() => {
    const key = ids.join(',') + '|' + focusId;
    width; height;
    if (key !== fitted && width > 0 && height > 0) { fitted = key; fit(); }
  });

  // drag to pan, wheel to zoom
  let drag: { sx: number; sy: number; tx: number; ty: number; moved: boolean; node: number | null } | null = null;
  function down(e: PointerEvent) {
    e.preventDefault(); // no text selection while dragging
    svg!.setPointerCapture(e.pointerId);
    const el = (e.target as Element).closest('[data-node]') as HTMLElement | null;
    drag = { sx: e.clientX, sy: e.clientY, tx: view.tx, ty: view.ty, moved: false, node: el ? +el.dataset.node! : null };
  }
  function move(e: PointerEvent) {
    if (!drag) return;
    if (Math.hypot(e.clientX - drag.sx, e.clientY - drag.sy) > 3) drag.moved = true;
    if (drag.moved) { view.tx = drag.tx + e.clientX - drag.sx; view.ty = drag.ty + e.clientY - drag.sy; }
  }
  function up() {
    if (drag && !drag.moved) selId = drag.node;
    drag = null;
  }
  function wheel(e: WheelEvent) {
    const b = svg!.getBoundingClientRect();
    const wx = (e.clientX - b.left - view.tx) / view.k, wy = (e.clientY - b.top - view.ty) / view.k;
    view.k = Math.min(3, Math.max(0.2, view.k * (e.deltaY < 0 ? 1.1 : 1 / 1.1)));
    view.tx = e.clientX - b.left - wx * view.k;
    view.ty = e.clientY - b.top - wy * view.k;
  }

  async function add() {
    if (!newA || !newB || newA === newB) return;
    const r: any = { a: newA, b: newB, type: 'family', label: '', description: '', since: '', directed: false, fam: '' };
    setFamRole(r, newRole);
    await db.save('relationship', r);
    newA = newB = null;
  }
  function changeRole(r: any, role: FamRole | '') { setFamRole(r, role); db.save('relationship', r); }

  const sel = $derived(db.get('character', selId));
  const selLinks = $derived(withRole.filter((r) => r.a === selId || r.b === selId));
  const phrase = (r: any) => {
    const other = db.name('character', r.a === selId ? r.b : r.a);
    if (r.fam === 'parent') return r.a === selId ? `parent of ${other}` : `child of ${other}`;
    return `${r.fam} of ${other}`;
  };
</script>

<div class="split">
  <aside class="side">
    <h3>Show</h3>
    <Select bind:value={focusId}>
      <option value={null}>Every family</option>
      {#each db.lists.character as c}<option value={c.id}>Family of {c.name}</option>{/each}
    </Select>

    <h3 style="margin-top:1.2rem">New family link</h3>
    <Select bind:value={newA}><option value={null}>Person A…</option>{#each db.lists.character as c}<option value={c.id}>{c.name}</option>{/each}</Select>
    <div style="height:.4rem"></div>
    <Select bind:value={newRole}>{#each FAM_ROLES as r}<option value={r.role}>{r.label}</option>{/each}</Select>
    <div style="height:.4rem"></div>
    <Select bind:value={newB}><option value={null}>Person B…</option>{#each db.lists.character as c}<option value={c.id}>{c.name}</option>{/each}</Select>
    <button class="primary" style="width:100%;margin-top:.5rem" disabled={!newA || !newB || newA === newB} onclick={add}>Add link</button>

    {#if needRole.length}
      <h3 style="margin-top:1.2rem">Pick a role ({needRole.length})</h3>
      <p class="muted">These links have the type “family”, but no role, so the tree cannot place them.</p>
      {#each needRole as r (r.id)}
        <div class="card">
          <strong>{db.name('character', r.a)}</strong> and <strong>{db.name('character', r.b)}</strong>
          <Select style="margin-top:.4rem" value="" onchange={(e) => changeRole(r, e.currentTarget.value as FamRole)}>
            <option value="">Role…</option>
            {#each FAM_ROLES as x}<option value={x.role}>{x.label}</option>{/each}
          </Select>
        </div>
      {/each}
    {/if}

    {#if sel}
      <div class="card">
        <div class="head"><strong>{sel.name}</strong><button class="small" onclick={() => openItem('character', sel.id)}>Open</button></div>
        {#each selLinks as r (r.id)}<div class="muted">{phrase(r)}</div>{:else}<div class="muted">No family links with a role.</div>{/each}
      </div>
    {/if}

    <p class="muted" style="margin-top:1.2rem">
      Links are the “family” links from the Relationships tab. Put the parent as person A. Drag to move around, scroll to zoom, click a person for details.
    </p>
  </aside>

  <div class="stage" bind:clientWidth={width} bind:clientHeight={height}>
    <div class="toolbar"><button onclick={fit}>Fit to screen</button></div>
    {#if layout.nodes.length === 0}
      <div class="empty" style="position:absolute;inset:3rem 1.5rem">
        No family yet. Add a link on the left, for example “Aldric is a parent of Brenna”.
      </div>
    {/if}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <svg bind:this={svg} onpointerdown={down} onpointermove={move} onpointerup={up} onwheel={wheel}>
      <g transform="translate({view.tx},{view.ty}) scale({view.k})">
        {#each layout.paths as p}
          <path d={p.d} fill="none" stroke={p.kind === 'partner' ? 'var(--accent-2)' : 'var(--muted)'}
            stroke-width={p.kind === 'partner' ? 4 : 2} stroke-dasharray={p.kind === 'sibling' ? '5 4' : undefined} stroke-linejoin="round" />
        {/each}
        {#each layout.nodes as n (n.id)}
          {@const c = db.get('character', n.id)}
          {#if c}
            <g data-node={n.id} transform="translate({n.x},{n.y})" style="cursor:pointer">
              <rect x={-n.w / 2} y={-NODE_H / 2} width={n.w} height={NODE_H} rx="12" fill={c.color}
                stroke={n.id === selId ? 'var(--text)' : 'var(--stage)'} stroke-width={n.id === selId ? 3 : 2} />
              <text y={c.age ? -1 : 5} text-anchor="middle" font-size="13" font-weight="600" fill="#0b0f14" pointer-events="none">{c.name}</text>
              {#if c.age}<text y="14" text-anchor="middle" font-size="10.5" fill="#0b0f14" opacity=".75" pointer-events="none">{c.age}</text>{/if}
            </g>
          {/if}
        {/each}
      </g>
    </svg>
  </div>
</div>
