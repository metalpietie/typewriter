<script lang="ts">
  import { onMount } from 'svelte';
  import { db } from './db.svelte';
  import { search, buildRegex, replaceIn, type SceneHits } from './findreplace';
  import { addVersion } from './versions';

  let { bookId, onjump, onchanged, onclose }: {
    bookId: number | null;
    onjump: (sceneId: number) => void;
    onchanged: () => void;
    onclose: () => void;
  } = $props();

  let query = $state('');
  let replacement = $state('');
  let matchCase = $state(false);
  let wholeWord = $state(false);
  let allBooks = $state(false);
  let message = $state('');
  let working = $state(false);
  let box: HTMLInputElement;

  onMount(() => {
    // start with the text that is selected in the editor, if any. Read it first: focusing the box clears the selection.
    const sel = window.getSelection()?.toString().trim();
    box.focus();
    if (sel && sel.length < 80 && !sel.includes('\n')) { query = sel; box.select(); }
  });

  const results = $derived<SceneHits[]>(search({ query, replacement, matchCase, wholeWord, allBooks, bookId }));
  const total = $derived(results.reduce((n, r) => n + r.hits.length, 0));

  async function apply(list: SceneHits[]) {
    const re = buildRegex({ query, matchCase, wholeWord });
    if (!re || working) return;
    working = true;
    message = '';
    try {
      db.flush(); // save what is being typed first
      let count = 0;
      for (const r of list) {
        const out = replaceIn(r.scene.body ?? '', re, replacement);
        if (!out.count) continue;
        // keep the old text, so History can bring it back
        await addVersion(r.scene.id, r.scene.body ?? '', `Before replacing “${query}” with “${replacement}”`);
        await db.save('scene', { ...r.scene, body: out.html });
        count += out.count;
      }
      onchanged();
      message = `Replaced ${count} in ${list.length} scene${list.length === 1 ? '' : 's'}. To undo: open History in that scene.`;
    } catch (e: any) {
      message = 'Could not replace: ' + (e?.message ?? e);
    } finally {
      working = false;
    }
  }
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); onclose(); } }} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="overlay" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
  <div class="palette fr">
    <div class="head"><strong>Find and replace</strong><button class="small" onclick={onclose}>Close</button></div>
    <div class="row">
      <div><label>Find</label><input bind:this={box} bind:value={query} placeholder="Text to find" onkeydown={(e) => { if (e.key === 'Enter' && e.ctrlKey) apply(results); }} /></div>
      <div><label>Replace with</label><input bind:value={replacement} placeholder="Leave empty to delete it" /></div>
    </div>
    <div class="opts">
      <label><input type="checkbox" bind:checked={matchCase} />Match case</label>
      <label><input type="checkbox" bind:checked={wholeWord} />Whole word</label>
      <label><input type="checkbox" bind:checked={allBooks} />All books</label>
      <span class="grow"></span>
      <button class="primary" disabled={!total || working} onclick={() => apply(results)}>Replace all ({total})</button>
    </div>
    {#if message}<p class="msg">{message}</p>{/if}

    <div class="results">
      {#if !query}
        <p class="muted">Type to search every scene{allBooks ? ' in all books' : ' in this book'}.</p>
      {:else if !results.length}
        <p class="muted">Nothing found.</p>
      {:else}
        <p class="muted">{total} match{total === 1 ? '' : 'es'} in {results.length} scene{results.length === 1 ? '' : 's'}.</p>
        {#each results as r (r.scene.id)}
          <div class="scene">
            <div class="head">
              <button class="link" onclick={() => { onjump(r.scene.id); onclose(); }}>{r.label}</button>
              <span class="row fit"><span class="muted">{r.hits.length}</span>
                <button class="small" disabled={working} onclick={() => apply([r])}>Replace here</button></span>
            </div>
            {#each r.hits.slice(0, 5) as h}
              <div class="snip">…{h.before}<mark>{h.match}</mark>{h.after}…</div>
            {/each}
            {#if r.hits.length > 5}<div class="muted">and {r.hits.length - 5} more</div>{/if}
          </div>
        {/each}
      {/if}
    </div>
  </div>
</div>

<style>
  .fr { width: min(760px, 94vw); max-height: 76vh; display: flex; flex-direction: column; }
  .opts { display: flex; gap: 1rem; align-items: center; margin: .6rem 0; flex-wrap: wrap; }
  .opts label { display: flex; gap: .35rem; align-items: center; margin: 0; color: var(--text); }
  .opts input { width: auto; }
  .grow { flex: 1; }
  .results { overflow: auto; min-height: 0; }
  .scene { border-top: 1px solid var(--line); padding: .5rem 0; }
  .snip { font-family: Georgia, serif; font-size: .9rem; color: var(--muted); padding: .1rem 0; white-space: pre-wrap; }
  mark { background: color-mix(in srgb, var(--accent) 35%, transparent); color: var(--text); border-radius: 3px; padding: 0 2px; }
  .link { background: none; border: 0; padding: 0; color: var(--accent); text-align: left; }
  .msg { color: var(--ok); margin: .2rem 0; }
</style>
