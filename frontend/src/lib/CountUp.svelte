<script lang="ts">
  import { untrack } from 'svelte';
  import { countAt } from './motion-core';
  import { params } from './motion.svelte';

  // A number that counts up to its value when it first shows, and counts to the new value when it changes.
  // Use it on the Dashboard and Statistics screens. Do not use it next to the editor, where the numbers change as you type.
  let { value }: { value: number } = $props();

  let shown = $state(Math.round(Number(value) || 0)); // the start value is the real one, so nothing is wrong if the animation never runs
  let first = true;

  $effect(() => {
    const to = Math.round(Number(value) || 0);
    const p = params();
    const from = untrack(() => (first ? 0 : shown));
    first = false;
    if (!p.dur || from === to) { shown = to; return; }
    const total = p.dur * 3.5;
    const start = performance.now();
    let frame = 0;
    const step = (now: number) => {
      const t = (now - start) / total;
      shown = countAt(from, to, t);
      if (t < 1) frame = requestAnimationFrame(step);
    };
    shown = from;
    frame = requestAnimationFrame(step);
    const finish = setTimeout(() => { shown = to; }, total + 80); // the number must end right even if frames are paused
    return () => { cancelAnimationFrame(frame); clearTimeout(finish); };
  });
</script>

<span class="cu">{shown.toLocaleString()}</span>

<style>
  .cu { font-variant-numeric: tabular-nums; }
</style>
