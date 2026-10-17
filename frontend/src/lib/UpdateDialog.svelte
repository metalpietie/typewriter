<script lang="ts">
  import { veil, dialog } from './motion.svelte';
  import { db } from './db.svelte';
  import { notesHtml } from './notes-core';
  import { updates, checkForUpdates, closeWindow, install, restartNow, restartLater, skipThisVersion, openReleasePage } from './updates.svelte';
  import ProgressBar from './ProgressBar.svelte';

  // The window for a new version: what is new, the download, and the restart.
  const info = $derived(updates.info);
  const mb = (n: number) => (n / 1048576).toFixed(1);
  // the release text often starts with the same heading as the title: it is shown once
  const notes = $derived.by(() => {
    if (!info?.notes) return '';
    const lines = info.notes.replace(/\r\n/g, '\n').split('\n');
    const first = lines.findIndex((l) => l.trim() && !l.trim().startsWith('```'));
    if (first >= 0 && lines[first].trim().replace(/^##\s+/, '') === info.title.trim()) lines.splice(first, 1);
    return notesHtml(lines.join('\n'));
  });
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape' && updates.open) { e.stopPropagation(); closeWindow(); } }} />

{#if updates.open}
  <div class="overlay" transition:veil|global role="presentation" onmousedown={(e) => { if (e.target === e.currentTarget) closeWindow(); }}>
    <div class="palette box" transition:dialog|global role="dialog" aria-label="Updates">
      {#if updates.phase === 'checking'}
        <h2>Updates</h2>
        <p class="muted">Looking for a new version…</p>

      {:else if updates.phase === 'none'}
        <h2>Updates</h2>
        <p>Typewriter {db.version} is the newest version.</p>
        <div class="btns"><button class="primary" onclick={closeWindow}>Close</button></div>

      {:else if updates.phase === 'error'}
        <h2>Updates</h2>
        <p style:color="var(--danger)">{updates.error}</p>
        <div class="btns">
          <button onclick={closeWindow}>Close</button>
          {#if info?.available}<button onclick={openReleasePage}>Open the release page</button>{/if}
          <button class="primary" onclick={() => checkForUpdates()}>Try again</button>
        </div>

      {:else if updates.phase === 'found' && info}
        <h2>{info.title}</h2>
        <p class="muted">You have Typewriter {info.current}. Version {info.version} is out.</p>
        {#if notes}<div class="notes">{@html notes}</div>{/if}
        {#if !info.canInstall}<p class="warn">{info.why}</p>{/if}
        <div class="btns">
          <button class="link-btn" onclick={skipThisVersion} title="The check at start does not show this version again">Skip this version</button>
          <span class="grow"></span>
          <button onclick={closeWindow}>Later</button>
          {#if info.canInstall}
            <button class="primary" onclick={install}>Update now</button>
          {:else}
            <button class="primary" onclick={openReleasePage}>Open the release page</button>
          {/if}
        </div>

      {:else if updates.phase === 'downloading'}
        <h2>Downloading {info?.version ?? ''}</h2>
        <ProgressBar value={updates.written} goal={Math.max(1, updates.total)} />
        <p class="muted">{updates.total ? `${mb(updates.written)} of ${mb(updates.total)} MB` : `${mb(updates.written)} MB`}. The file is checked before anything changes.</p>

      {:else if updates.phase === 'ready' || updates.phase === 'restarting'}
        <h2>Version {info?.version ?? ''} is ready</h2>
        <p>The download passed the check. Typewriter saves everything and makes a copy of your data, then starts again in the new version.</p>
        {#if updates.error}<p style:color="var(--danger)">{updates.error}</p>{/if}
        <div class="btns">
          <button disabled={updates.phase === 'restarting'} onclick={restartLater}>When I close the app</button>
          <button class="primary" disabled={updates.phase === 'restarting'} onclick={restartNow}>{updates.phase === 'restarting' ? 'Starting…' : 'Restart now'}</button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .box { width: min(560px, 94vw); max-height: 80vh; overflow: auto; padding: 1.1rem 1.2rem; }
  h2 { font-size: 1.15rem; }
  .notes { margin: .6rem 0; padding: .6rem .8rem; border: 1px solid var(--line); border-radius: 8px; background: var(--panel-2); max-height: 40vh; overflow: auto; font-size: .88rem; }
  .notes :global(h4) { font-size: .95rem; margin: .2rem 0 .4rem; }
  .notes :global(h5) { font-size: .82rem; margin: .7rem 0 .3rem; text-transform: uppercase; letter-spacing: .05em; color: var(--muted); }
  .notes :global(ul), .notes :global(ol) { margin: .2rem 0; padding-left: 1.2rem; }
  .notes :global(li) { margin: .15rem 0; }
  .notes :global(p) { margin: .3rem 0; }
  .notes :global(code) { font-size: .85em; padding: 0 .25em; border-radius: 4px; background: var(--panel); }
  .warn { color: var(--danger); font-size: .88rem; }
  .btns { display: flex; gap: .5rem; justify-content: flex-end; align-items: center; margin-top: 1rem; }
  .grow { flex: 1; }
  .link-btn { background: none; border-color: transparent; color: var(--muted); padding-left: 0; }
  .link-btn:hover { color: var(--text); border-color: transparent; }
</style>
