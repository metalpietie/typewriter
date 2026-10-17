<script lang="ts">
  import { pop } from './motion.svelte';
  import { db } from './db.svelte';
  import { peekOf, type PeekKind } from './peek-core';

  // The small card that shows when you point at an @ link: who or what it is, and buttons to open it.
  // at: the box of the link on the screen. The card goes under it, or over it when there is no room below.
  let { kind, id, at, onopen, onbeside, onenter, onleave }: {
    kind: PeekKind; id: number; at: DOMRect;
    onopen?: () => void; onbeside?: () => void; onenter?: () => void; onleave?: () => void;
  } = $props();

  const card = $derived(peekOf(kind, db.get(kind, id)));
  let h = $state(0);
  const W = 288;
  const left = $derived(Math.max(8, Math.min(at.left, innerWidth - W - 8)));
  const below = $derived(at.bottom + 8 + h <= innerHeight - 8);
  const top = $derived(below ? at.bottom + 6 : Math.max(8, at.top - 6 - h));
</script>

<div class="peek" bind:clientHeight={h} style:left="{left}px" style:top="{top}px" style:width="{W}px"
  role="tooltip" transition:pop={{ origin: below ? 'top left' : 'bottom left' }} onpointerenter={onenter} onpointerleave={onleave}>
  {#if card}
    <div class="pk-head">
      {#if card.color}<span class="dot" style:background={card.color}></span>{/if}
      <b>{card.title}</b>
    </div>
    {#if card.sub}<div class="pk-sub">{card.sub}</div>{/if}
    {#if card.text}<p>{card.text}</p>{:else}<p class="empty">No description yet.</p>{/if}
    {#if onopen || onbeside}
      <div class="pk-btns">
        {#if onopen}<button class="small" onclick={onopen} title="Open it in its own screen (Ctrl+click)">Open</button>{/if}
        {#if onbeside}<button class="small" onclick={onbeside} title="Show it in the pane next to the text (Alt+click)">Show beside</button>{/if}
      </div>
    {/if}
  {:else}
    <p class="empty">This was deleted. The link has nothing to show.</p>
  {/if}
</div>

<style>
  .peek {
    position: fixed; z-index: 70; background: var(--panel); border: 1px solid var(--line); border-radius: 10px;
    padding: .65rem .75rem; box-shadow: 0 12px 32px rgba(0, 0, 0, .4); font-size: .85rem; line-height: 1.45; color: var(--text);
  }
  .pk-head { display: flex; align-items: center; gap: .45rem; font-size: .95rem; }
  .dot { width: .65rem; height: .65rem; border-radius: 50%; flex: none; box-shadow: 0 0 0 2px var(--panel-2); }
  .pk-sub { color: var(--muted); font-size: .75rem; margin-top: .1rem; text-transform: capitalize; }
  p { margin: .45rem 0 0; color: var(--text); }
  .empty { color: var(--muted); font-style: italic; }
  .pk-btns { display: flex; gap: .4rem; margin-top: .6rem; }
</style>
