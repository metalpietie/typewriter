<script lang="ts">
  import { pomo, start, dismiss } from './pomodoro.svelte';
  import { LABEL } from './pomodoro-core';

  // Shown when a part of the timer ends. It sits on top of everything, also in focus mode,
  // so you see it without headphones. The screen edge glows, and a message waits until you answer it.
  const toBreak = $derived(pomo.phase !== 'work'); // the part that just ended was work
</script>

{#if pomo.alert}
  <div class="glow" class:rest={toBreak} aria-hidden="true"></div>
  <div class="toast" class:rest={toBreak} role="alert">
    <p>{pomo.message}</p>
    <div class="row">
      {#if !pomo.running}
        <button class="primary" onclick={start}>Start {LABEL[pomo.phase].toLowerCase()}</button>
        <button onclick={dismiss}>Dismiss</button>
      {:else}
        <button class="primary" onclick={dismiss}>OK</button>
      {/if}
    </div>
  </div>
{/if}

<style>
  .glow {
    position: fixed; inset: 0; z-index: 90; pointer-events: none;
    box-shadow: inset 0 0 0 4px var(--accent-2);
    animation: pulse 1s ease-in-out 6;
  }
  .glow.rest { box-shadow: inset 0 0 0 4px var(--ok); }
  @keyframes pulse { 50% { box-shadow: inset 0 0 0 6px var(--accent-2), inset 0 0 110px 12px var(--accent-2); } }
  .glow.rest { animation-name: pulse-rest; }
  @keyframes pulse-rest { 50% { box-shadow: inset 0 0 0 6px var(--ok), inset 0 0 110px 12px var(--ok); } }

  .toast {
    position: fixed; top: 1rem; left: 50%; transform: translateX(-50%); z-index: 95; width: min(30rem, 92vw);
    background: var(--panel); border: 2px solid var(--accent-2); border-radius: 12px; padding: .8rem 1rem;
    box-shadow: 0 16px 50px rgba(0, 0, 0, .55); animation: drop .25s ease-out;
  }
  .toast.rest { border-color: var(--ok); }
  .toast p { margin: 0 0 .6rem; font-size: 1.05rem; }
  .toast .row { display: flex; gap: .5rem; justify-content: flex-start; }
  .toast .row > * { flex: 0 0 auto; }
  @keyframes drop { from { transform: translate(-50%, -14px); opacity: 0; } }
  @media (prefers-reduced-motion: reduce) { .glow, .toast { animation: none; } }
</style>
