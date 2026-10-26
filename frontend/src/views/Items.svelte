<script lang="ts">
  import { takeCommand } from '../lib/commands';
  import CustomFields from '../lib/CustomFields.svelte';
  import Select from '../lib/Select.svelte';
  import { db, ITEM_TYPES } from '../lib/db.svelte';
  import { nav, openItem } from '../lib/nav.svelte';
  import { logsOf, nowOf } from '../lib/items';
  import { orderedScenes, sceneLabel } from '../lib/scenes';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import CopyButton from '../lib/CopyButton.svelte';
  import Backlinks from '../lib/Backlinks.svelte';

  let selId = $state<number | null>(null);
  let search = $state('');

  // Jump here when a search result or an @ link is opened.
  $effect(() => {
    const p = nav.pending;
    if (p?.kind === 'item') { search = ''; selId = p.id; nav.pending = null; }
  });

  const sel = $derived(db.get('item', selId));
  const shown = $derived(
    db.lists.item.filter((it) => (it.name + ' ' + (it.tags ?? '')).toLowerCase().includes(search.toLowerCase())),
  );
  const logs = $derived(sel ? logsOf(sel.id) : []);
  const now = $derived(sel ? nowOf(sel.id) : null);
  const scenes = $derived(orderedScenes());
  const where = (id: number) => {
    const n = nowOf(id);
    return n ? [n.holder?.name, n.place?.name].filter(Boolean).join(' · ') : '';
  };

  async function add() {
    const it = await db.save('item', { name: 'New item', type: 'relic', description: '', history: '', tags: '' });
    selId = it.id;
  }
  const save = () => { if (sel) db.queue('item', sel); };
  async function del() { if (selId) { await db.remove('item', selId); selId = null; } }

  /** A new step starts as a copy of the last one: usually only the holder or the place changes. */
  async function addStep() {
    if (!sel) return;
    const last = logs[logs.length - 1];
    await db.save('itemlog', {
      item: sel.id, order: (last?.order ?? 0) + 1,
      character: last?.character ?? null, location: last?.location ?? null,
      scene: null, when: '', note: '',
    });
  }
  const saveStep = (l: any) => db.queue('itemlog', l);
  const removeStep = (l: any) => db.remove('itemlog', l.id, false);
  async function move(l: any, dir: -1 | 1) {
    const other = logs[logs.indexOf(l) + dir];
    if (!other) return;
    const a = l.order; l.order = other.order; other.order = a;
    if (l.order === other.order) l.order += dir; // two steps that shared a number
    await db.save('itemlog', l); await db.save('itemlog', other);
  }

  $effect(() => { if (takeCommand('new')) add(); }); // the command menu
</script>

<div class="split">
  <aside class="side">
    <div class="row">
      <input placeholder="Search…" bind:value={search} />
      <button class="primary" onclick={add}>+ New</button>
    </div>
    <div class="list">
      {#each shown as it (it.id)}
        <button class="item" class:on={it.id === selId} onclick={() => (selId = it.id)}>
          {it.name}<small>{where(it.id) || it.type}</small>
        </button>
      {:else}
        <div class="empty">No items yet. Add a sword, a crown, a letter, anything that moves through your story.</div>
      {/each}
    </div>
  </aside>

  <section class="pane">
    {#if sel}
      <div class="head"><h2>{sel.name}</h2><span class="row fit" style="gap:.3rem"><CopyButton kind="item" id={sel.id} /><ConfirmButton onconfirm={del} /></span></div>
      <div class="grid2">
        <div><label>Name</label><input bind:value={sel.name} oninput={save} /></div>
        <div><label>Type</label>
          <Select bind:value={sel.type} onchange={save}>{#each ITEM_TYPES as t}<option>{t}</option>{/each}</Select>
        </div>
      </div>
      <label>Tags (comma separated)</label><input bind:value={sel.tags} oninput={save} />
      <label>Description</label>
      <textarea bind:value={sel.description} oninput={save}></textarea>
      <label>History and powers</label>
      <textarea bind:value={sel.history} oninput={save} placeholder="Who made it? What can it do? What does it cost?"></textarea>
      {#key sel.id}<CustomFields kind="item" item={sel} {save} />{/key}

      <h3 style="margin-top:1.4rem">Where it is now</h3>
      <div class="card">
        {#if now}
          {#if now.holder}Held by <button class="link" onclick={() => openItem('character', now!.holder!.id)}>{now.holder.name}</button>{:else}Nobody holds it{/if}
          {#if now.place} · at <strong>{now.place.name}</strong>{/if}
          {#if now.log.when} · {now.log.when}{/if}
        {:else}<span class="muted">Add a step below to say who has it and where it is.</span>{/if}
      </div>

      <h3 style="margin-top:1.4rem">Journey · {logs.length} step{logs.length === 1 ? '' : 's'}</h3>
      <div class="card">
        {#each logs as l, i (l.id)}
          <div class="step">
            <span class="num">{i + 1}</span>
            <div class="fields">
              <div class="row">
                <Select bind:value={l.character} onchange={() => saveStep(l)} title="Who has it">
                  <option value={null}>— nobody / unknown —</option>
                  {#each db.lists.character as c}<option value={c.id}>{c.name}</option>{/each}
                </Select>
                <Select bind:value={l.location} onchange={() => saveStep(l)} title="Where it is">
                  <option value={null}>— no place —</option>
                  {#each db.lists.location as p}<option value={p.id}>{p.name}</option>{/each}
                </Select>
              </div>
              <div class="row">
                <input placeholder="When (e.g. Year 312)" bind:value={l.when} oninput={() => saveStep(l)} />
                <Select bind:value={l.scene} onchange={() => saveStep(l)} title="Scene where this happens">
                  <option value={null}>— no scene —</option>
                  {#each scenes as s}<option value={s.id}>{sceneLabel(s)}</option>{/each}
                </Select>
              </div>
              <input placeholder="What happened (stolen, gift, lost…)" bind:value={l.note} oninput={() => saveStep(l)} />
            </div>
            <div class="acts">
              <button class="small" title="Earlier" disabled={i === 0} onclick={() => move(l, -1)}>↑</button>
              <button class="small" title="Later" disabled={i === logs.length - 1} onclick={() => move(l, 1)}>↓</button>
              <button class="small" title="Remove this step" onclick={() => removeStep(l)}>✕</button>
            </div>
          </div>
        {:else}<span class="muted">No steps yet.</span>{/each}
        <button class="primary" style="margin-top:.6rem" onclick={addStep}>+ Add a step</button>
      </div>

      <Backlinks kind="item" id={sel.id} />
    {:else}
      <div class="empty">Pick an item, or press “+ New”.</div>
    {/if}
  </section>
</div>

<style>
  .step { display: flex; gap: .6rem; align-items: flex-start; padding: .5rem 0; border-top: 1px solid var(--line); }
  .step:first-child { border-top: 0; }
  .num { flex: 0 0 1.4rem; height: 1.4rem; border-radius: 50%; background: var(--panel-2); border: 1px solid var(--line); font-size: .75rem; display: grid; place-items: center; margin-top: .3rem; }
  .fields { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: .3rem; }
  .acts { display: flex; flex-direction: column; gap: .2rem; }
  .link { background: none; border: 0; padding: 0; color: var(--accent); }
</style>
