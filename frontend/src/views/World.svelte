<script lang="ts">
  import { db, NOTE_CATEGORIES, PLACE_TYPES } from '../lib/db.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import Backlinks from '../lib/Backlinks.svelte';
  import Picture from '../lib/Picture.svelte';
  import { nav } from '../lib/nav.svelte';

  let mode = $state<'note' | 'location'>('note');
  let selId = $state<number | null>(null);
  // Jump here when a search result is picked.
  $effect(() => {
    const p = nav.pending;
    if (p && (p.kind === 'note' || p.kind === 'location')) {
      mode = p.kind; filter = 'All'; search = ''; selId = p.id; nav.pending = null;
    }
  });
  let filter = $state('All');
  let search = $state('');

  const kind = $derived(mode);
  const sel = $derived(db.get(kind, selId));
  const items = $derived(
    db.lists[kind].filter((i) => {
      const text = ((i.title ?? i.name ?? '') + ' ' + (i.tags ?? '')).toLowerCase();
      const cat = mode === 'note' ? i.category : i.type;
      return text.includes(search.toLowerCase()) && (filter === 'All' || cat === filter);
    }),
  );
  const cats = $derived(['All', ...(mode === 'note' ? NOTE_CATEGORIES : PLACE_TYPES)]);

  function switchMode(m: 'note' | 'location') { mode = m; selId = null; filter = 'All'; }

  async function add() {
    const item = mode === 'note'
      ? await db.save('note', { title: 'New note', category: filter === 'All' ? 'Other' : filter, body: '', tags: '' })
      : await db.save('location', { name: 'New place', type: filter === 'All' ? 'city' : filter, description: '', parent: null, tags: '' });
    selId = item.id;
  }
  function save() { if (sel) db.queue(kind, sel); }
  async function del() { if (selId) { await db.remove(kind, selId); selId = null; } }
</script>

<div class="split">
  <aside class="side">
    <div class="tabs">
      <button class:active={mode === 'note'} onclick={() => switchMode('note')}>Lore notes</button>
      <button class:active={mode === 'location'} onclick={() => switchMode('location')}>Places</button>
    </div>
    <div class="row"><input placeholder="Search…" bind:value={search} /><button class="primary" onclick={add}>+ New</button></div>
    {#if mode === 'location'}<button class="small" style="width:100%;margin-top:.4rem" onclick={() => (nav.names = 'place')}>Name generator…</button>{/if}
    <label>Filter</label>
    <select bind:value={filter}>{#each cats as c}<option>{c}</option>{/each}</select>
    <div class="list">
      {#each items as i (i.id)}
        <button class="item" class:on={i.id === selId} onclick={() => (selId = i.id)}>
          {i.title ?? i.name}<small>{i.category ?? i.type}</small>
        </button>
      {:else}<div class="empty">Nothing here yet.</div>{/each}
    </div>
  </aside>

  <section class="pane">
    {#if sel && mode === 'note'}
      <div class="head"><h2>{sel.title}</h2><ConfirmButton onconfirm={del} /></div>
      <div class="grid2">
        <div><label>Title</label><input bind:value={sel.title} oninput={save} /></div>
        <div><label>Category</label>
          <select bind:value={sel.category} onchange={save}>{#each NOTE_CATEGORIES as c}<option>{c}</option>{/each}</select>
        </div>
      </div>
      <label>Tags (comma separated)</label><input bind:value={sel.tags} oninput={save} />
      <label>Details</label>
      <textarea style="min-height:50vh" bind:value={sel.body} oninput={save}></textarea>
      <Backlinks kind="note" id={sel.id} />
    {:else if sel && mode === 'location'}
      <div class="head"><h2>{sel.name}</h2><ConfirmButton onconfirm={del} /></div>
      <div class="grid2">
        <div><label>Name</label><input bind:value={sel.name} oninput={save} /></div>
        <div><label>Type</label>
          <select bind:value={sel.type} onchange={save}>{#each PLACE_TYPES as c}<option>{c}</option>{/each}</select>
        </div>
        <div><label>Inside (parent place)</label>
          <select bind:value={sel.parent} onchange={save}>
            <option value={null}>— none —</option>
            {#each db.lists.location.filter((l) => l.id !== sel.id) as l}<option value={l.id}>{l.name}</option>{/each}
          </select>
        </div>
        <div><label>Tags</label><input bind:value={sel.tags} oninput={save} /></div>
      </div>
      <Picture label="Picture" value={sel.image} onchange={(v) => { sel.image = v; save(); }} />
      <label>Description</label>
      <textarea style="min-height:40vh" bind:value={sel.description} oninput={save}></textarea>
      <p class="muted">
        {#if sel.map}Pinned on map “{db.name('map', sel.map)}”.{:else}Not on a map yet. Place it in the Maps tab.{/if}
      </p>
      <Backlinks kind="location" id={sel.id} />
    {:else}
      <div class="empty">Pick an item, or press “+ New”.</div>
    {/if}
  </section>
</div>
