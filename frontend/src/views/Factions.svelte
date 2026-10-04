<script lang="ts">
  import Select from '../lib/Select.svelte';
  import { db, FACTION_TYPES, FACTION_REL } from '../lib/db.svelte';
  import { nav, openItem } from '../lib/nav.svelte';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import Backlinks from '../lib/Backlinks.svelte';

  const PALETTE = ['#d98a5b', '#8f7bf0', '#6bbf8a', '#5aa9d9', '#e0b25f', '#e0689f', '#2cd5e8', '#d9625b', '#b5c95b'];

  let selId = $state<number | null>(null);
  let search = $state('');
  let newMember = $state<number | null>(null);
  let newRel = $state('ally');
  let newOther = $state<number | null>(null);

  // Jump here when a search result or an @ link is opened.
  $effect(() => {
    const p = nav.pending;
    if (p?.kind === 'faction') { search = ''; selId = p.id; nav.pending = null; }
  });

  const sel = $derived(db.get('faction', selId));
  const shown = $derived(db.lists.faction.filter((f) => String(f.name).toLowerCase().includes(search.toLowerCase())));
  const ranks = $derived(String(sel?.ranks ?? '').split('\n').map((r) => r.trim()).filter(Boolean));
  const members = $derived(
    db.lists.membership
      .filter((m) => m.faction === selId && db.get('character', m.character))
      .sort((a, b) => {
        const ia = ranks.indexOf(a.rank), ib = ranks.indexOf(b.rank);
        return (ia < 0 ? 999 : ia) - (ib < 0 ? 999 : ib);
      }),
  );
  const candidates = $derived(db.lists.character.filter((c) => !members.some((m) => m.character === c.id)));
  const ties = $derived(db.lists.factionrel.filter((r) => (r.a === selId || r.b === selId) && db.get('faction', r.a) && db.get('faction', r.b)));
  const others = $derived(db.lists.faction.filter((f) => f.id !== selId));
  const count = (id: number) => db.lists.membership.filter((m) => m.faction === id).length;

  async function add() {
    const f = await db.save('faction', {
      name: 'New faction', type: 'guild', description: '', ranks: '', hq: null,
      color: PALETTE[db.lists.faction.length % PALETTE.length],
    });
    selId = f.id;
  }
  const save = () => { if (sel) db.queue('faction', sel); };
  async function del() { if (selId) { await db.remove('faction', selId); selId = null; } }

  async function addMember() {
    if (!sel || !newMember) return;
    await db.save('membership', { faction: sel.id, character: newMember, rank: ranks[ranks.length - 1] ?? '' });
    newMember = null;
  }
  const saveMember = (m: any) => db.queue('membership', m);
  const removeMember = (m: any) => db.remove('membership', m.id, false);

  async function addTie() {
    if (!sel || !newOther) return;
    await db.save('factionrel', { a: sel.id, b: newOther, type: newRel });
    newOther = null;
  }
  const saveTie = (r: any) => db.save('factionrel', r);
  const removeTie = (r: any) => db.remove('factionrel', r.id, false);
</script>

<div class="split">
  <aside class="side">
    <div class="row">
      <input placeholder="Search…" bind:value={search} />
      <button class="primary" onclick={add}>+ New</button>
    </div>
    <div class="list">
      {#each shown as f (f.id)}
        <button class="item" class:on={f.id === selId} onclick={() => (selId = f.id)}>
          <span class="dot" style:background={f.color}></span>{f.name}<small>{count(f.id)} member{count(f.id) === 1 ? '' : 's'}</small>
        </button>
      {:else}
        <div class="empty">No factions yet. A faction is a kingdom, guild, order, house, army, or any group.</div>
      {/each}
    </div>
  </aside>

  <section class="pane">
    {#if sel}
      <div class="head"><h2>{sel.name}</h2><ConfirmButton onconfirm={del} /></div>
      <div class="grid2">
        <div><label>Name</label><input bind:value={sel.name} oninput={save} /></div>
        <div><label>Type</label>
          <Select bind:value={sel.type} onchange={save}>{#each FACTION_TYPES as t}<option>{t}</option>{/each}</Select>
        </div>
        <div><label>Headquarters</label>
          <Select bind:value={sel.hq} onchange={save}>
            <option value={null}>— none —</option>
            {#each db.lists.location as l}<option value={l.id}>{l.name}</option>{/each}
          </Select>
        </div>
        <div><label>Colour</label><input type="color" bind:value={sel.color} oninput={save} /></div>
      </div>
      <label>Ranks (one per line, highest first)</label>
      <textarea style="min-height:6rem" bind:value={sel.ranks} oninput={save} placeholder={'King\nLord\nKnight\nSoldier'}></textarea>
      <label>Description</label>
      <textarea bind:value={sel.description} oninput={save}></textarea>

      <h3 style="margin-top:1.4rem">Members · {members.length}</h3>
      <div class="card">
        <datalist id="ranks-{sel.id}">{#each ranks as r}<option value={r}></option>{/each}</datalist>
        {#each members as m (m.id)}
          <div class="row line">
            <button class="link" onclick={() => openItem('character', m.character)}>
              <span class="dot" style:background={db.get('character', m.character)?.color}></span> {db.name('character', m.character)}
            </button>
            <input list="ranks-{sel.id}" placeholder="Rank" bind:value={m.rank} oninput={() => saveMember(m)} />
            <button class="small" title="Remove from this faction" onclick={() => removeMember(m)}>✕</button>
          </div>
        {:else}<span class="muted">No members yet.</span>{/each}
        <div class="row" style="margin-top:.6rem">
          <Select bind:value={newMember}>
            <option value={null}>Add a character…</option>
            {#each candidates as c}<option value={c.id}>{c.name}</option>{/each}
          </Select>
          <button class="primary" disabled={!newMember} onclick={addMember}>Add</button>
        </div>
      </div>

      <h3 style="margin-top:1.4rem">Allies and enemies</h3>
      <div class="card">
        {#each ties as r (r.id)}
          {@const other = r.a === sel.id ? r.b : r.a}
          <div class="row line">
            <Select bind:value={r.type} onchange={() => saveTie(r)} style="border-color: {FACTION_REL[r.type]}">
              {#each Object.keys(FACTION_REL) as t}<option>{t}</option>{/each}
            </Select>
            <button class="link" onclick={() => (selId = other)}>{db.name('faction', other)}</button>
            {#if r.type === 'rules'}<span class="muted fit">{r.a === sel.id ? 'rules them' : 'rules us'}</span>{/if}
            <button class="small" onclick={() => removeTie(r)}>✕</button>
          </div>
        {:else}<span class="muted">No ties to other factions yet.</span>{/each}
        {#if others.length}
          <div class="row" style="margin-top:.6rem">
            <Select bind:value={newRel}>{#each Object.keys(FACTION_REL) as t}<option>{t}</option>{/each}</Select>
            <Select bind:value={newOther}>
              <option value={null}>Faction…</option>
              {#each others as f}<option value={f.id}>{f.name}</option>{/each}
            </Select>
            <button class="primary" disabled={!newOther} onclick={addTie}>Add</button>
          </div>
          <p class="muted" style="margin:.4rem 0 0">“rules” means this faction rules the other one.</p>
        {/if}
      </div>

      <Backlinks kind="faction" id={sel.id} />
    {:else}
      <div class="empty">Pick a faction, or press “+ New”.</div>
    {/if}
  </section>
</div>

<style>
  .line { margin: .25rem 0; }
  .link { background: none; border: 0; padding: 0; color: var(--accent); text-align: left; flex: 1; }
</style>
