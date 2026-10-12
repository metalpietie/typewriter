<script lang="ts">
  import { veil, dialog } from './motion.svelte';
  import Select from './Select.svelte';
  import { onMount } from 'svelte';
  import { db, type Item } from './db.svelte';
  import { nav } from './nav.svelte';
  import { calendarRow, daysInYear, format, standard, toTime, usable, type Calendar } from './calendar';

  let row = $state<Item | null>(null);
  let weekdays = $state('');
  let yearNames = $state('');
  let year = $state(312);
  let month = $state(0);
  let day = $state(1);

  const lines = (s: string) => s.split('\n').map((l) => l.trim()).filter(Boolean);
  const asCal = $derived<Calendar | null>(row ? { months: row.months ?? [], weekdays: row.weekdays ?? [], weekdayShift: row.weekdayShift ?? 0, yearNames: row.yearNames ?? [], prefix: row.prefix ?? '', suffix: row.suffix ?? '' } : null);
  const ok = $derived(usable(asCal));
  const m = $derived(asCal && ok ? Math.min(month, asCal.months.length - 1) : 0);
  const sample = $derived(asCal && ok ? format(asCal, toTime(asCal, year, m, day)) : '');

  onMount(async () => {
    row = calendarRow() ?? (await db.save('setting', { key: 'calendar', months: [], weekdays: [], weekdayShift: 0, yearNames: [], prefix: 'Year', suffix: '' }));
    weekdays = (row.weekdays ?? []).join('\n');
    yearNames = (row.yearNames ?? []).join('\n');
  });

  const save = () => row && db.queue('setting', row);
  function useStandard() {
    if (!row) return;
    Object.assign(row, standard());
    weekdays = row.weekdays.join('\n');
    yearNames = '';
    save();
  }
  const addMonth = () => { row!.months = [...(row!.months ?? []), { name: 'New month', days: 30 }]; save(); };
  const removeMonth = (i: number) => { row!.months = row!.months.filter((_: unknown, k: number) => k !== i); save(); };
  function move(i: number, by: number) {
    const list = [...row!.months];
    const j = i + by;
    if (j < 0 || j >= list.length) return;
    [list[i], list[j]] = [list[j], list[i]];
    row!.months = list;
    save();
  }
  const close = () => { db.flush(); nav.calendar = false; };
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') close(); }} />

<div class="overlay" transition:veil|global role="presentation" onmousedown={(e) => { if (e.target === e.currentTarget) close(); }}>
  <div transition:dialog|global class="palette box" role="dialog" aria-label="Calendar">
    <div class="head"><h2 style="margin:0">Your calendar</h2><button class="small" onclick={close}>Close</button></div>
    {#if !row}
      <p class="muted">Loading…</p>
    {:else}
      <p class="muted" style="margin-top:0">
        Set your own months and days. The date fields in scenes, events and path steps then use them.
        {#if !row.months?.length}Start from a standard calendar, or add your own months below.{/if}
      </p>

      <div class="head">
        <h3 style="margin:0">Months{ok ? ` · ${daysInYear(asCal!)} days in a year` : ''}</h3>
        <span class="fit">
          <button class="small" onclick={useStandard}>Use a standard calendar</button>
          <button class="small" onclick={addMonth}>+ Month</button>
        </span>
      </div>
      {#each row.months ?? [] as mo, i (i)}
        <div class="mrow">
          <input bind:value={mo.name} oninput={save} aria-label="Month name" />
          <input type="number" min="1" step="1" bind:value={mo.days} oninput={save} aria-label="Days in this month" />
          <span class="muted">days</span>
          <button class="small" title="Move up" disabled={i === 0} onclick={() => move(i, -1)}>↑</button>
          <button class="small" title="Move down" disabled={i === row.months.length - 1} onclick={() => move(i, 1)}>↓</button>
          <button class="small danger" title="Remove" onclick={() => removeMonth(i)}>✕</button>
        </div>
      {/each}

      <div class="grid2">
        <div><label>Before the year number</label><input bind:value={row.prefix} oninput={save} placeholder="Year" /></div>
        <div><label>After the year number</label><input bind:value={row.suffix} oninput={save} placeholder="AE" /></div>
      </div>
      <div class="grid2">
        <div>
          <label>Weekday names (one per line, can be empty)</label>
          <textarea style="min-height:6rem" bind:value={weekdays} oninput={() => { row!.weekdays = lines(weekdays); save(); }}></textarea>
        </div>
        <div>
          <label>Year names that repeat (one per line, can be empty)</label>
          <textarea style="min-height:6rem" bind:value={yearNames} oninput={() => { row!.yearNames = lines(yearNames); save(); }} placeholder={'Dragon\nOwl\nWolf'}></textarea>
        </div>
      </div>
      {#if row.weekdays?.length}
        <label>First day of year 0 is a</label>
        <Select bind:value={row.weekdayShift} onchange={save}>
          {#each row.weekdays as w, i}<option value={i}>{w}</option>{/each}
        </Select>
      {/if}

      {#if ok}
        <div class="card" style="margin-bottom:0">
          <h3>Try it</h3>
          <div class="try">
            <input type="number" step="1" bind:value={year} aria-label="Year" />
            <Select bind:value={month} aria-label="Month">{#each asCal!.months as mo, i}<option value={i}>{mo.name}</option>{/each}</Select>
            <input type="number" min="1" step="1" bind:value={day} aria-label="Day" />
          </div>
          <p style="margin:.6rem 0 0"><b>{sample}</b></p>
          <p class="muted" style="margin:.2rem 0 0;font-size:.8rem">Timeline number: {toTime(asCal!, year, m, day)}</p>
        </div>
      {/if}
      <p class="muted" style="font-size:.8rem;margin-bottom:0">
        Changing the months later does not change the timeline numbers you already saved, so a date can then read differently. There are no leap years.
      </p>
    {/if}
  </div>
</div>

<style>
  .box { width: min(640px, 94vw); max-height: 80vh; overflow: auto; }
  .mrow { display: grid; grid-template-columns: 1fr 5rem auto auto auto auto; gap: .4rem; align-items: center; margin: .25rem 0; }
  .try { display: grid; grid-template-columns: 6rem 1fr 5rem; gap: .4rem; }
</style>
