<script lang="ts">
  import { takeCommand } from '../lib/commands';
  import Select from '../lib/Select.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import { db } from '../lib/db.svelte';
  import { nav } from '../lib/nav.svelte';
  import { word } from '../lib/projecttype.svelte';
  import { IDEA_KINDS, cleanIdea } from '../lib/boxes-core';
  import { citeLabel, sourceLine } from '../lib/cite-core';

  // Short items to keep: an idea, a quote, an anecdote, a statistic. A quote or a statistic can name its source.
  // One button puts an item in the text, with its citation.
  let selId = $state<number | null>(null);
  let search = $state('');
  let kindFilter = $state('');

  const items = $derived([...db.lists.idea].sort((a, b) => b.id - a.id));
  const shown = $derived(items.filter((i) => (!kindFilter || (i.form ?? 'idea') === kindFilter) && (String(i.text ?? '') + ' ' + (i.tags ?? '')).toLowerCase().includes(search.toLowerCase())));
  const sel = $derived(db.get('idea', selId));
  const sources = $derived([...db.lists.source].sort((a, b) => sourceLine(a).toLowerCase().localeCompare(sourceLine(b).toLowerCase())));
  const src = $derived(sel?.source ? db.get('source', sel.source) : undefined);

  // Jump here when a search result is picked.
  $effect(() => {
    const p = nav.pending;
    if (p?.kind === 'idea') { search = ''; kindFilter = ''; selId = p.id; nav.pending = null; }
  });

  const kindName = (k: string) => IDEA_KINDS.find(([x]) => x === k)?.[1] ?? 'Idea';
  async function add() {
    const made = await db.save('idea', { ...cleanIdea({ form: kindFilter || 'idea' }) });
    search = '';
    selId = made.id;
  }
  const save = () => { if (sel) db.queue('idea', sel); };
  const pickSource = (v: string) => { if (sel) { sel.source = v ? Number(v) : null; save(); } };
  async function del() { if (selId) { await db.remove('idea', selId); selId = null; } }

  /** Put the item at the end of the text that is open in the Write tab. */
  function use() {
    if (!sel || !String(sel.text ?? '').trim()) return;
    nav.insert = { kind: sel.form ?? 'idea', text: String(sel.text).trim(), source: sel.source ?? null, loc: sel.loc ?? '' };
    nav.tab = 'write';
  }

  $effect(() => { if (takeCommand('new')) add(); }); // the command menu
</script>

<div class="split">
  <aside class="side">
    <div class="row">
      <input placeholder="Search…" bind:value={search} />
      <button class="primary" onclick={add}>+ New</button>
    </div>
    <Select aria-label="Show items" bind:value={kindFilter}>
      <option value="">All items</option>
      {#each IDEA_KINDS as [k, name]}<option value={k}>{name}s</option>{/each}
    </Select>
    <div class="list">
      {#each shown as i (i.id)}
        <button class="item" class:on={i.id === selId} onclick={() => (selId = i.id)}>
          {String(i.text ?? '').slice(0, 70) || 'Empty ' + kindName(i.form ?? 'idea').toLowerCase()}
          <small>{kindName(i.form ?? 'idea')}{i.source ? ' · ' + citeLabel(db.get('source', i.source), '') : ''}</small>
        </button>
      {:else}
        <div class="empty">Nothing yet. Keep your ideas, quotes, stories and numbers here. Then put them in the text with one click.</div>
      {/each}
    </div>
  </aside>

  <section class="pane">
    {#if sel}
      <div class="head"><h2>{kindName(sel.form ?? 'idea')}</h2><ConfirmButton onconfirm={del} /></div>
      <div class="grid2">
        <div><label for="ikind">Kind</label>
          <Select id="ikind" bind:value={sel.form} onchange={save}>{#each IDEA_KINDS as [k, name]}<option value={k}>{name}</option>{/each}</Select>
        </div>
        <div><label for="isrc">Source</label>
          <Select id="isrc" value={sel.source ? String(sel.source) : ''} onchange={(e) => pickSource(e.currentTarget.value)}>
            <option value="">No source</option>
            {#each sources as s (s.id)}<option value={String(s.id)}>{sourceLine(s)}</option>{/each}
          </Select>
        </div>
      </div>
      <label for="itext">{sel.form === 'quote' ? 'The words' : 'Text'}</label>
      <textarea id="itext" rows="5" bind:value={sel.text} oninput={save} placeholder={sel.form === 'statistic' ? 'For example: 43% of habits are done in the same place every day.' : 'Write it as it should stand in the book.'}></textarea>
      {#if sel.source}
        <label for="iloc">Page or place in the source <small class="muted">(optional)</small></label>
        <input id="iloc" bind:value={sel.loc} oninput={save} placeholder="45 or 45-47 or ch. 3" />
        <p class="muted" style="margin-top:.4rem">In the text it ends with <b>{citeLabel(src, sel.loc)}</b>.</p>
      {:else if sel.form === 'quote' || sel.form === 'statistic'}
        <p class="muted" style="margin-top:.4rem">A {sel.form} should name its source, so the book can cite it.</p>
      {/if}
      <label for="itags">Tags</label>
      <input id="itags" bind:value={sel.tags} oninput={save} placeholder="For example: sleep, habits" />
      <label for="inote">Notes</label>
      <textarea id="inote" bind:value={sel.notes} oninput={save} placeholder="Where could it go? What does it show?"></textarea>
      <div style="margin-top:1rem">
        <button class="primary" disabled={!String(sel.text ?? '').trim()} onclick={use}>Put it in the text</button>
        <span class="muted"> It goes at the end of the {word('scene')} that is open in the Write tab.</span>
      </div>
    {:else}
      <div class="empty">Pick an item, or press “+ New”.</div>
    {/if}
  </section>
</div>

<style>
  .head { display: flex; gap: .5rem; align-items: center; justify-content: space-between; }
</style>
