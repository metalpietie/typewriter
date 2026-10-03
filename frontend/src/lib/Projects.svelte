<script lang="ts">
  import { Store } from '../../bindings/typewriter';
  import { onMount } from 'svelte';
  import { db } from './db.svelte';
  import { nav } from './nav.svelte';
  import ConfirmButton from './ConfirmButton.svelte';
  import type { ProjectItem } from '../../bindings/typewriter/models.js';

  let list = $state<ProjectItem[]>([]);
  let newName = $state('');
  let editing = $state<string | null>(null);
  let editName = $state('');
  let busy = $state(false);
  let message = $state('');
  let failed = $state(false);

  const size = (n: number) => (n > 1048576 ? (n / 1048576).toFixed(1) + ' MB' : Math.max(1, Math.round(n / 1024)) + ' KB');
  const refresh = () => Store.Projects().then((l) => (list = l ?? [])).catch(() => {});
  onMount(refresh);

  /** Wait until every edit has reached the data file, so nothing is lost when the file changes. */
  async function settle() {
    db.flush();
    for (let i = 0; i < 40 && db.waiting + db.busy > 0; i++) await new Promise((r) => setTimeout(r, 50));
  }

  /** Run one change. After a project opens, reload the screen so it shows the new data. */
  async function run(job: () => Promise<unknown>, reload = false) {
    busy = true; message = ''; failed = false;
    try {
      await settle();
      await job();
      if (reload) { location.reload(); return; }
      await refresh();
    } catch (e: any) {
      failed = true; message = e?.message ?? String(e);
    } finally { busy = false; }
  }

  const open = (p: ProjectItem) => run(() => Store.SwitchProject(p.id), true);
  const create = () => run(() => Store.NewProject(newName), true);
  async function rename(p: ProjectItem) {
    const name = editName;
    editing = null;
    await run(() => Store.RenameProject(p.id, name));
    if (p.current) db.project = name.trim() || db.project;
  }
  const remove = (p: ProjectItem) => run(async () => {
    const kept = await Store.RemoveProject(p.id);
    message = kept ? `Removed. The file is kept here: ${kept}` : 'Removed.';
  });
  const close = () => { db.flush(); nav.projects = false; };
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') close(); }} />

<div class="overlay" role="presentation" onmousedown={(e) => { if (e.target === e.currentTarget) close(); }}>
  <div class="palette box" role="dialog" aria-label="Projects">
    <div class="head"><h2 style="margin:0">Projects</h2><button class="small" onclick={close}>Close</button></div>
    <p class="muted" style="margin-top:.4rem">
      Each project is one book or one series. It has its own data file, its own backups, and its own characters, places, maps and text.
    </p>

    {#each list as p (p.id)}
      <div class="row proj" class:on={p.current}>
        {#if editing === p.id}
          <input bind:value={editName} onkeydown={(e) => { if (e.key === 'Enter') rename(p); }} />
          <button class="small primary" onclick={() => rename(p)}>Save</button>
          <button class="small" onclick={() => (editing = null)}>Cancel</button>
        {:else}
          <span class="name">{p.name}{#if p.current} <em class="muted">(open)</em>{/if}</span>
          <span class="muted fit">{size(p.size)}</span>
          {#if !p.current}<button class="small primary" disabled={busy} onclick={() => open(p)}>Open</button>{/if}
          <button class="small" disabled={busy} onclick={() => { editing = p.id; editName = p.name; }}>Rename</button>
          {#if !p.current}<ConfirmButton label="Remove" onconfirm={() => remove(p)} />{/if}
        {/if}
      </div>
    {/each}

    <h3 style="margin-top:1.2rem">New project</h3>
    <div class="row">
      <input placeholder="Name, for example “Book two”" bind:value={newName} onkeydown={(e) => { if (e.key === 'Enter') create(); }} />
      <button class="primary" disabled={busy} onclick={create}>Create and open</button>
    </div>
    <p class="muted" style="font-size:.8rem;margin-bottom:0">
      Opening a project reloads the app. Remove does not delete the file. It moves to the folder <code>projects\removed</code> in the app data folder.
    </p>
    {#if message}<p style:color={failed ? 'var(--danger)' : 'var(--ok)'} style="word-break:break-all">{message}</p>{/if}
  </div>
</div>

<style>
  .box { width: min(560px, 94vw); max-height: 80vh; overflow: auto; }
  .proj { padding: .35rem 0; border-top: 1px solid var(--line); }
  .proj.on { font-weight: 600; }
  .name { flex: 1; }
</style>
