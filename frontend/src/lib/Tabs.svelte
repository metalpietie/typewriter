<script lang="ts">
  import { onMount, tick } from 'svelte';

  // A row of pill tabs. The white pill slides to the tab that you pick.
  let { tabs, value = $bindable(), label = 'Tabs' }: {
    tabs: { id: string; label: string; badge?: number | string }[];
    value: string;
    label?: string;
  } = $props();

  let bar = $state<HTMLElement>();
  let pill = $state({ x: 0, w: 0, show: false, glide: false });

  /** Put the pill under the active tab. The first place is set without gliding. */
  function place() {
    const on = bar?.querySelector<HTMLElement>('button.on');
    if (!on) { pill.show = false; return; }
    const x = on.offsetLeft, w = on.offsetWidth;
    if (!pill.show || x !== pill.x || w !== pill.w) pill = { x, w, show: true, glide: pill.show };
  }
  $effect(() => { value; tabs.length; tabs.map((t) => t.badge).join(); tick().then(place); });
  onMount(() => {
    const watch = new ResizeObserver(place);
    if (bar) watch.observe(bar);
    return () => watch.disconnect();
  });

  function keys(e: KeyboardEvent) {
    const at = tabs.findIndex((t) => t.id === value);
    const step = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0;
    if (!step) return;
    e.preventDefault();
    const next = tabs[(at + step + tabs.length) % tabs.length];
    value = next.id;
    tick().then(() => bar?.querySelector<HTMLElement>('button.on')?.focus());
  }
</script>

<div class="tabs2" role="tablist" aria-label={label} bind:this={bar} onkeydown={keys}>
  <span class="slider" class:show={pill.show} class:glide={pill.glide} style:transform="translateX({pill.x}px)" style:width="{pill.w}px" aria-hidden="true"></span>
  {#each tabs as t (t.id)}
    <button type="button" role="tab" class:on={t.id === value} aria-selected={t.id === value} tabindex={t.id === value ? 0 : -1} onclick={() => (value = t.id)}>
      {t.label}{#if t.badge}<b>{t.badge}</b>{/if}
    </button>
  {/each}
</div>

<style>
  .tabs2 { position: relative; display: flex; gap: 2px; padding: 3px; background: var(--panel-2); border-radius: 10px; margin: .7rem 0 .9rem; }
  .tabs2:first-child { margin-top: 0; }
  .tabs2 button { position: relative; z-index: 1; flex: 1 1 auto; min-width: 0; background: none; border: 0; box-shadow: none; border-radius: 8px; padding: .3rem .45rem; font-size: .78rem; color: var(--muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .tabs2 button:hover { color: var(--text); border: 0; }
  .tabs2 button.on { color: var(--text); font-weight: 600; }
  .tabs2 button b { margin-left: .3rem; font-size: .65rem; background: var(--accent); color: var(--on-accent); border-radius: 99px; padding: 0 .35rem; font-weight: 600; }
  .tabs2 button:focus-visible { outline: 2px solid var(--accent); outline-offset: -2px; }
  .slider { position: absolute; top: 3px; bottom: 3px; left: 0; z-index: 0; background: var(--panel); border-radius: 8px; box-shadow: 0 1px 4px rgba(0, 0, 0, .18); opacity: 0; pointer-events: none; }
  .slider.show { opacity: 1; }
  .slider.glide { transition: transform var(--m-dur) var(--m-ease), width var(--m-dur) var(--m-ease); }
</style>
