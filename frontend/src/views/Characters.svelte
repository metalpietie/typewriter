<script lang="ts">
  import Select from '../lib/Select.svelte';
  import { db, ROLES, REL_TYPES, CHARACTER_COLORS as PALETTE } from '../lib/db.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import Backlinks from '../lib/Backlinks.svelte';
  import MoodBoard from '../lib/MoodBoard.svelte';
  import Picture from '../lib/Picture.svelte';
  import DatePick from '../lib/DatePick.svelte';
  import { ageAt, momentsOf } from '../lib/ages';
  import { nav, openItem, openScene } from '../lib/nav.svelte';
  import { scenesOf, sceneLabel } from '../lib/scenes';
  import { itemsHeldBy, membershipsOf } from '../lib/items';

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
  // the age tool: pick any date to see how old this character is then
  let probe = $state<number | null>(null);
  const probeAge = $derived(ageAt(sel, probe));
  const moments = $derived(selId ? momentsOf(selId) : []);
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
    <button class="small" style="width:100%;margin-top:.4rem" onclick={() => (nav.names = 'person')}>Name generator…</button>
    <div class="list">
      {#each shown as c (c.id)}
        <button class="item" class:on={c.id === selId} onclick={() => (selId = c.id)}>
          {#if c.portrait}<img class="thumb" src={c.portrait} alt="" style:border-color={c.color} />{:else}<span class="dot" style:background={c.color}></span>{/if}{c.name}<small>{c.role}</small>
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

      <Picture square label="Portrait" value={sel.portrait} onchange={(v) => { sel.portrait = v; save(); }} />

      <div class="grid2">
        <div><label>Name</label><input bind:value={sel.name} oninput={save} /></div>
        <div><label>Also called</label><input bind:value={sel.aliases} oninput={save} placeholder="nicknames, titles" /></div>
        <div><label>Role</label>
          <Select bind:value={sel.role} onchange={save}>{#each ROLES as r}<option>{r}</option>{/each}</Select>
        </div>
        <div><label>Age (a note, optional)</label><input bind:value={sel.age} oninput={save} placeholder="e.g. about thirty" /></div>
        <div><label>Colour (used on graph and maps)</label><input type="color" bind:value={sel.color} oninput={save} /></div>
      </div>

      <div class="grid2">
        <DatePick label="Born" bind:value={sel.born} onchange={save} hint="Set a birth date to see ages in your scenes." />
        <DatePick label="Died (leave empty if alive)" bind:value={sel.died} onchange={save} />
      </div>

      {#if sel.born !== null && sel.born !== undefined && sel.born !== ''}
        <div class="card ages">
          <strong>Age on a date</strong>
          <DatePick label="Pick a date" bind:value={probe} onchange={() => {}} hint="Pick any day in your story." />
          {#if probeAge}
            <div class="result" class:bad={probeAge.state !== 'alive'}>{probeAge.state === 'alive' ? `${sel.name} is ${probeAge.years} years old.` : probeAge.state === 'unborn' ? `${sel.name} is not born yet.` : `${sel.name} is dead. ${sel.name} died aged ${probeAge.years}.`}</div>
          {/if}
          {#if moments.length}
            <div class="muted" style="margin-top:.8rem">Age in your story</div>
            {#each moments as m (m.key)}
              <button class="moment" onclick={() => (m.open.kind === 'scene' ? openScene(m.open.id) : openItem('event', m.open.id))}>
                <span class="age" class:bad={m.age.state !== 'alive'}>{m.age.state === 'alive' ? m.age.years : m.age.state === 'unborn' ? '—' : '†'}</span>
                <span class="what">{m.what}</span>
                <span class="when muted">{m.when}</span>
              </button>
            {/each}
          {/if}
        </div>
      {/if}

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
      {#key sel.id}<MoodBoard owner={sel.id} />{/key}
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

<style>
  .result { margin-top: .6rem; font-weight: 600; color: var(--accent); }
  .result.bad { color: var(--danger); }
  .moment { display: flex; gap: .6rem; align-items: baseline; width: 100%; text-align: left; background: none; border: 0; border-top: 1px solid var(--line); border-radius: 0; padding: .35rem 0; }
  .moment:hover { background: var(--panel-2); }
  .moment .age { flex: 0 0 2rem; text-align: right; font-weight: 700; color: var(--accent); }
  .moment .age.bad { color: var(--danger); }
  .moment .what { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .moment .when { flex: 0 0 auto; font-size: .8rem; }
</style>
