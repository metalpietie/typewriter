<script lang="ts">
  import { db } from './db.svelte';
  import { tagsOf, withTag, withoutTag, tagCounts, MAX_TAG } from './tags-core';

  // The tags of one scene. onchange: the scene was changed, save it.
  let { scene, onchange }: { scene: any; onchange: () => void } = $props();

  let text = $state('');
  const mine = $derived(tagsOf(scene));
  // tags used in other scenes, so you can tap one instead of typing it again
  const known = $derived(tagCounts(db.lists.scene).map(([t]) => t).filter((t) => !mine.includes(t)));

  function add(raw: string) {
    const next = withTag(mine, raw);
    text = '';
    if (next === mine) return;
    scene.tags = next;
    onchange();
  }
  function remove(tag: string) { scene.tags = withoutTag(mine, tag); onchange(); }

  function onkey(e: KeyboardEvent) {
    if (e.key === 'Enter' || e.key === ',') { e.preventDefault(); add(text); }
    else if (e.key === 'Backspace' && !text && mine.length) remove(mine[mine.length - 1]);
  }
</script>

<div class="tags">
  {#each mine as t (t)}
    <span class="chip">{t}<button class="x" title="Take off the tag {t}" aria-label="Take off the tag {t}" onclick={() => remove(t)}>✕</button></span>
  {/each}
  <input class="new" bind:value={text} maxlength={MAX_TAG} placeholder={mine.length ? 'Add a tag' : 'For example: flashback'} aria-label="Add a tag"
    onkeydown={onkey} onblur={() => add(text)} />
</div>
{#if known.length}
  <div class="known">
    {#each known.slice(0, 12) as t (t)}<button class="small" title="Add the tag {t}" onclick={() => add(t)}>+ {t}</button>{/each}
  </div>
{/if}

<style>
  .tags { display: flex; flex-wrap: wrap; align-items: center; gap: .3rem; }
  .tags .chip { margin: 0; display: inline-flex; align-items: center; gap: .2rem; }
  .x { background: none; border: 0; padding: 0 .1rem; font-size: .7rem; color: var(--muted); line-height: 1; min-width: 0; }
  .x:hover { color: var(--danger); }
  .new { flex: 1 1 7rem; min-width: 6rem; width: auto; padding: .2rem .4rem; font-size: .85rem; }
  .known { display: flex; flex-wrap: wrap; gap: .25rem; margin-top: .35rem; }
  .known button { font-size: .72rem; padding: .05rem .45rem; border-radius: 99px; color: var(--muted); }
</style>
