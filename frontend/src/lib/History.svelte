<script lang="ts">
  import { word } from './projecttype.svelte';
  import { veil, dialog } from './motion.svelte';
  import { onMount } from 'svelte';
  import { toHtml, wordCount } from './db.svelte';
  import { listVersions, addVersion, deleteVersion, type Version } from './versions';
  import ConfirmButton from './ConfirmButton.svelte';
  import { blockText } from './diff';
  import { diffWords, countChanges, paragraphsOf } from './diff-core';

  let { sceneId, title, current, onrestore, onclose }: {
    sceneId: number;
    title: string;
    current: () => string; // the text as it is now
    onrestore: (body: string) => void | Promise<void>;
    onclose: () => void;
  } = $props();

  let versions = $state<Version[]>([]);
  let picked = $state<Version | null>(null);
  let label = $state('');
  let error = $state('');
  let loaded = $state(false);

  async function load() {
    try { versions = await listVersions(sceneId); } catch (e: any) { error = e?.message ?? String(e); }
    loaded = true;
    if (picked) picked = versions.find((v) => v.id === picked!.id) ?? null;
  }
  onMount(load);

  const when = (iso: string) => new Date(iso).toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' });
  const nowWords = $derived(wordCount(current()));
  const delta = (v: Version) => { const d = v.words - nowWords; return d === 0 ? 'same length as now' : `${Math.abs(d).toLocaleString()} words ${d > 0 ? 'more' : 'fewer'} than now`; };

  // What changed: the words added (green) and cut (red) between two texts.
  let mode = $state<'read' | 'changes'>('read');
  let against = $state<'now' | 'before'>('now'); // compare the picked copy with the scene now, or with the copy before it
  const older = $derived(picked ? versions[versions.findIndex((v) => v.id === picked!.id) + 1] ?? null : null);
  const pieces = $derived.by(() => {
    if (!picked || mode !== 'changes') return null;
    if (against === 'now') return diffWords(blockText(picked.body), blockText(current()));
    return older ? diffWords(blockText(older.body), blockText(picked.body)) : null;
  });
  const counts = $derived(pieces ? countChanges(pieces) : null);
  const plural = (n: number) => `${n.toLocaleString()} word${n === 1 ? '' : 's'}`;

  async function saveNow() {
    const body = current();
    if (!body.trim()) { error = 'The scene is empty. Nothing to save.'; return; }
    error = '';
    await addVersion(sceneId, body, label.trim() || 'Saved by me');
    label = '';
    await load();
  }
  async function restore() {
    if (!picked) return;
    const body = current();
    if (body.trim() && body !== picked.body) await addVersion(sceneId, body, 'Before going back');
    await onrestore(picked.body);
    onclose();
  }
  async function remove(v: Version) {
    await deleteVersion(v.id);
    if (picked?.id === v.id) picked = null;
    await load();
  }
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); onclose(); } }} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="overlay" transition:veil|global onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
  <div transition:dialog|global class="palette hist">
    <div class="head"><strong>History: {title || 'this ' + word('scene')}</strong><button class="small" onclick={onclose}>Close</button></div>
    <div class="row">
      <input bind:value={label} placeholder="Name for a copy (optional)" onkeydown={(e) => { if (e.key === 'Enter') saveNow(); }} />
      <button class="primary fit" onclick={saveNow}>Save a copy now</button>
    </div>
    {#if error}<p class="err">{error}</p>{/if}
    <p class="muted">The app also keeps a copy of the old text each time you start writing, at most every 10 minutes.</p>

    <div class="cols">
      <div class="list">
        {#each versions as v (v.id)}
          <button class="item" class:on={picked?.id === v.id} onclick={() => (picked = v)}>
            <span>{when(v.at)}</span>
            <small>{v.words.toLocaleString()} words{v.label ? ' · ' + v.label : v.auto ? ' · auto' : ''}</small>
          </button>
        {:else}
          <p class="muted">{loaded ? 'No old copies yet.' : 'Loading…'}</p>
        {/each}
      </div>
      <div class="view">
        {#if picked}
          <div class="head">
            <span class="muted">{when(picked.at)} · {delta(picked)}</span>
            <span class="row fit">
              <button class="primary small" onclick={restore}>Go back to this</button>
              <ConfirmButton onconfirm={() => remove(picked!)} />
            </span>
          </div>
          <div class="modes">
            <button class:active={mode === 'read'} onclick={() => (mode = 'read')}>Read</button>
            <button class:active={mode === 'changes'} onclick={() => (mode = 'changes')}>What changed</button>
            {#if mode === 'changes'}
              <span class="gap"></span>
              <button class:active={against === 'now'} onclick={() => (against = 'now')}>This copy → now</button>
              <button class:active={against === 'before'} onclick={() => (against = 'before')}>Copy before → this copy</button>
            {/if}
          </div>
          {#if mode === 'read'}
            <div class="preview">{@html toHtml(picked.body)}</div>
          {:else if !pieces}
            <p class="muted">This is the oldest copy. There is no copy before it to compare with.</p>
          {:else}
            <p class="muted legend">
              {#if against === 'now'}Since this copy, you{:else}In this copy, you{/if}
              <span class="add">added {plural(counts!.added)}</span> and <span class="del">cut {plural(counts!.cut)}</span>.
            </p>
            {#if !counts!.added && !counts!.cut}
              <p class="muted">The words are the same.</p>
            {/if}
            <div class="preview diff">
              {#each paragraphsOf(pieces) as para}
                <p>{#each para as pc}<span class={pc.kind === 'same' ? '' : pc.kind}>{pc.text}</span>{' '}{/each}</p>
              {/each}
            </div>
          {/if}
        {:else}
          <p class="muted">Pick a copy to read it.</p>
        {/if}
      </div>
    </div>
  </div>
</div>

<style>
  .hist { width: min(900px, 94vw); height: 76vh; display: flex; flex-direction: column; }
  .cols { display: grid; grid-template-columns: 250px 1fr; gap: .8rem; flex: 1; min-height: 0; }
  .list { overflow: auto; display: flex; flex-direction: column; gap: 3px; }
  .list .item { display: flex; flex-direction: column; align-items: flex-start; text-align: left; }
  .view { overflow: auto; border-left: 1px solid var(--line); padding-left: .8rem; }
  .preview { font-family: Georgia, 'Times New Roman', serif; line-height: 1.7; }
  .preview :global(p) { margin: 0 0 .8em; }
  .preview :global(li > p) { margin: 0; }
  .preview :global(hr)  { border: 0; text-align: center; }
  .preview :global(hr::after) { content: '* * *'; color: var(--muted); letter-spacing: .5em; }
  .modes { display: flex; gap: .3rem; flex-wrap: wrap; align-items: center; margin: .5rem 0; }
  .modes button { font-size: .8rem; padding: .2rem .6rem; }
  .gap { width: 1px; height: 1.2rem; background: var(--line); margin: 0 .3rem; }
  .legend { margin: .2rem 0 .6rem; }
  .add { background: color-mix(in srgb, var(--ok) 28%, transparent); border-bottom: 2px solid var(--ok); border-radius: 3px; padding: 0 .15em; }
  .del { background: color-mix(in srgb, var(--danger, #d9625b) 24%, transparent); text-decoration: line-through; border-radius: 3px; padding: 0 .15em; }
  .err { color: var(--danger, #d9625b); margin: .2rem 0; }
</style>
