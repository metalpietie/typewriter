<script lang="ts">
  import { untrack } from 'svelte';
  import Select from '../lib/Select.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import CopyButton from '../lib/CopyButton.svelte';
  import { db, plainText } from '../lib/db.svelte';
  import { nav, openScene } from '../lib/nav.svelte';
  import { orderedScenes, sceneLabel } from '../lib/scenes';
  import { findNameIssues, fixEverywhere, wordRe, type NameIssue } from '../lib/namespell';

  const TYPES = ['word', 'title', 'phrase', 'oath or curse', 'unit or measure', 'creature or plant', 'other'];

  let selId = $state<number | null>(null);
  let search = $state('');

  const terms = $derived([...db.lists.term].sort((a, b) => String(a.word).toLowerCase().localeCompare(String(b.word).toLowerCase())));
  const shown = $derived(terms.filter((t) => (t.word + ' ' + t.meaning + ' ' + (t.accepted ?? '')).toLowerCase().includes(search.toLowerCase())));
  const sel = $derived(db.get('term', selId));

  // Jump here when a search result is picked.
  $effect(() => {
    const p = nav.pending;
    if (p?.kind === 'term') { search = ''; selId = p.id; nav.pending = null; }
  });

  // Reading the whole book is slow, so it runs when you open the tab, pick a word, or press the button.
  // It does not run on every key you type.
  let issues = $state<NameIssue[]>([]);
  let used = $state<{ id: number; label: string; times: number }[]>([]);
  let checked = $state(false);
  const hidden = $derived(new Set(db.lists.plotignore.map((i) => i.key)));

  function recheck() {
    issues = findNameIssues();
    const w = String(sel?.word ?? '').trim();
    if (!w) { used = []; checked = true; return; }
    const re = new RegExp(wordRe(w).source, 'giu');
    used = orderedScenes()
      .map((s) => ({ id: s.id, label: sceneLabel(s), times: (plainText(s.body ?? '').match(re) ?? []).length }))
      .filter((u) => u.times > 0);
    checked = true;
  }
  $effect(() => { selId; untrack(recheck); });

  const mine = $derived(sel ? issues.filter((i) => !hidden.has(i.key) && i.suggest.toLowerCase() === String(sel.word ?? '').trim().toLowerCase()) : []);
  const flagged = (t: any) => issues.filter((i) => !hidden.has(i.key) && i.suggest.toLowerCase() === String(t.word ?? '').trim().toLowerCase()).length;
  const total = $derived(used.reduce((n, u) => n + u.times, 0));

  async function add() {
    const t = await db.save('term', { word: 'new word', type: 'word', pronunciation: '', meaning: '', accepted: '', notes: '' });
    search = '';
    selId = t.id;
  }
  const save = () => { if (sel) db.queue('term', sel); };
  async function del() { if (selId) { await db.remove('term', selId); selId = null; } }

  let msg = $state('');
  let fixing = $state(false);
  async function fix(n: NameIssue) {
    if (fixing) return;
    fixing = true; msg = '';
    try {
      const c = await fixEverywhere(n);
      msg = `Changed “${n.word}” to “${n.suggest}” ${c} time${c === 1 ? '' : 's'}. To undo: open History in that scene.`;
      recheck();
    } catch (e: any) {
      msg = 'Could not change it: ' + (e?.message ?? e);
    } finally { fixing = false; }
  }
  /** The word is right as it is: add it as another accepted spelling. */
  async function accept(n: NameIssue) {
    if (!sel) return;
    sel.accepted = [String(sel.accepted ?? '').trim(), n.word].filter(Boolean).join(', ');
    await db.save('term', sel);
    recheck();
  }
  const snip = (s: string) => s.split(/(⟦[^⟧]*⟧)/).filter(Boolean).map((t) => (t.startsWith('⟦') ? { t: t.slice(1, -1), hit: true } : { t, hit: false }));
</script>

<div class="split">
  <aside class="side">
    <div class="row">
      <input placeholder="Search…" bind:value={search} />
      <button class="primary" onclick={add}>+ New</button>
    </div>
    <div class="list">
      {#each shown as t (t.id)}
        <button class="item" class:on={t.id === selId} onclick={() => (selId = t.id)}>
          {t.word || 'Untitled word'}{#if flagged(t)} <span class="flag" title="Spelled another way in the text">⚠ {flagged(t)}</span>{/if}
          <small>{t.meaning || t.type}</small>
        </button>
      {:else}
        <div class="empty">No words yet. Add the made-up words of your world: a greeting, a curse, a title, a unit of time.</div>
      {/each}
    </div>
  </aside>

  <section class="pane">
    {#if sel}
      <div class="head"><h2>{sel.word || 'Untitled word'}</h2><span class="row fit" style="gap:.3rem"><CopyButton kind="term" id={sel.id} /><ConfirmButton onconfirm={del} /></span></div>
      <div class="grid2">
        <div><label for="gword">Word</label><input id="gword" bind:value={sel.word} oninput={save} onchange={recheck} /></div>
        <div><label for="gtype">Type</label>
          <Select id="gtype" bind:value={sel.type} onchange={save}>{#each TYPES as t}<option>{t}</option>{/each}</Select>
        </div>
      </div>
      <label for="gsaid">How it is said</label>
      <input id="gsaid" bind:value={sel.pronunciation} oninput={save} placeholder="For example: KETH-ah" />
      <label for="gmean">Meaning</label>
      <textarea id="gmean" bind:value={sel.meaning} oninput={save} placeholder="What does it mean? Who says it?"></textarea>
      <label for="gacc">Other spellings that are fine</label>
      <input id="gacc" bind:value={sel.accepted} oninput={save} onchange={recheck} placeholder="Comma separated. For example: kethaa, Ketha" />
      <label for="gnote">Notes</label>
      <textarea id="gnote" bind:value={sel.notes} oninput={save} placeholder="Where does it come from? Other words in the same family?"></textarea>

      <h3 style="margin-top:1.4rem">Spelling</h3>
      {#if msg}<p style="color:var(--ok)">{msg}</p>{/if}
      {#each mine as n (n.key)}
        <div class="card hole">
          <div class="head">
            <strong>“{n.word}” → “{n.suggest}”</strong>
            <span class="btns">
              <button class="small primary" disabled={fixing} onclick={() => fix(n)}>Change all to “{n.suggest}”</button>
              <button class="small" title="Keep “{n.word}” as another fine spelling" onclick={() => accept(n)}>It is fine</button>
            </span>
          </div>
          <div class="muted">Used {n.count} time{n.count === 1 ? '' : 's'}. It {n.reason}.</div>
          {#each n.places as p (p.scene.id)}
            <button class="row-link" onclick={() => openScene(p.scene.id)} title="Open it">
              <span class="chip">{p.label}{p.times > 1 ? ' · ' + p.times + '×' : ''}</span>
              <span class="t">{#each snip(p.snippet) as x}{#if x.hit}<mark>{x.t}</mark>{:else}{x.t}{/if}{/each}</span>
            </button>
          {/each}
        </div>
      {:else}
        <div class="muted">{checked ? 'No other spelling of this word was found in your scenes.' : 'Checking…'}</div>
      {/each}
      <button class="small" style="margin-top:.6rem" onclick={recheck}>Check again</button>

      <h3 style="margin-top:1.4rem">Used in the text · {total} time{total === 1 ? '' : 's'}</h3>
      <div class="card">
        {#each used as u (u.id)}
          <button class="row-link" onclick={() => openScene(u.id)} title="Open the scene">
            <span class="chip">{u.times}×</span><span class="t">{u.label}</span>
          </button>
        {:else}<span class="muted">Not used in any scene yet.</span>{/each}
      </div>
    {:else}
      <div class="empty">Pick a word, or press “+ New”.</div>
    {/if}
  </section>
</div>

<style>
  .head { display: flex; gap: .5rem; align-items: center; justify-content: space-between; }
  .btns { display: flex; gap: .3rem; flex-wrap: wrap; }
  .flag { color: var(--danger); font-size: .75rem; }
  .hole { border-color: var(--accent-2); margin-bottom: .6rem; }
  .row-link { display: flex; gap: .6rem; align-items: baseline; width: 100%; text-align: left; background: none; border: 0; border-top: 1px solid var(--line); padding: .35rem 0; border-radius: 0; }
  .row-link:first-of-type { border-top: 0; }
  .row-link:hover { background: var(--panel-2); }
  .chip { flex: 0 0 auto; font-size: .75rem; color: var(--muted); background: var(--panel-2); border: 1px solid var(--line); border-radius: 99px; padding: 0 .5rem; }
  .t { min-width: 0; overflow: hidden; text-overflow: ellipsis; }
  mark { background: color-mix(in srgb, var(--accent-2) 35%, transparent); color: inherit; border-radius: 3px; padding: 0 .1rem; }
</style>
