<script lang="ts">
  import { db } from '../lib/db.svelte';
  import { openItem, openScene } from '../lib/nav.svelte';
  import { findConflicts, type Conflict, type Presence } from '../lib/plotholes';
  import { findGaps, GROUPS, type Gap } from '../lib/consistency';
  import { findNameIssues, fixEverywhere, type NameIssue } from '../lib/namespell';

  let showDismissed = $state(false);
  let help = $state(false);
  let tab = $state<'holes' | 'gaps' | 'names'>('holes');
  let showHiddenGaps = $state(false);

  const gaps = $derived(findGaps());
  const hidden = $derived(new Set(db.lists.plotignore.map((i) => i.key as string)));
  const openGapList = $derived(gaps.filter((g) => !hidden.has(g.key)));
  const hiddenGapList = $derived(gaps.filter((g) => hidden.has(g.key)));
  const gapGroups = $derived(GROUPS.map((g) => ({ name: g, rows: openGapList.filter((x) => x.group === g) })).filter((g) => g.rows.length));
  function fix(g: Gap) {
    if (g.open.kind === 'scene') openScene(g.open.id);
    else openItem(g.open.kind, g.open.id);
  }
  const hideGap = (g: Gap) => db.save('plotignore', { key: g.key });
  const showGap = (g: Gap) => {
    const row = db.lists.plotignore.find((i) => i.key === g.key);
    if (row) db.remove('plotignore', row.id, false);
  };

  // names spelled two ways. Only worked out while that tab is open, because it reads the whole book.
  const nameIssues = $derived(tab === 'names' ? findNameIssues() : []);
  const openNames = $derived(nameIssues.filter((n) => !hidden.has(n.key)));
  const hiddenNames = $derived(nameIssues.filter((n) => hidden.has(n.key)));
  let showHiddenNames = $state(false);
  let nameMsg = $state('');
  let fixing = $state(false);
  async function fixName(n: NameIssue) {
    if (fixing) return;
    fixing = true; nameMsg = '';
    try {
      const c = await fixEverywhere(n);
      nameMsg = `Changed “${n.word}” to “${n.suggest}” ${c} time${c === 1 ? '' : 's'}. To undo: open History in that scene.`;
    } catch (e: any) {
      nameMsg = 'Could not change it: ' + (e?.message ?? e);
    } finally { fixing = false; }
  }
  const skipName = (n: NameIssue) => db.save('plotignore', { key: n.key });
  const backName = (n: NameIssue) => {
    const row = db.lists.plotignore.find((i) => i.key === n.key);
    if (row) db.remove('plotignore', row.id, false);
  };
  const snip = (s: string) => { const [a, rest] = s.split('⟦'); const [m, b] = rest.split('⟧'); return { a, m, b }; };

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
      : c.kind === 'travel'
        ? `${c.name} cannot travel that far in the time given`
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
  <div class="tabs">
    <button class:active={tab === 'holes'} onclick={() => (tab = 'holes')}>Plot holes{open.length ? ' · ' + open.length : ''}</button>
    <button class:active={tab === 'gaps'} onclick={() => (tab = 'gaps')}>Unfinished{openGapList.length ? ' · ' + openGapList.length : ''}</button>
    <button class:active={tab === 'names'} onclick={() => (tab = 'names')}>Names{tab === 'names' && openNames.length ? ' · ' + openNames.length : ''}</button>
  </div>

  {#if tab === 'names'}
    <p class="muted">
      Finds a name that is spelled two ways. It looks for a capitalised word that is almost a name from your Characters, Places, Factions or Items,
      and for a rare word that is almost a name you use often in the text. It reads your scenes only.
      Press <b>Skip</b> if the word is right (for example a different person with a close name).
    </p>
    {#if nameMsg}<p style="color:var(--ok)">{nameMsg}</p>{/if}
    {#each openNames as n (n.key)}
      <div class="card hole name">
        <div class="head">
          <strong>“{n.word}” → “{n.suggest}”</strong>
          <span class="row-btns">
            <button class="small primary" disabled={fixing} onclick={() => fixName(n)}>Change all to “{n.suggest}”</button>
            <button class="small" onclick={() => skipName(n)}>Skip</button>
          </span>
        </div>
        <div class="muted">Used {n.count} time{n.count === 1 ? '' : 's'}. It {n.reason}.</div>
        {#each n.places as p (p.scene.id)}
          {@const x = snip(p.snippet)}
          <button class="row-link" onclick={() => openScene(p.scene.id)} title="Open it">
            <span class="chip">{p.label}{p.times > 1 ? ' · ' + p.times + '×' : ''}</span>
            <span class="t">{x.a}<mark>{x.m}</mark>{x.b}</span>
          </button>
        {/each}
      </div>
    {:else}
      <div class="empty">{db.lists.scene.length ? 'No names look misspelled.' : 'Nothing to check yet. Write some scenes first.'}</div>
    {/each}
    {#if hiddenNames.length}
      <button class="small" style="margin-top:1rem" onclick={() => (showHiddenNames = !showHiddenNames)}>
        {showHiddenNames ? 'Hide' : 'Show'} {hiddenNames.length} skipped
      </button>
      {#if showHiddenNames}
        <div class="card">
          {#each hiddenNames as n (n.key)}
            <div class="gap done">
              <span class="t"><b>“{n.word}”</b> → “{n.suggest}”</span>
              <button class="small" onclick={() => backName(n)}>Bring back</button>
            </div>
          {/each}
        </div>
      {/if}
    {/if}
  {:else if tab === 'gaps'}
    <p class="muted">Things that look unfinished. Open one to fix it. Press <b>Skip</b> if it is fine as it is (for example a character with no age on purpose).</p>
    {#each gapGroups as g (g.name)}
      <div class="card">
        <h3>{g.name} · {g.rows.length}</h3>
        {#each g.rows as r (r.key)}
          <div class="gap">
            <button class="row-link" onclick={() => fix(r)} title="Open it"><span class="t"><b>{r.name}</b> {r.problem}</span></button>
            <button class="small" onclick={() => hideGap(r)}>Skip</button>
          </div>
        {/each}
      </div>
    {:else}
      <div class="empty">{gaps.length ? 'Everything is done or skipped.' : 'Nothing to check yet. Add characters, places and scenes first.'}</div>
    {/each}
    {#if hiddenGapList.length}
      <button class="small" style="margin-top:1rem" onclick={() => (showHiddenGaps = !showHiddenGaps)}>
        {showHiddenGaps ? 'Hide' : 'Show'} {hiddenGapList.length} skipped
      </button>
      {#if showHiddenGaps}
        <div class="card">
          {#each hiddenGapList as r (r.key)}
            <div class="gap done">
              <span class="t"><b>{r.name}</b> {r.problem}</span>
              <button class="small" onclick={() => showGap(r)}>Bring back</button>
            </div>
          {/each}
        </div>
      {/if}
    {/if}
  {:else}
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
        <li>If a map has a scale, the check also finds trips that are too long for the story time between two stops, even at your fastest way to travel.</li>
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
      <div class="muted">{c.note ?? `At ${timeText(c)}`}</div>
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
          <div class="muted">{c.note ?? `At ${timeText(c)}`}</div>
          {@render rows(c)}
        </div>
      {/each}
    {/if}
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
  .row-btns { display: flex; gap: .4rem; }
  mark { background: color-mix(in srgb, var(--accent-2) 35%, transparent); color: inherit; border-radius: 3px; padding: 0 .1rem; }
  .gap { display: flex; align-items: center; gap: .5rem; }
  .gap .row-link { flex: 1; width: auto; }
  .gap.done { opacity: .65; padding: .3rem .4rem; }
  .gap.done .t { flex: 1; }
  .row-link .chip { margin: 0; }
</style>
