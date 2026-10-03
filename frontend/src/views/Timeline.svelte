<script lang="ts">
  import { db } from '../lib/db.svelte';
  import { scenesOf, chapterCast, chapterHasPov } from '../lib/scenes';
  import ConfirmButton from '../lib/ConfirmButton.svelte';
  import { nav } from '../lib/nav.svelte';

  let mode = $state<'time' | 'lanes'>('time');
  let selId = $state<number | null>(null);
  // Open an event when the Checks tab sends you here.
  $effect(() => {
    const p = nav.pending;
    if (p?.kind !== 'event') return;
    nav.pending = null;
    mode = 'time';
    selId = p.id;
  });
  let show = $state({ event: true, chapter: true, step: true });
  let charFilter = $state<number | null>(null);

  const ev = $derived(db.get('event', selId));
  const books = $derived([...db.lists.book].sort((a, b) => a.order - b.order));
  const chapters = $derived(
    [...db.lists.chapter].sort(
      (a, b) => (db.get('book', a.book)?.order ?? 0) - (db.get('book', b.book)?.order ?? 0) || a.order - b.order,
    ),
  );

  type Row = { key: string; type: 'event' | 'chapter' | 'step'; id: number; time: number; title: string; when: string; sub: string; color: string; chars: number[]; label?: string };

  const all = $derived.by<Row[]>(() => {
    const rows: Row[] = [];
    for (const e of db.lists.event) {
      rows.push({
        key: 'e' + e.id, type: 'event', id: e.id, time: e.time, title: e.title, when: e.when ?? '',
        sub: e.description ?? '', color: 'var(--accent)', chars: (e.characters ?? []).filter((c: number) => db.get('character', c)),
      });
    }
    for (const c of chapters) {
      const sc = scenesOf(c.id);
      if (sc.length) {
        sc.forEach((s, i) => {
          const cast = new Set<number>([...(s.pov ? [s.pov] : []), ...(s.characters ?? [])]);
          rows.push({
            key: 's' + s.id, type: 'chapter', label: 'scene', id: s.id, time: s.time,
            title: `${db.name('book', c.book, 'title')} · ch. ${c.order}, scene ${i + 1}${s.title ? ': ' + s.title : ''}`,
            when: s.when ?? '', sub: s.summary ?? '', color: 'var(--accent-2)', chars: [...cast].filter((x) => db.get('character', x)),
          });
        });
        continue;
      }
      rows.push({
        key: 'c' + c.id, type: 'chapter', id: c.id, time: c.time, title: `${db.name('book', c.book, 'title')} · ch. ${c.order}: ${c.title}`,
        when: c.when ?? '', sub: c.summary ?? '', color: 'var(--accent-2)', chars: (c.characters ?? []).filter((x: number) => db.get('character', x)),
      });
    }
    for (const p of db.lists.pathpoint) {
      const who = db.get('character', p.character);
      if (!who) continue;
      rows.push({
        key: 'p' + p.id, type: 'step', id: p.id, time: p.time, title: `${who.name}: ${p.label || 'path step'}`,
        when: p.when ?? '', sub: p.location ? 'At ' + db.name('location', p.location) : '', color: who.color, chars: [who.id],
      });
    }
    return rows.filter((r) => show[r.type] && (!charFilter || r.chars.includes(charFilter)));
  });

  const placed = $derived(all.filter((r) => r.time != null).sort((a, b) => a.time - b.time));
  const unplacedEvents = $derived(all.filter((r) => r.time == null && r.type === 'event'));
  const unplacedOther = $derived(all.filter((r) => r.time == null && r.type !== 'event').length);

  async function add() {
    const last = Math.max(0, ...db.lists.event.map((e) => e.time ?? 0));
    const e = await db.save('event', { title: 'New event', time: last + 1, when: '', description: '', characters: [], location: null });
    selId = e.id;
  }
  function save() { if (ev) db.queue('event', ev); }
  function toggleChar(id: number) {
    if (!ev) return;
    const s = new Set<number>(ev.characters ?? []);
    s.has(id) ? s.delete(id) : s.add(id);
    ev.characters = [...s]; save();
  }
  async function del() { if (selId) { await db.remove('event', selId); selId = null; } }

  function cell(c: any, ch: any) {
    const steps = db.lists.pathpoint.filter((p) => p.character === c.id && p.chapter === ch.id).length;
    return { pov: chapterHasPov(ch, c.id), present: chapterCast(ch).includes(c.id), steps };
  }
</script>

<div class="split">
  <aside class="side">
    <div class="tabs">
      <button class:active={mode === 'time'} onclick={() => (mode = 'time')}>Story time</button>
      <button class:active={mode === 'lanes'} onclick={() => (mode = 'lanes')}>Who’s where</button>
    </div>

    {#if mode === 'time'}
      <button class="primary" style="width:100%" onclick={add}>+ New event</button>
      <h3 style="margin-top:1rem">Show</h3>
      {#each [['event', 'Events'], ['chapter', 'Chapters and scenes'], ['step', 'Path steps']] as [k, label]}
        <label style="display:flex;gap:.4rem;margin:.2rem 0;color:var(--text)">
          <input type="checkbox" style="width:auto" bind:checked={show[k as 'event']} />{label}
        </label>
      {/each}
      <label>Only this character</label>
      <select bind:value={charFilter}>
        <option value={null}>Everyone</option>
        {#each db.lists.character as c}<option value={c.id}>{c.name}</option>{/each}
      </select>

      {#if ev}
        <div class="card">
          <div class="head"><strong>Edit event</strong><ConfirmButton onconfirm={del} /></div>
          <label>Title</label><input bind:value={ev.title} oninput={save} />
          <div class="row">
            <div><label>Position (number)</label><input type="number" step="any" bind:value={ev.time} oninput={save} /></div>
            <div><label>Label</label><input bind:value={ev.when} oninput={save} placeholder="Year 312" /></div>
          </div>
          <label>Place</label>
          <select bind:value={ev.location} onchange={save}>
            <option value={null}>— none —</option>
            {#each db.lists.location as l}<option value={l.id}>{l.name}</option>{/each}
          </select>
          <label>Who is involved</label>
          {#each db.lists.character as c}
            <label style="display:flex;gap:.4rem;margin:.1rem 0;color:var(--text)">
              <input type="checkbox" style="width:auto" checked={(ev.characters ?? []).includes(c.id)} onchange={() => toggleChar(c.id)} />
              <span class="dot" style:background={c.color}></span>{c.name}
            </label>
          {/each}
          <label>What happened</label><textarea bind:value={ev.description} oninput={save}></textarea>
        </div>
      {/if}
    {:else}
      <p class="muted">Each row is a character. Each column is a chapter, in reading order.</p>
      <p class="muted">◉ point of view · ● in the chapter · number = map path steps</p>
    {/if}
  </aside>

  <section class="pane">
    {#if mode === 'time'}
      {#if placed.length === 0}
        <div class="empty">Nothing on the timeline yet. Add an event, or give a chapter or path step a “Timeline position”.</div>
      {/if}
      <div style="position:relative;padding-left:2rem;border-left:2px solid var(--line);margin-left:6.5rem">
        {#each placed as r (r.key)}
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <div class="card" role="button" tabindex="0" style="position:relative;cursor:{r.type === 'event' ? 'pointer' : 'default'};border-left:4px solid {r.color};outline:{r.type === 'event' && r.id === selId ? '1px solid var(--accent)' : 'none'}"
            onclick={() => { if (r.type === 'event') selId = r.id; }}>
            <span style="position:absolute;left:calc(-2rem - 7px - 6.5rem);width:5.5rem;text-align:right;top:.8rem;font-size:.8rem;color:var(--muted)">
              {r.when || r.time}
            </span>
            <span class="dot" style="position:absolute;left:calc(-2rem - 7px + 1px);top:1rem;background:{r.color}"></span>
            <div class="head" style="margin:0"><strong>{r.title}</strong><span class="chip" style="margin:0">{r.label ?? (r.type === 'step' ? 'path step' : r.type)}</span></div>
            {#if r.sub}<div class="muted" style="margin-top:.3rem">{r.sub}</div>{/if}
            <div style="margin-top:.4rem">
              {#each r.chars as id}<span class="chip"><span class="dot" style="display:inline-block;background:{db.get('character', id)?.color}"></span> {db.name('character', id)}</span>{/each}
            </div>
          </div>
        {/each}
      </div>
      {#if unplacedEvents.length}
        <h3 style="margin-top:1.5rem">Events without a position</h3>
        {#each unplacedEvents as r}<button class="small" style="margin:0 .3rem .3rem 0" onclick={() => (selId = r.id)}>{r.title}</button>{/each}
      {/if}
      {#if unplacedOther}
        <p class="muted" style="margin-top:1rem">{unplacedOther} chapter/path step{unplacedOther === 1 ? '' : 's'} have no “Timeline position”, so they are hidden here.</p>
      {/if}
    {:else}
      {#if chapters.length === 0 || db.lists.character.length === 0}
        <div class="empty">Add characters and chapters first.</div>
      {:else}
        <div style="overflow:auto;max-height:100%">
          <table style="border-collapse:collapse">
            <thead>
              <tr>
                <th style="position:sticky;left:0;background:var(--bg);min-width:140px"></th>
                {#each chapters as ch}
                  <th style="min-width:96px;max-width:96px;padding:.4rem;font-size:.75rem;font-weight:500;vertical-align:bottom;border-bottom:1px solid var(--line)">
                    <div class="muted">{db.name('book', ch.book, 'title')}</div>
                    {ch.order}. {ch.title}
                    {#if scenesOf(ch.id)[0]?.when || ch.when}<div class="muted">{scenesOf(ch.id)[0]?.when || ch.when}</div>{/if}
                  </th>
                {/each}
              </tr>
            </thead>
            <tbody>
              {#each db.lists.character as c (c.id)}
                <tr>
                  <td style="position:sticky;left:0;background:var(--bg);padding:.4rem .6rem;white-space:nowrap">
                    <span class="dot" style:background={c.color} style="display:inline-block;margin-right:.4rem"></span>{c.name}
                  </td>
                  {#each chapters as ch}
                    {@const x = cell(c, ch)}
                    <td style="text-align:center;padding:.4rem;border-bottom:1px solid var(--line);background:{x.pov || x.present || x.steps ? 'color-mix(in srgb, ' + c.color + ' 14%, transparent)' : 'transparent'}"
                      title="{c.name} in “{ch.title}”">
                      {#if x.pov}<span style="color:{c.color};font-size:1.1rem">◉</span>
                      {:else if x.present}<span style="color:{c.color}">●</span>{/if}
                      {#if x.steps}<sup style="color:var(--muted)">{x.steps}</sup>{/if}
                    </td>
                  {/each}
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    {/if}
  </section>
</div>
