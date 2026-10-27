<script lang="ts">
  import { db } from './db.svelte';

  const state = $derived(db.saveState);
  const time = $derived(db.lastSaved?.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) ?? '');
</script>

<span class="save" class:saving={state === 'saving'} class:error={state === 'error'} role="status" aria-live="polite">
  {#key db.lastSaved}<span class="dot ping"></span>{/key}
  {#if state === 'saving'}Saving…
  {:else if state === 'error'}Not saved
    <button class="small" onclick={() => db.retry()}>Retry</button>
  {:else}Saved{time ? ' ' + time : ''}{/if}
</span>

<style>
  .save { display: inline-flex; align-items: center; gap: .4rem; font-size: .8rem; color: var(--muted); white-space: nowrap; }
  .dot { width: 8px; height: 8px; min-width: 8px; min-height: 8px; display: inline-block; aspect-ratio: 1; border-radius: 50%; flex: 0 0 auto; vertical-align: middle; background: var(--ok); }
  .saving .dot { background: var(--accent); animation: pulse 1s ease-in-out infinite; }
  .error { color: var(--danger); }
  .error .dot { background: var(--danger); }
  .save:not(.saving):not(.error) .ping { animation: ping calc(var(--m-dur) * 3) var(--m-ease); }
  @keyframes ping { from { box-shadow: 0 0 0 0 color-mix(in srgb, var(--ok) 55%, transparent); } to { box-shadow: 0 0 0 .55rem transparent; } }
  @keyframes pulse { 50% { opacity: .3; } }
</style>
