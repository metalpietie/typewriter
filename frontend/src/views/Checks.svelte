<script lang="ts">
  import Select from '../lib/Select.svelte';
  import { db } from '../lib/db.svelte';
  import { openItem, openScene } from '../lib/nav.svelte';
  import { findConflicts, type Conflict, type Presence } from '../lib/plotholes';
  import { findGaps, GROUPS, type Gap } from '../lib/consistency';
  import { findNameIssues, fixEverywhere, type NameIssue } from '../lib/namespell';
  import { findRepeats } from '../lib/repeats';
  import StyleCheck from '../lib/StyleCheck.svelte';

  let showDismissed = $state(false);
  let help = $state(false);
  let tab = $state<'holes' | 'gaps' | 'names' | 'repeats' | 'style'>('holes');
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
  /** Split a snippet at its marks: text between ⟦ and ⟧ is shown highlighted. */
  const snip = (s: string) => s.split(/(⟦[^⟧]*⟧)/).filter(Boolean).map((t) => (t.startsWith('⟦') ? { t: t.slice(1, -1), hit: true } : { t, hit: false }));

  // repeated words. Only worked out while that tab is open, because it reads the whole book.
  let rScope = $state<number | 'all'>('all');
  let rPart = $state<'close' | 'starts' | 'top'>('close');
  let showHiddenReps = $state(false);
  const reps = $derived(tab === 'repeats' ? findRepeats(rScope) : null);
  const openClose = $derived((reps?.close ?? []).filter((r) => !hidden.has(r.key)));
  const openStarts = $derived((reps?.starts ?? []).filter((r) => !hidden.has(r.key)));
  const hiddenReps = $derived([...(reps?.close ?? []), ...(reps?.starts ?? [])].filter((r) => hidden.has(r.key)));
  /** Rows grouped by scene, in reading order, so each scene is one card. */
  function byScene<T extends { scene: { id: number }; label: string }>(rows: T[]) {
    const out: { id: number; label: string; rows: T[] }[] = [];
    for (const r of rows) {
      const last = out[out.length - 1];
      if (last && last.id === r.scene.id) last.rows.push(r); else out.push({ id: r.scene.id, label: r.label, rows: [r] });
    }
    return out;
  }
  const closeCards = $derived(byScene(openClose));
  const startCards = $derived(byScene(openStarts));
  let showAllCards = $state(false);
  const CARD_LIMIT = 40;
  const skipRep = (r: { key: string }) => db.save('plotignore', { key: r.key });
  const backRep = (r: { key: string }) => {
    const row = db.lists.plotignore.find((i) => i.key === r.key);
    if (row) db.remove('plotignore', row.id, false);
  };

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
    <button class:active={tab === 'repeats'} onclick={() => (tab = 'repeats')}>Repeats{tab === 'repeats' && openClose.length + openStarts.length ? ' · ' + (openClose.length + openStarts.length) : ''}</button>
    <button class:active={tab === 'style'} onclick={() => (tab = 'style')}>Style</button>
  </div>

  {#if tab === 'style'}
    <StyleCheck />
  {:else if tab === 'repeats'}
    <p class="muted">
      Finds words that come back too soon, sentences that start the same way, and the words you use most.
      It reads your scenes. Names from your Characters, Places, Factions and Items are left out, and so are everyday words.
      Press <b>Skip</b> if a repeat is on purpose.
    </p>
    <div class="row" style="justify-content:flex-start;margin-bottom:.6rem">
      <label class="fit" for="rscope" style="margin:0">Look in</label>
      <Select id="rscope" style="flex:0 1 16rem" bind:value={rScope} onchange={() => (showAllCards = false)}>
        <option value="all">All books</option>
        {#each [...db.lists.book].sort((a, b) => a.order - b.order) as b}<option value={b.id}>{b.title}</option>{/each}
      </Select>
    </div>
    <div class="seg" style="margin-bottom:.8rem">
      <button class:active={rPart === 'close'} onclick={() => (rPart = 'close')}>Words close together · {openClose.length}</button>
      <button class:active={rPart === 'starts'} onclick={() => (rPart = 'starts')}>Same sentence start · {openStarts.length}</button>
      <button class:active={rPart === 'top'} onclick={() => (rPart = 'top')}>Most used words</button>
    </div>

    {#if rPart === 'top'}
      <p class="muted">The words used most, out of {reps?.words.toLocaleString() ?? 0} words in {reps?.scenes ?? 0} scene{reps?.scenes === 1 ? '' : 's'}. A word used a lot is not always wrong. Look for words that surprise you.</p>
      {#each reps?.top ?? [] as t (t.word)}
        <div class="topw"><b>{t.word}</b><span class="bar" style:width={Math.max(4, (t.count / (reps?.top[0].count || 1)) * 100) + '%'}></span><span class="muted">{t.count} · {t.per1000} per 1,000</span></div>
      {:else}
        <div class="empty">No word is used 5 times or more yet.</div>
      {/each}
    {:else}
      {#if rPart === 'close'}
        <p class="muted" style="margin-top:0">A word of 4 or more letters used again within 20 words.</p>
      {:else}
        <p class="muted" style="margin-top:0">3 or more sentences in a row that begin with the same word.</p>
      {/if}
      {@const cards = rPart === 'close' ? closeCards : startCards}
      {#each showAllCards ? cards : cards.slice(0, CARD_LIMIT) as c (c.id)}
        <div class="card">
          <button class="row-link" onclick={() => openScene(c.id)} title="Open the scene"><span class="t"><b>{c.label}</b></span></button>
          {#each c.rows as r (r.key)}
            {@const x = snip(r.snippet)}
            <div class="gap">
              <button class="row-link" onclick={() => openScene(c.id)} title="Open the scene">
                <span class="chip">{r.word}{'times' in r ? ' · ' + r.times + '×' : ' · ' + r.run + ' in a row'}</span>
                <span class="t">{#each x as p}{#if p.hit}<mark>{p.t}</mark>{:else}{p.t}{/if}{/each}</span>
              </button>
              <button class="small" onclick={() => skipRep(r)}>Skip</button>
            </div>
          {/each}
        </div>
      {:else}
        <div class="empty">{(reps?.scenes ?? 0) ? 'Nothing found. Nice.' : 'Nothing to check yet. Write some scenes first.'}</div>
      {/each}
      {#if !showAllCards && cards.length > CARD_LIMIT}
        <button class="small" onclick={() => (showAllCards = true)}>Show the other {cards.length - CARD_LIMIT} scenes</button>
      {/if}
    {/if}

    {#if hiddenReps.length}
      <button class="small" style="margin-top:1rem" onclick={() => (showHiddenReps = !showHiddenReps)}>
        {showHiddenReps ? 'Hide' : 'Show'} {hiddenReps.length} skipped
      </button>
      {#if showHiddenReps}
        <div class="card">
          {#each hiddenReps as r (r.key)}
            <div class="gap done">
              <span class="t"><b>{r.word}</b> · {r.label}</span>
              <button class="small" onclick={() => backRep(r)}>Bring back</button>
            </div>
          {/each}
        </div>
      {/if}
    {/if}
  {:else if tab === 'names'}
    <p class="muted">
      Finds a name or a made-up word that is spelled two ways. It looks for a capitalised word that is almost a name from your Characters, Places, Factions or Items,
      for a word that is almost one of your Glossary words, and for a rare word that is almost a name you use often in the text. It reads your scenes only.
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
            <span class="t">{#each x as p}{#if p.hit}<mark>{p.t}</mark>{:else}{p.t}{/if}{/each}</span>
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
  .topw { display: flex; align-items: center; gap: .6rem; padding: .15rem 0; }
  .topw b { width: 9rem; flex: 0 0 auto; overflow: hidden; text-overflow: ellipsis; }
  .topw .bar { height: .6rem; border-radius: 3px; background: var(--accent); flex: 0 1 auto; max-width: 40%; opacity: .75; }
  .gap { display: flex; align-items: center; gap: .5rem; }
  .gap .row-link { flex: 1; width: auto; }
  .gap.done { opacity: .65; padding: .3rem .4rem; }
  .gap.done .t { flex: 1; }
  .row-link .chip { margin: 0; }
</style>
