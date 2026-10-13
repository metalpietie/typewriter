<script lang="ts">
  import { word } from './projecttype.svelte';
  import { backlinks, type LinkKind } from './backlinks';
  import { openScene } from './nav.svelte';

  let { kind, id }: { kind: LinkKind; id: number } = $props();
  const list = $derived(backlinks(kind, id));
</script>

<h3 style="margin-top:1.4rem">Appears in</h3>
<div class="card">
  {#each list as b (b.scene.id)}
    <button class="back" onclick={() => openScene(b.scene.id)} title="Open this {word('scene')}">
      <span class="t">{b.label}</span>
      {#if b.tagged}<span class="chip">@ tagged</span>{/if}
      {#if b.named}<span class="chip">{b.named}× by name</span>{/if}
    </button>
  {:else}
    <span class="muted">Not in any {word('scene')} yet. Type <b>@</b> in a {word('scene')} to tag it, or write its name.</span>
  {/each}
</div>

<style>
  .back { display: flex; align-items: center; gap: .4rem; width: 100%; text-align: left; background: none; border: 1px solid transparent; padding: .25rem .4rem; }
  .back:hover { background: var(--panel-2); border-color: var(--line); }
  .back .t { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .back .chip { margin: 0; }
</style>
