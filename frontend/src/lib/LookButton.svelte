<script lang="ts">
  import { look, setLook, resetLook } from './editorlook.svelte';
  import { FONTS, SIZES, SPACINGS, WIDTHS, isDefaultLook } from './editorlook-core';

  // A "Look" button for the tool row of the Write tab. It opens a small panel. The page changes while you click.
  let open = $state(false);
  let box: HTMLElement | undefined = $state();

  const groups = [
    { key: 'font', label: 'Font', list: FONTS },
    { key: 'size', label: 'Text size', list: SIZES },
    { key: 'spacing', label: 'Line spacing', list: SPACINGS },
    { key: 'width', label: 'Page width', list: WIDTHS },
  ] as const;
</script>

<svelte:window
  onclick={(e) => { if (open && box && !box.contains(e.target as Node)) open = false; }}
  onkeydown={(e) => { if (open && e.key === 'Escape') { e.stopPropagation(); open = false; } }} />

<span class="look" bind:this={box}>
  <button class="small" class:on={open} aria-expanded={open} onclick={() => (open = !open)}>Look</button>
  {#if open}
    <div class="panel" role="dialog" aria-label="How the page looks">
      {#each groups as g}
        <div class="lbl">{g.label}</div>
        <div class="seg" class:fonts={g.key === 'font'}>
          {#each g.list as [id, name, css]}
            <button class:active={look[g.key] === id} style:font-family={g.key === 'font' ? css : null} onclick={() => setLook({ [g.key]: id })}>{name}</button>
          {/each}
        </div>
      {/each}
      <label class="chk"><input type="checkbox" checked={look.typewriter} onchange={(e) => setLook({ typewriter: e.currentTarget.checked })} />
        <span><b>Typewriter mode</b><small>The line you write on stays in the middle of the screen.</small></span></label>
      <label class="chk"><input type="checkbox" checked={look.focusLine} onchange={(e) => setLook({ focusLine: e.currentTarget.checked })} />
        <span><b>Focus line</b><small>Every paragraph but the one you write in is dimmed.</small></span></label>
      <label class="chk"><input type="checkbox" checked={look.vim} onchange={(e) => setLook({ vim: e.currentTarget.checked })} />
        <span><b>Vim keys</b><small>Move and edit with Vim keys. Esc for Normal mode, i to type. Try it, it is new.</small></span></label>
      <button class="small" disabled={isDefaultLook(look)} onclick={resetLook}>Back to normal</button>
    </div>
  {/if}
</span>

<style>
  .look { position: relative; }
  .on { border-color: var(--accent); }
  .panel {
    position: absolute; top: calc(100% + 6px); left: 0; z-index: 40; width: 21rem; max-width: 90vw;
    background: var(--panel); border: 1px solid var(--line); border-radius: 10px; padding: .7rem .8rem;
    box-shadow: 0 14px 40px rgba(0, 0, 0, .45); display: flex; flex-direction: column; gap: .35rem;
  }
  .lbl { font-size: .72rem; text-transform: uppercase; letter-spacing: .06em; color: var(--muted); margin-top: .2rem; }
  .seg { display: flex; gap: 3px; }
  .seg button { flex: 1 1 0; min-width: 0; padding: .25rem .3rem; font-size: .8rem; }
  /* the font names have different lengths: each button is as wide as its name, so "Typewriter" fits */
  .seg.fonts button { flex: 1 1 auto; white-space: nowrap; }
  .chk { display: flex; gap: .5rem; align-items: flex-start; margin: .35rem 0 0; color: var(--text); font-size: .85rem; }
  .chk input { width: auto; margin-top: .2rem; }
  .chk small { display: block; color: var(--muted); font-size: .75rem; }
  .panel > button.small { align-self: flex-start; margin-top: .4rem; }
</style>
