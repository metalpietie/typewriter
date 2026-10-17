<script module lang="ts">
  // The first time in a run of the app, the shelf is just there, so the app does not show through while it fades in.
  // When it is opened again later (the Bookshelf button), it fades in. This lives outside the component, which is made anew each time.
  let firstShow = true;
</script>

<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { flip } from 'svelte/animate';
  import { fade } from 'svelte/transition';
  import { Store } from '../../bindings/typewriter/internal/app';
  import type { Shelf as ApiShelf, ShelfBook } from '../../bindings/typewriter/internal/app/models.js';
  import { db } from './db.svelte';
  import { nav, closeShelf, reloadInto, lastWriteKey } from './nav.svelte';
  import { params, veil, dialog } from './motion.svelte';
  import { bezier } from './motion-core';
  import { generatedCover, savedCover, forgetCovers } from './shelfcover';
  import ProgressBar from './ProgressBar.svelte';
  import { bookLine, dragStarted, dropIndex, edgeScroll, insertAt, shelfAt } from './shelf-core';

  type Shelf = Omit<ApiShelf, 'books'> & { books: ShelfBook[] }; // the list of books is never empty-null here
  let shelves = $state<Shelf[]>([]);
  let loaded = $state(false);
  let intro = $state(true); // the first arrival of the books is animated, later changes are not
  let busy = $state(false);
  let error = $state('');
  let saved = $state<Record<string, string>>({}); // covers that the writer made, by "project/book"
  let scroller = $state<HTMLElement>();

  const arrive = (node: Element) => { if (firstShow) { firstShow = false; return { duration: 0 }; } return veil(node); };

  const key = (project: string, book: number) => project + '/' + book;

  async function load() {
    try {
      const list = (await Store.Shelves()) ?? [];
      shelves = list.map((s) => ({ ...s, books: s.books ?? [] }));
      loaded = true;
      for (const s of shelves) for (const b of s.books) if (b.hasCover && saved[key(s.id, b.id)] == null) savedCover(s.id, b.id).then((url) => (saved[key(s.id, b.id)] = url));
    } catch (e: any) {
      error = e?.message ?? String(e);
    }
  }
  onMount(() => {
    forgetCovers(); // a cover may have changed in Export & Backup since the shelf was last open
    load();
    const t = setTimeout(() => (intro = false), 1500);
    return () => clearTimeout(t);
  });

  // The Projects window (Manage shelves) and Project settings can rename or remove a shelf: read the shelves again when they close.
  let windowOpen = false;
  $effect(() => {
    const open = nav.projects || nav.projectSettings;
    if (windowOpen && !open) { saved = {}; forgetCovers(); load(); }
    windowOpen = open;
  });

  /** Wait until every edit has reached the data file, so nothing is lost when the file is changed from here. */
  const settle = () => db.settle();

  /** Run a change of the data. When the open project is touched, the app reads its data again afterwards. */
  async function mutate(projects: string[], job: () => Promise<void>): Promise<boolean> {
    busy = true; error = '';
    try {
      const open = shelves.some((s) => s.current && projects.includes(s.id));
      if (open) await settle();
      await job();
      if (open) await db.load();
      return true;
    } catch (e: any) {
      error = e?.message ?? String(e);
      return false;
    } finally { busy = false; }
  }

  // ---- the cover of a book ----
  function coverOf(s: Shelf, b: ShelfBook): string {
    if (b.hasCover) return saved[key(s.id, b.id)] ?? '';
    return generatedCover(b.title, b.seed);
  }

  let seeding = $state<Record<string, boolean>>({}); // a new cover is being saved for these books
  async function refresh(s: Shelf, b: ShelfBook) {
    const k = key(s.id, b.id);
    if (seeding[k] || busy) return;
    seeding[k] = true;
    const before = b.seed || 0, seed = before + 1;
    b.seed = seed; // show it at once
    const ok = await mutate([s.id], async () => {
      await Store.SetBookSeed(s.id, b.id, seed);
      const mine = s.current ? db.get('book', b.id) : undefined;
      if (mine) mine.shelfSeed = seed;
    });
    if (!ok) b.seed = before; // it was not saved: show the cover that is saved
    seeding[k] = false;
  }

  // ---- opening a book ----
  async function open(s: Shelf, b: ShelfBook) {
    if (busy || drag) return;
    // the place that Write opens; the book you were writing in keeps its chapter and scene
    try {
      const place = lastWriteKey(s.id);
      if (JSON.parse(localStorage.getItem(place) ?? 'null')?.book !== b.id) localStorage.setItem(place, JSON.stringify({ book: b.id, chapter: null, scene: null }));
    } catch { /* fine without it */ }
    if (s.current) { nav.book = b.id; closeShelf('write'); return; }
    busy = true; error = '';
    try {
      await settle();
      await Store.SwitchProject(s.id); // the app reloads, and opens on the Write tab
      reloadInto('write');
      location.reload();
    } catch (e: any) {
      error = e?.message ?? String(e);
      busy = false;
    }
  }

  // ---- adding a book, adding and renaming a shelf ----
  let adding = $state<{ project: string; title: string } | null>(null);
  async function addBook() {
    const a = adding;
    adding = null;
    if (!a || !a.title.trim()) return;
    await mutate([a.project], async () => { await Store.AddBook(a.project, a.title); });
    await load();
  }

  let newShelf = $state<string | null>(null);
  async function addShelf() {
    const name = newShelf;
    newShelf = null;
    if (name == null || !name.trim()) return;
    await mutate([], async () => { await Store.AddProject(name); });
    await load();
  }

  let renaming = $state<{ id: string; name: string } | null>(null);
  async function renameShelf() {
    const r = renaming;
    renaming = null;
    const s = shelves.find((x) => x.id === r?.id);
    if (!r || !s || !r.name.trim() || r.name.trim() === s.name) return;
    const ok = await mutate([], async () => { await Store.RenameProject(r.id, r.name); });
    if (ok && s.current) db.project = r.name.trim();
    await load();
  }

  // ---- dragging a book ----
  type Drag = {
    book: ShelfBook; from: string; origIndex: number; src: string;
    w: number; h: number; offX: number; offY: number; x: number; y: number; // the size of the card, where it was grabbed, where the pointer is
    left: number; top: number; // where the floating card is drawn
    over: string; index: number; settling: boolean; parked: boolean;
  };
  let drag = $state<Drag | null>(null);
  let press: { shelf: Shelf; book: ShelfBook; x: number; y: number; el: HTMLElement } | null = null;

  type Slot = { book?: ShelfBook; ph?: boolean };
  /** The books of a shelf as they are drawn now: the dragged book is lifted out, and a placeholder shows where it would land. */
  function slots(s: Shelf): Slot[] {
    if (!drag) return s.books.map((book) => ({ book }));
    const list: Slot[] = s.books.filter((b) => b.id !== drag!.book.id || s.id !== drag!.from).map((book) => ({ book }));
    if (s.id === drag.over) list.splice(Math.min(drag.index, list.length), 0, { ph: true });
    return list;
  }

  function down(e: PointerEvent, shelf: Shelf, book: ShelfBook) {
    if (e.button !== 0 || busy || drag) return;
    if ((e.target as HTMLElement).closest('.refresh')) return;
    press = { shelf, book, x: e.clientX, y: e.clientY, el: e.currentTarget as HTMLElement };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
    window.addEventListener('pointercancel', cancelPress);
  }

  function endPress() {
    press = null;
    window.removeEventListener('pointermove', move);
    window.removeEventListener('pointerup', up);
    window.removeEventListener('pointercancel', cancelPress);
  }
  function cancelPress() { endPress(); if (drag && !drag.parked) giveBack(); }

  function move(e: PointerEvent) {
    if (!drag) {
      if (!press || !dragStarted(e.clientX - press.x, e.clientY - press.y)) return;
      const r = press.el.getBoundingClientRect();
      const { shelf, book } = press;
      const origIndex = shelf.books.findIndex((b) => b.id === book.id);
      drag = {
        book, from: shelf.id, origIndex, src: coverOf(shelf, book), w: r.width, h: r.height,
        offX: press.x - r.left, offY: press.y - r.top, x: e.clientX, y: e.clientY, left: r.left, top: r.top,
        over: shelf.id, index: origIndex, settling: false, parked: false,
      };
      scrolling();
    }
    if (!drag || drag.parked || drag.settling) return;
    drag.x = e.clientX; drag.y = e.clientY;
    drag.left = e.clientX - drag.offX; drag.top = e.clientY - drag.offY;
    retarget();
  }

  /** Which shelf is the pointer over, and between which books. */
  function retarget() {
    if (!drag || !scroller) return;
    const els = [...scroller.querySelectorAll<HTMLElement>('.shelf[data-id]')];
    const at = shelfAt(els.map((el) => { const r = el.getBoundingClientRect(); return { x: r.left, y: r.top, w: r.width, h: r.height }; }), drag.x, drag.y);
    if (at < 0) return;
    const row = els[at].querySelector<HTMLElement>('.row')!;
    const rr = row.getBoundingClientRect();
    // the places are measured without the animation that is running, so a book that glides aside does not confuse the choice
    const boxes = [...row.querySelectorAll<HTMLElement>('.slot[data-book]')].map((s) => ({ x: rr.left + s.offsetLeft, y: rr.top + s.offsetTop, w: s.offsetWidth, h: s.offsetHeight }));
    drag.over = els[at].dataset.id!;
    drag.index = dropIndex(boxes, drag.x, drag.y);
  }

  // scroll when the book is held near the top or the bottom edge
  function scrolling() {
    const step = () => {
      if (!drag || drag.parked || drag.settling) return;
      const dy = edgeScroll(drag.y, 0, window.innerHeight);
      if (dy && scroller) { scroller.scrollTop += dy; retarget(); }
      requestAnimationFrame(step);
    };
    requestAnimationFrame(step);
  }

  /** Let the floating book glide to the placeholder, then call done. */
  async function flyToPlaceholder(done: () => void) {
    await tick();
    const slot = scroller?.querySelector<HTMLElement>('.ph')?.parentElement;
    const row = slot?.parentElement;
    if (!drag || !slot || !row) { done(); return; }
    // the place is measured without the gliding animation that is running, so the book flies to where the placeholder ends up
    const rr = row.getBoundingClientRect();
    const ms = params().dur * 1.4;
    drag.settling = true; drag.left = rr.left + slot.offsetLeft; drag.top = rr.top + slot.offsetTop;
    setTimeout(done, ms + 30);
  }

  /** The book goes back to the place where it was picked up. */
  function giveBack() {
    if (!drag) return;
    drag.over = drag.from; drag.index = drag.origIndex; drag.parked = false;
    flyToPlaceholder(() => (drag = null));
  }

  let ask = $state<{ book: ShelfBook; from: Shelf; to: Shelf; index: number } | null>(null);

  function up() {
    endPress();
    if (!drag) return;
    const d = drag;
    if (d.settling) return;
    const to = shelves.find((s) => s.id === d.over);
    const from = shelves.find((s) => s.id === d.from);
    if (!to || !from) { giveBack(); return; }
    flyToPlaceholder(() => {
      if (to.id === from.id) { drag = null; reorder(from, d.book, d.index); }
      else { if (drag) drag.parked = true; ask = { book: d.book, from, to, index: d.index }; }
    });
  }

  async function reorder(s: Shelf, book: ShelfBook, index: number) {
    const ids = insertAt(s.books.map((b) => b.id), book.id, index);
    if (ids.every((id, i) => id === s.books[i].id)) return;
    s.books = ids.map((id) => s.books.find((b) => b.id === id)!);
    await mutate([s.id], async () => { await Store.SetBookOrder(s.id, ids); });
  }

  async function confirmMove() {
    const a = ask;
    if (!a) return;
    ask = null;
    await mutate([a.from.id, a.to.id], async () => {
      const id = await Store.MoveBook(a.from.id, a.book.id, a.to.id);
      const ids = insertAt([...a.to.books.map((b) => b.id), id], id, a.index);
      await Store.SetBookOrder(a.to.id, ids);
      // show the book on its new shelf at once, then read the truth from the data
      a.from.books = a.from.books.filter((b) => b.id !== a.book.id);
      a.to.books = ids.map((i) => (i === id ? { ...a.book, id, hasCover: a.book.hasCover } : a.to.books.find((b) => b.id === i)!));
      forgetCovers();
      saved = {};
    });
    drag = null;
    await load();
  }

  function cancelMove() {
    ask = null;
    giveBack();
  }

  function keys(e: KeyboardEvent) {
    if (e.key !== 'Escape') return;
    if ((e.target as HTMLElement | null)?.closest?.('input, textarea')) return; // Escape in a text box only closes that box
    if (ask) { cancelMove(); return; }
    if (drag) { if (!drag.parked && !drag.settling) { endPress(); giveBack(); } return; }
    // a window on top of the shelf closes first (each one listens for Esc itself)
    if (adding || newShelf != null || renaming || nav.projects || nav.projectSettings || nav.searchOpen || nav.dataFolder || nav.calendar || nav.names) return;
    closeShelf();
  }

  /** The tool tip of the progress bar: what it counts. */
  function goalTip(b: ShelfBook): string {
    const what = b.goalKind === 'book' ? 'Book goal' : b.goalKind === 'chapter' ? 'Chapter goals together' : 'Scene goals together';
    return `${what}: ${b.goalWords.toLocaleString()} / ${b.goal.toLocaleString()} words (${Math.round((b.goalWords / b.goal) * 100)}%)`;
  }

  const ease = $derived(bezier(params().ease)); // kept so the screen follows a change of the motion level
  const flipOpts = () => ({ duration: params().dur * 1.4, easing: ease });
  const current = $derived(shelves.find((s) => s.current));
  const total = $derived(shelves.reduce((n, s) => n + s.books.length, 0));
</script>

<svelte:window onkeydown={keys} />

<div class="screen" class:dragging={!!drag} bind:this={scroller} in:arrive|global out:veil|global
  onscroll={() => { if (drag && !drag.parked && !drag.settling) retarget(); }}>
  <div class="page">
    <header>
      <div>
        <h1>Bookshelf</h1>
        <p class="muted">{total} {total === 1 ? 'book' : 'books'} on {shelves.length} {shelves.length === 1 ? 'shelf' : 'shelves'}. Each shelf is a project. Click a book to write. Drag a book to another shelf to move it.</p>
      </div>
      <div class="head-actions">
        <button title="Rename, remove and set up the projects" onclick={() => (nav.projects = true)}>Manage shelves…</button>
        <button class="primary" title="Go to the app without choosing a book" onclick={() => closeShelf()}>Open {current?.name ?? 'the app'} →</button>
      </div>
    </header>

    {#if error}<p class="err" role="alert">{error}</p>{/if}
    {#if !loaded && !error}<p class="muted">Loading…</p>{/if}

    {#each shelves as s (s.id)}
      <section class="shelf" class:over={drag && drag.over === s.id && drag.from !== s.id} data-id={s.id}>
        <div class="shelf-head">
          {#if renaming?.id === s.id}
            <!-- svelte-ignore a11y_autofocus -->
            <input class="name-edit" autofocus bind:value={renaming.name} onkeydown={(e) => { if (e.key === 'Enter') renameShelf(); else if (e.key === 'Escape') { e.stopPropagation(); renaming = null; } }} onblur={renameShelf} />
          {:else}
            <button class="name" title="Rename this shelf" onclick={() => (renaming = { id: s.id, name: s.name })}>{s.name}</button>
          {/if}
          <span class="muted count">{s.books.length} {s.books.length === 1 ? 'book' : 'books'}{s.current ? ' · open now' : ''}</span>
        </div>
        {#if s.problem}<p class="err">This project could not be read: {s.problem}</p>{/if}

        <div class="row" class:intro>
          {#each slots(s) as slot, i (slot.ph ? 'ph' : slot.book!.id)}
            <div class="slot" data-book={slot.book?.id} style:--i={i} animate:flip={flipOpts()}>
              {#if slot.ph}
                <div class="ph"><span>Drop here</span></div>
              {:else}
                {@const b = slot.book!}
                <div class="book" role="button" tabindex="0" aria-label="{b.title || 'Untitled'}. Open to write."
                  onpointerdown={(e) => down(e, s, b)} onkeydown={(e) => { if (e.target === e.currentTarget && (e.key === 'Enter' || e.key === ' ')) { e.preventDefault(); open(s, b); } }}
                  onclick={() => open(s, b)}>
                  <div class="cover">
                    {#key b.seed + '/' + (b.hasCover ? 'own' : 'made')}
                      {#if coverOf(s, b)}<img src={coverOf(s, b)} alt="" draggable="false" in:fade={{ duration: params().dur * 2 }} />{/if}
                    {/key}
                    {#if !b.hasCover}
                      <button class="refresh" title="Make another cover" aria-label="Make another cover for {b.title || 'this book'}" onclick={(e) => { e.stopPropagation(); refresh(s, b); }}>↻</button>
                    {/if}
                  </div>
                  <div class="meta"><b>{b.title || 'Untitled'}</b><small>{bookLine(b.chapters, b.words)}</small>
                    {#if b.goal > 0}<div class="goal" title={goalTip(b)}><ProgressBar value={b.goalWords} goal={b.goal} thin /></div>{/if}</div>
                </div>
              {/if}
            </div>
          {/each}

          <div class="slot add-slot">
            {#if adding?.project === s.id}
              <div class="add editing">
                <!-- svelte-ignore a11y_autofocus -->
                <input autofocus placeholder="Title" bind:value={adding.title} onkeydown={(e) => { if (e.key === 'Enter') addBook(); else if (e.key === 'Escape') { e.stopPropagation(); adding = null; } }} onblur={addBook} />
                <small class="muted">Press Enter</small>
              </div>
            {:else}
              <button class="add" title="Add a book to {s.name}" onclick={() => (adding = { project: s.id, title: '' })}>
                <svg viewBox="0 0 24 24" width="30" height="30" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>
                <span>Add</span>
              </button>
            {/if}
          </div>
        </div>
        <div class="board" aria-hidden="true"></div>
      </section>
    {/each}

    {#if loaded}
      {#if newShelf == null}
        <button class="new-shelf" onclick={() => (newShelf = '')}>+ New shelf <span class="muted">(a new project)</span></button>
      {:else}
        <div class="new-shelf editing">
          <!-- svelte-ignore a11y_autofocus -->
          <input autofocus placeholder="Name of the shelf, for example “Book two”" bind:value={newShelf} onkeydown={(e) => { if (e.key === 'Enter') addShelf(); else if (e.key === 'Escape') { e.stopPropagation(); newShelf = null; } }} onblur={addShelf} />
        </div>
      {/if}
    {/if}
  </div>
</div>

{#if drag}
  <div class="float" class:settling={drag.settling} style:left="{drag.left}px" style:top="{drag.top}px" style:width="{drag.w}px" style:height="{drag.h}px" aria-hidden="true">
    <div class="cover">{#if drag.src}<img src={drag.src} alt="" draggable="false" />{/if}</div>
    <div class="meta"><b>{drag.book.title || 'Untitled'}</b><small>{bookLine(drag.book.chapters, drag.book.words)}</small>
      {#if drag.book.goal > 0}<div class="goal"><ProgressBar value={drag.book.goalWords} goal={drag.book.goal} thin /></div>{/if}</div>
  </div>
{/if}

{#if ask}
  <div class="overlay" transition:veil|global role="presentation">
    <div class="palette ask" transition:dialog|global role="dialog" aria-label="Move book">
      <h2>Move “{ask.book.title || 'Untitled'}” to “{ask.to.name}”?</h2>
      <p>The chapters, text, comments, snapshots and cover move with the book.</p>
      <p class="muted">Characters, places, sources and arcs stay on “{ask.from.name}”. In the text, @ links to them lose their link (the name stays), and citations become plain text marked “needs a source”.</p>
      <div class="row-btns">
        <button onclick={cancelMove}>Cancel</button>
        <button class="primary" onclick={confirmMove}>Move the book</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .screen {
    --bw: 150px; /* the width of a book */
    --cover-h: calc(var(--bw) * 1.5);
    --meta-h: 4.5rem;
    --shelf-shade: rgba(0, 0, 0, .55); /* the shadow inside a shelf */
    position: fixed; inset: 0; z-index: 40; overflow-y: auto;
    background: radial-gradient(1200px 500px at 50% -10%, color-mix(in srgb, var(--accent) 9%, transparent), transparent), var(--stage);
  }
  :global(:root[data-theme='light']) .screen { --shelf-shade: rgba(40, 40, 90, .12); }
  .screen.dragging { user-select: none; cursor: grabbing; }
  .page { max-width: 1100px; margin: 0 auto; padding: 2.2rem 1.6rem 4rem; }
  header { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; margin-bottom: 1.4rem; }
  h1 { font-size: 1.7rem; margin: 0 0 .25rem; }
  header p { margin: 0; max-width: 46rem; }
  .head-actions { display: flex; gap: .5rem; }
  .err { color: var(--danger); }

  .shelf {
    position: relative; margin-bottom: 1.6rem; border-radius: 14px; border: 1px solid var(--line);
    background: linear-gradient(180deg, color-mix(in srgb, var(--panel) 70%, transparent), var(--panel));
    box-shadow: inset 0 14px 30px -18px var(--shelf-shade);
    padding: .9rem 1.2rem 0;
    transition: border-color var(--m-dur) var(--m-ease), box-shadow var(--m-dur) var(--m-ease);
  }
  .shelf.over { border-color: var(--accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 25%, transparent), inset 0 14px 30px -18px var(--shelf-shade); }
  .shelf-head { display: flex; align-items: baseline; gap: .8rem; margin-bottom: .9rem; }
  .name { background: none; border: 1px solid transparent; font-size: 1.05rem; font-weight: 600; padding: .1rem .4rem; margin-left: -.4rem; }
  .name:hover { border-color: var(--line); background: var(--panel-2); }
  .name-edit { font-size: 1.05rem; font-weight: 600; width: 18rem; max-width: 100%; }
  .count { font-size: .8rem; }

  .row { position: relative; display: flex; flex-wrap: wrap; gap: 1.2rem 1.3rem; align-items: flex-start; padding-bottom: 1.1rem; min-height: calc(var(--cover-h) + 1.1rem); }
  .slot { width: var(--bw); flex: none; }

  .book {
    display: block; cursor: grab; touch-action: none; border-radius: 6px; outline: none;
    translate: 0 0; transition: translate var(--m-dur) var(--m-ease), filter var(--m-dur) var(--m-ease);
  }
  .book .cover {
    position: relative; width: var(--bw); height: var(--cover-h); border-radius: 3px 7px 7px 3px; overflow: hidden;
    background: var(--panel-2); border: 1px solid var(--line);
    box-shadow: 0 3px 8px rgba(0, 0, 0, .35), inset 3px 0 5px -2px rgba(0, 0, 0, .35);
    transition: box-shadow var(--m-dur) var(--m-ease);
  }
  .book .cover::after { content: ''; position: absolute; inset: 0; pointer-events: none; background: linear-gradient(90deg, rgba(255, 255, 255, .16), transparent 7%, transparent 93%, rgba(0, 0, 0, .12)); }
  .book .cover img { display: block; width: 100%; height: 100%; object-fit: cover; }
  .screen:not(.dragging) .book:hover, .book:focus-visible { translate: 0 -7px; }
  .screen:not(.dragging) .book:hover .cover, .book:focus-visible .cover { box-shadow: 0 14px 22px -6px rgba(0, 0, 0, .5), inset 3px 0 5px -2px rgba(0, 0, 0, .35); }
  .book:focus-visible .cover { outline: 2px solid var(--accent); outline-offset: 2px; }
  .meta { height: var(--meta-h); padding: .5rem .1rem 0; display: flex; flex-direction: column; gap: .15rem; }
  .meta b { font-size: .88rem; line-height: 1.2; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  .meta small { color: var(--muted); font-size: .72rem; }
  .goal { margin-top: .3rem; }

  .refresh {
    position: absolute; right: 6px; top: 6px; width: 28px; height: 28px; padding: 0; border-radius: 50%; line-height: 1;
    background: rgba(15, 16, 20, .72); color: #fff; border: 1px solid rgba(255, 255, 255, .25); font-size: 1rem;
    opacity: 0; transform: scale(.85); transition: opacity var(--m-dur) var(--m-ease), transform var(--m-dur) var(--m-ease), background-color var(--m-dur) var(--m-ease);
  }
  .book:hover .refresh, .book:focus-within .refresh { opacity: 1; transform: scale(1); }
  .refresh:hover { background: var(--accent); border-color: var(--accent); transform: rotate(180deg) scale(1.05) !important; }

  /* the empty place where a dragged book will land */
  .ph {
    width: var(--bw); height: calc(var(--cover-h) + var(--meta-h)); border-radius: 4px 9px 9px 4px; box-sizing: border-box;
    border: 2px dashed var(--accent); display: grid; place-items: center; color: var(--accent); font-size: .8rem; font-weight: 600;
    background: color-mix(in srgb, var(--accent) 9%, transparent);
    animation: breathe 1.4s ease-in-out infinite;
  }
  @keyframes breathe { 50% { background: color-mix(in srgb, var(--accent) 18%, transparent); } }

  /* the place to add a book */
  .add {
    width: var(--bw); height: var(--cover-h); box-sizing: border-box; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: .35rem;
    background: transparent; border: 2px dashed var(--line); border-radius: 4px 9px 9px 4px; color: var(--muted); font-size: .9rem;
    transition: border-color var(--m-dur) var(--m-ease), color var(--m-dur) var(--m-ease), background-color var(--m-dur) var(--m-ease), translate var(--m-dur) var(--m-ease);
  }
  .add:hover { border-color: var(--accent); color: var(--accent); background: color-mix(in srgb, var(--accent) 7%, transparent); translate: 0 -4px; }
  .add:active { transform: none; }
  .add.editing { padding: .6rem; gap: .5rem; border-style: solid; border-color: var(--accent); }
  .add.editing input { width: 100%; box-sizing: border-box; }

  .board {
    height: 14px; margin: 0 -1.2rem; border-radius: 0 0 13px 13px;
    background: linear-gradient(180deg, color-mix(in srgb, var(--text) 14%, var(--panel-2)), var(--panel-2) 40%, var(--line));
    box-shadow: 0 6px 12px -6px var(--shelf-shade), inset 0 1px 0 color-mix(in srgb, var(--text) 18%, transparent);
  }

  .new-shelf {
    display: block; width: 100%; box-sizing: border-box; padding: .9rem; border: 2px dashed var(--line); border-radius: 14px; background: transparent; color: var(--muted); font-size: .95rem;
  }
  .new-shelf:hover { border-color: var(--accent); color: var(--accent); }
  .new-shelf.editing { border-style: solid; border-color: var(--accent); padding: .6rem; }
  .new-shelf.editing input { width: 100%; box-sizing: border-box; }

  /* the book that follows the pointer */
  .float {
    position: fixed; z-index: 45; pointer-events: none; transform: rotate(3deg) scale(1.06); transform-origin: 50% 30%;
    filter: drop-shadow(0 22px 22px rgba(0, 0, 0, .45));
  }
  .float.settling { transform: none; filter: drop-shadow(0 4px 6px rgba(0, 0, 0, .3)); transition: left calc(var(--m-dur) * 1.4) var(--m-ease), top calc(var(--m-dur) * 1.4) var(--m-ease), transform calc(var(--m-dur) * 1.4) var(--m-ease), filter calc(var(--m-dur) * 1.4) var(--m-ease); }
  .float { --meta-h: 4.5rem; }
  .float .cover { position: relative; width: 100%; height: calc(100% - var(--meta-h)); border-radius: 3px 7px 7px 3px; overflow: hidden; border: 1px solid var(--line); background: var(--panel-2); }
  .float .cover img { display: block; width: 100%; height: 100%; object-fit: cover; }
  .float .meta { height: var(--meta-h); padding: .5rem .1rem 0; }
  .float .meta b { font-size: .88rem; display: block; }
  .float .meta small { color: var(--muted); font-size: .72rem; }

  /* the books arrive one after the other when the screen opens */
  .row.intro .slot[data-book] { animation: arrive calc(var(--m-dur) * 2.2) var(--m-ease) backwards; animation-delay: calc(var(--i) * var(--m-stagger)); }
  @keyframes arrive { from { opacity: 0; translate: 0 calc(var(--m-travel) * 2); } }

  .ask { padding: 1.1rem 1.2rem; }
  .ask h2 { font-size: 1.1rem; }
  .row-btns { display: flex; justify-content: flex-end; gap: .5rem; margin-top: 1rem; }
</style>
