<script lang="ts">
  import Select from '../lib/Select.svelte';
  import { Exporter } from '../../bindings/typewriter';
  import { db } from '../lib/db.svelte';
  import Backup from '../lib/Backup.svelte';
  import CoverEditor from '../lib/CoverEditor.svelte';
  import { getCover } from '../lib/cover';
  import { cleanPages, copyrightStarter, SCENE_BREAKS, CHAPTER_STYLES } from '../lib/exportpages-core';
  import { scenesOf } from '../lib/scenes';
  import { synopsisWords, outlineWords, missingSummaries, pagesOf } from '../lib/summary-core';

  let what = $state<'manuscript' | 'outline' | 'synopsis' | 'bible'>('manuscript');
  let outlineDetails = $state(false);
  let outlineEmpty = $state(false);
  let bookId = $state<number>(0);
  let format = $state<'docx' | 'pdf' | 'epub' | 'md'>('docx');
  let pageSize = $state('letter');
  let print = $state(false);
  let contents = $state(saved('typewriter.contents') !== 'no'); // on, until you turn it off
  let author = $state(saved('typewriter.author'));
  // The pages and style of a manuscript are kept with the project, so they are in backups too.
  const row = $derived(db.lists.setting.find((r) => r.key === 'exportPages'));
  let pages = $state(cleanPages(null));
  let loaded = false;
  $effect(() => { if (!loaded && db.ready) { loaded = true; pages = cleanPages(row?.value); } });
  async function keepPages() { await db.save('setting', { ...(row ?? {}), key: 'exportPages', value: { ...pages } }); }
  const fillCopyright = () => { if (!pages.copyright.trim()) { pages.copyright = copyrightStarter(author); keepPages(); } };
  let busy = $state(false);
  let message = $state('');
  let failed = $state(false);

  function saved(key: string): string {
    try { return localStorage.getItem(key) ?? ''; } catch { return ''; }
  }
  const SIZES = [
    ['letter', 'Letter (8.5 × 11 in)'], ['a4', 'A4 (210 × 297 mm)'], ['a5', 'A5 (148 × 210 mm)'],
    ['6x9', 'US trade book (6 × 9 in)'], ['5.5x8.5', 'Digest (5.5 × 8.5 in)'], ['5x8', 'Pocket book (5 × 8 in)'],
  ];

  const books = $derived([...db.lists.book].sort((a, b) => a.order - b.order));

  // how long the outline or the synopsis will be, from the summaries of the chosen book (or of all books)
  const summaries = $derived.by(() => {
    const chapters = db.lists.chapter.filter((c) => !bookId || c.book === bookId)
      .map((c) => ({ summary: c.summary as string, scenes: scenesOf(c.id).map((s) => ({ summary: s.summary as string })) }));
    return { synopsis: synopsisWords(chapters), outline: outlineWords(chapters), missing: missingSummaries(chapters) };
  });
  const toBook = $derived(what !== 'bible');
  // the cover is the cover of the chosen book, or of the first book when all books are chosen
  const coverBook = $derived(bookId || books[0]?.id || null); // an outline and a synopsis are made for one book or for all

  async function run() {
    busy = true; message = ''; failed = false;
    try {
      localStorage.setItem('typewriter.author', author.trim());
      localStorage.setItem('typewriter.contents', contents ? 'yes' : 'no');
    } catch { /* these are only conveniences */ }
    try {
      await keepPages();
      const cover = what === 'manuscript' && coverBook ? (await getCover(coverBook))?.image ?? '' : '';
      const path = await Exporter.Export(what, toBook ? bookId : 0, format, { author: author.trim(), pageSize, print: format === 'pdf' && print, contents: contents && (what === 'manuscript' || what === 'bible'), ...pages, outlineDetails, outlineEmpty, cover });
      message = path ? `Saved: ${path}` : 'Cancelled.';
    } catch (e: any) {
      failed = true; message = e?.message ?? String(e);
    } finally {
      busy = false;
    }
  }
</script>

<div class="pane" style="max-width:640px">
  <h2>Export</h2>

  <h3 style="margin-top:1.2rem">What to export</h3>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={what} value="manuscript" />
    Manuscript — chapter text, with a title page</label>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={what} value="outline" />
    Outline — each chapter and scene with its summary</label>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={what} value="synopsis" />
    Synopsis — the story in a few pages, from your summaries (for an agent or an editor)</label>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={what} value="bible" />
    Story bible — characters, relationships, places, lore, timeline</label>

  {#if what !== 'bible'}
    <label>Author name (on the title page and in the e-book)</label>
    <input bind:value={author} placeholder="Your name" />
    <label>Which book</label>
    <Select bind:value={bookId}>
      <option value={0}>All books (whole series)</option>
      {#each books as b}<option value={b.id}>{b.order}. {b.title}</option>{/each}
    </Select>
  {/if}

  {#if what === 'manuscript'}
    <details class="pages" style="margin-top:1rem">
      <summary><b>Pages and style</b> <span class="muted">(optional)</span></summary>
      <p class="muted" style="margin:.4rem 0">An empty box means that page is left out. Each page gets its own page. Your text is saved with the project.</p>
      <label for="px-copy">Copyright page (after the title page)</label>
      <textarea id="px-copy" rows="3" bind:value={pages.copyright} onfocus={fillCopyright} onchange={keepPages} placeholder="© 2026 Your name. All rights reserved."></textarea>
      <label for="px-ded">Dedication</label>
      <textarea id="px-ded" rows="2" bind:value={pages.dedication} onchange={keepPages} placeholder="For …"></textarea>
      <label for="px-thanks">Acknowledgments (after the last chapter)</label>
      <textarea id="px-thanks" rows="4" bind:value={pages.thanks} onchange={keepPages}></textarea>
      <label for="px-about">About the author (at the end)</label>
      <textarea id="px-about" rows="4" bind:value={pages.about} onchange={keepPages}></textarea>
      <label for="px-break">Between scenes</label>
      <Select id="px-break" bind:value={pages.sceneBreak} onchange={keepPages}>{#each SCENE_BREAKS as [key, name]}<option value={key}>{name}</option>{/each}</Select>
      <label for="px-chap">Chapter headings</label>
      <Select id="px-chap" bind:value={pages.chapterStyle} onchange={keepPages}>{#each CHAPTER_STYLES as [key, name]}<option value={key}>{name}</option>{/each}</Select>
    </details>
    <CoverEditor bookId={coverBook} {author} />
  {/if}

  {#if what === 'outline' || what === 'synopsis'}
    <div class="card" style="margin:1rem 0 0">
      {#if what === 'outline'}
        <label style="display:flex;gap:.5rem;color:var(--text);margin:0"><input type="checkbox" style="width:auto" bind:checked={outlineDetails} />
          Add the point of view and the place of each scene</label>
        <label style="display:flex;gap:.5rem;color:var(--text);margin:.4rem 0 0"><input type="checkbox" style="width:auto" bind:checked={outlineEmpty} />
          List scenes that have no summary yet</label>
      {/if}
      <p class="muted" style="margin:{what === 'outline' ? '.7rem' : '0'} 0 0;font-size:.85rem">
        {#if what === 'synopsis'}
          Your summaries make a synopsis of about {summaries.synopsis.toLocaleString()} words ({pagesOf(summaries.synopsis) || 'under 1'} page{pagesOf(summaries.synopsis) === 1 ? '' : 's'}).
          Each chapter is one paragraph: its own summary, or else the summaries of its scenes joined. Agents often ask for 1 to 2 pages (500 to 1,000 words).
        {:else}
          Your summaries come to {summaries.outline.toLocaleString()} words.
        {/if}
        {#if summaries.missing}{summaries.missing} scene{summaries.missing === 1 ? ' has' : 's have'} no summary yet. Write it in the Scene panel in the Write tab, or on the Board.{/if}
      </p>
    </div>
  {/if}

  <h3 style="margin-top:1.2rem">File type</h3>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={format} value="docx" />Word (.docx)</label>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={format} value="pdf" />PDF (.pdf)</label>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={format} value="epub" />E-book (.epub): for e-readers and phones</label>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={format} value="md" />Markdown (.md): plain text with simple marks</label>

  {#if what === 'manuscript' || what === 'bible'}
  <label style="display:flex;gap:.5rem;color:var(--text);margin-top:1rem"><input type="checkbox" style="width:auto" bind:checked={contents} />
    Table of contents after the title page</label>
  {#if contents}
    <p class="muted" style="margin:.2rem 0 0;font-size:.85rem">
      {#if format === 'pdf'}Each chapter has its page number. Click a line to jump to that chapter.
      {:else if format === 'docx'}Each line is a link to its chapter. Word does not know the page numbers, so there are none.
      {:else if format === 'epub'}A Contents page with a link for each chapter. Your e-reader also has its own list.
      {:else}A list of links to the chapter headings.{/if}
    </p>
  {/if}
  {/if}

  {#if format === 'docx' || format === 'pdf'}
    <label>Page size</label>
    <Select bind:value={pageSize}>{#each SIZES as [key, name]}<option value={key}>{name}</option>{/each}</Select>
  {/if}
  {#if format === 'pdf'}
    <label style="display:flex;gap:.5rem;color:var(--text);margin-top:.8rem"><input type="checkbox" style="width:auto" bind:checked={print} />
      Print layout: wider inside margin, page numbers in the outer corners, chapters start on a right-hand page</label>
  {/if}

  <div style="margin-top:1.4rem"><button class="primary" disabled={busy} onclick={run}>{busy ? 'Exporting…' : 'Export…'}</button></div>
  {#if message}<p style:color={failed ? 'var(--danger)' : 'var(--ok)'}>{message}</p>{/if}
  {#if format === 'epub'}<p class="muted">One page per chapter, with a contents list. It has no cover image yet. Open it in any e-reader app to check it.</p>{/if}
  {#if format === 'pdf'}<p class="muted">PDF uses standard fonts. Letters outside Western languages may show as “?”. Use Word for those.</p>{/if}

  <Backup />
</div>
