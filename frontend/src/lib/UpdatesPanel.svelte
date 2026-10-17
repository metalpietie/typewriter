<script lang="ts">
  import { db } from './db.svelte';
  import { updates, checkForUpdates, setAuto } from './updates.svelte';

  // Updates, in Export & Backup: the button, and the choice to check when the app starts.
</script>

<h3 style="margin-top:1.6rem">Updates</h3>
<p class="muted" style="margin-top:.2rem">
  You have Typewriter {db.version}. New versions come from GitHub. Only the newest version number is asked for: nothing of your writing is sent.
</p>
<div class="row fit" style="gap:.6rem;align-items:center;flex-wrap:wrap">
  <button disabled={updates.phase === 'checking' || updates.phase === 'downloading'} onclick={() => checkForUpdates()}>Check for updates</button>
  {#if updates.later}<span class="muted" style="font-size:.85rem">Version {updates.info?.version} starts when you close the app.</span>{/if}
</div>
<label class="chk">
  <input type="checkbox" checked={updates.auto} onchange={(e) => setAuto(e.currentTarget.checked)} />
  <span>Check for updates when the app starts, and once a day <small class="muted">(this computer only)</small></span>
</label>

<style>
  .chk { display: flex; gap: .5rem; align-items: flex-start; margin-top: .6rem; color: var(--text); font-size: .9rem; }
  .chk input { width: auto; margin-top: .2rem; }
</style>
