<script lang="ts">
  import { db } from '../lib/db.svelte';
  import { openItem, openScene } from '../lib/nav.svelte';
  import { findConflicts, type Conflict, type Presence } from '../lib/plotholes';

  let showDismissed = $state(false);
  let help = $state(false);

  const result = $derived(findConflicts());
  const ignored = $derived(new Set(db.lists.plotignore.map((i) => i.key as string)));
  const open = $derived(result.conflicts.filter((c) => !ignored.has(c.key)));
  const dismissed = $derived(result.conflicts.filter((c) => ignored.has(c.key)));
  const stats = $derived(result.stats);
  const checked = $derived(stats.scenes + stats.events + stats.steps + stats.itemSteps);

  const timeText = (c: Conflict) => (c.when ? `${c.when} (${c.time})` : String(c.time));
  const title = (c: Conflict) =>
    c.kind === 'character'
      ? `${c.name} is in different places at the same time`
      : `“${c.name}” has two owners or places at the same time`;

  function go(r: Presence) {
    if (r.source === 'scene') openScene(r.id);
    else if (r.source === 'event') openItem('event', r.id);
    else if (r.source === 'path') openItem('pathpoint', r.id);
    else openItem('item', r.id);
  }
  const dismiss = (c: Conflict) => db.save('plotignore', { key: c.key });
  const bring = (c: Conflict) => {
    const row = db.lists.plotignore.find((i) => i.key === c.key);
    if (row) db.remove('plotignore', row.id, false);
  };
  const SOURCE = { scene: 'scene', event: 'event', path: 'path step', item: 'item' } as const;
</script>

<section class="pane">
  <div class="head">
    <h2>Plot holes{open.length ? ` · ${open.length}` : ''}</h2>
    <button class="small" onclick={() => (help = !help)}>{help ? 'Hide' : 'How this works'}</button>
  </div>

  {#if help}
    <div class="card">
      <p style="margin-top:0">The check looks for one person (or item) in two places at the same moment of story time.</p>
      <ul>
        <li>It reads the <b>timeline number</b> and the <b>place</b> of each scene, each timeline event, each map path step, and each item step that names a scene.</li>
        <li>Only things with <b>exactly the same number</b> are compared. If one number covers a whole year, use decimals for the days: 312.1, 312.2.</li>
        <li>A place inside another place is fine. A person in the market and in the city that holds the market is not a problem.</li>
        <li>Two map spots very close together count as the same spot.</li>
        <li>If it is not a real problem (the scenes are really days apart), press <b>Not a problem</b>. If you change the scenes later and the problem comes back in a new shape, it shows up again.</li>
      </ul>
    </div>
  {/if}

  <p class="muted">
    Checked {stats.scenes} scene{stats.scenes === 1 ? '' : 's'}, {stats.events} event{stats.events === 1 ? '' : 's'},
    {stats.steps} path step{stats.steps === 1 ? '' : 's'} and {stats.itemSteps} item step{stats.itemSteps === 1 ? '' : 's'}.
    {#if stats.scenesNoTime || stats.scenesNoPlace}
      Not checked: {stats.scenesNoTime} scene{stats.scenesNoTime === 1 ? '' : 's'} with no timeline number, {stats.scenesNoPlace} with a number but no place.
    {/if}
  </p>

  {#each open as c (c.key)}
    <div class="card hole">
      <div class="head">
        <strong>{title(c)}</strong>
        <button class="small" title="Hide this one" onclick={() => dismiss(c)}>Not a problem</button>
      </div>
      <div class="muted">At {timeText(c)}</div>
      {@render rows(c)}
    </div>
  {:else}
    {#if checked === 0}
      <div class="empty">Nothing to check yet. Give scenes a timeline number and a place in the Write tab, then come back.</div>
    {:else}
      <div class="empty">No plot holes found. Nobody is in two places at once.</div>
    {/if}
  {/each}

  {#if dismissed.length}
    <button class="small" style="margin-top:1rem" onclick={() => (showDismissed = !showDismissed)}>
      {showDismissed ? 'Hide' : 'Show'} {dismissed.length} marked “not a problem”
    </button>
    {#if showDismissed}
      {#each dismissed as c (c.key)}
        <div class="card hole done">
          <div class="head"><strong>{title(c)}</strong><button class="small" onclick={() => bring(c)}>Bring back</button></div>
          <div class="muted">At {timeText(c)}</div>
          {@render rows(c)}
        </div>
      {/each}
    {/if}
  {/if}
</section>

{#snippet rows(c: Conflict)}
  {#each c.rows as r}
    <button class="row-link" onclick={() => go(r)} title="Open it">
      <span class="chip">{SOURCE[r.source]}</span>
      <span class="t">{r.text}</span>
      {#if r.place.label}<span class="muted">{r.place.label}</span>{/if}
    </button>
  {/each}
{/snippet}

<style>
  .hole { border-left: 3px solid var(--danger); }
  .hole.done { border-left-color: var(--muted); opacity: .65; }
  .row-link { display: flex; align-items: center; gap: .5rem; width: 100%; text-align: left; background: none; border: 1px solid transparent; padding: .3rem .4rem; }
  .row-link:hover { background: var(--panel-2); border-color: var(--line); }
  .row-link .t { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .row-link .chip { margin: 0; }
</style>
