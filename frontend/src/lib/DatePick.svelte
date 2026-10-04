<script lang="ts">
  import Select from './Select.svelte';
  import { nav } from './nav.svelte';
  import { calendar, fromTime, monthLength, timeText, toTime } from './calendar';

  // A date on its own, without a label text. Use it for a birth date, or to pick a day to look at.
  // The value is a timeline number: the same number the scenes and events use.
  let { value = $bindable(), label, onchange, hint = '' }: {
    value: number | null | undefined;
    label: string;
    onchange: () => void;
    hint?: string;
  } = $props();

  const cal = $derived(calendar());
  const has = $derived(value !== null && value !== undefined && (value as unknown) !== '' && Number.isFinite(Number(value)));
  const p = $derived(cal && has ? fromTime(cal, Number(value)) : null);
  const num = (v: string) => (v.trim() === '' ? null : Number(v));

  function set(year: number | null, month: number | null, day: number | null) {
    if (year === null || !Number.isFinite(year)) { value = null; onchange(); return; }
    year = Math.trunc(year);
    value = cal && month !== null ? toTime(cal, year, month, day ?? 1) : year;
    onchange();
  }
</script>

<div class="dp">
  <span class="lab">{label}</span>
  {#if cal}
    <div class="date">
      <input type="number" step="1" placeholder="Year" aria-label="{label}: year" value={p?.year ?? ''} oninput={(e) => set(num(e.currentTarget.value), p?.month ?? null, p?.day ?? null)} />
      <Select aria-label="{label}: month" disabled={!p} value={p?.month ?? ''} onchange={(e) => { const v = e.currentTarget.value; set(p!.year, v === '' ? null : +v, v === '' ? null : p?.day ?? 1); }}>
        <option value="">— whole year —</option>
        {#each cal.months as m, i}<option value={i}>{m.name}</option>{/each}
      </Select>
      {#if p && p.month !== null}
        <input type="number" min="1" max={monthLength(cal.months[p.month])} step="1" aria-label="{label}: day" value={p.day ?? 1}
          oninput={(e) => { const d = num(e.currentTarget.value); if (d !== null) set(p.year, p.month, d); }} />
      {/if}
      {#if has}<button class="small" title="Clear the date" aria-label="Clear {label}" onclick={() => set(null, null, null)}>✕</button>{/if}
    </div>
    <div class="muted note">{has ? timeText(value) : hint}</div>
  {:else}
    <div class="date plain">
      <input type="number" step="any" placeholder="Year (a number)" aria-label={label} bind:value={value} oninput={onchange} />
      {#if has}<button class="small" title="Clear the date" aria-label="Clear {label}" onclick={() => { value = null; onchange(); }}>✕</button>{/if}
    </div>
    <div class="muted note">{hint || ''} <button class="link" onclick={() => (nav.calendar = true)}>Set up your own calendar</button> to pick dates with your own months.</div>
  {/if}
</div>

<style>
  .lab { display: block; font-size: .75rem; color: var(--muted); margin: .7rem 0 .2rem; }
  .date { display: grid; grid-template-columns: 5.5rem minmax(0, 1fr) auto auto; gap: .4rem; align-items: center; }
  .date.plain { grid-template-columns: minmax(0, 1fr) auto; }
  .date input[aria-label$='day'] { width: 4.5rem; }
  .note { font-size: .75rem; margin-top: .2rem; }
  .link { background: none; border: 0; padding: 0; color: var(--accent); text-decoration: underline; font-size: inherit; }
</style>
