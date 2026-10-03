<script lang="ts">
  import { nav } from './nav.svelte';
  import { calendar, format, fromTime, monthLength, toTime } from './calendar';

  let { time = $bindable(), when = $bindable(), onchange, placeholder = 'e.g. Year 312, dawn', timeLabel = 'Timeline position (number, e.g. year)' }: {
    time: number | null | undefined;
    when: string | undefined;
    onchange: () => void;
    placeholder?: string;
    timeLabel?: string;
  } = $props();

  const cal = $derived(calendar());
  const has = $derived(time !== null && time !== undefined && (time as unknown) !== '' && Number.isFinite(Number(time)));
  const p = $derived(cal && has ? fromTime(cal, Number(time)) : null);

  /** Set the date. The label follows the date, unless you wrote your own label. */
  function set(year: number | null, month: number | null, day: number | null) {
    const old = cal && has ? format(cal, Number(time)) : '';
    const follows = !when || when === old;
    if (year === null || !Number.isFinite(year) || !cal) { time = null; if (follows) when = ''; onchange(); return; }
    year = Math.trunc(year);
    time = month === null ? year : toTime(cal, year, month, day ?? 1);
    if (follows) when = format(cal, time);
    onchange();
  }
  const num = (v: string) => (v.trim() === '' ? null : Number(v));
</script>

{#if cal}
  <label>Date</label>
  <div class="date">
    <input type="number" step="1" placeholder="Year" aria-label="Year" value={p?.year ?? ''} oninput={(e) => set(num(e.currentTarget.value), p?.month ?? null, p?.day ?? null)} />
    <select aria-label="Month" disabled={!p} value={p?.month ?? ''} onchange={(e) => { const v = e.currentTarget.value; set(p!.year, v === '' ? null : +v, v === '' ? null : p?.day ?? 1); }}>
      <option value="">— whole year —</option>
      {#each cal.months as m, i}<option value={i}>{m.name}</option>{/each}
    </select>
    {#if p && p.month !== null}
      <input type="number" min="1" max={monthLength(cal.months[p.month])} step="1" aria-label="Day" value={p.day ?? 1}
        oninput={(e) => { const d = num(e.currentTarget.value); if (d !== null) set(p.year, p.month, d); }} />
    {/if}
  </div>
  <label>Label (shown in lists)</label>
  <input bind:value={when} oninput={onchange} {placeholder} />
  <div class="muted note">{has ? `Timeline number: ${time}` : 'Pick a year to place it on the timeline.'}</div>
{:else}
  <label>When (story time)</label><input bind:value={when} oninput={onchange} {placeholder} />
  <label>{timeLabel}</label><input type="number" step="any" bind:value={time} oninput={onchange} />
  <div class="muted note"><button class="link" onclick={() => (nav.calendar = true)}>Set up your own calendar</button> to pick dates with your own months.</div>
{/if}

<style>
  .date { display: grid; grid-template-columns: 5.5rem minmax(0, 1fr) auto; gap: .4rem; }
  .date input[aria-label='Day'] { width: 4.5rem; }
  .note { font-size: .75rem; margin-top: .2rem; }
  .link { background: none; border: 0; padding: 0; color: var(--accent); text-decoration: underline; font-size: inherit; }
</style>
