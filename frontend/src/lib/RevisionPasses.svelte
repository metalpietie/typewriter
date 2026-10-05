<script lang="ts">
  import { passesOf, scenesInBook, addPass, addUsualPasses, renamePass, movePass, removePass } from './passes';
  import { progress } from './passes-core';
  import ConfirmButton from './ConfirmButton.svelte';

  // The list of revision passes of one book, for the book panel in the Write tab.
  let { bookId }: { bookId: number } = $props();

  let text = $state('');
  const passes = $derived(passesOf(bookId));
  const scenes = $derived(scenesInBook(bookId));

  async function add() {
    const ok = await addPass(bookId, text);
    if (ok) text = '';
  }
  function rename(e: Event, p: any) {
    const input = e.currentTarget as HTMLInputElement;
    input.value = renamePass(p, input.value);
  }
</script>

<details class="passes">
  <summary>Revision passes{passes.length ? ' · ' + passes.length : ''}</summary>
  <p class="muted note">A pass is one round of revising: plot, dialogue, line edit. Tick a scene in the Scene panel when a pass is done for it. The Board shows what is left.</p>

  {#each passes as p, i (p.id)}
    {@const pr = progress(scenes, p.id)}
    <div class="p">
      <input class="nm" value={p.name} maxlength="40" aria-label="Name of the pass" onchange={(e) => rename(e, p)} />
      <span class="muted cnt" title="Scenes done in this pass">{pr.done}/{pr.total}</span>
      <button class="small" title="Move up" disabled={i === 0} onclick={() => movePass(bookId, p.id, -1)}>↑</button>
      <button class="small" title="Move down" disabled={i === passes.length - 1} onclick={() => movePass(bookId, p.id, 1)}>↓</button>
      <ConfirmButton label="✕" onconfirm={() => removePass(p)} />
    </div>
  {/each}

  <div class="row" style="margin-top:.4rem">
    <input bind:value={text} maxlength="40" placeholder="New pass, for example: Plot" aria-label="New pass"
      onkeydown={(e) => { if (e.key === 'Enter') add(); }} />
    <button class="small fit" onclick={add}>Add</button>
  </div>
  {#if !passes.length}
    <button class="small" style="margin-top:.4rem" onclick={() => addUsualPasses(bookId)}>Add the usual passes</button>
    <div class="muted note">Plot, Characters, Dialogue, Line edit, Proofread.</div>
  {/if}
</details>

<style>
  .passes { margin-top: 1rem; }
  summary { cursor: pointer; font-weight: 600; margin-bottom: .3rem; }
  .note { font-size: .8rem; margin: 0 0 .5rem; }
  .p { display: flex; align-items: center; gap: .25rem; margin: .2rem 0; }
  .nm { flex: 1; min-width: 0; padding: .25rem .4rem; font-size: .85rem; }
  .cnt { font-size: .75rem; min-width: 2.2rem; text-align: right; }
  .p button { padding: .1rem .4rem; min-width: 0; }
</style>
