<script lang="ts">
  import Select from './Select.svelte';
  import { db, toHtml, type Item } from './db.svelte';
  import { nav, openItem } from './nav.svelte';
  import { openWindow } from './windows';
  import { hasKind } from './projecttype.svelte';
  import { orderedScenes, sceneLabel } from './scenes';
  import { REF_KINDS, characterFields, loreFields, placeFields, type RefKind, type RefPick } from './reference-core';

  // Something you pin next to the editor and read while you write. It only shows. To change it, press Open.
  let { pick = $bindable(), hereScene = null, onclose }: { pick: RefPick; hereScene?: number | null; onclose: () => void } = $props();

  const bookTitle = (id: number) => db.get('book', id)?.title ?? '';

  /** The items of the chosen kind, as a list for the drop-down. */
  const options = $derived.by((): { id: number; label: string }[] => {
    switch (pick.kind) {
      case 'scene': return orderedScenes().map((s) => ({ id: s.id, label: (db.lists.book.length > 1 ? bookTitle(s.book) + ', ' : '') + sceneLabel(s) }));
      case 'draft': return [...db.lists.draft].sort((a, b) => (a.order ?? a.id) - (b.order ?? b.id)).map((p) => ({ id: p.id, label: p.title || 'Untitled page' }));
      case 'character': return [...db.lists.character].sort((a, b) => String(a.name).localeCompare(String(b.name))).map((c) => ({ id: c.id, label: String(c.name || 'Unnamed') }));
      case 'location': return [...db.lists.location].sort((a, b) => String(a.name).localeCompare(String(b.name))).map((l) => ({ id: l.id, label: String(l.name || 'Unnamed') }));
      default: return [...db.lists.note].sort((a, b) => String(a.title).localeCompare(String(b.title))).map((n) => ({ id: n.id, label: String(n.title || 'Untitled') }));
    }
  });

  const item = $derived<Item | undefined>(pick.id ? db.get(pick.kind as any, pick.id) : undefined);
  const fields = $derived.by(() => {
    if (!item) return [];
    if (pick.kind === 'character') return characterFields(item);
    if (pick.kind === 'location') return placeFields(item, db.get('location', item.parent)?.name ?? '');
    if (pick.kind === 'note') return loreFields(item);
    return [];
  });
  const title = $derived(!item ? '' : pick.kind === 'scene' ? sceneLabel(item) : String(item.name ?? item.title ?? ''));

  function changeKind(k: RefKind) {
    pick = { ...pick, kind: k, id: null };
  }
  function choose(v: unknown) {
    pick = { ...pick, id: v ? Number(v) : null };
  }
</script>

<aside class="side ref" aria-label="Reference">
  <div class="head">
    <strong>Reference</strong>
    <span class="row fit" style="gap:.3rem">
      {#if item}<button class="small" title="Open it to change it" onclick={() => { db.flush(); openItem(pick.kind, item!.id); }}>Open</button>{/if}
      {#if nav.tab !== 'reference'}<button class="small" title="Open the reference in its own window, for example on a second screen" onclick={() => { openWindow('reference', 'Reference'); onclose(); }}>⧉</button>{/if}
      <button class="small" aria-label="Close the reference pane" onclick={onclose}>✕</button>
    </span>
  </div>
  <label for="refkind">Show a…</label>
  <Select id="refkind" value={pick.kind} onchange={(e) => changeKind(e.currentTarget.value as RefKind)}>
    {#each REF_KINDS.filter(([k]) => hasKind(k)) as [k, name]}<option value={k}>{name}</option>{/each}
  </Select>
  <label for="refitem">Which one</label>
  <Select id="refitem" value={pick.id ?? ''} onchange={(e) => choose(e.currentTarget.value)}>
    <option value="">— choose —</option>
    {#each options as o (o.id)}<option value={o.id}>{o.label}</option>{/each}
  </Select>

  <div class="body">
    {#if !item}
      <p class="muted">Pick something to read next to your text. It stays here while you write, and when you change scene.</p>
    {:else}
      <h3 class="rt">{title}{#if pick.kind === 'scene' && hereScene === item.id} <small class="muted">(open in the editor)</small>{/if}</h3>
      {#if pick.kind === 'scene'}
        {#if item.summary}<p class="sum"><em>{item.summary}</em></p>{/if}
        <div class="text">{@html toHtml(item.body ?? '') || '<p class="muted">No text yet.</p>'}</div>
      {:else if pick.kind === 'draft'}
        <div class="text">{@html toHtml(item.body ?? '') || '<p class="muted">Empty page.</p>'}</div>
      {:else}
        {#if pick.kind === 'character' && item.portrait}<img class="portrait" src={item.portrait} alt="" />{/if}
        {#each fields as f}
          <div class="fld"><div class="lab">{f.label}</div><div class="val">{f.text}</div></div>
        {:else}
          <p class="muted">Nothing filled in yet.</p>
        {/each}
      {/if}
    {/if}
  </div>
</aside>

<style>
  .ref { border-left: 1px solid var(--line); border-right: 0; min-width: 0; }
  .body { margin-top: .8rem; }
  .rt { margin: 0 0 .4rem; color: var(--text); text-transform: none; letter-spacing: 0; font-size: 1rem; }
  .sum { margin: 0 0 .6rem; color: var(--muted); }
  .text :global(p) { margin: 0 0 .8em; line-height: 1.6; }
  .text :global(li > p) { margin: 0; }
  .text :global(h1), .text :global(h2), .text :global(h3) { margin: 1em 0 .4em; color: var(--text); text-transform: none; letter-spacing: 0; }
  .text :global(blockquote) { margin: .8em 0; padding-left: .8rem; border-left: 3px solid var(--accent); color: var(--muted); }
  .fld { margin: 0 0 .7rem; }
  .lab { font-size: .75rem; color: var(--muted); margin-bottom: .1rem; }
  .val { white-space: pre-wrap; line-height: 1.5; }
  .portrait { width: 5.5rem; height: 5.5rem; border-radius: 50%; object-fit: cover; border: 2px solid var(--line); margin-bottom: .6rem; }
</style>
