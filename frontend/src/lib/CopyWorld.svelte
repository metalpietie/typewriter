<script lang="ts">
  import { onMount } from 'svelte';
  import { veil, dialog } from './motion.svelte';
  import { Store } from '../../bindings/typewriter/internal/app';
  import type { ProjectItem } from '../../bindings/typewriter/internal/app/models.js';
  import { db, type Kind } from './db.svelte';
  import { nav } from './nav.svelte';
  import { hasKind } from './projecttype.svelte';
  import { sourceLine } from './cite-core';
  import Select from './Select.svelte';

  // Copies world items of the open project to another shelf (a project, or a world of a group). The originals stay.
  // The links between the copied items come along (see copyworld.go).
  const KINDS: { kind: Kind; title: string }[] = [
    { kind: 'character', title: 'Characters' }, { kind: 'location', title: 'Places' }, { kind: 'note', title: 'Lore notes' },
    { kind: 'faction', title: 'Factions' }, { kind: 'item', title: 'Items' }, { kind: 'term', title: 'Glossary' }, { kind: 'source', title: 'Sources' }, { kind: 'draft', title: 'Notes pages' },
  ];
  const label = (kind: Kind, it: any): string =>
    String((kind === 'note' || kind === 'draft' ? it.title : kind === 'term' ? it.word : kind === 'source' ? sourceLine(it) : it.name) ?? '').trim() || 'Untitled';

  let projects = $state<ProjectItem[]>([]);
  let to = $state('');
  let filter = $state('');
  let picked = $state<Record<number, boolean>>(nav.copy?.id ? { [nav.copy.id]: true } : {});
  let busy = $state(false);
  let message = $state('');
  let failed = $state(false);

  onMount(async () => {
    projects = ((await Store.Projects()) ?? []).filter((p) => !p.current);
    // a world of the same group first: that is where a copy most often goes
    const mine = ((await Store.Projects()) ?? []).find((p) => p.current);
    to = (projects.find((p) => mine?.group && p.group === mine.group) ?? projects[0])?.id ?? '';
  });

  const groups = $derived(KINDS.filter((k) => hasKind(k.kind)).map((k) => ({
    ...k,
    items: [...db.lists[k.kind]].map((it) => ({ id: it.id, name: label(k.kind, it) }))
      .filter((it) => !filter.trim() || it.name.toLowerCase().includes(filter.trim().toLowerCase()))
      .sort((a, b) => a.name.localeCompare(b.name)),
  })).filter((g) => g.items.length));
  const count = $derived(Object.values(picked).filter(Boolean).length);
  const target = $derived(projects.find((p) => p.id === to));
  const kindOf = (id: number) => KINDS.find((k) => db.get(k.kind, id))?.kind;

  function toggleAll(items: { id: number }[]) {
    const on = !items.every((it) => picked[it.id]);
    for (const it of items) picked[it.id] = on;
  }

  async function copy() {
    if (!count || !target || busy) return;
    busy = true; message = ''; failed = false;
    try {
      await db.settle(); // the newest edits are in the file before it is read
      const refs = Object.entries(picked).filter(([, on]) => on).map(([id]) => ({ kind: kindOf(+id) ?? '', id: +id })).filter((r) => r.kind);
      const rep = await Store.CopyItems(db.projectId, to, refs);
      message = `Copied ${rep.items} ${rep.items === 1 ? 'item' : 'items'} to “${target.name}”` +
        (rep.links ? `, with ${rep.links} ${rep.links === 1 ? 'link' : 'links'} between them (relationships, members, pictures).` : '.');
      picked = {};
    } catch (e: any) {
      failed = true; message = e?.message ?? String(e);
    } finally { busy = false; }
  }
  const close = () => (nav.copy = null);
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); close(); } }} />

<div class="overlay" transition:veil|global role="presentation" onmousedown={(e) => { if (e.target === e.currentTarget) close(); }}>
  <div class="palette box" transition:dialog|global role="dialog" aria-label="Copy to another shelf">
    <div class="head"><h2 style="margin:0">Copy to another shelf</h2><button class="small" onclick={close}>Close</button></div>
    <p class="muted" style="margin:.4rem 0 .8rem">
      Copies characters, places, notes and more into another project, for example a book with its own world in a group.
      The originals stay here. Relationships, faction members, mood board pictures, @ links and citations come along when
      both ends are copied. Maps and the timeline are not copied.
    </p>

    {#if !projects.length}
      <p>There is no other shelf yet. Make one on the bookshelf first.</p>
    {:else}
      <label for="copy-to">Copy to</label>
      <Select id="copy-to" bind:value={to}>
        {#each projects as p}<option value={p.id}>{p.name}{p.group ? ` (${p.group})` : ''}</option>{/each}
      </Select>

      <input class="filter" placeholder="Filter by name" bind:value={filter} />
      <div class="lists">
        {#each groups as g (g.kind)}
          <div class="group">
            <div class="ghead">
              <b>{g.title}</b>
              <button class="link" onclick={() => toggleAll(g.items)}>{g.items.every((it) => picked[it.id]) ? 'none' : 'all'}</button>
            </div>
            {#each g.items as it (it.id)}
              <label class="pick"><input type="checkbox" bind:checked={picked[it.id]} /> <span>{it.name}</span></label>
            {/each}
          </div>
        {:else}
          <p class="muted">{filter ? 'Nothing matches.' : 'This shelf has no world items to copy.'}</p>
        {/each}
      </div>

      <div class="foot">
        {#if message}<p class="msg" style:color={failed ? 'var(--danger)' : 'var(--ok)'}>{message}</p>{/if}
        <span class="grow"></span>
        <button class="primary" disabled={!count || !target || busy} onclick={copy}>
          {busy ? 'Copying…' : `Copy ${count || ''} ${count === 1 ? 'item' : 'items'}`.replace('  ', ' ')}
        </button>
      </div>
    {/if}
  </div>
</div>

<style>
  .box { width: min(620px, 94vw); max-height: 86vh; display: flex; flex-direction: column; padding: 1rem 1.1rem; }
  .head { display: flex; justify-content: space-between; align-items: center; }
  .filter { margin: .7rem 0 .5rem; }
  .lists { overflow: auto; min-height: 8rem; border: 1px solid var(--line); border-radius: 8px; padding: .5rem .7rem; columns: 2 14rem; column-gap: 1.2rem; }
  .group { break-inside: avoid; margin-bottom: .7rem; }
  .ghead { display: flex; justify-content: space-between; align-items: baseline; font-size: .85rem; margin-bottom: .2rem; }
  .pick { display: flex; gap: .45rem; align-items: center; font-size: .88rem; padding: .12rem 0; margin: 0; color: var(--text); cursor: pointer; }
  .pick input { width: auto; margin: 0; }
  .pick span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .foot { display: flex; align-items: center; gap: .6rem; margin-top: .8rem; }
  .grow { flex: 1; }
  .msg { margin: 0; font-size: .85rem; }
</style>
