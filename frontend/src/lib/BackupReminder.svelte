<script lang="ts">
  import { db } from './db.svelte';
  import { agoText, backupNow, daysSince, snoozeToday } from './backupReminder';

  // A card for the Dashboard. The Dashboard shows it only when a backup is due.
  // onkeep: keep this card on the screen. The Dashboard would hide it as soon as the backup is saved,
  // and then you would not see that it worked.
  let { onhide, onkeep }: { onhide: () => void; onkeep: () => void } = $props();

  let busy = $state(false);
  let message = $state('');
  let failed = $state(false);
  let saved = $state(''); // the path of the backup you just saved

  const days = $derived(db.lastBackup ? daysSince(db.lastBackup) : null);
  const when = $derived(db.lastBackup ? new Date(db.lastBackup).toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' }) : '');

  async function run() {
    busy = true; message = ''; failed = false;
    onkeep();
    try {
      const path = await backupNow();
      if (path) saved = path;
    } catch (e: any) {
      failed = true; message = e?.message ?? String(e);
    } finally { busy = false; }
  }
  const later = () => { snoozeToday(); onhide(); };
</script>

{#if saved}
<div class="card due">
  <h3>Backup saved</h3>
  <p style="margin:.2rem 0 .6rem;word-break:break-all;color:var(--ok)">{saved}</p>
  <button class="fit" onclick={onhide}>Close</button>
</div>
{:else}
<div class="card due">
  <h3>Time for a backup</h3>
  <p style="margin:.2rem 0 .6rem">
    {#if days === null}
      You have not saved a backup copy yet.
    {:else}
      Your last backup was <b>{agoText(days)}</b> <span class="muted">({when})</span>.
    {/if}
    Keep a copy outside the app, for example on a USB drive.
  </p>
  <div class="row" style="justify-content:flex-start">
    <button class="primary fit" disabled={busy} onclick={run}>{busy ? 'Working…' : 'Back up now…'}</button>
    <button class="fit" disabled={busy} onclick={later}>Remind me tomorrow</button>
  </div>
  {#if message}<p style:color={failed ? 'var(--danger)' : 'var(--ok)'} style="word-break:break-all;margin:.6rem 0 0">{message}</p>{/if}
</div>
{/if}

<style>
  .due { border-left: 3px solid var(--accent-2); }
</style>
