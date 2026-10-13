<script lang="ts">
  import { word } from './projecttype.svelte';
  import { db } from './db.svelte';
  import ConfirmButton from './ConfirmButton.svelte';

  let { sceneId, body, active, onshow, onremove }: {
    sceneId: number;
    body: string; // the scene text, to see which comments still have their text
    active: number | null;
    onshow: (id: number) => void;
    onremove: (id: number) => void;
  } = $props();

  const mine = $derived(db.lists.comment.filter((c) => c.scene === sceneId).sort((a, b) => String(a.at).localeCompare(String(b.at))));
  const open = $derived(mine.filter((c) => !c.resolved));
  const done = $derived(mine.filter((c) => c.resolved));
  let showDone = $state(false);

  const lost = (id: number) => !body.includes(`data-comment="${id}"`);
  const save = (c: any) => db.queue('comment', c);
  function toggle(c: any) { c.resolved = !c.resolved; db.save('comment', c); }

  // a new comment (or a click on commented text) brings its card into view
  $effect(() => {
    if (active == null) return;
    const card = document.getElementById('cmt-' + active);
    card?.scrollIntoView({ block: 'nearest' });
    const box = card?.querySelector('textarea');
    if (box && !box.value) box.focus();
  });
</script>

{#snippet card(c: any)}
  <div class="cmt-card" id={'cmt-' + c.id} class:active={active === c.id} class:done={c.resolved}>
    <button class="quote" onclick={() => onshow(c.id)} disabled={lost(c.id)} title={lost(c.id) ? 'The text was deleted' : 'Show in the text'}>
      “{c.quote}”{#if lost(c.id)} <em>(text deleted)</em>{/if}
    </button>
    <textarea bind:value={c.text} oninput={() => save(c)} placeholder="Your note" rows="2"></textarea>
    <div class="acts">
      <button class="small" onclick={() => toggle(c)}>{c.resolved ? 'Reopen' : 'Resolve'}</button>
      <ConfirmButton onconfirm={() => onremove(c.id)} />
    </div>
  </div>
{/snippet}

<div class="head"><h3 style="margin:0">Comments{open.length ? ` · ${open.length} open` : ''}</h3></div>
{#if !mine.length}
  <p class="muted">Select text in the {word('scene')}, then press 💬 to add a note to it.</p>
{/if}
{#each open as c (c.id)}{@render card(c)}{/each}
{#if done.length}
  <button class="small" onclick={() => (showDone = !showDone)}>{showDone ? 'Hide' : 'Show'} {done.length} resolved</button>
  {#if showDone}{#each done as c (c.id)}{@render card(c)}{/each}{/if}
{/if}

<style>
  .cmt-card { border: 1px solid var(--line); border-left: 3px solid var(--accent-2); border-radius: 6px; padding: .4rem .5rem; margin: .4rem 0; background: var(--panel-2); }
  .cmt-card.active { border-color: var(--accent); box-shadow: 0 0 0 2px color-mix(in srgb, var(--accent) 30%, transparent); }
  .cmt-card.done { opacity: .6; border-left-color: var(--ok); }
  .quote { display: block; width: 100%; text-align: left; background: none; border: 0; padding: 0 0 .3rem; color: var(--muted); font-style: italic; font-size: .85rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .cmt-card textarea { min-height: 0; font-size: .9rem; }
  .acts { display: flex; gap: .3rem; margin-top: .3rem; justify-content: space-between; }
</style>
