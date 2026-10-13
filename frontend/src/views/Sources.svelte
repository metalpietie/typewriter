<script lang="ts">
  import Select from '../lib/Select.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import { db } from '../lib/db.svelte';
  import { nav, openScene } from '../lib/nav.svelte';
  import { orderedScenes, sceneLabel } from '../lib/scenes';
  import { word } from '../lib/projecttype.svelte';
  import { SOURCE_STATUS, SOURCE_TYPES, citeLabel, citedIds, sourceLine, stripCitations } from '../lib/cite-core';

  // The library of sources: the books, articles and pages that the writer used. A citation in the text points to one of them.
  let selId = $state<number | null>(null);
  let search = $state('');
  let statusFilter = $state('');
  let asking = $state(false); // the source is cited: ask before it goes

  const sources = $derived([...db.lists.source].sort((a, b) => sourceLine(a).toLowerCase().localeCompare(sourceLine(b).toLowerCase())));
  const shown = $derived(sources.filter((s) => {
    if (statusFilter && (s.status ?? 'to-read') !== statusFilter) return false;
    return (sourceLine(s) + ' ' + (s.note ?? '') + ' ' + (s.url ?? '')).toLowerCase().includes(search.toLowerCase());
  }));
  const sel = $derived(db.get('source', selId));

  // Jump here when a search result is picked.
  $effect(() => {
    const p = nav.pending;
    if (p?.kind === 'source') { search = ''; statusFilter = ''; selId = p.id; nav.pending = null; }
  });
  $effect(() => { selId; asking = false; });

  /** Where each source is cited, worked out from the text of the scenes. */
  const citedIn = $derived.by(() => {
    const map = new Map<number, { id: number; label: string; times: number }[]>();
    for (const s of orderedScenes()) {
      const counts = new Map<number, number>();
      for (const id of citedIds(String(s.body ?? ''))) counts.set(id, (counts.get(id) ?? 0) + 1);
      for (const [id, times] of counts) map.set(id, [...(map.get(id) ?? []), { id: s.id, label: sceneLabel(s), times }]);
    }
    return map;
  });
  const here = $derived(sel ? citedIn.get(sel.id) ?? [] : []);
  const total = $derived(here.reduce((n, h) => n + h.times, 0));

  async function add() {
    const s = await db.save('source', { type: 'book', title: '', authors: '', year: '', container: '', publisher: '', volume: '', issue: '', pages: '', url: '', accessed: '', edition: '', status: 'to-read', note: '' });
    search = ''; statusFilter = '';
    selId = s.id;
  }
  const save = () => { if (sel) db.queue('source', sel); };

  /** Take the citations of a source out of the text, then delete it. */
  async function del(removeCites: boolean) {
    if (!selId) return;
    const id = selId;
    if (removeCites) {
      db.flush();
      for (const s of db.lists.scene) {
        const body = String(s.body ?? '');
        if (!citedIds(body).includes(id)) continue;
        s.body = stripCitations(body, id);
        await db.save('scene', s);
      }
    }
    // ideas and quotes that named this source no longer do
    for (const i of db.lists.idea) if (i.source === id) { i.source = null; await db.save('idea', i); }
    selId = null; asking = false;
    await db.remove('source', id);
  }
  const askOrDelete = () => (total > 0 ? (asking = true) : del(false));

  const isWeb = $derived(sel?.type === 'web');
  const isArticle = $derived(sel?.type === 'article');
</script>

<div class="split">
  <aside class="side">
    <div class="row">
      <input placeholder="Search…" bind:value={search} />
      <button class="primary" onclick={add}>+ New</button>
    </div>
    <Select aria-label="Show sources" bind:value={statusFilter}>
      <option value="">All sources</option>
      {#each SOURCE_STATUS as [k, name]}<option value={k}>{name}</option>{/each}
    </Select>
    <div class="list">
      {#each shown as s (s.id)}
        <button class="item" class:on={s.id === selId} onclick={() => (selId = s.id)}>
          {sourceLine(s)}
          <small>{SOURCE_TYPES.find(([k]) => k === s.type)?.[1] ?? 'Other'}{(citedIn.get(s.id)?.length ?? 0) ? ' · cited' : ''}</small>
        </button>
      {:else}
        <div class="empty">No sources yet. Add the books, articles and pages that you use. Then cite them in your text.</div>
      {/each}
    </div>
  </aside>

  <section class="pane">
    {#if sel}
      <div class="head"><h2>{sel.title || 'Untitled source'}</h2><ConfirmButton onconfirm={askOrDelete} /></div>
      {#if asking}
        <div class="card warn">
          <p style="margin-top:0"><b>This source is cited {total} time{total === 1 ? '' : 's'} in {here.length} {here.length === 1 ? word('scene') : word('scenes')}.</b>
            If you delete it, those citations show as [?] in an export. You can take them out of the text now.</p>
          <div class="btns">
            <button class="small danger" onclick={() => del(true)}>Delete and remove the citations</button>
            <button class="small" onclick={() => del(false)}>Delete, keep the citations</button>
            <button class="small" onclick={() => (asking = false)}>Cancel</button>
          </div>
        </div>
      {/if}
      <div class="grid2">
        <div><label for="stitle">Title</label><input id="stitle" bind:value={sel.title} oninput={save} /></div>
        <div><label for="stype">Type</label>
          <Select id="stype" bind:value={sel.type} onchange={save}>{#each SOURCE_TYPES as [k, name]}<option value={k}>{name}</option>{/each}</Select>
        </div>
      </div>
      <label for="sauth">Authors <small class="muted">one on each line, as “Last, First”. A group ends with a comma, like “World Health Organization,”</small></label>
      <textarea id="sauth" rows="3" bind:value={sel.authors} oninput={save} placeholder="Clear, James"></textarea>
      <div class="grid2">
        <div><label for="syear">Year</label><input id="syear" bind:value={sel.year} oninput={save} placeholder="2018" /></div>
        <div><label for="sstat">Status</label>
          <Select id="sstat" bind:value={sel.status} onchange={save}>{#each SOURCE_STATUS as [k, name]}<option value={k}>{name}</option>{/each}</Select>
        </div>
      </div>
      {#if isArticle || isWeb}
        <label for="scont">{isArticle ? 'Journal or magazine' : 'Website'}</label>
        <input id="scont" bind:value={sel.container} oninput={save} />
      {/if}
      {#if !isWeb}
        <div class="grid2">
          <div><label for="spub">{isArticle ? 'Publisher (optional)' : 'Publisher'}</label><input id="spub" bind:value={sel.publisher} oninput={save} /></div>
          <div><label for="sedn">Edition</label><input id="sedn" bind:value={sel.edition} oninput={save} placeholder="2nd" /></div>
        </div>
      {/if}
      {#if isArticle}
        <div class="grid3">
          <div><label for="svol">Volume</label><input id="svol" bind:value={sel.volume} oninput={save} /></div>
          <div><label for="sissue">Issue</label><input id="sissue" bind:value={sel.issue} oninput={save} /></div>
          <div><label for="spages">Pages</label><input id="spages" bind:value={sel.pages} oninput={save} placeholder="998-1009" /></div>
        </div>
      {/if}
      {#if isWeb || isArticle}
        <div class="grid2">
          <div><label for="surl">Web address</label><input id="surl" bind:value={sel.url} oninput={save} placeholder="https://…" /></div>
          <div><label for="sacc">Looked at on</label><input id="sacc" bind:value={sel.accessed} oninput={save} placeholder="2026-01-31" /></div>
        </div>
      {/if}
      <label for="snote">Notes</label>
      <textarea id="snote" bind:value={sel.note} oninput={save} placeholder="What is in it that you can use? Page numbers, quotes, what to check."></textarea>

      <p class="muted" style="margin-top:.8rem">In the text it shows as <b>{citeLabel(sel, '45')}</b>. The exported book uses the style that you choose on the Export screen.</p>

      <h3 style="margin-top:1.4rem">Cited in · {total} time{total === 1 ? '' : 's'}</h3>
      <div class="card">
        {#each here as u (u.id)}
          <button class="row-link" onclick={() => openScene(u.id)} title="Open the {word('scene')}">
            <span class="chip">{u.times}×</span><span class="t">{u.label}</span>
          </button>
        {:else}<span class="muted">Not cited yet. Select text in the Write tab and press Cite.</span>{/each}
      </div>
    {:else}
      <div class="empty">Pick a source, or press “+ New”.</div>
    {/if}
  </section>
</div>

<style>
  .head { display: flex; gap: .5rem; align-items: center; justify-content: space-between; }
  .btns { display: flex; gap: .3rem; flex-wrap: wrap; }
  .warn { border-color: var(--danger); }
  .grid3 { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: .6rem; }
  .row-link { display: flex; gap: .6rem; align-items: baseline; width: 100%; text-align: left; background: none; border: 0; border-top: 1px solid var(--line); padding: .35rem 0; border-radius: 0; }
  .row-link:first-of-type { border-top: 0; }
  .row-link:hover { background: var(--panel-2); }
  .chip { flex: 0 0 auto; font-size: .75rem; color: var(--muted); background: var(--panel-2); border: 1px solid var(--line); border-radius: 99px; padding: 0 .5rem; }
  .t { min-width: 0; overflow: hidden; text-overflow: ellipsis; }
</style>
