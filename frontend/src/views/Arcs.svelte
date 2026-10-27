<script lang="ts">
  import { takeCommand } from '../lib/commands';
  import Select from '../lib/Select.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import { db } from '../lib/db.svelte';
  import { nav, openScene } from '../lib/nav.svelte';
  import { scenesOf, sceneLabel, orderedScenes } from '../lib/scenes';
  import { BEATS, BEAT_LETTER, arcsSorted, arcLinks, arcWarnings, addArc, tagScene, untagScene, setBeat, linkFor, type Beat } from '../lib/arcs';

  let selId = $state<number | null>(null); // null shows the overview of all arcs
  const arcs = $derived(arcsSorted());
  const sel = $derived(db.get('arc', selId));

  // Jump here when a search result is picked.
  $effect(() => {
    const p = nav.pending;
    if (p?.kind === 'arc') { selId = p.id; nav.pending = null; }
  });

  // the overview shows one book at a time; start on the book you were writing in
  const lastBook = () => {
    try { return JSON.parse(localStorage.getItem('typewriter.write') ?? 'null')?.book ?? null; } catch { return null; }
  };
  const books = $derived([...db.lists.book].sort((a, b) => a.order - b.order));
  let pick = $state<number | null>(lastBook());
  const bookId = $derived(db.get('book', pick) ? pick : (books[0]?.id ?? null));
  const chapters = $derived(db.lists.chapter.filter((c) => c.book === bookId).sort((a, b) => a.order - b.order));

  const beatName = (b: string) => BEATS.find(([k]) => k === b)?.[1] ?? 'touches';

  /** One row of the overview: for each chapter, the scenes that belong to the arc. */
  const rows = $derived(arcs.map((arc) => {
    const cells = chapters.map((c) => scenesOf(c.id).filter((s) => linkFor(arc.id, s.id)).map((s) => ({ scene: s, beat: (linkFor(arc.id, s.id)!.beat ?? '') as Beat })));
    const used = cells.map((x) => x.length > 0);
    return { arc, cells, from: used.indexOf(true), to: used.lastIndexOf(true), total: arcLinks(arc.id).length, warnings: arcWarnings(arc.id) };
  }));

  const links = $derived(sel ? arcLinks(sel.id) : []);
  const warnings = $derived(sel ? arcWarnings(sel.id) : []);
  const bookPrefix = (s: any) => (db.lists.book.length > 1 ? (db.get('book', s.book)?.title || 'Book') + ' · ' : '');
  const untagged = $derived(sel ? orderedScenes().filter((s) => !linkFor(sel.id, s.id)) : []);
  let addPick = $state<number | null>(null);

  async function add() { selId = (await addArc()).id; }
  const save = () => { if (sel) db.queue('arc', sel); };
  async function del() { if (selId) { await db.remove('arc', selId); selId = null; } }
  async function move(dir: -1 | 1) {
    if (!sel) return;
    const other = arcs[arcs.indexOf(sel) + dir];
    if (!other) return;
    const a = sel.order ?? sel.id, b = other.order ?? other.id;
    sel.order = a === b ? b + dir : b;
    other.order = a;
    await db.save('arc', sel); await db.save('arc', other);
  }

  $effect(() => { if (takeCommand('new')) add(); }); // the command menu
</script>

<div class="split">
  <aside class="side">
    <div class="row"><button class="primary" onclick={add}>+ New arc</button></div>
    <div class="list">
      <button class="item" class:on={selId === null} onclick={() => (selId = null)}>All arcs<small>Overview of the book</small></button>
      {#each rows as r (r.arc.id)}
        <button class="item" class:on={r.arc.id === selId} onclick={() => (selId = r.arc.id)}>
          <span class="dot" style:background={r.arc.color}></span> {r.arc.name || 'Untitled arc'}
          <small>{r.total} scene{r.total === 1 ? '' : 's'}{r.warnings.length ? ` · ${r.warnings.length} to check` : ''}</small>
        </button>
      {/each}
    </div>
  </aside>

  <section class="pane">
    {#if sel}
      <div class="head">
        <input type="color" class="swatch" aria-label="Colour" bind:value={sel.color} oninput={save} />
        <input class="title" placeholder="Arc name" aria-label="Arc name" bind:value={sel.name} oninput={save} />
        <button class="small" title="Move up in the list" disabled={arcs.indexOf(sel) === 0} onclick={() => move(-1)}>↑</button>
        <button class="small" title="Move down in the list" disabled={arcs.indexOf(sel) === arcs.length - 1} onclick={() => move(1)}>↓</button>
        <ConfirmButton onconfirm={del} />
      </div>

      {#if warnings.length}
        <div class="warn">
          <strong>To check</strong>
          {#each warnings as w}
            <div class="wrow"><span>{w.text}</span>{#if w.scene}<button class="small" onclick={() => openScene(w.scene!)}>Open scene</button>{/if}</div>
          {/each}
        </div>
      {:else}
        <div class="ok">This arc has a start, an end, and no long gaps.</div>
      {/if}

      <label for="arcnotes">Notes</label>
      <textarea id="arcnotes" bind:value={sel.notes} oninput={save} placeholder="What is this thread about? How does it start, turn and end?"></textarea>

      <h3 style="margin-top:1.4rem">Scenes · {links.length}</h3>
      <div class="card">
        {#each links as { link, scene } (link.id)}
          <div class="srow">
            <Select style="flex:0 0 6.5rem" aria-label="What the scene does for the arc" value={link.beat ?? ''} onchange={(e) => setBeat(link, e.currentTarget.value as Beat)}>
              {#each BEATS as [v, l]}<option value={v}>{l}</option>{/each}
            </Select>
            <button class="link" title="Write this scene" onclick={() => openScene(scene.id)}>{bookPrefix(scene)}{sceneLabel(scene)}</button>
            <button class="small" title="Take this scene out of the arc" onclick={() => untagScene(sel.id, scene.id)}>✕</button>
          </div>
        {:else}<span class="muted">No scenes yet. Add one below, or tick this arc inside a scene in the Write tab.</span>{/each}
        <div class="row" style="margin-top:.6rem">
          <Select bind:value={addPick} aria-label="Add a scene to this arc" onchange={(e) => { const id = +e.currentTarget.value; if (id && sel) tagScene(sel.id, id); setTimeout(() => (addPick = null)); }}>
            <option value={null}>Add a scene…</option>
            {#each untagged as s (s.id)}<option value={s.id}>{bookPrefix(s)}{sceneLabel(s)}</option>{/each}
          </Select>
        </div>
      </div>
    {:else}
      <div class="head">
        <h2 style="margin:0">Story arcs</h2>
        {#if books.length > 1}
          <Select aria-label="Book" style="flex:0 1 16rem" bind:value={pick}>
            {#each books as b}<option value={b.id}>{b.order}. {b.title}</option>{/each}
          </Select>
        {/if}
      </div>
      <p class="muted help">One row for each plot thread. One column for each chapter. A dot is a scene that belongs to the thread. Click a dot to write that scene. Click a name to edit the arc.</p>

      {#if !arcs.length}
        <div class="empty">No arcs yet. Press “+ New arc”, for example “The crown” or “The war with the Guild”. Then tick it inside the scenes it touches.</div>
      {:else if !chapters.length}
        <div class="empty">This book has no chapters yet.</div>
      {:else}
        <div class="gridwrap">
          <table class="grid">
            <thead>
              <tr><th class="name"></th>{#each chapters as c (c.id)}<th title={c.title}>{c.order}</th>{/each}</tr>
            </thead>
            <tbody>
              {#each rows as r (r.arc.id)}
                <tr>
                  <th class="name"><button class="link" onclick={() => (selId = r.arc.id)}><span class="dot" style:background={r.arc.color}></span> {r.arc.name || 'Untitled arc'}</button></th>
                  {#each r.cells as cell, i}
                    <td class:span={r.from >= 0 && i >= r.from && i <= r.to} class:first={i === r.from} class:last={i === r.to} style:--c={r.arc.color}>
                      <div class="dots">
                        {#each cell as d (d.scene.id)}
                          <button class="pt" class:big={d.beat !== ''} title="{sceneLabel(d.scene)} · {beatName(d.beat)}" aria-label="{sceneLabel(d.scene)}, {beatName(d.beat)} {r.arc.name}" onclick={() => openScene(d.scene.id)}>{BEAT_LETTER[d.beat]}</button>
                        {/each}
                      </div>
                    </td>
                  {/each}
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
        <p class="muted legend"><b>S</b> starts the arc · <b>T</b> a turn · <b>E</b> ends it · a plain dot only touches it. The line shows how long the arc runs.</p>

        {#if rows.some((r) => r.warnings.length)}
          <h3 style="margin-top:1.2rem">To check</h3>
          <div class="card">
            {#each rows as r (r.arc.id)}
              {#if r.warnings.length}
                <div class="chk">
                  <button class="link" onclick={() => (selId = r.arc.id)}><span class="dot" style:background={r.arc.color}></span> {r.arc.name || 'Untitled arc'}</button>
                  <ul>{#each r.warnings as w}<li>{w.text}</li>{/each}</ul>
                </div>
              {/if}
            {/each}
          </div>
        {/if}
      {/if}
    {/if}
  </section>
</div>

<style>
  .head { display: flex; gap: .5rem; align-items: center; margin-bottom: .8rem; }
  .title { flex: 1; font-size: 1.3rem; font-weight: 700; }
  .swatch { flex: 0 0 auto; }
  .help { margin: 0 0 .8rem; font-size: .85rem; }
  .link { background: none; border: 0; padding: 0; color: var(--accent); text-align: left; }
  .link:hover { text-decoration: underline; }

  .warn { border: 1px solid var(--danger); border-radius: 8px; padding: .6rem .8rem; margin-bottom: .6rem; background: color-mix(in srgb, var(--danger) 8%, transparent); }
  .wrow { display: flex; gap: .6rem; align-items: center; justify-content: space-between; margin-top: .3rem; }
  .ok { color: var(--ok); margin-bottom: .6rem; }
  .srow { display: flex; gap: .5rem; align-items: center; padding: .3rem 0; border-top: 1px solid var(--line); }
  .srow:first-child { border-top: 0; }
  .srow .link { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .gridwrap { overflow: auto; border: 1px solid var(--line); border-radius: 10px; background: var(--panel); }
  .grid { border-collapse: collapse; width: 100%; }
  .grid th, .grid td { padding: 0; border-bottom: 1px solid var(--line); }
  .grid tr:last-child th, .grid tr:last-child td { border-bottom: 0; }
  .grid thead th { font-size: .75rem; color: var(--muted); font-weight: 600; padding: .4rem 0; min-width: 2.6rem; border-bottom: 1px solid var(--line); }
  .grid .name { position: sticky; left: 0; z-index: 1; background: var(--panel); text-align: left; padding: .4rem .8rem; min-width: 11rem; max-width: 16rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; border-right: 1px solid var(--line); }
  .grid .name .link { max-width: 100%; overflow: hidden; text-overflow: ellipsis; color: var(--text); font-weight: 600; }
  .grid td { position: relative; height: 2.6rem; text-align: center; }
  /* the line that shows how long the arc runs */
  .grid td.span::before { content: ''; position: absolute; top: 50%; left: 0; right: 0; height: 3px; margin-top: -1.5px; background: color-mix(in srgb, var(--c) 40%, transparent); }
  .grid td.first::before { left: 50%; }
  .grid td.last::before { right: 50%; }
  .grid td.first.last::before { display: none; }
  .dots { position: relative; display: flex; flex-wrap: wrap; gap: 3px; justify-content: center; align-items: center; padding: .2rem; min-height: 2.6rem; }
  .pt { width: 14px; height: 14px; min-width: 14px; flex: 0 0 auto; aspect-ratio: 1; padding: 0; border-radius: 50%; border: 0; background: var(--c); color: var(--on-accent); font-size: .55rem; font-weight: 800; line-height: 14px; }
  .pt.big { width: 20px; height: 20px; line-height: 20px; font-size: .65rem; box-shadow: 0 0 0 2px var(--panel), 0 0 0 3px var(--c); }
  .pt:hover { transform: scale(1.25); border: 0; }
  .legend { font-size: .8rem; margin: .5rem 0 0; }
  .chk { padding: .4rem 0; border-top: 1px solid var(--line); }
  .chk:first-child { border-top: 0; }
  .chk ul { margin: .2rem 0 0; padding-left: 1.4rem; color: var(--muted); }
</style>
