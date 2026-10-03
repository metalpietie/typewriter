<script lang="ts">
  import { db, ROLES, REL_TYPES } from '../lib/db.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import Backlinks from '../lib/Backlinks.svelte';
  import { nav, openItem } from '../lib/nav.svelte';
  import { scenesOf, sceneLabel } from '../lib/scenes';
  import { itemsHeldBy, membershipsOf } from '../lib/items';

  const PALETTE = ['#2cd5e8', '#e0b25f', '#6bbf8a', '#5aa9d9', '#e0689f', '#8f6bd1', '#d98a5b', '#d9625b', '#b5c95b'];

  let selId = $state<number | null>(null);
  // Jump here when a search result is picked.
  $effect(() => {
    const p = nav.pending;
    if (p?.kind === 'character') { search = ''; selId = p.id; nav.pending = null; }
  });
  let search = $state('');

  const sel = $derived(db.get('character', selId));
  const shown = $derived(
    db.lists.character.filter((c) =>
      (c.name + ' ' + (c.aliases ?? '')).toLowerCase().includes(search.toLowerCase())),
  );
  const rels = $derived(
    db.lists.relationship.filter((r) => r.a === selId || r.b === selId),
  );
  // Scenes the character is in, plus older chapters that have no scenes yet.
  const scenes = $derived([
    ...db.lists.scene
      .filter((s) => s.pov === selId || (s.characters ?? []).includes(selId))
      .map((s) => ({ id: 's' + s.id, title: sceneLabel(s) })),
    ...db.lists.chapter
      .filter((c) => !scenesOf(c.id).length && (c.pov === selId || (c.characters ?? []).includes(selId)))
      .map((c) => ({ id: 'c' + c.id, title: c.title || 'Untitled' })),
  ]);
  const pathCount = $derived(db.lists.pathpoint.filter((p) => p.character === selId).length);

  async function add() {
    const c = await db.save('character', {
      name: 'New character', role: 'supporting', aliases: '', age: '', description: '',
      traits: '', backstory: '', goals: '', notes: '', fields: [],
      color: PALETTE[db.lists.character.length % PALETTE.length],
    });
    selId = c.id;
  }

  function save() { if (sel) db.queue('character', sel); }
  function addField() { if (sel) { sel.fields = [...(sel.fields ?? []), { k: '', v: '' }]; save(); } }
  function dropField(i: number) { if (sel) { sel.fields = sel.fields.filter((_: any, n: number) => n !== i); save(); } }
  async function del() { if (selId) { await db.remove('character', selId); selId = null; } }
</script>

<div class="split">
  <aside class="side">
    <div class="row">
      <input placeholder="Search…" bind:value={search} />
      <button class="primary" onclick={add}>+ New</button>
    </div>
    <div class="list">
      {#each shown as c (c.id)}
        <button class="item" class:on={c.id === selId} onclick={() => (selId = c.id)}>
          <span class="dot" style:background={c.color}></span>{c.name}<small>{c.role}</small>
        </button>
      {:else}
        <div class="empty">No characters yet.</div>
      {/each}
    </div>
  </aside>

  <section class="pane">
    {#if sel}
      <div class="head">
        <h2>{sel.name}</h2>
        <ConfirmButton onconfirm={del} />
      </div>

      <div class="grid2">
        <div><label>Name</label><input bind:value={sel.name} oninput={save} /></div>
        <div><label>Also called</label><input bind:value={sel.aliases} oninput={save} placeholder="nicknames, titles" /></div>
        <div><label>Role</label>
          <select bind:value={sel.role} onchange={save}>{#each ROLES as r}<option>{r}</option>{/each}</select>
        </div>
        <div><label>Age / born / died</label><input bind:value={sel.age} oninput={save} /></div>
        <div><label>Colour (used on graph and maps)</label><input type="color" bind:value={sel.color} oninput={save} /></div>
      </div>

      <label>Looks & description</label>
      <textarea bind:value={sel.description} oninput={save}></textarea>
      <label>Personality traits</label>
      <textarea bind:value={sel.traits} oninput={save}></textarea>
      <label>Backstory</label>
      <textarea bind:value={sel.backstory} oninput={save}></textarea>
      <label>Goals & fears</label>
      <textarea bind:value={sel.goals} oninput={save}></textarea>
      <label>Notes</label>
      <textarea bind:value={sel.notes} oninput={save}></textarea>

      <h3 style="margin-top:1.2rem">Custom fields</h3>
      {#each sel.fields ?? [] as f, i}
        <div class="row" style="margin-bottom:.3rem">
          <input placeholder="Field (e.g. Eye colour)" bind:value={f.k} oninput={save} />
          <input placeholder="Value" bind:value={f.v} oninput={save} />
          <button class="small" onclick={() => dropField(i)}>✕</button>
        </div>
      {/each}
      <button class="small" onclick={addField}>+ Add field</button>

      <h3 style="margin-top:1.4rem">Connections</h3>
      <div class="card">
        {#each rels as r}
          {@const other = r.a === selId ? r.b : r.a}
          <span class="chip" style:border-color={REL_TYPES[r.type]}>
            {r.type}: {db.name('character', other)}
          </span>
        {:else}<span class="muted">No relationships yet. Add them in the Relationships tab.</span>{/each}
      </div>
      {@const inFactions = membershipsOf(sel.id)}
      {@const held = itemsHeldBy(sel.id)}
      {#if inFactions.length || held.length}
        <div class="card">
          {#if inFactions.length}
            <div><strong>Factions</strong>
              {#each inFactions as m}<button class="chip" onclick={() => openItem('faction', m.faction.id)}><span class="dot" style="display:inline-block;background:{m.faction.color}"></span> {m.faction.name}{m.rank ? ' · ' + m.rank : ''}</button>{/each}
            </div>
          {/if}
          {#if held.length}
            <div style="margin-top:.4rem"><strong>Holds</strong>
              {#each held as it}<button class="chip" onclick={() => openItem('item', it.id)}>{it.name}</button>{/each}
            </div>
          {/if}
        </div>
      {/if}
      <Backlinks kind="character" id={sel.id} />
      <div class="card">
        <strong>{scenes.length}</strong> scene{scenes.length === 1 ? '' : 's'} ·
        <strong>{pathCount}</strong> map path point{pathCount === 1 ? '' : 's'}
        <div style="margin-top:.5rem">
          {#each scenes as c (c.id)}<span class="chip">{c.title}</span>{/each}
        </div>
      </div>
    {:else}
      <div class="empty">Pick a character, or press “+ New”.</div>
    {/if}
  </section>
</div>
