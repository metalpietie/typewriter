<script lang="ts">
  import { db, wordCount } from '../lib/db.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import ProgressBar from '../lib/ProgressBar.svelte';

  let bookId = $state<number | null>(null);
  let chapId = $state<number | null>(null);

  const book = $derived(db.get('book', bookId));
  const chapters = $derived(db.lists.chapter.filter((c) => c.book === bookId).sort((a, b) => a.order - b.order));
  const chap = $derived(db.get('chapter', chapId));
  const goalOf = (c: any) => Number(c.goal) || Number(book?.chapterGoal) || 0;
  const pctText = (n: number, goal: number) => (goal > 0 ? ` (${Math.round((n / goal) * 100)}%)` : '');
  const total = $derived(chapters.reduce((n, c) => n + wordCount(c.body ?? ''), 0));

  async function addBook() {
    const b = await db.save('book', { title: 'New book', synopsis: '', order: db.lists.book.length + 1 });
    bookId = b.id; chapId = null;
  }
  async function addChapter() {
    if (!bookId) return;
    const c = await db.save('chapter', {
      book: bookId, order: chapters.length + 1, title: 'New chapter', summary: '', body: '',
      status: 'draft', pov: null, location: null, characters: [], when: '',
    });
    chapId = c.id;
  }
  async function move(c: any, dir: -1 | 1) {
    const other = chapters[chapters.indexOf(c) + dir];
    if (!other) return;
    const a = c.order; c.order = other.order; other.order = a;
    await db.save('chapter', c); await db.save('chapter', other);
  }
  function saveBook() { if (book) db.queue('book', book); }
  function save() { if (chap) db.queue('chapter', chap); }
  function toggleChar(id: number) {
    if (!chap) return;
    const set = new Set<number>(chap.characters ?? []);
    set.has(id) ? set.delete(id) : set.add(id);
    chap.characters = [...set]; save();
  }
  async function delBook() { if (bookId) { await db.remove('book', bookId); bookId = chapId = null; } }
  async function delChap() { if (chapId) { await db.remove('chapter', chapId); chapId = null; } }
</script>

<div class="split wide">
  <aside class="side">
    <div class="row"><select bind:value={bookId} onchange={() => (chapId = null)}>
      <option value={null}>— choose a book —</option>
      {#each [...db.lists.book].sort((a, b) => a.order - b.order) as b}<option value={b.id}>{b.order}. {b.title}</option>{/each}
    </select><button class="primary" onclick={addBook}>+ Book</button></div>

    {#if book}
      <label>Book title</label><input bind:value={book.title} oninput={saveBook} />
      <label>Order in series</label><input type="number" bind:value={book.order} oninput={saveBook} />
      <label>Synopsis</label><textarea bind:value={book.synopsis} oninput={saveBook}></textarea>
      <div class="row">
        <div><label>Book goal (words)</label><input type="number" min="0" bind:value={book.goal} oninput={saveBook} /></div>
        <div><label>Default per chapter</label><input type="number" min="0" bind:value={book.chapterGoal} oninput={saveBook} /></div>
      </div>
      {#if book.goal > 0}
        <div style="margin-top:.6rem"><ProgressBar value={total} goal={book.goal} /></div>
        <div class="muted" style="font-size:.8rem;margin-top:.2rem">{total.toLocaleString()} / {Number(book.goal).toLocaleString()} words{pctText(total, book.goal)}</div>
      {/if}
      <div class="head" style="margin-top:1rem"><h3 style="margin:0">Chapters · {total.toLocaleString()} words</h3>
        <button class="small primary" onclick={addChapter}>+ Chapter</button></div>
      <div class="list">
        {#each chapters as c (c.id)}
          <button class="item" class:on={c.id === chapId} onclick={() => (chapId = c.id)}>
            {c.order}. {c.title}<small>{wordCount(c.body ?? '')}{goalOf(c) ? ' / ' + goalOf(c) : ''}</small>
            {#if goalOf(c)}<span style="flex-basis:100%"><ProgressBar value={wordCount(c.body ?? '')} goal={goalOf(c)} thin /></span>{/if}
          </button>
        {:else}<div class="empty">No chapters yet.</div>{/each}
      </div>
      <div style="margin-top:1rem"><ConfirmButton label="Delete book" onconfirm={delBook} /></div>
    {/if}
  </aside>

  <section class="pane">
    {#if chap}
      <input class="editor" style="font-size:1.4rem;min-height:0;margin-bottom:.8rem" bind:value={chap.title} oninput={save} />
      <textarea class="editor" placeholder="Start writing…" bind:value={chap.body} oninput={save}></textarea>
      {@const words = wordCount(chap.body ?? '')}
      {@const goal = goalOf(chap)}
      <div style="margin-top:.6rem"><ProgressBar value={words} goal={goal} /></div>
      <p class="muted">
        {words.toLocaleString()}{goal ? ' / ' + goal.toLocaleString() : ''} words{pctText(words, goal)}
        {#if goal && words >= goal} · goal reached{:else if goal} · {(goal - words).toLocaleString()} to go{/if}
        · saved {chap.updatedAt}
      </p>
    {:else}
      <div class="empty">{book ? 'Pick a chapter, or add one.' : 'Add a book to begin.'}</div>
    {/if}
  </section>

  <aside class="side right">
    {#if chap}
      <div class="head"><h3 style="margin:0">Chapter info</h3>
        <span class="row fit" style="gap:.2rem">
          <button class="small" onclick={() => move(chap, -1)}>↑</button>
          <button class="small" onclick={() => move(chap, 1)}>↓</button>
          <ConfirmButton onconfirm={delChap} />
        </span>
      </div>
      <label>Status</label>
      <select bind:value={chap.status} onchange={save}>
        {#each ['idea', 'draft', 'revised', 'final'] as s}<option>{s}</option>{/each}
      </select>
      <label>Word goal for this chapter</label>
      <input type="number" min="0" bind:value={chap.goal} oninput={save} placeholder={book?.chapterGoal ? 'Default: ' + book.chapterGoal : 'No goal'} />
      <label>Point of view</label>
      <select bind:value={chap.pov} onchange={save}>
        <option value={null}>— none —</option>
        {#each db.lists.character as c}<option value={c.id}>{c.name}</option>{/each}
      </select>
      <label>Main place</label>
      <select bind:value={chap.location} onchange={save}>
        <option value={null}>— none —</option>
        {#each db.lists.location as l}<option value={l.id}>{l.name}</option>{/each}
      </select>
      <label>When (story time)</label><input bind:value={chap.when} oninput={save} placeholder="e.g. Year 312, autumn" />
      <label>Timeline position (number, e.g. year)</label><input type="number" step="any" bind:value={chap.time} oninput={save} />
      <label>Characters in this chapter</label>
      {#each db.lists.character as c}
        <div><label style="display:flex;gap:.4rem;margin:.15rem 0;color:var(--text)">
          <input type="checkbox" style="width:auto" checked={(chap.characters ?? []).includes(c.id)} onchange={() => toggleChar(c.id)} />
          <span class="dot" style:background={c.color} style="margin-top:.2rem"></span>{c.name}
        </label></div>
      {/each}
      <label>Summary</label><textarea bind:value={chap.summary} oninput={save}></textarea>
    {/if}
  </aside>
</div>
