<script lang="ts">
  import Select from '../lib/Select.svelte';
  import { Exporter } from '../../bindings/typewriter';
  import { db } from '../lib/db.svelte';
  import Backup from '../lib/Backup.svelte';

  let what = $state<'manuscript' | 'bible'>('manuscript');
  let bookId = $state<number>(0);
  let format = $state<'docx' | 'pdf' | 'epub' | 'md'>('docx');
  let pageSize = $state('letter');
  let print = $state(false);
  let author = $state(saved('typewriter.author'));
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

  async function run() {
    busy = true; message = ''; failed = false;
    try { localStorage.setItem('typewriter.author', author.trim()); } catch { /* the name is only a convenience */ }
    try {
      const path = await Exporter.Export(what, what === 'manuscript' ? bookId : 0, format, { author: author.trim(), pageSize, print: format === 'pdf' && print });
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
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={what} value="bible" />
    Story bible — characters, relationships, places, lore, timeline</label>

  {#if what === 'manuscript'}
    <label>Author name (on the title page and in the e-book)</label>
    <input bind:value={author} placeholder="Your name" />
    <label>Which book</label>
    <Select bind:value={bookId}>
      <option value={0}>All books (whole series)</option>
      {#each books as b}<option value={b.id}>{b.order}. {b.title}</option>{/each}
    </Select>
  {/if}

  <h3 style="margin-top:1.2rem">File type</h3>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={format} value="docx" />Word (.docx)</label>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={format} value="pdf" />PDF (.pdf)</label>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={format} value="epub" />E-book (.epub): for e-readers and phones</label>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={format} value="md" />Markdown (.md): plain text with simple marks</label>

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
