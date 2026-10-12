<script lang="ts">
  import { db } from './db.svelte';
  import { slideUp } from './motion.svelte';
</script>

<svelte:window
  onkeydown={(e) => {
    // Ctrl+Z undoes a delete, but only when you are not typing, so text undo still works.
    const el = e.target as HTMLElement;
    const typing = el?.closest?.('input, textarea, select, [contenteditable]');
    if ((e.ctrlKey || e.metaKey) && !e.shiftKey && e.key.toLowerCase() === 'z' && !typing && db.undoable) {
      e.preventDefault();
      db.undo();
    }
  }}
/>

{#if db.undoable}
  <div class="toast" role="status" transition:slideUp>
    <span>{db.undoable.label}</span>
    <button class="primary small" onclick={() => db.undo()}>Undo</button>
    <button class="small" aria-label="Dismiss" onclick={() => (db.undoable = null)}>✕</button>
  </div>
{/if}

<style>
  .toast {
    position: fixed; left: 0; right: 0; width: max-content; margin: 0 auto; bottom: 1.5rem; z-index: 60;
    display: flex; align-items: center; gap: .7rem; max-width: 90vw;
    background: var(--panel); border: 1px solid var(--line); border-radius: 10px;
    padding: .6rem .9rem; box-shadow: 0 10px 40px rgba(0, 0, 0, .45);
  }
  span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
