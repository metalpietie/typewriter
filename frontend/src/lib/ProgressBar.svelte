<script lang="ts">
  let { value, goal, thin = false }: { value: number; goal: number; thin?: boolean } = $props();
  const pct = $derived(goal > 0 ? Math.min(100, (value / goal) * 100) : 0);
  const done = $derived(goal > 0 && value >= goal);
</script>

{#if goal > 0}
  <div class="bar" class:thin role="progressbar" aria-valuemin="0" aria-valuemax={goal} aria-valuenow={Math.min(value, goal)}>
    <div class="fill" class:done style:width="{pct}%"></div>
  </div>
{/if}

<style>
  .bar { height: 8px; background: var(--panel-2); border: 1px solid var(--line); border-radius: 99px; overflow: hidden; width: 100%; }
  .bar.thin { height: 4px; border: 0; }
  .fill { height: 100%; background: var(--accent); border-radius: 99px; transition: width calc(var(--m-dur) * 2.2) var(--m-ease); }
  .fill.done { background: var(--ok); }
</style>
