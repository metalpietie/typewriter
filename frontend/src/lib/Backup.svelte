<script lang="ts">
  import { Store } from '../../bindings/typewriter';
  import { db } from './db.svelte';

  const KEY = 'typewriter-last-backup';
  const read = () => { try { return localStorage.getItem(KEY) ?? ''; } catch { return ''; } };

  let busy = $state(false);
  let confirming = $state(false);
  let message = $state('');
  let failed = $state(false);
  let last = $state(read());

  /** Wait until every edit has reached the database, so the backup is complete. */
  async function settle() {
    db.flush();
    for (let i = 0; i < 40 && db.waiting + db.busy > 0; i++) await new Promise((r) => setTimeout(r, 50));
  }

  async function backup() {
    busy = true; message = ''; failed = false;
    try {
      await settle();
      const path = await Store.Backup();
      if (path) {
        message = `Backup saved: ${path}`;
        last = new Date().toLocaleString();
        try { localStorage.setItem(KEY, last); } catch { /* storage may be blocked */ }
      }
    } catch (e: any) {
      failed = true; message = e?.message ?? String(e);
    } finally { busy = false; }
  }

  async function restore() {
    confirming = false;
    busy = true; message = ''; failed = false;
    try {
      await settle();
      const res = await Store.RestoreBackup();
      if (res.restored) {
        await db.load();
        db.undoable = null; // an old undo would point at data that is gone
        message = `Restored ${res.items} item${res.items === 1 ? '' : 's'}. Your previous data was saved first: ${res.safetyCopy}`;
      }
    } catch (e: any) {
      failed = true; message = e?.message ?? String(e);
    } finally { busy = false; }
  }
</script>

<h2 style="margin-top:2.4rem">Backup</h2>
<p class="muted" style="margin-top:0">
  A backup is one file with everything: books, characters, maps, and drawings.
  {#if last}Last backup from this app: {last}.{:else}You have not made a backup yet.{/if}
</p>

<div class="row" style="justify-content:flex-start">
  <button class="primary" disabled={busy} onclick={backup}>{busy && !confirming ? 'Working…' : 'Back up my data…'}</button>
  {#if !confirming}
    <button disabled={busy} onclick={() => (confirming = true)}>Restore from backup…</button>
  {/if}
</div>

{#if confirming}
  <div class="card" style="border-color:var(--danger)">
    <strong>Replace everything with a backup?</strong>
    <p class="muted" style="margin:.4rem 0 .8rem">
      All books, characters, maps, and drawings in the app are replaced by the ones in the backup file.
      Your current data is saved to a safety copy first, so you can go back.
    </p>
    <div class="row" style="justify-content:flex-start">
      <button class="danger" onclick={restore}>Choose backup file…</button>
      <button onclick={() => (confirming = false)}>Cancel</button>
    </div>
  </div>
{/if}

{#if message}<p style:color={failed ? 'var(--danger)' : 'var(--ok)'} style="word-break:break-all">{message}</p>{/if}
