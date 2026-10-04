<script lang="ts">
  import { Store } from '../../bindings/typewriter';
  import { onMount } from 'svelte';
  import { db } from './db.svelte';
  import ConfirmButton from './ConfirmButton.svelte';
  import Select from './Select.svelte';
  import { REMIND_CHOICES, agoText, backupNow, daysSince, reminderDays, setReminderDays, settle } from './backupReminder';
  import type { AutoBackup } from '../../bindings/typewriter/models.js';

  let busy = $state(false);
  let confirming = $state(false);
  let message = $state('');
  let failed = $state(false);
  const days = $derived(db.lastBackup ? daysSince(db.lastBackup) : null);
  const when = $derived(db.lastBackup ? new Date(db.lastBackup).toLocaleString() : '');

  async function backup() {
    busy = true; message = ''; failed = false;
    try {
      const path = await backupNow();
      if (path) message = `Backup saved: ${path}`;
    } catch (e: any) {
      failed = true; message = e?.message ?? String(e);
    } finally { busy = false; }
  }

  let autos = $state<AutoBackup[]>([]);
  onMount(() => { Store.AutoBackups().then((l) => (autos = l ?? [])).catch(() => {}); });
  const day = (d: string) => new Date(d + 'T00:00:00').toLocaleDateString([], { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' });
  const size = (n: number) => (n > 1048576 ? (n / 1048576).toFixed(1) + ' MB' : Math.max(1, Math.round(n / 1024)) + ' KB');

  /** Restore from the file dialog, or from one of the daily copies when a name is given. */
  async function restore(auto?: string) {
    confirming = false;
    busy = true; message = ''; failed = false;
    try {
      await settle();
      const res = auto ? await Store.RestoreAuto(auto) : await Store.RestoreBackup();
      if (res.restored) {
        await db.load();
        db.undoable = null; // an old undo would point at data that is gone
        message = `Restored ${res.items} item${res.items === 1 ? '' : 's'}. Your previous data was saved first: ${res.safetyCopy}`;
        Store.AutoBackups().then((l) => (autos = l ?? [])).catch(() => {});
      }
    } catch (e: any) {
      failed = true; message = e?.message ?? String(e);
    } finally { busy = false; }
  }
</script>

<h2 style="margin-top:2.4rem">Backup</h2>
<p class="muted" style="margin-top:0">
  A backup is one file with everything: books, characters, maps, and drawings.
  {#if days !== null}Last backup: {agoText(days)} <span>({when})</span>.{:else}You have not made a backup yet.{/if}
</p>

<div class="row" style="justify-content:flex-start">
  <button class="primary fit" disabled={busy} onclick={backup}>{busy && !confirming ? 'Working…' : 'Back up my data…'}</button>
  {#if !confirming}
    <button class="fit" disabled={busy} onclick={() => (confirming = true)}>Restore from backup…</button>
  {/if}
</div>

<div class="row" style="justify-content:flex-start;margin-top:.8rem">
  <label class="fit" for="remind" style="margin:0">Remind me to back up</label>
  <Select id="remind" style="flex:0 1 11rem" value={reminderDays()} onchange={(e) => setReminderDays(+e.currentTarget.value)}>
    {#each REMIND_CHOICES as [v, label]}<option value={v}>{label}</option>{/each}
  </Select>
</div>
<p class="muted" style="margin:.3rem 0 0;font-size:.85rem">The reminder shows on the Dashboard. The automatic copies below stay in the app's folder, so they do not count: save a copy somewhere else too.</p>

{#if confirming}
  <div class="card" style="border-color:var(--danger)">
    <strong>Replace everything with a backup?</strong>
    <p class="muted" style="margin:.4rem 0 .8rem">
      All books, characters, maps, and drawings in the app are replaced by the ones in the backup file.
      Your current data is saved to a safety copy first, so you can go back.
    </p>
    <div class="row" style="justify-content:flex-start">
      <button class="danger" onclick={() => restore()}>Choose backup file…</button>
      <button onclick={() => (confirming = false)}>Cancel</button>
    </div>
  </div>
{/if}

<h3 style="margin-top:1.6rem">Automatic backups</h3>
<p class="muted" style="margin-top:0">
  The app saves a copy of your data the first time you open it each day. It keeps the last 7 days.
  A copy is the way your data was when that day began, so it does not hold a mistake made later that day.
</p>
{#each autos as a (a.name)}
  <div class="row auto">
    <span>{day(a.date)}</span><span class="muted fit">{size(a.size)}</span>
    <ConfirmButton label="Restore" onconfirm={() => restore(a.name)} />
  </div>
{:else}
  <p class="muted">No automatic copy yet. The first one is made the next time you open the app.</p>
{/each}
<button style="margin-top:.6rem" onclick={() => Store.OpenBackupFolder()}>Open the backup folder</button>

{#if message}<p style:color={failed ? 'var(--danger)' : 'var(--ok)'} style="word-break:break-all">{message}</p>{/if}

<style>
  .auto { padding: .25rem 0; border-top: 1px solid var(--line); max-width: 30rem; }
  .auto > span:first-child { flex: 1; }
</style>
