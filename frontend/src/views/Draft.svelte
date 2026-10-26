<script lang="ts">
  import { takeCommand } from '../lib/commands';
  import { db, wordCount } from '../lib/db.svelte';
  import { nav } from '../lib/nav.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import CopyButton from '../lib/CopyButton.svelte';
  import RichEditor from '../lib/RichEditor.svelte';

  let selId = $state<number | null>(null);
  let search = $state('');

  const pages = $derived([...db.lists.draft].sort((a, b) => (a.order ?? a.id) - (b.order ?? b.id)));
  const shown = $derived(pages.filter((p) => (p.title + ' ' + p.body).toLowerCase().includes(search.toLowerCase())));
  const sel = $derived(db.get('draft', selId));
  // a page that came from a phone says so
  const origin = $derived(db.lists.syncnote.find((n) => n.draft === selId && !n.deleted)?.from || '');

  // Jump here when a search result is picked.
  $effect(() => {
    const p = nav.pending;
    if (p?.kind === 'draft') { search = ''; selId = p.id; nav.pending = null; }
  });

  // Open the first page when the tab opens.
  $effect(() => { if (selId === null && pages.length) selId = pages[0].id; });

  async function add() {
    const last = pages[pages.length - 1];
    const p = await db.save('draft', { title: 'Untitled page', body: '', order: (last?.order ?? last?.id ?? 0) + 1 });
    search = '';
    selId = p.id;
  }
  const save = () => { if (sel) db.queue('draft', sel); };
  const change = (html: string) => { if (sel) { sel.body = html; save(); } };
  async function del() { if (selId) { await db.remove('draft', selId); selId = null; } }

  /** Swap the order of two pages. */
  async function move(dir: -1 | 1) {
    if (!sel) return;
    const other = pages[pages.indexOf(sel) + dir];
    if (!other) return;
    const a = sel.order ?? sel.id, b = other.order ?? other.id;
    sel.order = a === b ? b + dir : b;
    other.order = a;
    await db.save('draft', sel); await db.save('draft', other);
  }
  const preview = (body: string) => {
    const t = new DOMParser().parseFromString((body || '').replace(/<\/(p|h[1-6]|li)>/g, ' '), 'text/html').body.textContent?.trim() ?? '';
    return t.length > 60 ? t.slice(0, 60) + '…' : t;
  };

  $effect(() => { if (takeCommand('new')) add(); }); // the command menu
</script>

<div class="split">
  <aside class="side">
    <div class="row">
      <input placeholder="Search…" bind:value={search} />
      <button class="primary" onclick={add}>+ New</button>
    </div>
    <div class="list">
      {#each shown as p (p.id)}
        <button class="item" class:on={p.id === selId} onclick={() => (selId = p.id)}>
          {p.title || 'Untitled page'}<small>{preview(p.body) || 'Empty'}</small>
        </button>
      {:else}
        <div class="empty">No pages yet. Press “+ New” and write down your first idea. It can be rough.</div>
      {/each}
    </div>
  </aside>

  <section class="pane">
    {#if sel}
      <div class="head">
        <input class="title" placeholder="Page title" bind:value={sel.title} oninput={save} />
        <button class="small" title="Move up in the list" disabled={pages.indexOf(sel) === 0} onclick={() => move(-1)}>↑</button>
        <button class="small" title="Move down in the list" disabled={pages.indexOf(sel) === pages.length - 1} onclick={() => move(1)}>↓</button>
        <CopyButton kind="draft" id={sel.id} />
        <ConfirmButton onconfirm={del} />
      </div>
      <div class="muted count">{wordCount(sel.body)} words · a page of notes. It is not counted in your book.{origin ? ' · From ' + origin : ''}</div>
      {#key sel.id}
        <RichEditor autofocus value={sel.body} onchange={change} placeholder="Write anything. Who is it about? What goes wrong? How does it end?" />
      {/key}
    {:else}
      <div class="empty">Pick a page, or press “+ New”.</div>
    {/if}
  </section>
</div>

<style>
  .head { display: flex; gap: .4rem; align-items: center; margin-bottom: .3rem; }
  .title { flex: 1; font-size: 1.3rem; font-weight: 700; }
  .count { font-size: .8rem; margin-bottom: .6rem; }
</style>
