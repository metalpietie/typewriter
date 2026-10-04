<script lang="ts">
  import Select from './Select.svelte';
  import { db } from './db.svelte';
  import { openScene } from './nav.svelte';
  import { findStyle, weakExamples, type LongRow } from './style';
  import { GROUPS } from './style-core';

  // The Style tab of the Checks screen: how much is dialogue, very long sentences, and weak words.
  let scope = $state<number | 'all'>('all');
  let part = $state<'dialogue' | 'long' | 'weak'>('dialogue');
  let minLong = $state(40);
  let openWord = $state('');
  let showAll = $state(false);
  let showSkipped = $state(false);

  const LIMIT = 30; // long sentences shown before "show more"
  const report = $derived(findStyle(scope, minLong));
  const hidden = $derived(new Set(db.lists.plotignore.map((i) => i.key)));
  const openLong = $derived(report.long.filter((r) => !hidden.has(r.key)));
  const skipped = $derived(report.long.filter((r) => hidden.has(r.key)));
  const shown = $derived(showAll ? openLong : openLong.slice(0, LIMIT));
  const examples = $derived(openWord ? weakExamples(scope, openWord) : []);

  const pct = (a: number, b: number) => (b ? Math.round((a / b) * 100) : 0);
  const n = (v: number) => v.toLocaleString();
  const skip = (r: LongRow) => db.save('plotignore', { key: r.key });
  const bring = (r: LongRow) => {
    const row = db.lists.plotignore.find((i) => i.key === r.key);
    if (row) db.remove('plotignore', row.id, false);
  };
  /** Cut a very long sentence for the list: the start and the end. */
  const short = (s: string) => (s.length > 220 ? s.slice(0, 130).trimEnd() + ' … ' + s.slice(-70).trimStart() : s);
  const snip = (s: string) => s.split(/(⟦[^⟧]*⟧)/).filter(Boolean).map((t) => (t.startsWith('⟦') ? { t: t.slice(1, -1), hit: true } : { t, hit: false }));
</script>

<p class="muted">
  Shows how much of your text is dialogue, finds very long sentences, and counts weak words.
  These are hints about style, not mistakes. Many are fine when you meant them.
</p>
<div class="row" style="justify-content:flex-start;margin-bottom:.6rem">
  <label class="fit" for="sscope" style="margin:0">Look in</label>
  <Select id="sscope" style="flex:0 1 16rem" bind:value={scope} onchange={() => { showAll = false; openWord = ''; }}>
    <option value="all">All books</option>
    {#each [...db.lists.book].sort((a, b) => a.order - b.order) as b}<option value={b.id}>{b.title}</option>{/each}
  </Select>
</div>
<div class="seg">
  <button class:active={part === 'dialogue'} onclick={() => (part = 'dialogue')}>Dialogue</button>
  <button class:active={part === 'long'} onclick={() => (part = 'long')}>Long sentences · {openLong.length}</button>
  <button class:active={part === 'weak'} onclick={() => (part = 'weak')}>Weak words</button>
</div>

{#if !report.scenes}
  <div class="empty">Nothing to check yet. Write some scenes first.</div>
{:else if part === 'dialogue'}
  <p class="lead">
    Dialogue is <b>{pct(report.dialogue.dialogue, report.dialogue.total)}%</b> of the text
    <span class="muted">({n(report.dialogue.dialogue)} of {n(report.dialogue.total)} words)</span>
  </p>
  <p class="muted">Words inside quotation marks, chapter by chapter. There is no right number. A chapter with a very low or high share may be just what you want.</p>
  {#each report.chapters as c (c.id)}
    <button class="chap" disabled={!c.first} onclick={() => c.first && openScene(c.first)} title="Open the first scene of this chapter">
      <span class="name">{c.label}</span>
      <span class="track"><span class="fill" style:width={pct(c.dialogue, c.total) + '%'}></span></span>
      <span class="num">{pct(c.dialogue, c.total)}%</span>
      <span class="muted words">{n(c.total)} words</span>
    </button>
  {/each}
{:else if part === 'long'}
  <div class="row" style="justify-content:flex-start;margin:.6rem 0">
    <label class="fit" for="slong" style="margin:0">Longer than</label>
    <Select id="slong" style="flex:0 1 9rem" bind:value={minLong} onchange={() => (showAll = false)}>
      {#each [30, 40, 50, 60] as v}<option value={v}>{v} words</option>{/each}
    </Select>
  </div>
  <p class="muted" style="margin-top:0">A long sentence can be good. Press <b>Skip</b> when it is on purpose.</p>
  {#each shown as r (r.key)}
    <div class="card">
      <div class="gap">
        <button class="row-link" onclick={() => openScene(r.scene.id)} title="Open the scene">
          <span class="chip">{r.words} words</span>
          <span class="t wrap">{short(r.text)}</span>
        </button>
        <button class="small" onclick={() => skip(r)}>Skip</button>
      </div>
      <div class="muted from">{r.label}</div>
    </div>
  {:else}
    <div class="empty">No sentence is longer than {minLong} words. Nice.</div>
  {/each}
  {#if !showAll && openLong.length > LIMIT}
    <button class="small" onclick={() => (showAll = true)}>Show the other {openLong.length - LIMIT}</button>
  {/if}
  {#if skipped.length}
    <button class="small" style="margin-top:1rem" onclick={() => (showSkipped = !showSkipped)}>{showSkipped ? 'Hide' : 'Show'} {skipped.length} skipped</button>
    {#if showSkipped}
      <div class="card">
        {#each skipped as r (r.key)}
          <div class="gap done"><span class="t">{r.words} words · {r.label}</span><button class="small" onclick={() => bring(r)}>Bring back</button></div>
        {/each}
      </div>
    {/if}
  {/if}
{:else}
  <p class="muted">How often each weak word is used, out of {n(report.dialogue.total)} words. Click a word to see where.</p>
  {#each GROUPS as g (g.id)}
    {@const list = report.weak.filter((w) => w.group === g.id).slice(0, 15)}
    <h3 style="margin-top:1.2rem">{g.title}</h3>
    <p class="muted" style="margin:0 0 .4rem">{g.help}</p>
    {#each list as w (w.word)}
      <button class="topw" class:on={openWord === w.word} onclick={() => (openWord = openWord === w.word ? '' : w.word)}>
        <b>{w.word}</b><span class="bar" style:width={Math.max(4, (w.count / (list[0].count || 1)) * 100) + '%'}></span>
        <span class="muted">{w.count} · {w.per1000} per 1,000</span>
      </button>
      {#if openWord === w.word}
        <div class="card examples">
          {#each examples as e}
            <button class="row-link" onclick={() => openScene(e.scene.id)} title="Open the scene">
              <span class="chip">{e.label}</span>
              <span class="t">{#each snip(e.snippet) as p}{#if p.hit}<mark>{p.t}</mark>{:else}{p.t}{/if}{/each}</span>
            </button>
          {/each}
          {#if w.count > examples.length}<div class="muted from">Showing the first {examples.length} of {w.count}.</div>{/if}
        </div>
      {/if}
    {:else}
      <div class="muted" style="padding:.3rem 0">None found.</div>
    {/each}
  {/each}
{/if}

<style>
  .seg { display: flex; gap: .3rem; flex-wrap: wrap; margin-bottom: .8rem; }
  .lead { font-size: 1.1rem; margin: .4rem 0; }
  .chap { display: flex; align-items: center; gap: .6rem; width: 100%; text-align: left; background: none; border: 1px solid transparent; padding: .3rem .4rem; }
  .chap:hover:not(:disabled) { background: var(--panel-2); border-color: var(--line); }
  .chap .name { flex: 0 0 14rem; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .chap .track { flex: 1; height: .6rem; border-radius: 3px; background: var(--panel-2); overflow: hidden; }
  .chap .fill { display: block; height: 100%; background: var(--accent); opacity: .8; }
  .chap .num { flex: 0 0 2.6rem; text-align: right; font-weight: 600; }
  .chap .words { flex: 0 0 6rem; text-align: right; font-size: .8rem; }

  .row-link { display: flex; align-items: center; gap: .5rem; width: 100%; text-align: left; background: none; border: 1px solid transparent; padding: .3rem .4rem; }
  .row-link:hover { background: var(--panel-2); border-color: var(--line); }
  .row-link .t { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .row-link .t.wrap { white-space: normal; }
  .row-link .chip { margin: 0; flex: 0 0 auto; }
  .gap { display: flex; align-items: center; gap: .5rem; }
  .gap .row-link { flex: 1; width: auto; }
  .gap.done { opacity: .65; padding: .3rem .4rem; }
  .gap.done .t { flex: 1; }
  .from { font-size: .8rem; padding: 0 .4rem; }

  .topw { display: flex; align-items: center; gap: .6rem; width: 100%; text-align: left; background: none; border: 1px solid transparent; padding: .15rem .4rem; }
  .topw:hover, .topw.on { background: var(--panel-2); border-color: var(--line); }
  .topw b { width: 9rem; flex: 0 0 auto; overflow: hidden; text-overflow: ellipsis; }
  .topw .bar { height: .6rem; border-radius: 3px; background: var(--accent); flex: 0 1 auto; max-width: 40%; opacity: .75; }
  .examples { margin: .2rem 0 .6rem; }
  mark { background: color-mix(in srgb, var(--accent-2) 35%, transparent); color: inherit; border-radius: 3px; padding: 0 .1rem; }
</style>
