<script lang="ts">
  import { veil, dialog } from './motion.svelte';
  import { onMount } from 'svelte';
  import { db, toHtml } from './db.svelte';
  import { listSnapshots, takeSnapshot, deleteSnapshot, renameSnapshot, defaultName, currentBook, snapshotItems, type Snapshot } from './snapshots';
  import { compare, totals, isDifferent, type Row } from './snapshot-core';
  import { diffWords, countChanges, paragraphsOf } from './diff-core';
  import { blockText } from './diff';
  import ConfirmButton from './ConfirmButton.svelte';

  let { bookId, onrestore, onclose }: {
    bookId: number;
    /** Put the old text of one scene back. */
    onrestore: (sceneId: number, body: string) => void | Promise<void>;
    onclose: () => void;
  } = $props();

  let snaps = $state<Snapshot[]>([]);
  let picked = $state<Snapshot | null>(null);
  let name = $state('');
  let error = $state('');
  let loaded = $state(false);
  let open = $state<number | null>(null); // the scene shown in detail
  let onlyChanged = $state(true);
  let note = $state('');
  let busy = $state(false);

  const book = $derived(db.get('book', bookId));

  async function load() {
    try { snaps = await listSnapshots(bookId); } catch (e: any) { error = e?.message ?? String(e); }
    loaded = true;
    if (picked) picked = snaps.find((s) => s.id === picked!.id) ?? null;
  }
  onMount(load);

  const when = (iso: string) => new Date(iso).toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' });
  const n = (x: number) => x.toLocaleString();
  const plural = (x: number, one: string) => `${n(x)} ${one}${x === 1 ? '' : 's'}`;

  async function save() {
    if (busy) return;
    busy = true; error = '';
    try {
      const s = await takeSnapshot(bookId, name);
      name = '';
      await load();
      picked = snaps.find((x) => x.id === s.id) ?? null;
      open = null;
    } catch (e: any) { error = e?.message ?? String(e); }
    busy = false;
  }
  async function remove(s: Snapshot) {
    await deleteSnapshot(s.id);
    if (picked?.id === s.id) { picked = null; open = null; }
    await load();
  }
  /** Save the new name when you leave the box or press Enter. An empty name goes back to the old one. */
  async function rename(s: Snapshot, e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    try {
      const next = await renameSnapshot(s, input.value);
      if (next !== s.name) await load();
      input.value = next;
    } catch (err: any) { error = err?.message ?? String(err); input.value = s.name; }
  }

  function pick(s: Snapshot) { picked = s; open = null; note = ''; }

  // The comparison. It is worked out again when you pick another snapshot or write more.
  const rows = $derived.by(() => {
    if (!picked) return [] as Row[];
    const now = currentBook(bookId);
    return compare(snapshotItems(picked), picked.chapters, now.scenes, now.chapters);
  });
  const sum = $derived(totals(rows));
  const shown = $derived(onlyChanged ? rows.filter(isDifferent) : rows);
  const row = $derived(open == null ? null : rows.find((r) => r.id === open) ?? null);

  const oldScene = $derived(picked && row ? picked.scenes.find((s) => s.id === row.id) ?? null : null);
  const nowScene = $derived(row ? db.get('scene', row.id) : null);
  const pieces = $derived(row?.state === 'changed' && oldScene && nowScene ? diffWords(blockText(oldScene.body), blockText(nowScene.body ?? '')) : null);
  const counts = $derived(pieces ? countChanges(pieces) : null);

  const label: Record<string, string> = { same: 'same', changed: 'changed', new: 'new', gone: 'removed' };

  async function putBack() {
    if (!oldScene || !row) return;
    await onrestore(row.id, oldScene.body);
    note = 'Done. The old text is back in the scene. The text from before is saved in the History of that scene.';
  }
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); if (open != null) open = null; else onclose(); } }} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="overlay" transition:veil|global onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
  <div transition:dialog|global class="palette snaps">
    <div class="head"><strong>Snapshots: {book?.title || 'this book'}</strong><button class="small" onclick={onclose}>Close</button></div>
    <div class="row">
      <input bind:value={name} placeholder={defaultName() + ' (name it, for example: after beta readers)'} onkeydown={(e) => { if (e.key === 'Enter') save(); }} />
      <button class="primary fit" disabled={busy} onclick={save}>Save a snapshot now</button>
    </div>
    {#if error}<p class="err">{error}</p>{/if}
    <p class="muted">A snapshot is a copy of the whole book as it is now. Later you can see what changed in each scene. Snapshots stay until you delete them.</p>

    <div class="cols">
      <div class="list">
        {#each snaps as s (s.id)}
          <button class="item" class:on={picked?.id === s.id} onclick={() => pick(s)}>
            <span class="nm">{s.name}</span>
            <small>{when(s.at)}</small>
            <small>{n(s.words)} words · {plural(s.scenes.length, 'scene')}</small>
          </button>
        {:else}
          <p class="muted">{loaded ? 'No snapshots yet.' : 'Loading…'}</p>
        {/each}
      </div>

      <div class="view">
        {#if !picked}
          <p class="muted">Pick a snapshot to see what changed since then.</p>
        {:else if !row}
          <div class="head">
            <label class="namebox">
              <span class="muted">Name</span>
              {#key picked.id + picked.name}
                <input class="nameinput" value={picked.name} maxlength="80" aria-label="Name of this snapshot" title="Click to change the name"
                  onchange={(e) => rename(picked!, e)} onblur={(e) => rename(picked!, e)}
                  onkeydown={(e) => { if (e.key === 'Enter') (e.currentTarget as HTMLInputElement).blur(); else if (e.key === 'Escape') { e.stopPropagation(); (e.currentTarget as HTMLInputElement).value = picked!.name; (e.currentTarget as HTMLInputElement).blur(); } }} />
              {/key}
              <span class="muted">{when(picked.at)}</span>
            </label>
            <ConfirmButton onconfirm={() => remove(picked!)} />
          </div>
          <p class="summary">
            The book had <b>{n(sum.wordsThen)}</b> words then and has <b>{n(sum.wordsNow)}</b> now.
            You <span class="add">added {plural(sum.wordsAdded, 'word')}</span> and <span class="del">cut {plural(sum.wordsCut, 'word')}</span>.
          </p>
          <p class="muted counts">
            {plural(sum.changed, 'scene')} changed · {n(sum.added)} new · {n(sum.gone)} removed · {n(sum.moved)} moved to another chapter
          </p>
          <label class="chk"><input type="checkbox" bind:checked={onlyChanged} /> Show only what changed</label>
          <div class="rows">
            {#each shown as r (r.id)}
              <button class="r {r.state}" onclick={() => { open = r.id; note = ''; }}>
                <span class="t">{r.title || 'Untitled'}<small>{r.chapter ? ' · ' + r.chapter : ''}</small></span>
                <span class="tag {r.state}">{label[r.state]}{r.moved ? ' · moved' : ''}</span>
                <small class="nums">
                  {#if r.state === 'changed'}<span class="add">+{n(r.added)}</span> <span class="del">−{n(r.cut)}</span>
                  {:else if r.state === 'new'}<span class="add">+{n(r.added)}</span>
                  {:else if r.state === 'gone'}<span class="del">−{n(r.cut)}</span>{/if}
                </small>
              </button>
            {:else}
              <p class="muted">{rows.length ? 'No scene is different. The book is the same as the snapshot.' : 'The book had no scenes then, and has none now.'}</p>
            {/each}
          </div>
        {:else}
          <div class="head">
            <button class="small" onclick={() => (open = null)}>← Back to the list</button>
            <span class="muted">{row.title || 'Untitled'}{row.chapter ? ' · ' + row.chapter : ''} · {label[row.state]}{row.moved ? ' · moved to another chapter' : ''}</span>
          </div>
          {#if note}<p class="ok">{note}</p>{/if}
          {#if row.state === 'changed' && pieces && counts}
            <div class="head">
              <p class="muted legend">
                {n(row.wordsThen)} → {n(row.wordsNow)} words. Since the snapshot, you <span class="add">added {plural(counts.added, 'word')}</span> and <span class="del">cut {plural(counts.cut, 'word')}</span>.
              </p>
              <button class="primary small fit" onclick={putBack}>Put back the old text</button>
            </div>
            <div class="preview diff">
              {#each paragraphsOf(pieces) as para}
                <p>{#each para as pc}<span class={pc.kind === 'same' ? '' : pc.kind}>{pc.text}</span>{' '}{/each}</p>
              {/each}
            </div>
          {:else if row.state === 'gone' && oldScene}
            <p class="muted legend">This scene is gone now. This is its text at the snapshot. You can copy from it.</p>
            <div class="preview">{@html toHtml(oldScene.body)}</div>
          {:else if row.state === 'new'}
            <p class="muted">You wrote this scene after the snapshot. It has {plural(row.wordsNow, 'word')}.</p>
          {:else}
            <p class="muted">The text is the same as in the snapshot.{row.moved ? ' The scene was moved to another chapter.' : ''}</p>
          {/if}
        {/if}
      </div>
    </div>
  </div>
</div>

<style>
  .snaps { width: min(960px, 94vw); height: 78vh; display: flex; flex-direction: column; }
  .cols { display: grid; grid-template-columns: 240px 1fr; gap: .8rem; flex: 1; min-height: 0; }
  .list { overflow: auto; display: flex; flex-direction: column; gap: 3px; }
  .list .item { display: flex; flex-direction: column; align-items: flex-start; text-align: left; gap: .05rem; }
  .list .nm { font-weight: 600; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .view { overflow: auto; border-left: 1px solid var(--line); padding-left: .8rem; min-width: 0; }
  .view .head { gap: .6rem; align-items: center; }
  .namebox { display: flex; align-items: center; gap: .5rem; flex: 1; min-width: 0; margin: 0; }
  .nameinput { flex: 1; min-width: 0; font-weight: 600; padding: .25rem .5rem; }
  .namebox .muted { white-space: nowrap; font-size: .8rem; }
  .summary { margin: .4rem 0 .2rem; }
  .counts { margin: 0 0 .5rem; font-size: .85rem; }
  .chk { display: flex; gap: .4rem; align-items: center; margin: 0 0 .5rem; color: var(--text); font-size: .85rem; }
  .chk input { width: auto; }
  .rows { display: flex; flex-direction: column; gap: 3px; }
  .r { display: flex; align-items: center; gap: .6rem; text-align: left; padding: .35rem .6rem; }
  .r .t { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .r .t small { color: var(--muted); }
  .tag { font-size: .72rem; border: 1px solid var(--line); border-radius: 99px; padding: 0 .5rem; white-space: nowrap; color: var(--muted); }
  .tag.changed { color: var(--accent-2); border-color: var(--accent-2); }
  .tag.new { color: var(--ok); border-color: var(--ok); }
  .tag.gone { color: var(--danger, #d9625b); border-color: var(--danger, #d9625b); }
  .nums { min-width: 5.5rem; text-align: right; white-space: nowrap; }
  .preview { font-family: Georgia, 'Times New Roman', serif; line-height: 1.7; }
  .preview :global(p) { margin: 0 0 .8em; }
  .preview :global(li > p) { margin: 0; }
  .preview :global(hr) { border: 0; text-align: center; }
  .preview :global(hr::after) { content: '* * *'; color: var(--muted); letter-spacing: .5em; }
  .legend { margin: .2rem 0 .6rem; }
  .add { background: color-mix(in srgb, var(--ok) 28%, transparent); border-bottom: 2px solid var(--ok); border-radius: 3px; padding: 0 .15em; }
  .del { background: color-mix(in srgb, var(--danger, #d9625b) 24%, transparent); text-decoration: line-through; border-radius: 3px; padding: 0 .15em; }
  .err { color: var(--danger, #d9625b); margin: .2rem 0; }
  .ok { color: var(--ok); margin: .2rem 0 .6rem; }
</style>
