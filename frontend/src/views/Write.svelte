<script lang="ts">
  import Select from '../lib/Select.svelte';
  import { shortAge, impossible } from '../lib/ages';
  import FinishLine from '../lib/FinishLine.svelte';
  import { BEATS, arcsSorted, linkFor, tagScene, untagScene, setBeat, type Beat } from '../lib/arcs';
  import { onDestroy } from 'svelte';
  import { flip } from 'svelte/animate';
  import { Window } from '@wailsio/runtime';
  import { db, toHtml, wordCount, type Item } from '../lib/db.svelte';
  import { nav, openItem } from '../lib/nav.svelte';
  import { STATUSES, scenesOf, sceneWords, chapterWords, chapterCast, addScene, sceneLabel } from '../lib/scenes';
  import { maybeAutoVersion, addVersion } from '../lib/versions';
  import Snapshots from '../lib/Snapshots.svelte';
  import DateField from '../lib/DateField.svelte';
  import { addWords, dailyGoal, wordsToday } from '../lib/progress';
  import { missingCast } from '../lib/mentions';
  import Comments from '../lib/Comments.svelte';
  import RichEditor from '../lib/RichEditor.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import { splitScene, mergeWithNext, sceneAfter } from '../lib/scenesplit';
  import ProgressBar from '../lib/ProgressBar.svelte';
  import FindReplace from '../lib/FindReplace.svelte';
  import History from '../lib/History.svelte';
  import SaveIndicator from '../lib/SaveIndicator.svelte';
  import LookButton from '../lib/LookButton.svelte';
  import SceneTags from '../lib/SceneTags.svelte';
  import MoodBoard from '../lib/MoodBoard.svelte';
  import ReferencePane from '../lib/ReferencePane.svelte';
  import { cleanPick, refKindOf, type RefPick } from '../lib/reference-core';
  import RevisionPasses from '../lib/RevisionPasses.svelte';
  import { passesOf, tick } from '../lib/passes';
  import { isDone, summary as passSummary } from '../lib/passes-core';
  import { miniText } from '../lib/pomodoro.svelte';
  import { tagCounts, hasTag } from '../lib/tags-core';

  // Reopen what you were writing last, also after the app was closed.
  const LAST = 'typewriter.write';
  function lastOpen(): { book: number | null; chapter: number | null; scene: number | null } {
    const none = { book: null, chapter: null, scene: null };
    try {
      const saved = JSON.parse(localStorage.getItem(LAST) ?? 'null');
      if (!saved) return none;
      // the item may have been deleted since: fall back from scene to chapter to book
      const s = db.get('scene', saved.scene);
      const c = db.get('chapter', s?.chapter ?? saved.chapter);
      if (s && c && db.get('book', c.book)) return { book: c.book, chapter: c.id, scene: s.id };
      if (c && db.get('book', c.book)) return { book: c.book, chapter: c.id, scene: scenesOf(c.id)[0]?.id ?? null };
      const b = db.get('book', saved.book);
      if (b) return { ...none, book: b.id };
    } catch { /* no saved place, or storage is blocked */ }
    return none;
  }
  const start = lastOpen();

  let bookId = $state<number | null>(start.book);
  let chapId = $state<number | null>(start.chapter);
  let sceneId = $state<number | null>(start.scene);

  $effect(() => {
    const here = JSON.stringify({ book: bookId, chapter: chapId, scene: sceneId });
    try { localStorage.setItem(LAST, here); } catch { /* fine without it */ }
  });

  // The reference pane: something pinned next to the text. Its choice is kept on this computer.
  const REF_KEY = 'typewriter-reference';
  let ref = $state<RefPick>(cleanPick((() => { try { return JSON.parse(localStorage.getItem(REF_KEY) ?? 'null'); } catch { return null; } })()));
  $effect(() => { try { localStorage.setItem(REF_KEY, JSON.stringify(ref)); } catch { /* fine without it */ } });
  /** Show an item beside the text (Alt+click on an @ link does this). */
  function showBeside(kind: string, id: number) {
    const k = refKindOf(kind);
    if (k) ref = { kind: k, id, open: true };
  }

  let showFind = $state(false);
  let showHistory = $state(false);
  let showSnaps = $state(false);
  let rich = $state<any>(); // the open editor, to point at a comment
  let activeComment = $state<number | null>(null);
  let rev = $state(0); // bump to reload the editor after the text was changed from outside it

  /** Focus mode hides everything but the text, and fills the screen. */
  async function setFocus(on: boolean) {
    nav.focus = on;
    try { await (on ? Window.Fullscreen() : Window.UnFullscreen()); } catch { /* still fine without full screen */ }
  }
  onDestroy(() => { if (nav.focus) setFocus(false); });

  function onKey(e: KeyboardEvent) {
    const ctrl = e.ctrlKey || e.metaKey;
    if (ctrl && e.key.toLowerCase() === 'h') { e.preventDefault(); showFind = true; }
    else if (e.key === 'F11' || (ctrl && e.shiftKey && e.key.toLowerCase() === 'f')) { e.preventDefault(); setFocus(!nav.focus); }
    // the find and history windows are still open when this runs, so Esc closes them first
    else if (e.key === 'Escape' && nav.focus && !showFind && !showHistory && !showSnaps && !nav.searchOpen) setFocus(false);
  }

  async function newComment(quote: string): Promise<number | null> {
    if (!scene) return null;
    const c = await db.save('comment', { scene: scene.id, quote: quote.slice(0, 200), text: '', resolved: false, at: new Date().toISOString() });
    activeComment = c.id;
    return c.id;
  }
  async function removeComment(id: number) {
    rich?.removeComment(id);
    await db.remove('comment', id, false);
  }
  // comments that are resolved lose their highlight in the text
  const resolvedCss = $derived.by(() => {
    const sel = db.lists.comment.filter((c) => c.resolved && c.scene === sceneId).map((c) => `.ProseMirror span[data-comment="${c.id}"]`);
    return sel.length ? sel.join(',') + '{background:none !important;border-bottom-color:transparent !important}' : '';
  });
  const openComments = (id: number) => db.lists.comment.filter((c) => c.scene === id && !c.resolved).length;
  function addToCast(id: number) { if (scene) { scene.characters = [...(scene.characters ?? []), id]; saveScene(); } }
  function ignoreMention(id: number) { if (scene) { scene.ignoreMentions = [...(scene.ignoreMentions ?? []), id]; saveScene(); } }

  // another tab asked for a scene (a backlink was clicked)
  $effect(() => {
    if (nav.scene == null || !db.ready) return;
    const id = nav.scene;
    nav.scene = null;
    jumpToScene(id);
  });

  function jumpToScene(id: number) {
    const s = db.get('scene', id);
    const c = db.get('chapter', s?.chapter);
    if (!s || !c) return;
    bookId = c.book; chapId = c.id; sceneId = s.id;
  }

  const book = $derived(db.get('book', bookId));
  const chapters = $derived(db.lists.chapter.filter((c) => c.book === bookId).sort((a, b) => a.order - b.order));
  const chap = $derived(db.get('chapter', chapId));
  const scenes = $derived(scenesOf(chapId));
  const bookPasses = $derived(passesOf(bookId));
  // Scene tags: the filter above the chapter list shows only the scenes with one tag.
  let tagFilter = $state('');
  const bookTags = $derived(tagCounts(db.lists.scene.filter((s) => chapters.some((c) => c.id === s.chapter))));
  const activeTag = $derived(bookTags.some(([t]) => t === tagFilter) ? tagFilter : '');
  const shown = $derived(scenes.filter((s) => hasTag(s, activeTag)));
  const tagged = (cid: number) => scenesOf(cid).filter((s) => hasTag(s, activeTag)).length;
  const scene = $derived(db.get('scene', sceneId));
  // Who is named in the text but not in the cast. It reads the text and every character, so it waits 0.3 seconds while you type.
  let miss = $state<Item[]>([]);
  let missFor = -1;
  $effect(() => {
    const s = scene;
    void [s?.body, s?.pov, JSON.stringify(s?.characters), JSON.stringify(s?.ignoreMentions), db.lists.character.map((c) => c.name + '|' + c.aliases).join('~')];
    const run = () => { miss = s ? missingCast(s) : []; };
    if ((s?.id ?? -1) !== missFor) { missFor = s?.id ?? -1; run(); return; }
    const t = setTimeout(run, 300);
    return () => clearTimeout(t);
  });
  const goalOf = (c: any) => Number(c.goal) || Number(book?.chapterGoal) || 0;
  const pctText = (n: number, goal: number) => (goal > 0 ? ` (${Math.round((n / goal) * 100)}%)` : '');
  const total = $derived(chapters.reduce((n, c) => n + chapterWords(c), 0));
  const chapWords = $derived(chap ? chapterWords(chap) : 0);

  function pickChapter(id: number | null) {
    chapId = id;
    sceneId = scenesOf(id)[0]?.id ?? null;
  }

  async function addBook() {
    const b = await db.save('book', { title: 'New book', synopsis: '', order: db.lists.book.length + 1 });
    bookId = b.id; pickChapter(null);
  }
  async function addChapter() {
    if (!bookId) return;
    const c = await db.save('chapter', { book: bookId, order: chapters.length + 1, title: 'New chapter', summary: '' });
    const s = await addScene(c);
    chapId = c.id; sceneId = s.id;
  }
  async function newScene() {
    if (!chap) return;
    sceneId = (await addScene(chap)).id;
  }
  async function swapOrder(kind: 'chapter' | 'scene', list: any[], item: any, dir: -1 | 1) {
    const other = list[list.indexOf(item) + dir];
    if (!other) return;
    const a = item.order; item.order = other.order; other.order = a;
    await db.save(kind, item); await db.save(kind, other);
  }
  /** Move the open scene to the end of another chapter. */
  async function moveScene(targetId: number) {
    if (!scene || !targetId || targetId === scene.chapter) return;
    const target = db.get('chapter', targetId);
    if (!target) return;
    const from = scenesOf(scene.chapter).filter((s) => s.id !== scene.id);
    scene.chapter = targetId;
    scene.order = scenesOf(targetId).length + 1;
    await db.save('scene', scene);
    for (const [i, s] of from.entries()) if (s.order !== i + 1) { s.order = i + 1; await db.save('scene', s); }
    chapId = targetId;
  }

  // ----- drag to reorder -----
  type DragItem = { type: 'scene' | 'chapter'; id: number; label: string };
  type Drop = { type: 'scene' | 'chapter'; id: number; after: boolean; into?: boolean };
  let drag = $state<DragItem | null>(null);
  let drop = $state<Drop | null>(null);
  let ghost = $state({ x: 0, y: 0, tilt: 0 });
  let landed = $state(''); // the row that was just dropped, so it can flash
  let landedTimer: ReturnType<typeof setTimeout>;
  let tiltTimer: ReturnType<typeof setTimeout>;
  let lastX = 0;
  const flash = (key: string) => { landed = key; clearTimeout(landedTimer); landedTimer = setTimeout(() => (landed = ''), 900); };
  let suppressClick = false; // a drag must not also count as a click

  function startDrag(e: PointerEvent, item: DragItem) {
    if (e.button !== 0) return;
    const sx = e.clientX, sy = e.clientY;
    let active = false;
    const move = (ev: PointerEvent) => {
      if (!active) {
        if (Math.hypot(ev.clientX - sx, ev.clientY - sy) < 6) return;
        active = true; drag = item; lastX = ev.clientX;
      }
      // lean a little in the direction you drag, then settle
      const tilt = Math.max(-7, Math.min(7, (ev.clientX - lastX) * 0.8));
      lastX = ev.clientX;
      ghost = { x: ev.clientX, y: ev.clientY, tilt };
      clearTimeout(tiltTimer);
      tiltTimer = setTimeout(() => (ghost = { ...ghost, tilt: 0 }), 90);
      drop = findDrop(ev.clientX, ev.clientY, item);
    };
    const up = async () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
      window.removeEventListener('pointercancel', up);
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

  function findDrop(x: number, y: number, item: DragItem): Drop | null {
    const el = document.elementFromPoint(x, y)?.closest('[data-scene],[data-chapter]') as HTMLElement | null;
    if (!el) return null;
    const r = el.getBoundingClientRect();
    const after = y > r.top + r.height / 2;
    if (el.dataset.scene) {
      const id = +el.dataset.scene;
      return item.type === 'scene' && id !== item.id ? { type: 'scene', id, after } : null;
    }
    const id = +el.dataset.chapter!;
    if (item.type === 'chapter') return id !== item.id ? { type: 'chapter', id, after } : null;
    return { type: 'chapter', id, after: false, into: true }; // drop a scene onto a chapter: it goes to the end
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
    chapId = targetChapter;
    sceneId = moving.id;
    flash('scene' + moving.id);
  }

  const dropClass = (type: 'scene' | 'chapter', id: number) =>
    drop && drop.type === type && drop.id === id ? (drop.into ? 'drop-into' : drop.after ? 'drop-after' : 'drop-before') : '';

  function saveBook() { if (book) db.queue('book', book); }
  function saveChap() { if (chap) db.queue('chapter', chap); }
  function saveScene() { if (scene) db.queue('scene', scene); }
  function toggleChar(id: number) {
    if (!scene) return;
    const set = new Set<number>(scene.characters ?? []);
    set.has(id) ? set.delete(id) : set.add(id);
    scene.characters = [...set]; saveScene();
  }
  async function delBook() { if (bookId) { await db.remove('book', bookId); bookId = null; pickChapter(null); } }
  async function delChap() { if (chapId) { await db.remove('chapter', chapId); pickChapter(null); } }
  // Split the open scene at the cursor, or join it to the next scene.
  let sceneMsg = $state('');
  let msgTimer: ReturnType<typeof setTimeout> | undefined;
  function say(text: string) { sceneMsg = text; clearTimeout(msgTimer); msgTimer = setTimeout(() => (sceneMsg = ''), 6000); }
  async function splitHere() {
    if (!scene) return;
    const parts = rich?.splitAtCursor();
    if (!parts) return;
    db.flush();
    const made = await splitScene(scene, parts.before, parts.after);
    if (typeof made === 'string') { say(made); return; }
    say('Split. The text after the cursor is now the next scene. The old text is saved in History.');
    sceneId = made.id; // the editor opens the new scene, with the cursor at its start
  }
  async function mergeNext() {
    if (!scene) return;
    db.flush();
    const err = await mergeWithNext(scene);
    if (err) { say(err); return; }
    rev++; // show the joined text
    say('Merged. The text of the next scene now follows this text. The old text is saved in History.');
  }
  async function delScene() {
    if (!sceneId) return;
    const left = scenes.filter((s) => s.id !== sceneId);
    await db.remove('scene', sceneId);
    sceneId = left[0]?.id ?? null;
  }
</script>

<svelte:window onkeydown={onKey} />
<svelte:head>{@html `<style>${resolvedCss}</style>`}</svelte:head>

<div class="split wide" class:focus={nav.focus} class:withref={ref.open && !nav.focus}>
  <aside class="side">
    <div class="row"><Select bind:value={bookId} onchange={() => pickChapter(null)}>
      <option value={null}>— choose a book —</option>
      {#each [...db.lists.book].sort((a, b) => a.order - b.order) as b}<option value={b.id}>{b.order}. {b.title}</option>{/each}
    </Select><button class="primary" onclick={addBook}>+ Book</button></div>

    {#if book}
      <label>Book title</label><input bind:value={book.title} oninput={saveBook} />
      <label>Order in series</label><input type="number" bind:value={book.order} oninput={saveBook} />
      <label>Synopsis</label><textarea bind:value={book.synopsis} oninput={saveBook}></textarea>
      <div class="row">
        <div><label>Book goal (words)</label><input type="number" min="0" bind:value={book.goal} oninput={saveBook} /></div>
        <div><label>Default per chapter</label><input type="number" min="0" bind:value={book.chapterGoal} oninput={saveBook} /></div>
      </div>
      <label for="finishby">Finish by</label>
      <div class="row">
        <input id="finishby" type="date" bind:value={book.finishBy} oninput={saveBook} onchange={saveBook} />
        {#if book.finishBy}<button class="small" title="Clear the finish date" onclick={() => { book.finishBy = ''; saveBook(); }}>✕</button>{/if}
      </div>
      <div style="margin-top:.4rem"><FinishLine {book} /></div>
      {#if book.goal > 0}
        <div style="margin-top:.6rem"><ProgressBar value={total} goal={book.goal} /></div>
        <div class="muted" style="font-size:.8rem;margin-top:.2rem">{total.toLocaleString()} / {Number(book.goal).toLocaleString()} words{pctText(total, book.goal)}</div>
      {/if}
      <RevisionPasses bookId={book.id} />
      {#if bookTags.length}
        <label for="tagfilter">Show scenes tagged</label>
        <Select id="tagfilter" bind:value={tagFilter}>
          <option value="">All scenes</option>
          {#each bookTags as [t, n]}<option value={t}>{t} ({n})</option>{/each}
        </Select>
      {/if}
      <div class="head" style="margin-top:1rem"><h3 style="margin:0">Chapters · {total.toLocaleString()} words</h3>
        <button class="small primary" onclick={addChapter}>+ Chapter</button></div>
      <div class="list" class:sorting={!!drag}>
        {#each chapters as c (c.id)}
          {@const words = chapterWords(c)}
          <div class="chap" animate:flip={{ duration: 220 }}>
          <button class="item {dropClass('chapter', c.id)}" class:landed={landed === 'chapter' + c.id} class:on={c.id === chapId} class:nomatch={activeTag && !tagged(c.id)} class:dragging={drag?.type === 'chapter' && drag.id === c.id}
            data-chapter={c.id} onpointerdown={(e) => startDrag(e, { type: 'chapter', id: c.id, label: c.title || 'Chapter' })}
            onclick={() => { if (!suppressClick) pickChapter(c.id); }}>
            <span class="ct">{c.order}. {c.title}</span><small class="cw">{words.toLocaleString()}{goalOf(c) ? ' / ' + goalOf(c).toLocaleString() : ''} words{activeTag ? ' · ' + tagged(c.id) + ' tagged' : ''}</small>
            {#if goalOf(c)}<span style="flex-basis:100%"><ProgressBar value={words} goal={goalOf(c)} thin /></span>{/if}
          </button>
          {#if c.id === chapId}
            <div class="scenes">
              {#each shown as s (s.id)}
                <button animate:flip={{ duration: 200 }} class="scene {dropClass('scene', s.id)}" class:landed={landed === 'scene' + s.id} class:on={s.id === sceneId} class:dragging={drag?.type === 'scene' && drag.id === s.id}
                  data-scene={s.id} onpointerdown={(e) => startDrag(e, { type: 'scene', id: s.id, label: s.title || 'Untitled scene' })}
                  onclick={() => { if (!suppressClick) sceneId = s.id; }}>
                  <span class="status {s.status}" title={s.status}></span>
                  <span class="t">{scenes.indexOf(s) + 1}. {s.title || 'Untitled'}</span>
                  {#if missingCast(s).length}<span class="badge" title="Names a character who is not in the cast">⚠</span>{/if}
                  {#if openComments(s.id)}<span class="badge" title="Open comments">💬{openComments(s.id)}</span>{/if}
                  <small>{sceneWords(s)}</small>
                </button>
              {/each}
              <button class="small" onclick={newScene}>+ Scene</button>
            </div>
          {/if}
          </div>
        {:else}<div class="empty">No chapters yet.</div>{/each}
      </div>
      <div style="margin-top:1rem"><ConfirmButton label="Delete book" onconfirm={delBook} /></div>
    {/if}
  </aside>

  <section class="pane">
    {#if !nav.focus}
      {#if sceneMsg}<div class="chip scenemsg" role="status">{sceneMsg}</div>{/if}
      <div class="tools">
        <button class="small" onclick={() => (showFind = true)}>Find &amp; replace <kbd>Ctrl+H</kbd></button>
        <LookButton />
        <button class="small" class:active={ref.open} title="Pin a scene, note, character or place next to your text" onclick={() => (ref = { ...ref, open: !ref.open })}>Reference</button>
        {#if scene}<button class="small" onclick={() => (showHistory = true)}>History</button>{/if}
        {#if scene}<button class="small" title="Split this scene in two at the cursor" onclick={splitHere}>Split here</button>{/if}
        {#if scene && sceneAfter(scene)}<ConfirmButton label="Merge with next" onconfirm={mergeNext} />{/if}
        {#if book}<button class="small" title="Save a copy of the whole book, and see later what changed" onclick={() => (showSnaps = true)}>Snapshots</button>{/if}
        <button class="small" onclick={() => setFocus(true)}>Focus mode <kbd>F11</kbd></button>
      </div>
    {/if}
    {#if chap && scene}
      {#if !nav.focus}
        <input class="editor" style="font-size:1.4rem;min-height:0;margin-bottom:.4rem" bind:value={chap.title} oninput={saveChap} placeholder="Chapter title" />
        <input class="scenetitle" bind:value={scene.title} oninput={saveScene} placeholder="Scene title (only you see this)" />
      {/if}
      {#key scene.id + ':' + rev}
        <RichEditor bind:this={rich} bare={nav.focus} onfind={() => (showFind = true)} onbeside={showBeside} source={[book?.title, sceneLabel(scene)].filter(Boolean).join(', ')} onopenref={(k, id) => { db.flush(); openItem(k, id); }} oncomment={newComment} onfocuscomment={(id) => { if (id) activeComment = id; }} value={toHtml(scene.body ?? '')}
          onchange={(html) => { const before = scene.body ?? ''; scene.body = html; saveScene(); maybeAutoVersion(scene.id, before); addWords(wordCount(html) - wordCount(before)); }} />
      {/key}
      {@const words = sceneWords(scene)}
      {#if nav.focus}
        <div class="focus-hud">{#if miniText()}<span>{miniText()}</span>{/if}{words.toLocaleString()} words · today {wordsToday().toLocaleString()}{dailyGoal() ? ' / ' + dailyGoal().toLocaleString() : ''} <SaveIndicator /> <span>Esc to leave</span></div>
      {:else}
        <div class="statusbar">
        {#if Number(scene.goal) > 0}<div style="margin-top:.6rem"><ProgressBar value={words} goal={Number(scene.goal)} /></div>{/if}
        <p class="muted">
          Scene: {words.toLocaleString()}{Number(scene.goal) ? ' / ' + Number(scene.goal).toLocaleString() : ''} words{pctText(words, Number(scene.goal))}
          · Today: {wordsToday().toLocaleString()}{dailyGoal() ? ' / ' + dailyGoal().toLocaleString() : ''}
          · Chapter: {chapWords.toLocaleString()}{goalOf(chap) ? ' / ' + goalOf(chap).toLocaleString() : ''} words{pctText(chapWords, goalOf(chap))}
          {#if goalOf(chap) && chapWords >= goalOf(chap)} · chapter goal reached{/if}
        </p>
        </div>
      {/if}
    {:else if chap}
      <input class="editor" style="font-size:1.4rem;min-height:0;margin-bottom:.8rem" bind:value={chap.title} oninput={saveChap} />
      <div class="empty">This chapter has no scenes yet. <button class="primary" onclick={newScene}>+ Add the first scene</button></div>
    {:else}
      <div class="empty">{book ? 'Pick a chapter, or add one.' : 'Add a book to begin.'}</div>
    {/if}
  </section>

  {#if ref.open && !nav.focus}
    <ReferencePane bind:pick={ref} hereScene={sceneId} onclose={() => (ref = { ...ref, open: false })} />
  {/if}

  <aside class="side right">
    {#if scene}
      <Comments sceneId={scene.id} body={scene.body ?? ''} active={activeComment} onshow={(id) => { activeComment = id; rich?.showComment(id); }} onremove={removeComment} />
      <div class="head" style="margin-top:1.4rem"><h3 style="margin:0">Scene</h3>
        <span class="row fit" style="gap:.2rem">
          <button class="small" title="Move up" onclick={() => swapOrder('scene', scenes, scene, -1)}>↑</button>
          <button class="small" title="Move down" onclick={() => swapOrder('scene', scenes, scene, 1)}>↓</button>
          <ConfirmButton onconfirm={delScene} />
        </span>
      </div>
      <label>Status</label>
      <Select bind:value={scene.status} onchange={saveScene}>{#each STATUSES as s}<option>{s}</option>{/each}</Select>
      <label>Point of view</label>
      <Select bind:value={scene.pov} onchange={saveScene}>
        <option value={null}>— none —</option>
        {#each db.lists.character as c}<option value={c.id}>{c.name}</option>{/each}
      </Select>
      <label>Place</label>
      <Select bind:value={scene.location} onchange={saveScene}>
        <option value={null}>— none —</option>
        {#each db.lists.location as l}<option value={l.id}>{l.name}</option>{/each}
      </Select>
      <DateField bind:time={scene.time} bind:when={scene.when} onchange={saveScene} />
      <label>Tags</label>
      {#key scene.id}<SceneTags {scene} onchange={saveScene} />{/key}
      {#key scene.id}<MoodBoard owner={scene.id} />{/key}
      {#if bookPasses.length}
        <label>Revision passes <small class="muted">{passSummary(scene, bookPasses).done}/{bookPasses.length} done</small></label>
        {#each bookPasses as p (p.id)}
          <div><label style="display:flex;gap:.4rem;margin:.15rem 0;color:var(--text)">
            <input type="checkbox" style="width:auto" checked={isDone(scene, p.id)} onchange={() => tick(scene!, p.id)} />{p.name}
          </label></div>
        {/each}
      {/if}
      <label>Story arcs</label>
      {#each arcsSorted() as a (a.id)}
        {@const lk = linkFor(a.id, scene.id)}
        <div class="arcrow">
          <label class="arcchk">
            <input type="checkbox" style="width:auto" checked={!!lk} onchange={() => (lk ? untagScene(a.id, scene!.id) : tagScene(a.id, scene!.id))} />
            <span class="dot" style:background={a.color}></span><span class="arcname">{a.name}</span>
          </label>
          {#if lk}
            <Select style="flex:0 0 6rem" aria-label="What this scene does for {a.name}" value={lk.beat ?? ''} onchange={(e) => setBeat(lk, e.currentTarget.value as Beat)}>
              {#each BEATS as [v, l]}<option value={v}>{l}</option>{/each}
            </Select>
          {/if}
        </div>
      {:else}
        <div class="muted" style="font-size:.85rem">No arcs yet. <button class="small" onclick={() => (nav.tab = 'arcs')}>Add one</button></div>
      {/each}
      <label>Who is in this scene</label>
      {#each db.lists.character as c}
        <div><label style="display:flex;gap:.4rem;margin:.15rem 0;color:var(--text)">
          <input type="checkbox" style="width:auto" checked={(scene.characters ?? []).includes(c.id)} onchange={() => toggleChar(c.id)} />
          <span class="dot" style:background={c.color} style="margin-top:.2rem"></span>{c.name}
          {#if shortAge(c, scene.time)}<small class="age" class:bad={impossible(c, scene.time)}>{shortAge(c, scene.time)}</small>{/if}
        </label></div>
      {/each}
      {#if miss.length}
        <div class="warn">
          <strong>Named in the text, not in the cast</strong>
          {#each miss as c (c.id)}
            <div class="mrow">
              <span><span class="dot" style:background={c.color}></span> {c.name}</span>
              <span class="row fit" style="gap:.2rem">
                <button class="small" onclick={() => addToCast(c.id)}>Add to cast</button>
                <button class="small" title="Only mentioned. Do not warn again for this scene." onclick={() => ignoreMention(c.id)}>Not here</button>
              </span>
            </div>
          {/each}
        </div>
      {/if}
      <label>Notes and plan</label>
      <textarea bind:value={scene.summary} oninput={saveScene} placeholder="What happens? What must the reader learn?"></textarea>
      <label>Word goal for this scene</label>
      <input type="number" min="0" bind:value={scene.goal} oninput={saveScene} />
      <label>Move to chapter</label>
      <Select value={scene.chapter} onchange={(e) => moveScene(+e.currentTarget.value)}>
        {#each chapters as c}<option value={c.id}>{c.order}. {c.title}</option>{/each}
      </Select>
    {/if}

    {#if chap}
      <div class="head" style="margin-top:1.4rem"><h3 style="margin:0">Chapter</h3>
        <span class="row fit" style="gap:.2rem">
          <button class="small" title="Move up" onclick={() => swapOrder('chapter', chapters, chap, -1)}>↑</button>
          <button class="small" title="Move down" onclick={() => swapOrder('chapter', chapters, chap, 1)}>↓</button>
          <ConfirmButton onconfirm={delChap} />
        </span>
      </div>
      <label>Word goal for this chapter</label>
      <input type="number" min="0" bind:value={chap.goal} oninput={saveChap} placeholder={book?.chapterGoal ? 'Default: ' + book.chapterGoal : 'No goal'} />
      <label>In this chapter</label>
      <div>
        {#each chapterCast(chap) as id}<span class="chip"><span class="dot" style="display:inline-block;background:{db.get('character', id)?.color}"></span> {db.name('character', id)}</span>
        {:else}<span class="muted">No one yet. Tick people in a scene.</span>{/each}
      </div>
      <label>Summary</label><textarea bind:value={chap.summary} oninput={saveChap}></textarea>
    {/if}
  </aside>
</div>

{#if showFind}
  <FindReplace {bookId} onjump={jumpToScene} onchanged={() => rev++} onclose={() => (showFind = false)} />
{/if}
{#if showSnaps && book}
  <Snapshots bookId={book.id}
    onrestore={async (id, body) => {
      const s = db.get('scene', id);
      if (!s) return;
      const now = s.body ?? '';
      if (now.trim() && now !== body) await addVersion(id, now, 'Before going back to a snapshot');
      s.body = body; await db.save('scene', s); rev++;
    }}
    onclose={() => (showSnaps = false)} />
{/if}
{#if showHistory && scene}
  <History sceneId={scene.id} title={scene.title} current={() => scene.body ?? ''}
    onrestore={async (body) => { scene.body = body; await db.save('scene', scene); rev++; }}
    onclose={() => (showHistory = false)} />
{/if}

{#if drag}<div class="ghost" style="transform: translate({ghost.x + 14}px, {ghost.y + 8}px) rotate({ghost.tilt}deg)"><span>{drag.label}</span></div>{/if}

<style>
  .list :global(.item) { flex-wrap: wrap; row-gap: .1rem; }
  .ct { flex: 1 1 100%; min-width: 0; overflow-wrap: anywhere; }
  .list :global(.item small.cw) { flex: 1 1 100%; margin-left: 0; font-size: .75rem; }
  .scenes { margin: 0 0 .5rem 1rem; padding-left: .6rem; border-left: 2px solid var(--line); display: flex; flex-direction: column; gap: 2px; }
  .scene { display: flex; align-items: center; gap: .45rem; text-align: left; background: none; border: 1px solid transparent; padding: .25rem .5rem; font-size: .85rem; }
  .list :global(.item.nomatch) { opacity: .45; }
  .scene.on { background: var(--panel-2); border-color: var(--line); }
  .scene .t { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .scene small { color: var(--muted); }
  .scenes > button.small { align-self: flex-start; margin-top: .2rem; }
  .status { width: .55rem; height: .55rem; border-radius: 50%; flex: 0 0 auto; background: var(--muted); }
  .status.draft { background: var(--accent); }
  /* the counts stay at the bottom of the pane while you scroll or type */
  .statusbar { position: sticky; bottom: -1.2rem; z-index: 5; margin: 0 -1.6rem -1.2rem; padding: .1rem 1.6rem .5rem; background: var(--bg); border-top: 1px solid var(--line); }
  .statusbar p { margin: .4rem 0 0; }
  .status.revised { background: var(--accent-2); }
  .status.final { background: var(--ok); }
  .list.sorting { user-select: none; -webkit-user-select: none; cursor: grabbing; }
  .list :global(.item), .scene { cursor: grab; position: relative; transition: opacity .15s, transform .15s, background .15s; }
  .dragging { opacity: 0.35; transform: scale(0.97); }

  /* a line grows from the middle to show where the item will land */
  .drop-before::before, .drop-after::before {
    content: ''; position: absolute; left: 0; right: 0; height: 3px; border-radius: 2px; background: var(--accent);
    box-shadow: 0 0 8px var(--accent); animation: grow .16s ease-out; pointer-events: none;
  }
  .drop-before::before { top: -2px; }
  .drop-after::before { bottom: -2px; }
  .drop-into { outline: 2px dashed var(--accent); outline-offset: -2px; background: color-mix(in srgb, var(--accent) 14%, transparent); animation: breathe .9s ease-in-out infinite; }

  /* the row flashes where it landed */
  .landed { animation: landed .9s ease-out; }

  .ghost {
    position: fixed; left: 0; top: 0; z-index: 70; pointer-events: none; max-width: 230px;
    transition: transform 70ms ease-out; will-change: transform;
  }
  .ghost span {
    display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
    background: var(--panel); border: 1px solid var(--accent); border-radius: 8px; padding: .3rem .7rem; font-size: .85rem;
    box-shadow: 0 12px 28px rgba(0, 0, 0, .45), 0 0 0 3px color-mix(in srgb, var(--accent) 25%, transparent);
    animation: pop .18s cubic-bezier(.2, 1.4, .4, 1);
  }

  @keyframes grow { from { transform: scaleX(0.1); opacity: 0; } to { transform: scaleX(1); opacity: 1; } }
  @keyframes pop { from { transform: scale(0.7); opacity: 0; } to { transform: scale(1.04); opacity: 1; } }
  @keyframes breathe { 50% { background: color-mix(in srgb, var(--accent) 26%, transparent); } }
  @keyframes landed { 0% { background: color-mix(in srgb, var(--accent) 55%, transparent); transform: scale(1.03); } 100% { background: transparent; transform: scale(1); } }
  @media (prefers-reduced-motion: reduce) {
    .ghost, .ghost span, .drop-before::before, .drop-after::before, .drop-into, .landed, .dragging { animation: none; transition: none; }
  }
  .scenetitle { margin-bottom: .6rem; font-size: .95rem; color: var(--muted); }
  .badge { font-size: .7rem; color: var(--muted); }
  .warn { border: 1px solid var(--accent-2); border-radius: 6px; padding: .5rem; margin-top: .6rem; font-size: .85rem; background: color-mix(in srgb, var(--accent-2) 10%, transparent); }
  .age { margin-left: auto; color: var(--muted); }
  .age.bad { color: var(--danger); }
  .arcrow { display: flex; align-items: center; gap: .4rem; margin: .15rem 0; }
  .arcchk { flex: 1; min-width: 0; display: flex; align-items: center; gap: .4rem; margin: 0; color: var(--text); }
  .arcname { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .mrow { display: flex; align-items: center; justify-content: space-between; gap: .4rem; margin-top: .3rem; }
  .tools { display: flex; flex-wrap: wrap; gap: .4rem; margin-bottom: .6rem; }
  .scenemsg { margin-bottom: .5rem; }
  .tools kbd { font-size: .7rem; color: var(--muted); margin-left: .3rem; }
</style>
