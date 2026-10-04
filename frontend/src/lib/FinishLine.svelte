<script lang="ts">
  import { type Item } from './db.svelte';
  import { bookWords, dayText, normalPace, planOf } from './finish';

  // The plan for one book: how many words to write each day to finish by the date.
  let { book }: { book: Item } = $props();

  const plan = $derived(planOf(book));
  const pace = $derived(normalPace());
  const n = (v: number) => v.toLocaleString();
  const plural = (v: number, one: string, many: string) => `${n(v)} ${v === 1 ? one : many}`;
</script>

<div class="plan">
  {#if plan.state === 'nodate'}
    <span class="muted">Pick a finish date to see how many words to write each day.</span>
  {:else if plan.state === 'nogoal'}
    <span class="muted">Set a book goal (words) to see how many words to write each day. {plural(plan.days, 'day', 'days')} left.</span>
  {:else if plan.state === 'done'}
    <span class="good">The goal of {n(plan.goal)} words is reached. Well done.</span>
  {:else if plan.state === 'late'}
    <span class="bad">The finish date passed {plural(plan.daysLate, 'day', 'days')} ago. {n(plan.left)} words are left.</span>
    <span class="muted">Pick a new date to get a new plan.</span>
  {:else}
    <div><b>{n(plan.perDay)}</b> words a day for {plural(plan.days, 'day', 'days')}</div>
    <div class="muted">{n(plan.left)} words to go · {n(bookWords(book.id))} of {n(Number(book.goal))} written</div>
    <div class="verdict" class:good={plan.verdict === 'on'} class:bad={plan.verdict !== 'on'}>
      {#if plan.verdict === 'on'}
        On track. Your last 7 days: {n(pace)} words a day.
      {:else if plan.verdict === 'behind'}
        Behind. Your last 7 days: {n(pace)} words a day. At that pace you finish on {dayText(plan.projected!)}.
      {:else}
        You have not written in the last 7 days.
      {/if}
    </div>
  {/if}
</div>

<style>
  .plan { font-size: .9rem; display: flex; flex-direction: column; gap: .15rem; }
  .verdict { margin-top: .2rem; }
  .good { color: var(--ok); }
  .bad { color: var(--danger); }
</style>
