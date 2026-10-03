<script lang="ts">
  import { Exporter } from '../../bindings/typewriter';
  import { db } from '../lib/db.svelte';
  import Backup from '../lib/Backup.svelte';

  let what = $state<'manuscript' | 'bible'>('manuscript');
  let bookId = $state<number>(0);
  let format = $state<'docx' | 'pdf'>('docx');
  let busy = $state(false);
  let message = $state('');
  let failed = $state(false);

  const books = $derived([...db.lists.book].sort((a, b) => a.order - b.order));

  async function run() {
    busy = true; message = ''; failed = false;
    try {
      const path = await Exporter.Export(what, what === 'manuscript' ? bookId : 0, format);
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
    <label>Which book</label>
    <select bind:value={bookId}>
      <option value={0}>All books (whole series)</option>
      {#each books as b}<option value={b.id}>{b.order}. {b.title}</option>{/each}
    </select>
  {/if}

  <h3 style="margin-top:1.2rem">File type</h3>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={format} value="docx" />Word (.docx)</label>
  <label style="display:flex;gap:.5rem;color:var(--text)"><input type="radio" style="width:auto" bind:group={format} value="pdf" />PDF (.pdf)</label>

  <div style="margin-top:1.4rem"><button class="primary" disabled={busy} onclick={run}>{busy ? 'Exporting…' : 'Export…'}</button></div>
  {#if message}<p style:color={failed ? 'var(--danger)' : 'var(--ok)'}>{message}</p>{/if}
  {#if format === 'pdf'}<p class="muted">PDF uses standard fonts. Letters outside Western languages may show as “?”. Use Word for those.</p>{/if}

  <Backup />
</div>
