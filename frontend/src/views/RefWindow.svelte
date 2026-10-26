<script lang="ts">
  import ReferencePane from '../lib/ReferencePane.svelte';
  import { cleanPick, type RefPick } from '../lib/reference-core';

  // The reference pane on its own, in a second window: something to read next to the text on another screen.
  const KEY = 'typewriter-reference-window';
  let pick = $state<RefPick>(cleanPick((() => { try { return JSON.parse(localStorage.getItem(KEY) ?? 'null'); } catch { return null; } })()));
  $effect(() => { try { localStorage.setItem(KEY, JSON.stringify(pick)); } catch { /* fine without it */ } });
</script>

<div class="refwin">
  <ReferencePane bind:pick onclose={() => window.close()} />
</div>

<style>
  .refwin { height: 100%; min-height: 0; display: flex; }
  .refwin :global(.ref) { flex: 1; max-width: 52rem; margin: 0 auto; border-right: 0; }
</style>
