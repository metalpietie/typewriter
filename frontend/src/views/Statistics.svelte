<script lang="ts">
  import { db } from '../lib/db.svelte';
  import ProgressBar from '../lib/ProgressBar.svelte';
  import Readability from '../lib/Readability.svelte';
  import CountUp from '../lib/CountUp.svelte';
  import Tabs from '../lib/Tabs.svelte';
  import Select from '../lib/Select.svelte';
  import { chapterWords, sceneWords, scenesOf, STATUSES } from '../lib/scenes';
  import { has, word } from '../lib/projecttype.svelte';
  import { addDays, dailyGoal, dayKey, daysOf, lastDays, setDailyGoal, streak, type DayRow } from '../lib/progress';
  import { Store } from '../../bindings/typewriter/internal/app';
  import type { ProjectDays, Shelf } from '../../bindings/typewriter/internal/app/models.js';

  // What the numbers count: one book, the open project, or every project on the bookshelf.
  type Scope = 'book' | 'project' | 'all';
  const SCOPE_KEY = 'typewriter.statsScope', BOOK_KEY = 'typewriter.statsBook';
  const stored = (key: string) => { try { return localStorage.getItem(key); } catch { return null; } };
  const keep = (key: string, v: string) => { try { localStorage.setItem(key, v); } catch { /* the choice lasts until the app closes */ } };

  let scope = $state<Scope>((['book', 'project', 'all'] as const).find((s) => s === stored(SCOPE_KEY)) ?? 'project');
  let bookPick = $state(Number(stored(BOOK_KEY)) || 0);
  $effect(() => keep(SCOPE_KEY, scope));

  const sortedBooks = $derived([...db.lists.book].sort((a, b) => a.order - b.order));
  const bookId = $derived(sortedBooks.find((b) => b.id === bookPick)?.id ?? sortedBooks[0]?.id ?? 0);
  const bookTitle = $derived(db.get('book', bookId)?.title || 'Untitled');
  const scopes = $derived([
    ...(sortedBooks.length > 1 ? [{ id: 'book', label: 'This book' }] : []),
    { id: 'project', label: 'This project' },
    { id: 'all', label: 'All projects' },
  ]);
  $effect(() => { if (scope === 'book' && sortedBooks.length < 2) scope = 'project'; }); // one book: the project is the book

  // the other projects are read from their files when "All projects" opens; the open one counts live
  let others = $state<ProjectDays[]>([]);
  let shelves = $state<Shelf[]>([]);
  let loadFailed = $state('');
  $effect(() => {
    if (scope !== 'all') return;
    db.settle().then(() => Promise.all([Store.WordDays(), Store.Shelves()])).then(([d, s]) => {
      others = (d ?? []).filter((p) => !p.current);
      shelves = s ?? [];
      loadFailed = [...others, ...shelves].find((p) => p.problem)?.problem ?? '';
    }).catch((e) => (loadFailed = e?.message ?? String(e)));
  });

  const rows = $derived(db.lists.wordday as DayRow[]);
  const days = $derived(
    scope === 'book' ? daysOf(rows, bookId) :
    scope === 'all' ? addDays([daysOf(rows), ...others.map((p) => daysOf((p.days ?? []) as DayRow[]))]) :
    daysOf(rows),
  );

  let range = $state(30);
  const goal = $derived(dailyGoal());
  const today = $derived(days.get(dayKey()) ?? 0);
  const run = $derived(streak(days, goal));
  const chart = $derived(lastDays(range, days));
  const top = $derived(Math.max(goal, ...chart.map((d) => d.words), 10));
  const week = $derived(lastDays(7, days).reduce((a, d) => a + d.words, 0));
  const active = $derived([...days].filter(([, w]) => w > 0).map(([date, words]) => ({ date, words })));
  const best = $derived(active.reduce((m, d) => (d.words > (m?.words ?? 0) ? d : m), null as (typeof active)[number] | null));
  const average = $derived(active.length ? Math.round(active.reduce((a, d) => a + d.words, 0) / active.length) : 0);

  const books = $derived(
    sortedBooks.filter((b) => scope !== 'book' || b.id === bookId).map((b) => ({
      book: b,
      chapters: db.lists.chapter.filter((c) => c.book === b.id).sort((x, y) => x.order - y.order)
        .map((c) => ({ c, words: chapterWords(c), scenes: scenesOf(c.id).length })),
    })),
  );
  const total = $derived(books.reduce((a, b) => a + b.chapters.reduce((x, c) => x + c.words, 0), 0));
  const biggest = $derived(Math.max(1, ...books.flatMap((b) => b.chapters.map((c) => c.words))));
  // the scenes the numbers below count: all of the project, or the ones of the book
  const scenes = $derived.by(() => {
    if (scope !== 'book') return db.lists.scene;
    const chapters = new Set(db.lists.chapter.filter((c) => c.book === bookId).map((c) => c.id));
    return db.lists.scene.filter((s) => chapters.has(s.chapter));
  });
  const status = $derived(STATUSES.map((s) => ({ s, n: scenes.filter((x) => (x.status ?? 'draft') === s).length })));

  // All projects: the words of each book on the bookshelf, by project. The open project counts live.
  const shelfRows = $derived(shelves.map((s) => ({
    s,
    books: s.current
      ? books.map((b) => ({ id: b.book.id, title: b.book.title, words: b.chapters.reduce((x, c) => x + c.words, 0) }))
      : (s.books ?? []).map((b) => ({ id: b.id, title: b.title, words: b.words })),
  })));
  const allWords = $derived(shelfRows.reduce((a, r) => a + r.books.reduce((x, b) => x + b.words, 0), 0));
  const biggestBook = $derived(Math.max(1, ...shelfRows.flatMap((r) => r.books.map((b) => b.words))));

  const people = $derived.by(() => {
    const rows = new Map<number, { pov: number; inScenes: number; words: number }>();
    for (const s of scenes) {
      const w = sceneWords(s);
      const cast = new Set<number>([...(s.pov ? [s.pov] : []), ...(s.characters ?? [])]);
      for (const id of cast) {
        const r = rows.get(id) ?? { pov: 0, inScenes: 0, words: 0 };
        r.inScenes++;
        r.words += w;
        if (s.pov === id) r.pov += w;
        rows.set(id, r);
      }
    }
    return db.lists.character
      .map((c) => ({ c, ...(rows.get(c.id) ?? { pov: 0, inScenes: 0, words: 0 }) }))
      .filter((p) => scope !== 'book' || p.inScenes) // a book shows the characters that are in it
      .sort((a, b) => b.words - a.words);
  });
  const mostWords = $derived(Math.max(1, ...people.map((p) => p.words)));

  const n = (v: number) => v.toLocaleString();
  const barW = $derived(600 / chart.length);
  const label = (d: Date) => d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
</script>

<section class="pane">
  <div class="top">
    <h2>Statistics</h2>
    <div class="scope"><Tabs tabs={scopes} bind:value={scope} label="What the statistics count" /></div>
  </div>
  {#if scope === 'book'}
    <div class="bookpick">
      <label for="stats-book">Book</label>
      <Select id="stats-book" value={bookId} onchange={(e) => { bookPick = +e.currentTarget.value; keep(BOOK_KEY, String(bookPick)); }}>
        {#each sortedBooks as b (b.id)}<option value={b.id}>{b.title || 'Untitled'}</option>{/each}
      </Select>
    </div>
  {/if}
  {#if scope === 'all' && loadFailed}<p class="muted" style="color:var(--danger)">Some projects could not be read: {loadFailed}</p>{/if}

  <div class="card">
    <h3>Daily writing goal</h3>
    <div class="row" style="max-width:26rem">
      <div>
        <label for="goal">Words per day (0 means no goal)</label>
        <input id="goal" type="number" min="0" step="50" value={goal} onchange={(e) => setDailyGoal(+e.currentTarget.value)} />
      </div>
    </div>
    <p style="margin:.8rem 0 .3rem"><b>Today{scope === 'all' ? ', in all projects' : scope === 'book' ? `, in ${bookTitle}` : ''}: <CountUp value={today} /></b>{goal ? ` / ${n(goal)} words` : ' words'}{goal && today >= goal ? ' · goal reached' : ''}</p>
    <ProgressBar value={today} {goal} />
    <p class="muted">
      Streak: <b><CountUp value={run.current} /></b> day{run.current === 1 ? '' : 's'} · Best streak: <b><CountUp value={run.best} /></b> ·
      Last 7 days: <b><CountUp value={week} /></b> words · Average on writing days: <b><CountUp value={average} /></b>
      {#if best}· Best day: <b>{n(best.words)}</b> ({best.date}){/if}
    </p>
    <p class="muted" style="font-size:.8rem;margin:0">
      Counts words added in the Write tab, one tally per day. Words you cut are taken off the day, but a day never goes below 0.
      {#if scope === 'all'}The goal is the one of this project. Each project has its own goal.{/if}
      {#if scope === 'book'}Days from before version 1.16 are not split by book, so they do not count here.{/if}
    </p>
  </div>

  <div class="card">
    <div class="head">
      <h3 style="margin:0">Words per day</h3>
      <span class="fit">
        <button class="small" class:active={range === 14} onclick={() => (range = 14)}>14 days</button>
        <button class="small" class:active={range === 30} onclick={() => (range = 30)}>30 days</button>
        <button class="small" class:active={range === 90} onclick={() => (range = 90)}>90 days</button>
      </span>
    </div>
    <svg viewBox="0 0 600 160" role="img" aria-label="Words written each day" style="width:100%;height:auto">
      {#each chart as d, i (d.key)}
        {@const h = (d.words / top) * 130}
        <rect x={i * barW + 1} y={140 - h} width={Math.max(1, barW - 2)} height={h} rx="2"
          fill={goal && d.words >= goal ? 'var(--ok)' : 'var(--accent)'} opacity={d.words ? 1 : 0.15}>
          <title>{label(d.date)}: {n(d.words)} words</title>
        </rect>
      {/each}
      {#if goal}
        <line x1="0" x2="600" y1={140 - (goal / top) * 130} y2={140 - (goal / top) * 130} stroke="var(--danger)" stroke-dasharray="4 4" />
        <text x="598" y={136 - (goal / top) * 130} text-anchor="end" font-size="10" fill="var(--danger)">goal {n(goal)}</text>
      {/if}
      <text x="0" y="156" font-size="10" fill="var(--muted)">{label(chart[0].date)}</text>
      <text x="600" y="156" font-size="10" text-anchor="end" fill="var(--muted)">{label(chart[chart.length - 1].date)}</text>
    </svg>
  </div>

  {#if scope === 'all'}
    <div class="card">
      <h3>Words per book · <CountUp value={allWords} /> words in all projects</h3>
      {#each shelfRows as r (r.s.id)}
        <div style="margin:.6rem 0 .2rem"><b>{r.s.name}</b>{#if r.s.group}<span class="muted">{' · ' + r.s.group}</span>{/if}{#if r.s.current}<span class="muted">{' · open now'}</span>{/if}</div>
        {#each r.books as b (b.id)}
          <div class="line">
            <span class="nm">{b.title || 'Untitled'}</span>
            <span class="bar"><ProgressBar value={b.words} goal={biggestBook} thin /></span>
            <span class="num">{n(b.words)}</span>
          </div>
        {:else}
          <div class="muted">No books yet.</div>
        {/each}
      {/each}
    </div>
  {:else}
    <div class="card">
      <h3>Words per chapter · <CountUp value={total} /> words in {scope === 'book' ? 'this book' : 'all'}</h3>
      {#each books as b (b.book.id)}
        {#if scope !== 'book'}<div style="margin:.6rem 0 .2rem"><b>{b.book.title}</b></div>{/if}
        {#each b.chapters as r (r.c.id)}
          <div class="line">
            <span class="nm">{r.c.order}. {r.c.title}</span>
            <span class="bar"><ProgressBar value={r.words} goal={biggest} thin /></span>
            <span class="num">{n(r.words)}</span>
          </div>
        {:else}
          <div class="muted">No chapters yet.</div>
        {/each}
      {:else}
        <div class="empty">No books yet.</div>
      {/each}
      <p class="muted" style="margin-bottom:0">
        {word('Scenes')} by stage: {#each status as x, i}{i ? ' · ' : ''}{x.s} {x.n}{/each}
      </p>
    </div>

    {#if scope === 'project'}<Readability />{/if}

    {#if has('characters')}
    <div class="card">
      <h3>Words per character</h3>
      <p class="muted" style="margin-top:0">The first number is the words in every scene they are in, as point of view or cast. “pov” is the words in scenes told through them.</p>
      {#each people as p (p.c.id)}
        <div class="line">
          <span class="nm"><span class="dot" style:background={p.c.color || 'var(--muted)'}></span> {p.c.name || 'Unnamed'}</span>
          <span class="bar"><ProgressBar value={p.words} goal={mostWords} thin /></span>
          <span class="num" title="Words in all scenes they are in">{n(p.words)}</span>
          <span class="num muted" title="Words in scenes where they are the point of view">pov {n(p.pov)}</span>
          <span class="num muted">{p.inScenes} scene{p.inScenes === 1 ? '' : 's'}</span>
        </div>
      {:else}
        <div class="empty">{scope === 'book' ? 'No characters in the scenes of this book yet.' : 'No characters yet.'}</div>
      {/each}
    </div>
    {/if}
  {/if}
</section>

<style>
  .top { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: .4rem 1rem; }
  .top h2 { margin: 0; }
  .scope { flex: 0 1 22rem; min-width: 14rem; }
  .scope :global(.tabs2) { margin: 0; }
  .bookpick { display: flex; align-items: center; gap: .6rem; max-width: 26rem; margin: .7rem 0 0; }
  .bookpick label { margin: 0; }
  .top + * { margin-top: .9rem; }
  .line { display: flex; flex-wrap: wrap; align-items: center; gap: .2rem .6rem; padding: .15rem 0; }
  .nm { flex: 0 1 14rem; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; display: flex; align-items: center; gap: .4rem; }
  .bar { flex: 1; min-width: 4rem; }
  .num { flex: 0 0 5.5rem; text-align: right; font-variant-numeric: tabular-nums; }
</style>
