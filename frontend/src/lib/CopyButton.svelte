<script lang="ts">
  import { nav } from './nav.svelte';
  // Opens the window that copies this item (and others you tick) to another shelf.
  let { kind, id }: { kind: string; id: number } = $props();
</script>

<button class="small" title="Copy this, and other items you pick, to another shelf (for example a book with its own world)" onclick={() => (nav.copy = { kind, id })}>Copy to…</button>
