<script lang="ts">
  import { onMount } from 'svelte';
  import { Events } from '@wailsio/runtime';
  import { PhoneSync } from '../../bindings/typewriter';
  import type { PairingInfo, SyncStatus } from '../../bindings/typewriter/models.js';
  import ConfirmButton from './ConfirmButton.svelte';

  // Notes on your phone, copied both ways over the Wi-Fi. Switched on here, and a phone is added once with a QR code.
  let status = $state<SyncStatus | null>(null);
  let pairing = $state<PairingInfo | null>(null);
  let message = $state('');
  let failed = $state(false);
  let busy = $state(false);
  let now = $state(Date.now());

  const refresh = () => PhoneSync.Status().then((s) => (status = s)).catch((e) => say(e, true));
  function say(e: unknown, bad = false) { message = String((e as any)?.message ?? e); failed = bad; }

  onMount(() => {
    refresh();
    const off = Events.On('typewriter:synced', () => refresh());
    const tick = setInterval(() => {
      now = Date.now();
      if (pairing) {
        // a phone was added while the code was open: close it
        const before = (status?.devices ?? []).length;
        PhoneSync.Status().then((s) => {
          status = s;
          const all = s.devices ?? [];
          if (pairing && all.length > before) { message = 'Phone added: ' + all[all.length - 1].name + '.'; failed = false; pairing = null; }
        }).catch(() => {});
        if (pairing && pairing.expires * 1000 < Date.now()) { pairing = null; message = 'The code ran out. Press Add a phone again.'; failed = true; }
      }
    }, 2000);
    return () => { off(); clearInterval(tick); PhoneSync.CancelPairing().catch(() => {}); };
  });

  async function toggle() {
    busy = true; message = '';
    try { status = status?.on ? await PhoneSync.Stop() : await PhoneSync.Start(); if (!status.on) pairing = null; }
    catch (e) { say(e, true); } finally { busy = false; }
  }
  async function addPhone() {
    message = '';
    try { pairing = await PhoneSync.StartPairing(); } catch (e) { say(e, true); }
  }
  async function closePairing() { pairing = null; await PhoneSync.CancelPairing().catch(() => {}); }
  async function remove(id: string) { try { await PhoneSync.RemoveDevice(id); await refresh(); } catch (e) { say(e, true); } }

  const left = $derived(pairing ? Math.max(0, Math.round((pairing.expires * 1000 - now) / 1000)) : 0);
  const ago = (iso: string) => {
    if (!iso) return 'never';
    const d = new Date(iso);
    return isNaN(+d) ? '' : d.toLocaleString();
  };
</script>

<section class="phones">
  <h3 style="margin-top:1.6rem">Phones</h3>
  <p class="muted" style="margin:.2rem 0 .6rem">
    Write notes on your phone and read your Notes pages there. They copy both ways over your home Wi-Fi when Typewriter is open and this is switched on. Nothing goes over the internet.
  </p>

  {#if status}
    <div class="row" style="justify-content:flex-start;gap:.6rem">
      <button class:primary={!status.on} disabled={busy} onclick={toggle}>{status.on ? 'Switch off' : 'Switch on'}</button>
      <span class="muted">{status.on ? 'On: phones can find this computer.' : 'Off: no phone can reach this computer.'}</span>
    </div>
    <p class="muted" style="margin:.5rem 0 0;font-size:.85rem">Notes sync with the project <b>{status.project.name || 'open now'}</b>. To sync another project, open it first.</p>

    {#if status.on}
      <p class="muted" style="margin:.3rem 0 0;font-size:.85rem">
        This computer: <b>{status.computer}</b>, {(status.addresses ?? []).length ? (status.addresses ?? []).map((a) => a + ':' + status?.port).join(', ') : 'no Wi-Fi address found'}.
        {status.announced ? 'It can be found by name.' : 'It cannot be found by name here. Add a phone with the QR code.'}
      </p>
      <p class="muted" style="margin:.3rem 0 0;font-size:.85rem">
        Windows may ask once to let Typewriter use the network. Choose <b>private networks</b>. A guest Wi-Fi often stops phones from reaching the computer.
      </p>

      <h3 style="margin-top:1rem">Added phones</h3>
      {#each status.devices ?? [] as d (d.id)}
        <div class="row dev">
          <span class="nm"><b>{d.name}</b><br /><small class="muted">Last sync: {ago(d.last)}</small></span>
          <ConfirmButton label="Remove" onconfirm={() => remove(d.id)} />
        </div>
      {:else}
        <p class="muted">No phone yet.</p>
      {/each}

      {#if !pairing}
        <button style="margin-top:.6rem" onclick={addPhone}>Add a phone…</button>
      {:else}
        <div class="card pair">
          <div class="qr"><img src={pairing.qr} alt="A QR code to scan with the Typewriter app on your phone" /></div>
          <div>
            <p style="margin:0 0 .4rem"><b>Open the Typewriter app on your phone, press “Add a computer”, and scan this code.</b></p>
            <p class="muted" style="margin:0 0 .4rem">If it cannot scan, pick this computer in the app and type the code: <b class="code">{pairing.code}</b></p>
            <p class="muted" style="margin:0 0 .6rem">The code works once, for {Math.floor(left / 60)}:{String(left % 60).padStart(2, '0')} more.</p>
            <button onclick={closePairing}>Cancel</button>
          </div>
        </div>
      {/if}

      <p class="muted" style="margin-top:.8rem;font-size:.85rem">
        A note that you delete here is removed from your phones at the next sync, and it cannot be brought back from the phone. Use <b>Undo</b> or a backup. A note with links, comments or citations shows on the phone, and can only be changed here.
      </p>
    {/if}
  {/if}
  {#if message}<p style:color={failed ? 'var(--danger)' : 'var(--ok)'}>{message}</p>{/if}
</section>

<style>
  .dev { padding: .35rem 0; border-top: 1px solid var(--line); }
  .dev .nm { flex: 1; }
  .pair { display: flex; gap: 1rem; align-items: flex-start; flex-wrap: wrap; margin-top: .8rem; }
  .qr img { width: 220px; height: 220px; background: #fff; padding: 8px; border-radius: 10px; display: block; image-rendering: pixelated; }
  .code { font-size: 1.2rem; letter-spacing: .15em; color: var(--text); }
</style>
