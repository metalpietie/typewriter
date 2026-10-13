<script lang="ts">
  import { veil, dialog } from './motion.svelte';
  import { tick } from 'svelte';
  import { db, KINDS } from './db.svelte';
  import { hasKind, word, isNonfiction } from './projecttype.svelte';
  import { nav, openItem } from './nav.svelte';
  import { search } from './search';

  let query = $state('');
  let active = $state(0);
  let input = $state<HTMLInputElement>();
  let list = $state<HTMLElement>();

  // a kind that belongs to a tool that is switched off is not searched
  const lists = $derived(Object.fromEntries(KINDS.map((k) => [k, hasKind(k) ? db.lists[k] : []])) as typeof db.lists);
  const hits = $derived(search(query, lists, 30, word('scene')));
  $effect(() => { query; active = 0; });
  $effect(() => { if (nav.searchOpen) tick().then(() => input?.focus()); else query = ''; });

  function key(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') { active = Math.min(hits.length - 1, active + 1); e.preventDefault(); }
    else if (e.key === 'ArrowUp') { active = Math.max(0, active - 1); e.preventDefault(); }
    else if (e.key === 'Enter' && hits[active]) { openItem(hits[active].kind, hits[active].id); }
    else if (e.key === 'Escape') nav.searchOpen = false;
    tick().then(() => list?.querySelector('.on')?.scrollIntoView({ block: 'nearest' }));
  }
</script>

{#if nav.searchOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="overlay" transition:veil|global onclick={() => (nav.searchOpen = false)}>
    <div transition:dialog|global class="palette" onclick={(e) => e.stopPropagation()}>
      <input bind:this={input} bind:value={query} onkeydown={key} placeholder={isNonfiction() ? `Search your ${word('scenes')}, notes and glossary…` : 'Search your scenes, notes, characters, places, lore, glossary and arcs…'} />
      <div class="results" bind:this={list}>
        {#each hits as h, i (h.kind + h.id)}
          <button class="hit" class:on={i === active} onmouseenter={() => (active = i)} onclick={() => openItem(h.kind, h.id)}>
            <span class="hit-title">{h.title}<small>{h.sub}</small></span>
            {#if h.snippet}<span class="hit-snip">{h.snippet}</span>{/if}
          </button>
        {:else}
          <div class="empty" style="padding:1.2rem">{query ? 'Nothing found.' : 'Type to search. Typos are fine.'}</div>
        {/each}
      </div>
    </div>
  </div>
{/if}
