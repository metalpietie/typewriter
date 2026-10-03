<script lang="ts">
  import { db } from './db.svelte';

  const state = $derived(db.saveState);
  const time = $derived(db.lastSaved?.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) ?? '');
</script>

<span class="save" class:saving={state === 'saving'} class:error={state === 'error'} role="status" aria-live="polite">
  <span class="dot"></span>
  {#if state === 'saving'}Saving…
  {:else if state === 'error'}Not saved
    <button class="small" onclick={() => db.retry()}>Retry</button>
  {:else}Saved{time ? ' ' + time : ''}{/if}
</span>

<style>
  .save { display: inline-flex; align-items: center; gap: .4rem; font-size: .8rem; color: var(--muted); white-space: nowrap; }
  .dot { width: .55rem; height: .55rem; border-radius: 50%; background: var(--ok); }
  .saving .dot { background: var(--accent); animation: pulse 1s ease-in-out infinite; }
  .error { color: var(--danger); }
  .error .dot { background: var(--danger); }
  @keyframes pulse { 50% { opacity: .3; } }
</style>
