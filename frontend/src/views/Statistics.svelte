<script lang="ts">
  import { db } from '../lib/db.svelte';
  import ProgressBar from '../lib/ProgressBar.svelte';
  import Readability from '../lib/Readability.svelte';
  import CountUp from '../lib/CountUp.svelte';
  import { chapterWords, sceneWords, scenesOf, STATUSES } from '../lib/scenes';
  import { dailyGoal, lastDays, setDailyGoal, streak, wordsToday } from '../lib/progress';

  let range = $state(30);
  const goal = $derived(dailyGoal());
  const today = $derived(wordsToday());
  const run = $derived(streak());
  const days = $derived(lastDays(range));
  const top = $derived(Math.max(goal, ...days.map((d) => d.words), 10));
  const week = $derived(lastDays(7).reduce((a, d) => a + d.words, 0));
  const active = $derived(db.lists.wordday.filter((d) => d.words > 0));
  const best = $derived(active.reduce((m, d) => (d.words > (m?.words ?? 0) ? d : m), null as (typeof active)[number] | null));
  const average = $derived(active.length ? Math.round(active.reduce((a, d) => a + d.words, 0) / active.length) : 0);

  const total = $derived(db.lists.scene.reduce((a, s) => a + sceneWords(s), 0));
  const books = $derived(
    [...db.lists.book].sort((a, b) => a.order - b.order).map((b) => ({
      book: b,
      chapters: db.lists.chapter.filter((c) => c.book === b.id).sort((x, y) => x.order - y.order)
        .map((c) => ({ c, words: chapterWords(c), scenes: scenesOf(c.id).length })),
    })),
  );
  const biggest = $derived(Math.max(1, ...books.flatMap((b) => b.chapters.map((c) => c.words))));
  const status = $derived(STATUSES.map((s) => ({ s, n: db.lists.scene.filter((x) => (x.status ?? 'draft') === s).length })));

  const people = $derived.by(() => {
    const rows = new Map<number, { pov: number; inScenes: number; words: number }>();
    for (const s of db.lists.scene) {
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
      .sort((a, b) => b.words - a.words);
  });
  const mostWords = $derived(Math.max(1, ...people.map((p) => p.words)));

  const n = (v: number) => v.toLocaleString();
  const barW = $derived(600 / days.length);
  const label = (d: Date) => d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
</script>

<section class="pane">
  <h2>Statistics</h2>

  <div class="card">
    <h3>Daily writing goal</h3>
    <div class="row" style="max-width:26rem">
      <div>
        <label for="goal">Words per day (0 means no goal)</label>
        <input id="goal" type="number" min="0" step="50" value={goal} onchange={(e) => setDailyGoal(+e.currentTarget.value)} />
      </div>
    </div>
    <p style="margin:.8rem 0 .3rem"><b>Today: <CountUp value={today} /></b>{goal ? ` / ${n(goal)} words` : ' words'}{goal && today >= goal ? ' · goal reached' : ''}</p>
    <ProgressBar value={today} {goal} />
    <p class="muted">
      Streak: <b><CountUp value={run.current} /></b> day{run.current === 1 ? '' : 's'} · Best streak: <b><CountUp value={run.best} /></b> ·
      Last 7 days: <b><CountUp value={week} /></b> words · Average on writing days: <b><CountUp value={average} /></b>
      {#if best}· Best day: <b>{n(best.words)}</b> ({best.date}){/if}
    </p>
    <p class="muted" style="font-size:.8rem;margin:0">Counts words added in the Write tab, one tally per day. Words you cut are taken off the day, but a day never goes below 0.</p>
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
      {#each days as d, i (d.key)}
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
      <text x="0" y="156" font-size="10" fill="var(--muted)">{label(days[0].date)}</text>
      <text x="600" y="156" font-size="10" text-anchor="end" fill="var(--muted)">{label(days[days.length - 1].date)}</text>
    </svg>
  </div>

  <div class="card">
    <h3>Words per chapter · <CountUp value={total} /> words in all</h3>
    {#each books as b (b.book.id)}
      <div style="margin:.6rem 0 .2rem"><b>{b.book.title}</b></div>
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
      Scenes by stage: {#each status as x, i}{i ? ' · ' : ''}{x.s} {x.n}{/each}
    </p>
  </div>

  <Readability />

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
      <div class="empty">No characters yet.</div>
    {/each}
  </div>
</section>

<style>
  .line { display: flex; flex-wrap: wrap; align-items: center; gap: .2rem .6rem; padding: .15rem 0; }
  .nm { flex: 0 1 14rem; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; display: flex; align-items: center; gap: .4rem; }
  .bar { flex: 1; min-width: 4rem; }
  .num { flex: 0 0 5.5rem; text-align: right; font-variant-numeric: tabular-nums; }
</style>
