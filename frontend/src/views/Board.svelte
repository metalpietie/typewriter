<script lang="ts">
  import Select from '../lib/Select.svelte';
  import { flip } from 'svelte/animate';
  import { db } from '../lib/db.svelte';
  import { openScene } from '../lib/nav.svelte';
  import { STATUSES, scenesOf, sceneWords, chapterWords, addScene } from '../lib/scenes';
  import ProgressBar from '../lib/ProgressBar.svelte';
  import { arcsOfScene } from '../lib/arcs';
  import { tagCounts, tagsOf, hasTag } from '../lib/tags-core';

  // start on the book you were writing in
  const lastBook = () => {
    try { return JSON.parse(localStorage.getItem('typewriter.write') ?? 'null')?.book ?? null; } catch { return null; }
  };
  const books = $derived([...db.lists.book].sort((a, b) => a.order - b.order));
  let pick = $state<number | null>(lastBook());
  const bookId = $derived(db.get('book', pick) ? pick : (books[0]?.id ?? null));
  const book = $derived(db.get('book', bookId));
  const chapters = $derived(db.lists.chapter.filter((c) => c.book === bookId).sort((a, b) => a.order - b.order));
  const total = $derived(chapters.reduce((n, c) => n + chapterWords(c), 0));

  // show only the scenes with one tag
  let tagFilter = $state('');
  const bookTags = $derived(tagCounts(db.lists.scene.filter((s) => chapters.some((c) => c.id === s.chapter))));
  const activeTag = $derived(bookTags.some(([t]) => t === tagFilter) ? tagFilter : '');

  let colour = $state<'status' | 'pov'>('status');
  const STATUS_COLOUR: Record<string, string> = { idea: 'var(--muted)', draft: 'var(--accent)', revised: 'var(--accent-2)', final: 'var(--ok)' };
  const barColour = (s: any) => {
    if (colour === 'pov') return db.get('character', s.pov)?.color ?? 'var(--line)';
    return STATUS_COLOUR[s.status] ?? 'var(--muted)';
  };
  const placeOf = (s: any) => db.get('location', s.location)?.name ?? '';
  const povOf = (s: any) => db.get('character', s.pov)?.name ?? '';
  const goalOf = (c: any) => Number(c.goal) || Number(book?.chapterGoal) || 0;

  const saveScene = (s: any) => db.queue('scene', s);
  const saveChap = (c: any) => db.queue('chapter', c);

  async function newScene(chapter: any) {
    const s = await addScene(chapter);
    openScene(s.id);
  }
  async function newChapter() {
    if (!bookId) return;
    const c = await db.save('chapter', { book: bookId, order: chapters.length + 1, title: 'New chapter', summary: '' });
    await addScene(c);
  }

  // ----- drag to reorder (scenes up and down, and across chapters; chapters left and right) -----
  type DragItem = { type: 'scene' | 'chapter'; id: number; label: string };
  type Drop = { type: 'scene' | 'chapter'; id: number; after: boolean; into?: boolean };
  let drag = $state<DragItem | null>(null);
  let drop = $state<Drop | null>(null);
  let ghost = $state({ x: 0, y: 0 });
  let landed = $state('');
  let landedTimer: ReturnType<typeof setTimeout>;
  const flash = (key: string) => { landed = key; clearTimeout(landedTimer); landedTimer = setTimeout(() => (landed = ''), 900); };
  let suppressClick = false;

  function startDrag(e: PointerEvent, item: DragItem) {
    if (e.button !== 0) return;
    e.preventDefault(); // no text selection while you drag
    window.getSelection()?.removeAllRanges();
    const sx = e.clientX, sy = e.clientY;
    let active = false;
    const noSelect = (ev: Event) => ev.preventDefault();
    window.addEventListener('selectstart', noSelect);
    const move = (ev: PointerEvent) => {
      if (!active) {
        if (Math.hypot(ev.clientX - sx, ev.clientY - sy) < 6) return;
        active = true; drag = item;
        (document.activeElement as HTMLElement | null)?.blur?.();
      }
      ghost = { x: ev.clientX, y: ev.clientY };
      drop = findDrop(ev.clientX, ev.clientY, item);
      scrollNear(ev.clientX);
    };
    const up = async () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
      window.removeEventListener('pointercancel', up);
      window.removeEventListener('selectstart', noSelect);
      clearInterval(scroller); scroller = undefined;
      if (!active) return;
      suppressClick = true;
      setTimeout(() => (suppressClick = false), 0);
      const d = drop;
      drag = null; drop = null;
      if (d) await applyDrop(item, d);
    };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
    window.addEventListener('pointercancel', up);
  }

  // the board scrolls sideways by itself when a card is held near its edge
  let board: HTMLElement | undefined = $state();
  let scroller: ReturnType<typeof setInterval> | undefined;
  function scrollNear(x: number) {
    if (!board) return;
    const r = board.getBoundingClientRect();
    const speed = x < r.left + 60 ? -14 : x > r.right - 60 ? 14 : 0;
    clearInterval(scroller); scroller = undefined;
    if (speed) scroller = setInterval(() => board!.scrollBy(speed, 0), 16);
  }

  function findDrop(x: number, y: number, item: DragItem): Drop | null {
    const el = document.elementFromPoint(x, y)?.closest('[data-scene],[data-col]') as HTMLElement | null;
    if (!el) return null;
    const r = el.getBoundingClientRect();
    if (el.dataset.scene && item.type === 'scene') {
      const id = +el.dataset.scene;
      return id !== item.id ? { type: 'scene', id, after: y > r.top + r.height / 2 } : null;
    }
    const col = (el.closest('[data-col]') as HTMLElement | null) ?? el;
    const cid = +col.dataset.col!;
    if (item.type === 'chapter') {
      const cr = col.getBoundingClientRect();
      return cid !== item.id ? { type: 'chapter', id: cid, after: x > cr.left + cr.width / 2 } : null;
    }
    return { type: 'chapter', id: cid, after: false, into: true }; // a scene dropped on a column goes to its end
  }

  /** Put the dragged item in its new place and renumber what changed. */
  async function applyDrop(item: DragItem, d: Drop) {
    if (item.type === 'chapter') {
      const moving = db.get('chapter', item.id);
      const target = db.get('chapter', d.id);
      if (!moving || !target) return;
      const list = chapters.filter((c) => c.id !== moving.id);
      list.splice(list.findIndex((c) => c.id === target.id) + (d.after ? 1 : 0), 0, moving);
      for (const [i, c] of list.entries()) if (c.order !== i + 1) { c.order = i + 1; await db.save('chapter', c); }
      flash('chapter' + moving.id);
      return;
    }
    const moving = db.get('scene', item.id);
    if (!moving) return;
    const fromChapter = moving.chapter;
    let targetChapter: number;
    let list: any[];
    let at: number;
    if (d.type === 'scene') {
      const t = db.get('scene', d.id);
      if (!t) return;
      targetChapter = t.chapter;
      list = scenesOf(targetChapter).filter((x) => x.id !== moving.id);
      at = list.findIndex((x) => x.id === t.id) + (d.after ? 1 : 0);
    } else {
      targetChapter = d.id;
      list = scenesOf(targetChapter).filter((x) => x.id !== moving.id);
      at = list.length;
    }
    moving.chapter = targetChapter;
    list.splice(at, 0, moving);
    for (const [i, x] of list.entries()) if (x === moving || x.order !== i + 1) { x.order = i + 1; await db.save('scene', x); }
    if (fromChapter !== targetChapter) {
      const rest = scenesOf(fromChapter);
      for (const [i, x] of rest.entries()) if (x.order !== i + 1) { x.order = i + 1; await db.save('scene', x); }
    }
    flash('scene' + moving.id);
  }

  const dropClass = (type: 'scene' | 'chapter', id: number) =>
    drop && drop.type === type && drop.id === id ? (drop.into ? 'drop-into' : drop.after ? 'drop-after' : 'drop-before') : '';
</script>

<section class="pane boardpane">
  <div class="top">
    <h2 style="margin:0">Board</h2>
    <Select aria-label="Book" bind:value={pick}>
      {#each books as b}<option value={b.id}>{b.order}. {b.title}</option>{/each}
    </Select>
    {#if bookTags.length}
      <Select aria-label="Show scenes with a tag" bind:value={tagFilter}>
        <option value="">All scenes</option>
        {#each bookTags as [t, n]}<option value={t}>Tag: {t} ({n})</option>{/each}
      </Select>
    {/if}
    <div class="seg">
      <button class:active={colour === 'status'} onclick={() => (colour = 'status')}>Colour by status</button>
      <button class:active={colour === 'pov'} onclick={() => (colour = 'pov')}>Colour by point of view</button>
    </div>
    <span class="muted">{total.toLocaleString()} words</span>
  </div>
  <p class="muted" style="margin:.3rem 0 .6rem;font-size:.85rem">
    One card for each scene. Drag the grip <b>⠿</b> on a card to put it in another place or another chapter. Drag the grip next to a chapter's title to move the whole chapter. Click a card's title to write that scene.
  </p>

  {#if !book}
    <div class="empty">No book yet. Add one in the Write tab.</div>
  {:else}
    <div class="board" class:sorting={!!drag} bind:this={board}>
      {#each chapters as c (c.id)}
        {@const scenes = scenesOf(c.id).filter((s) => hasTag(s, activeTag))}
        {@const words = chapterWords(c)}
        <div class="col {dropClass('chapter', c.id)}" class:landed={landed === 'chapter' + c.id} class:dragging={drag?.type === 'chapter' && drag.id === c.id}
          data-col={c.id} animate:flip={{ duration: 220 }}>
          <div class="colhead">
            <div class="hrow">
              <span class="grip" role="button" tabindex="-1" title="Drag to move this chapter" aria-label="Drag to move this chapter"
                onpointerdown={(e) => startDrag(e, { type: 'chapter', id: c.id, label: c.title || 'Chapter' })}>⠿</span>
              <input class="ctitle" aria-label="Chapter title" bind:value={c.title} oninput={() => saveChap(c)} />
            </div>
            <div class="muted small">{c.order}. {scenes.length} scene{scenes.length === 1 ? '' : 's'} · {words.toLocaleString()}{goalOf(c) ? ' / ' + goalOf(c).toLocaleString() : ''} words</div>
            {#if goalOf(c)}<div style="margin-top:.2rem"><ProgressBar value={words} goal={goalOf(c)} thin /></div>{/if}
          </div>
          <div class="cards">
            {#each scenes as s (s.id)}
              <div class="card idx {dropClass('scene', s.id)}" class:landed={landed === 'scene' + s.id} class:dragging={drag?.type === 'scene' && drag.id === s.id}
                style:border-left-color={barColour(s)} data-scene={s.id} animate:flip={{ duration: 200 }}>
                <div class="hrow">
                  <button class="title" title="Write this scene" onclick={() => { if (!suppressClick) openScene(s.id); }}>{s.title || 'Untitled scene'}</button>
                  <span class="grip" role="button" tabindex="-1" title="Drag to move this scene" aria-label="Drag to move this scene"
                    onpointerdown={(e) => startDrag(e, { type: 'scene', id: s.id, label: s.title || 'Untitled scene' })}>⠿</span>
                </div>
                <textarea rows="3" placeholder="What happens in this scene?" aria-label="Summary" bind:value={s.summary} oninput={() => saveScene(s)}></textarea>
                <div class="foot muted">
                  <Select class="st" aria-label="Status" bind:value={s.status} onchange={() => saveScene(s)}>{#each STATUSES as t}<option>{t}</option>{/each}</Select>
                  {#each tagsOf(s) as t (t)}<span class="tag hash" title="Tag">#{t}</span>{/each}
                  {#each arcsOfScene(s.id) as a (a.id)}<span class="adot" style:background={a.color} title="Arc: {a.name}"></span>{/each}
                  {#if povOf(s)}<span class="tag" title="Point of view">{povOf(s)}</span>{/if}
                  {#if placeOf(s)}<span class="tag" title="Place">{placeOf(s)}</span>{/if}
                  <span class="fit">{sceneWords(s).toLocaleString()} words</span>
                </div>
              </div>
            {/each}
            <button class="small addscene" onclick={() => newScene(c)}>+ Scene</button>
          </div>
        </div>
      {/each}
      <div class="col addcol"><button class="small primary" onclick={newChapter}>+ Chapter</button></div>
    </div>
  {/if}
</section>

{#if drag}<div class="ghost" style="transform: translate({ghost.x + 14}px, {ghost.y + 8}px)"><span>{drag.label}</span></div>{/if}

<style>
  .boardpane { display: flex; flex-direction: column; min-height: 0; }
  .top { display: flex; gap: .8rem; align-items: center; flex-wrap: wrap; }
  .top :global(.select) { flex: 0 1 16rem; }
  .board { flex: 1; min-height: 0; display: flex; gap: .8rem; overflow: auto; align-items: flex-start; padding-bottom: .8rem; }
  .board.sorting { user-select: none; -webkit-user-select: none; cursor: grabbing; }
  .board.sorting :global(*) { cursor: grabbing !important; }
  .idx .foot, .idx .title { user-select: none; -webkit-user-select: none; }
  .col { flex: 0 0 17rem; background: var(--panel); border: 1px solid var(--line); border-radius: 10px; padding: .6rem; display: flex; flex-direction: column; gap: .5rem; max-height: 100%; position: relative; }
  .col.addcol { background: none; border-style: dashed; flex-basis: 8rem; align-items: center; padding: 1rem; }
  .colhead { user-select: none; -webkit-user-select: none; }
  .hrow { display: flex; align-items: center; gap: .3rem; }
  .grip { flex: 0 0 auto; width: 1.5rem; height: 1.8rem; display: grid; place-items: center; border-radius: 6px; color: var(--muted); font-size: 1.15rem; line-height: 1; cursor: grab; user-select: none; -webkit-user-select: none; touch-action: none; }
  .grip:hover { background: var(--panel-2); color: var(--accent); }
  .grip:active { cursor: grabbing; color: var(--accent); }
  .ctitle { flex: 1; min-width: 0; font-weight: 600; background: none; border: 1px solid transparent; padding: .15rem .3rem; }
  .ctitle:hover, .ctitle:focus { border-color: var(--line); background: var(--panel-2); }
  .small { font-size: .75rem; }
  .cards { display: flex; flex-direction: column; gap: .5rem; overflow: auto; min-height: 2rem; }
  .idx { margin: 0; padding: .5rem .6rem; border-left: 4px solid var(--line); position: relative; background: var(--panel-2); transition: opacity .15s, transform .15s; }
  .idx .title { display: block; flex: 1; min-width: 0; text-align: left; background: none; border: 0; padding: 0; font-weight: 600; color: var(--text, inherit); cursor: pointer; }
  .idx .title:hover { text-decoration: underline; }
  .idx textarea { min-height: 3.6rem; margin: .3rem 0; font-size: .85rem; resize: vertical; background: var(--panel); }
  .foot { display: flex; gap: .4rem; align-items: center; flex-wrap: wrap; font-size: .75rem; }
  .foot :global(.st) { width: auto; padding: .05rem .3rem; font-size: .75rem; }
  .foot .fit { margin-left: auto; }
  .tag { background: var(--panel); border: 1px solid var(--line); border-radius: 99px; padding: 0 .5rem; max-width: 7rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .hash { color: var(--accent); }
  .adot { width: .6rem; height: .6rem; border-radius: 50%; flex: 0 0 auto; }
  .addscene { align-self: flex-start; }
  .dragging { opacity: .35; transform: scale(.97); }

  /* a line shows where the card or column will land */
  .drop-before::before, .drop-after::before { content: ''; position: absolute; background: var(--accent); box-shadow: 0 0 8px var(--accent); border-radius: 2px; pointer-events: none; z-index: 2; }
  .idx.drop-before::before, .idx.drop-after::before { left: 0; right: 0; height: 3px; }
  .idx.drop-before::before { top: -4px; }
  .idx.drop-after::before { bottom: -4px; }
  .col.drop-before::before, .col.drop-after::before { top: 0; bottom: 0; width: 3px; }
  .col.drop-before::before { left: -6px; }
  .col.drop-after::before { right: -6px; }
  .drop-into { outline: 2px dashed var(--accent); outline-offset: -2px; background: color-mix(in srgb, var(--accent) 12%, var(--panel)); }
  .landed { animation: landed .9s ease-out; }
  @keyframes landed { 0% { background: color-mix(in srgb, var(--accent) 45%, transparent); } 100% { background: transparent; } }

  .ghost { position: fixed; left: 0; top: 0; z-index: 70; pointer-events: none; max-width: 230px; }
  .ghost span { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; background: var(--panel); border: 1px solid var(--accent); border-radius: 8px; padding: .3rem .7rem; font-size: .85rem; box-shadow: 0 12px 28px rgba(0, 0, 0, .45); }
  @media (prefers-reduced-motion: reduce) { .landed { animation: none; } .idx { transition: none; } }
</style>
