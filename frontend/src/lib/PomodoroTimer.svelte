<script lang="ts">
  import { pomo, clock, progress, started, start, pause, skip, reset, dismiss, setSettings, resetSettings, todaySummary, clearToday } from './pomodoro.svelte';
  import { LABEL, LIMITS, DEFAULTS } from './pomodoro-core';

  // The timer in the menu. While it runs (or waits for you after a part), a card shows above Search.
  // When it does not run, a plain button shows above the data file. Both open the panel with all controls and the settings.
  let open = $state(false);
  let box: HTMLElement | undefined = $state();

  const sum = $derived(todaySummary());
  const label = $derived(pomo.phase === 'work' ? `Work · round ${pomo.round} of ${pomo.settings.rounds}` : LABEL[pomo.phase]);
  const words = $derived(Math.max(0, pomo.words));
  const changed = $derived(JSON.stringify(pomo.settings) !== JSON.stringify(DEFAULTS));
  const active = $derived(started() || pomo.alert);
  const rest = $derived(pomo.phase !== 'work');

  const fields = [
    { key: 'work', label: 'Work (minutes)' },
    { key: 'short', label: 'Short break (minutes)' },
    { key: 'long', label: 'Long break (minutes)' },
    { key: 'rounds', label: 'Rounds before the long break' },
  ] as const;

  // A number is saved when you leave the box, so you can clear it and type a new one. A wrong number is put right.
  function commit(key: (typeof fields)[number]['key'], e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    setSettings({ [key]: input.value });
    input.value = String(pomo.settings[key]);
  }
</script>

<svelte:window
  onclick={(e) => { if (open && box && !box.contains(e.target as Node)) open = false; }}
  onkeydown={(e) => { if (open && e.key === 'Escape') { e.stopPropagation(); open = false; } }} />

<div class="pomo" class:active bind:this={box}>
  {#if active}
    <div class="mini" class:rest class:done={pomo.alert} class:run={pomo.running}>
      <button class="mhead" aria-expanded={open} title="Open the timer" onclick={() => (open = !open)}>
        <span class="dotp"></span>{label}
      </button>
      <div class="mtime">{clock()}</div>
      <div class="bar"><span style:width="{Math.round(progress() * 100)}%"></span></div>
      {#if pomo.alert}<p class="mmsg">{pomo.message}</p>{/if}
      <div class="mctl">
        {#if pomo.running}
          <button class="primary" onclick={pause}>Pause</button>
        {:else}
          <button class="primary" onclick={start}>{pomo.leftMs !== null ? 'Continue' : 'Start'}</button>
        {/if}
        {#if pomo.alert && !started()}
          <button onclick={dismiss}>Dismiss</button>
        {:else}
          <button onclick={skip} title="End this part now and go to the next one">Skip</button>
        {/if}
      </div>
    </div>
  {:else}
    <button class="pbtn" aria-expanded={open} onclick={() => (open = !open)}>Pomodoro timer</button>
  {/if}

  {#if open}
    <div class="panel" role="dialog" aria-label="Pomodoro timer">
      <div class="lbl">{label}</div>
      <div class="time" class:rest>{clock()}</div>
      <div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(progress() * 100)}>
        <span style:width="{Math.round(progress() * 100)}%"></span>
      </div>

      <div class="ctl">
        {#if pomo.running}
          <button class="primary" onclick={pause}>Pause</button>
        {:else}
          <button class="primary" onclick={start}>{pomo.leftMs !== null ? 'Continue' : 'Start'}</button>
        {/if}
        <button onclick={skip} title="End this part now and go to the next one">Skip</button>
        <button onclick={reset} disabled={!started() && pomo.round === 1 && pomo.phase === 'work'} title="Stop and go back to round 1">Reset</button>
      </div>

      {#if pomo.message}<p class="msg">{pomo.message}</p>{/if}
      {#if pomo.phase === 'work' && started()}<p class="muted note">This round: {words.toLocaleString()} word{words === 1 ? '' : 's'}</p>{/if}

      <p class="muted note">
        Today: {sum.count} round{sum.count === 1 ? '' : 's'}, {sum.words.toLocaleString()} word{sum.words === 1 ? '' : 's'}, {sum.minutes} minute{sum.minutes === 1 ? '' : 's'} of work.
        {#if sum.count}<button class="link" onclick={clearToday}>Clear</button>{/if}
      </p>

      <details>
        <summary>Settings</summary>
        <div class="set">
          {#each fields as f}
            <label for="pomo-{f.key}">{f.label}</label>
            <input id="pomo-{f.key}" type="number" min={LIMITS[f.key][0]} max={LIMITS[f.key][1]} value={pomo.settings[f.key]}
              onchange={(e) => commit(f.key, e)} onblur={(e) => commit(f.key, e)} />
          {/each}
          <label class="chk"><input type="checkbox" checked={pomo.settings.auto} onchange={(e) => setSettings({ auto: e.currentTarget.checked })} />
            <span>Start the next part by itself</span></label>
          <label class="chk"><input type="checkbox" checked={pomo.settings.sound} onchange={(e) => setSettings({ sound: e.currentTarget.checked })} />
            <span>Ring when a part ends</span></label>
          <label class="chk"><input type="checkbox" checked={pomo.settings.flash} onchange={(e) => setSettings({ flash: e.currentTarget.checked })} />
            <span>Show a message and flash the screen when a part ends</span></label>
          <p class="muted note">A new length counts from the next part. The part that runs now keeps its end time.</p>
          <button class="small" disabled={!changed} onclick={resetSettings}>Back to 25 / 5 / 15 / 4</button>
        </div>
      </details>
    </div>
  {/if}
</div>

<style>
  /* Idle: after the menu items, above the data file. Active: right after the project button, above Search. */
  .pomo { order: 5; margin-top: .8rem; padding: 0 .1rem; }
  .pomo.active { order: 0; margin: .2rem 0 .4rem; }

  /* the menu styles its buttons flat. These buttons need a look of their own. */
  .pbtn, .mini button, .panel button { text-align: center; background: var(--panel-2); border: 1px solid var(--line); padding: .35rem .6rem; }
  .pbtn:hover, .mini button:hover:not(:disabled), .panel button:hover:not(:disabled) { border-color: var(--accent); }
  .mini button.primary, .panel button.primary { background: var(--accent); color: var(--on-accent); border-color: var(--accent); font-weight: 600; }
  .panel button.small { padding: .15rem .5rem; }
  .pbtn { width: 100%; font-size: .85rem; padding: .55rem .7rem; }
  .pbtn[aria-expanded='true'] { border-color: var(--accent); }

  .mini { background: var(--panel-2); border: 1px solid var(--accent); border-radius: 8px; padding: .5rem .6rem; display: flex; flex-direction: column; gap: .3rem; }
  .mini.rest { border-color: var(--ok); }
  .mini .mhead { background: none; border: 0; padding: 0; display: flex; align-items: center; gap: .4rem; font-size: .72rem; text-transform: uppercase; letter-spacing: .06em; color: var(--muted); text-align: left; }
  .mini .mhead:hover { color: var(--text); }
  .mtime { font-size: 1.7rem; font-weight: 600; line-height: 1.1; font-variant-numeric: tabular-nums; color: var(--accent); }
  .mini.rest .mtime { color: var(--ok); }
  .mmsg { margin: .1rem 0; font-size: .8rem; color: var(--text); }
  .mctl { display: flex; gap: .3rem; }
  .mctl button { flex: 1 1 0; min-width: 0; padding: .25rem .4rem; font-size: .8rem; }
  .dotp { width: .5rem; height: .5rem; border-radius: 50%; background: var(--accent); flex: 0 0 auto; }
  .rest .dotp { background: var(--ok); }
  .run .dotp { animation: blink 1.6s ease-in-out infinite; }
  /* a part ended: the card flashes until you answer it */
  .mini.done { animation: card 1s ease-in-out infinite; }
  @keyframes blink { 50% { opacity: .3; } }
  @keyframes card { 50% { background: color-mix(in srgb, var(--accent) 35%, var(--panel-2)); } }
  .mini.done.rest { animation-name: cardrest; }
  @keyframes cardrest { 50% { background: color-mix(in srgb, var(--ok) 35%, var(--panel-2)); } }
  @media (prefers-reduced-motion: reduce) { .run .dotp, .mini.done { animation: none; } }

  .panel {
    position: fixed; left: 218px; bottom: 1rem; z-index: 60; width: 19rem; max-width: calc(100vw - 230px); max-height: calc(100vh - 2rem); overflow: auto;
    background: var(--panel); border: 1px solid var(--line); border-radius: 10px; padding: .8rem .9rem;
    box-shadow: 0 14px 40px rgba(0, 0, 0, .45); display: flex; flex-direction: column; gap: .4rem;
  }
  .lbl { font-size: .72rem; text-transform: uppercase; letter-spacing: .06em; color: var(--muted); }
  .time { font-size: 2.6rem; font-weight: 600; line-height: 1.1; font-variant-numeric: tabular-nums; color: var(--accent); }
  .time.rest { color: var(--ok); }
  .bar { height: 5px; border-radius: 3px; background: var(--bg); overflow: hidden; }
  .panel .bar { background: var(--panel-2); }
  .bar span { display: block; height: 100%; background: var(--accent); transition: width .5s linear; }
  .rest .bar span { background: var(--ok); }
  .ctl { display: flex; gap: .4rem; margin-top: .3rem; }
  .ctl button { flex: 1 1 0; min-width: 0; }
  .msg { margin: .2rem 0 0; font-size: .85rem; color: var(--text); }
  .note { margin: 0; font-size: .78rem; }
  .panel button.link { background: none; border: 0; padding: 0 .2rem; color: var(--accent); text-decoration: underline; font-size: .78rem; min-width: 0; }
  details { margin-top: .3rem; border-top: 1px solid var(--line); padding-top: .5rem; }
  summary { cursor: pointer; font-size: .85rem; }
  .set { display: grid; grid-template-columns: 1fr 4.2rem; gap: .35rem .6rem; align-items: center; margin-top: .5rem; }
  .set label { margin: 0; font-size: .8rem; }
  .set input[type='number'] { width: 100%; padding: .2rem .4rem; }
  .set .chk { grid-column: 1 / -1; display: flex; gap: .5rem; align-items: center; color: var(--text); }
  .set .chk input { width: auto; }
  .set .note, .set button { grid-column: 1 / -1; }
  .set button { justify-self: start; }
</style>
