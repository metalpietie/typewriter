<script lang="ts">
  import ProgressBar from './ProgressBar.svelte';
  import { mergeBars, type Level, type Scope } from './goals-core';
  import { word } from './projecttype.svelte';

  // The word goals of the book, the chapter and the scene. A level with no goal has no bar.
  // Levels that have the same goal share one bar.
  let { levels }: { levels: Level[] } = $props();
  const bars = $derived(mergeBars(levels));
  const name = (s: Scope) => (s === 'book' ? 'Book' : s === 'chapter' ? 'Chapter' : word('Scene'));
  const pct = (n: number, g: number) => Math.round((n / g) * 100);
</script>

{#if bars.length}
  <div class="goals">
    {#each bars as b (b.scopes.join('+') + b.goal)}
      <div class="goal">
        <div class="line"><span>{b.scopes.map(name).join(' + ')}</span><span class="muted">{b.words.toLocaleString()} / {b.goal.toLocaleString()} words ({pct(b.words, b.goal)}%)</span></div>
        <ProgressBar value={b.words} goal={b.goal} />
      </div>
    {/each}
  </div>
{/if}

<style>
  .goals { display: flex; flex-direction: column; gap: .5rem; margin-top: .6rem; }
  .line { display: flex; justify-content: space-between; gap: 1rem; font-size: .78rem; margin-bottom: .2rem; }
</style>
