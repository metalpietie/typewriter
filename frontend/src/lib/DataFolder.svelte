<script lang="ts">
  import { veil, dialog } from './motion.svelte';
  import { Store } from '../../bindings/typewriter';
  import { onMount } from 'svelte';
  import { db } from './db.svelte';
  import { nav } from './nav.svelte';
  import type { DataFolderInfo, FolderCheck } from '../../bindings/typewriter/models.js';

  let info = $state<DataFolderInfo | null>(null);
  let picked = $state<FolderCheck | null>(null); // a folder you chose, waiting for your answer
  let busy = $state(false);
  let message = $state('');
  let failed = $state(false);

  onMount(() => { Store.DataFolder().then((i) => (info = i)).catch(() => {}); });

  /** Wait until every edit has reached the data file, so nothing is lost when the folder changes. */
  async function settle() {
    db.flush();
    for (let i = 0; i < 40 && db.waiting + db.busy > 0; i++) await new Promise((r) => setTimeout(r, 50));
  }

  async function choose() {
    busy = true; message = ''; failed = false; picked = null;
    try {
      const path = await Store.ChooseDataFolder();
      if (path) picked = await Store.InspectDataFolder(path);
    } catch (e: any) {
      failed = true; message = e?.message ?? String(e);
    } finally { busy = false; }
  }

  /** Check the default folder the same way as any other folder. */
  async function chooseDefault() {
    if (!info) return;
    busy = true; message = ''; failed = false; picked = null;
    try { picked = await Store.InspectDataFolder(info.default); }
    catch (e: any) { failed = true; message = e?.message ?? String(e); }
    finally { busy = false; }
  }

  async function apply(mode: 'copy' | 'use') {
    if (!picked) return;
    busy = true; message = ''; failed = false;
    try {
      await settle();
      await Store.MoveDataFolder(picked.path, mode);
      location.reload(); // show the data of the new folder
    } catch (e: any) {
      failed = true; message = e?.message ?? String(e);
      busy = false;
    }
  }
  const close = () => { db.flush(); nav.dataFolder = false; };
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') close(); }} />

<div class="overlay" transition:veil|global role="presentation" onmousedown={(e) => { if (e.target === e.currentTarget) close(); }}>
  <div transition:dialog|global class="palette box" role="dialog" aria-label="Data folder">
    <div class="head"><h2 style="margin:0">Data folder</h2><button class="small" onclick={close}>Close</button></div>
    <p class="muted" style="margin-top:.4rem">
      Your books, characters, maps and backups are saved in one folder. You can keep it somewhere else, for example on another drive.
    </p>

    {#if info}
      {#if info.problem}<div class="card" style="border-color:var(--danger)">{info.problem}</div>{/if}
      <label for="cur">In use now</label>
      <input id="cur" readonly value={info.path} />
      {#if !info.isDefault}<p class="muted" style="font-size:.8rem;margin:.3rem 0 0">The normal folder is <code>{info.default}</code>.</p>{/if}
    {/if}

    {#if !picked}
      <div class="row" style="justify-content:flex-start;margin-top:.8rem">
        <button class="primary" disabled={busy} onclick={choose}>Choose another folder…</button>
        {#if info && !info.isDefault}<button disabled={busy} onclick={chooseDefault}>Go back to the normal folder</button>{/if}
      </div>
    {:else if picked.same}
      <div class="card"><b>That is the folder in use already.</b> Nothing to change.
        <div class="row" style="justify-content:flex-start;margin-top:.6rem"><button onclick={() => (picked = null)}>OK</button></div></div>
    {:else if picked.hasData}
      <div class="card">
        <strong>Use the data in this folder?</strong>
        <p style="margin:.4rem 0;word-break:break-all"><code>{picked.path}</code></p>
        <p class="muted" style="margin:.4rem 0 .8rem">
          This folder already has Typewriter data. The app will show that data. Your current data stays where it is. Nothing is deleted or overwritten.
          You can come back to it with <b>Go back to the normal folder</b> or by choosing it again.
        </p>
        <div class="row" style="justify-content:flex-start">
          <button class="primary" disabled={busy} onclick={() => apply('use')}>{busy ? 'Working…' : 'Use that data'}</button>
          <button disabled={busy} onclick={() => (picked = null)}>Cancel</button>
        </div>
      </div>
    {:else}
      <div class="card">
        <strong>Copy your data to this folder?</strong>
        <p style="margin:.4rem 0;word-break:break-all"><code>{picked.path}</code></p>
        <p class="muted" style="margin:.4rem 0 .8rem">
          The app copies all your projects and backups there, then uses that copy. The old files stay where they are, so you can delete them yourself when you are sure the copy is good.
        </p>
        <div class="row" style="justify-content:flex-start">
          <button class="primary" disabled={busy} onclick={() => apply('copy')}>{busy ? 'Copying…' : 'Copy and use'}</button>
          <button disabled={busy} onclick={() => (picked = null)}>Cancel</button>
        </div>
      </div>
    {/if}

    {#if message}<p style:color={failed ? 'var(--danger)' : 'var(--ok)'} style="word-break:break-all">{message}</p>{/if}

    <p class="muted" style="font-size:.8rem;margin-bottom:0">
      Pick a normal folder on a local drive. Do not pick a folder that Dropbox, OneDrive or Google Drive keeps in sync while the app is open. They can damage the data file. For a safe second copy, use the Backup buttons on the Export &amp; Backup screen.
      If the folder is on a drive that is not plugged in when the app starts, the app opens the normal folder and tells you.
    </p>
  </div>
</div>

<style>
  .box { width: min(600px, 94vw); max-height: 80vh; overflow: auto; }
</style>
