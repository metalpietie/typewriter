<script lang="ts">
  import { db } from './db.svelte';
  import { chapterCounts } from './readability';
  import { addCounts, cleanTarget, cleanWpm, DEFAULT_WPM, ease, easeLabel, grade, marks, NO_COUNTS, overTarget, readingTime, TARGET_MARGIN } from './readability-core';

  // Reading time and reading level, for each chapter. Shown in the Statistics tab.
  const KEY = 'typewriter-wpm';
  let wpm = $state(DEFAULT_WPM);
  try { wpm = cleanWpm(localStorage.getItem(KEY)); } catch { /* storage may be blocked */ }
  function setWpm(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    wpm = cleanWpm(input.value);
    input.value = String(wpm);
    try { localStorage.setItem(KEY, String(wpm)); } catch { /* it lasts until the app closes */ }
  }

  // The target grade is kept with the project: a chapter well above it is marked.
  const targetRow = $derived(db.lists.setting.find((s) => s.key === 'readingTarget'));
  const target = $derived(cleanTarget(targetRow?.value));
  async function setTarget(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    const v = cleanTarget(input.value);
    input.value = v ? String(v) : '';
    await db.save('setting', { ...(targetRow ?? {}), key: 'readingTarget', value: v });
  }

  const books = $derived(
    [...db.lists.book].sort((a, b) => a.order - b.order).map((b) => {
      const chapters = db.lists.chapter.filter((c) => c.book === b.id).sort((x, y) => x.order - y.order)
        .map((c) => ({ c, counts: chapterCounts(c) }));
      const flags = marks(chapters.map((r) => r.counts));
      const all = chapters.reduce((sum, r) => addCounts(sum, r.counts), NO_COUNTS);
      return { book: b, all, rows: chapters.map((r, i) => ({ ...r, flag: flags[i] })) };
    }),
  );
  const n = (v: number) => v.toLocaleString();
  const level = (c: typeof NO_COUNTS) => {
    const e = ease(c), g = grade(c);
    return e === null || g === null ? 'too short to judge' : `${easeLabel(e)} · grade ${g >= 20 ? '20+' : g.toFixed(1)}`;
  };
</script>

<div class="card">
  <h3>Reading time and level</h3>
  <p class="muted" style="margin-top:0">
    The level comes from the length of sentences and words (Flesch reading ease and grade). It is a rough guide, and it is made for English.
    Chapters with a lot of dialogue read as easy. A chapter much harder or easier than the rest of its book is marked.
    Chapters under 300 words are not judged.
  </p>
  <div class="row" style="justify-content:flex-start;margin-bottom:.6rem">
    <label class="fit" for="wpm" style="margin:0">Reading speed</label>
    <input id="wpm" class="wpmbox" type="number" min="80" max="600" value={wpm} onchange={setWpm} />
    <span class="fit muted">words a minute</span>
  </div>
  <div class="row" style="justify-content:flex-start;margin-bottom:.6rem">
    <label class="fit" for="tgrade" style="margin:0">Target grade</label>
    <input id="tgrade" class="wpmbox" type="number" min="1" max="20" placeholder="none" value={target || ''} onchange={setTarget} />
    <span class="fit muted">a chapter more than {TARGET_MARGIN} grades above it is marked (for example 8 for a book anyone can read)</span>
  </div>

  {#each books as b (b.book.id)}
    <div style="margin:.8rem 0 .3rem"><b>{b.book.title}</b>
      <span class="muted"> · {n(b.all.words)} words · {readingTime(b.all.words, wpm)} · {level(b.all)}</span></div>
    {#each b.rows as r (r.c.id)}
      <div class="rl">
        <span class="nm">{r.c.order}. {r.c.title}</span>
        <span class="num">{n(r.counts.words)}</span>
        <span class="num">{readingTime(r.counts.words, wpm)}</span>
        <span class="lv">{level(r.counts)}</span>
        <span class="mk" class:harder={r.flag === 'harder' || overTarget(r.counts, target)} class:easier={r.flag === 'easier' && !overTarget(r.counts, target)}>{overTarget(r.counts, target) ? 'above your target' : r.flag ? r.flag + ' than the rest' : ''}</span>
      </div>
    {:else}
      <div class="muted">No chapters yet.</div>
    {/each}
  {:else}
    <div class="empty">No books yet.</div>
  {/each}
</div>

<style>
  .rl { display: grid; grid-template-columns: minmax(8rem, 1.4fr) 5rem 5rem minmax(10rem, 1.6fr) 9rem; gap: .6rem; align-items: baseline; padding: .15rem 0; border-top: 1px solid var(--line); font-size: .9rem; }
  .nm { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .num { text-align: right; font-variant-numeric: tabular-nums; }
  .lv { color: var(--muted); }
  .mk { font-size: .8rem; }
  .mk.harder { color: var(--danger); }
  .mk.easier { color: var(--accent); }
  .wpmbox { width: 5.5rem; flex: 0 0 auto; padding: .3rem .5rem; }
  @media (max-width: 760px) { .rl { grid-template-columns: 1fr 4rem 4rem; } .lv, .mk { grid-column: 1 / -1; } }
</style>
