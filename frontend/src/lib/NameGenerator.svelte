<script lang="ts">
  import { veil, dialog } from './motion.svelte';
  import Select from './Select.svelte';
  import { db, CHARACTER_COLORS, type Item } from './db.svelte';
  import { nav, openItem } from './nav.svelte';
  import {
    CULTURES, CUSTOM, FORMS, GENDERS, LENGTHS, PLACE_KINDS, customCulture, lettersOf, makeNames,
    type Culture, type Form, type Gender, type Length, type Made, type PlaceKind,
  } from './namegen';

  const COUNT = 12;
  const PREFS = 'typewriter-namegen';

  // the choices you made last time, kept on this computer
  type Prefs = { culture: string; gender: Gender; form: Form; length: Length; letter: string; place: PlaceKind };
  const pickOf = <T extends string>(v: unknown, allowed: readonly (readonly [T, string])[], fallback: T): T => (allowed.some(([k]) => k === v) ? (v as T) : fallback);
  function loadPrefs(): Prefs {
    let raw: any = null;
    try { raw = JSON.parse(localStorage.getItem(PREFS) ?? 'null'); } catch { /* storage may be blocked */ }
    const known = (k: unknown) => k === CUSTOM || CULTURES.some((c) => c.key === k);
    return {
      culture: known(raw?.culture) ? raw.culture : CULTURES[0].key,
      gender: pickOf(raw?.gender, GENDERS, 'any'), form: pickOf(raw?.form, FORMS, 'first'), length: pickOf(raw?.length, LENGTHS, 'any'),
      letter: typeof raw?.letter === 'string' && /^[A-Za-z]?$/.test(raw.letter) ? raw.letter : '', place: pickOf(raw?.place, PLACE_KINDS, 'settlement'),
    };
  }
  const start = loadPrefs();

  let kind = $state<'person' | 'place'>(nav.names === 'place' ? 'place' : 'person');
  let cultureKey = $state(start.culture);
  let gender = $state<Gender>(start.gender);
  let form = $state<Form>(start.form);
  let length = $state<Length>(start.length);
  let letter = $state(start.letter);
  let placeKind = $state<PlaceKind>(start.place);
  let names = $state<Made[]>([]);
  let copied = $state('');
  let words = $state('');
  let row = $state<Item | null>(null);
  let favRow = $state<Item | null>(null);

  const culture = $derived<Culture | null>(cultureKey === CUSTOM ? customCulture(words) : CULTURES.find((c) => c.key === cultureKey) ?? null);
  const letters = $derived(culture ? lettersOf(culture) : []);
  // a letter that this style does not have is dropped
  $effect(() => { if (letter && culture && !letters.includes(letter.toUpperCase())) letter = ''; });
  $effect(() => {
    const p: Prefs = { culture: cultureKey, gender, form, length, letter, place: placeKind };
    try { localStorage.setItem(PREFS, JSON.stringify(p)); } catch { /* it lasts until the app closes */ }
  });

  // names already in the book: new names never repeat them. A first name that is taken is left out too.
  const used = $derived.by(() => {
    const out = new Set<string>();
    const add = (v: unknown, first = false) => {
      for (const p of String(v ?? '').split(',')) {
        const t = p.trim().toLowerCase();
        if (!t) continue;
        out.add(t);
        if (first) out.add(t.split(' ')[0]);
      }
    };
    for (const c of db.lists.character) { add(c.name, true); add(c.aliases); }
    for (const k of ['location', 'faction', 'item'] as const) for (const i of db.lists[k]) add(i.name);
    return out;
  });

  function generate() {
    names = culture ? makeNames({ kind, culture, gender, form, length: kind === 'person' ? length : 'any', letter, place: placeKind }, COUNT, used) : [];
  }

  // the word list and the favourites are saved with the other settings
  $effect(() => {
    row = db.lists.setting.find((s) => s.key === 'nameWords') ?? null;
    if (row && !words) words = row.value ?? '';
    favRow = db.lists.setting.find((s) => s.key === 'nameFavs') ?? null;
  });
  async function saveWords() {
    if (row) { row.value = words; db.queue('setting', row); return; }
    row = await db.save('setting', { key: 'nameWords', value: words });
  }

  type Fav = { name: string; kind: 'person' | 'place'; gender: string };
  const favs = $derived<Fav[]>(Array.isArray(favRow?.value) ? favRow!.value.filter((f: any) => f && typeof f.name === 'string') : []);
  const isFav = (name: string) => favs.some((f) => f.name === name && f.kind === kind);
  async function saveFavs(list: Fav[]) {
    favRow = await db.save('setting', { ...(favRow ?? {}), key: 'nameFavs', value: list });
  }
  const toggleFav = (m: Made) => saveFavs(isFav(m.name) ? favs.filter((f) => !(f.name === m.name && f.kind === kind)) : [...favs, { name: m.name, kind, gender: m.gender }]);
  const dropFav = (f: Fav) => saveFavs(favs.filter((x) => !(x.name === f.name && x.kind === f.kind)));

  // make a first set of names when the panel opens, and again when a choice changes
  $effect(() => { kind; culture; gender; form; length; letter; placeKind; generate(); });

  async function copy(name: string) {
    try {
      await navigator.clipboard.writeText(name);
    } catch {
      const { Clipboard } = await import('@wailsio/runtime');
      await Clipboard.SetText(name);
    }
    copied = name;
    setTimeout(() => { if (copied === name) copied = ''; }, 1500);
  }

  async function make(name: string, as: 'person' | 'place', g: string) {
    db.flush();
    nav.names = '';
    if (as === 'person') {
      const c = await db.save('character', {
        name, role: 'supporting', aliases: '', age: '', description: '',
        traits: '', backstory: '', goals: '', notes: '', fields: g ? [{ k: 'Gender', v: g === 'female' ? 'Female' : 'Male' }] : [],
        color: CHARACTER_COLORS[db.lists.character.length % CHARACTER_COLORS.length],
      });
      openItem('character', c.id);
    } else {
      const l = await db.save('location', { name, type: 'city', description: '', parent: null, tags: '' });
      openItem('location', l.id);
    }
  }
  const close = () => { db.flush(); nav.names = ''; };
  const mark = (g: string) => (g === 'female' ? '♀' : g === 'male' ? '♂' : '');
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') close(); }} />

<div class="overlay" transition:veil|global role="presentation" onmousedown={(e) => { if (e.target === e.currentTarget) close(); }}>
  <div transition:dialog|global class="palette box" role="dialog" aria-label="Name generator">
    <div class="head"><h2 style="margin:0">Name generator</h2><button class="small" onclick={close}>Close</button></div>

    <div class="tabs">
      <button class:active={kind === 'person'} onclick={() => (kind = 'person')}>People</button>
      <button class:active={kind === 'place'} onclick={() => (kind = 'place')}>Places</button>
    </div>

    <label for="style">Style</label>
    <Select id="style" bind:value={cultureKey}>
      {#each CULTURES as c}<option value={c.key}>{c.label}</option>{/each}
      <option value={CUSTOM}>My own words…</option>
    </Select>

    {#if cultureKey === CUSTOM}
      <label for="words">Your words (at least 3, separated by spaces, commas or new lines). Names are built from them.</label>
      <textarea id="words" style="min-height:4.5rem" bind:value={words} oninput={saveWords} placeholder="frost, ember, hollow, stone, wind"></textarea>
      {#if !culture}<p class="muted" style="margin:.4rem 0 0">Add at least 3 words to make names.</p>{/if}
    {/if}

    <div class="opts">
      {#if kind === 'person'}
        {#if culture?.gendered}
          <div><label for="gender">Gender</label>
            <Select id="gender" bind:value={gender}>{#each GENDERS as [k, name]}<option value={k}>{name}</option>{/each}</Select></div>
        {/if}
        <div><label for="form">Name</label>
          <Select id="form" bind:value={form}>{#each FORMS as [k, name]}<option value={k}>{name}</option>{/each}</Select></div>
        <div><label for="length">First name</label>
          <Select id="length" bind:value={length}>{#each LENGTHS as [k, name]}<option value={k}>{name}</option>{/each}</Select></div>
      {:else}
        <div><label for="pkind">Kind of place</label>
          <Select id="pkind" bind:value={placeKind}>{#each PLACE_KINDS as [k, name]}<option value={k}>{name}</option>{/each}</Select></div>
      {/if}
      <div><label for="letter">Starts with</label>
        <Select id="letter" bind:value={letter}><option value="">Any letter</option>{#each letters as l}<option value={l}>{l}</option>{/each}</Select></div>
    </div>

    <div class="head" style="margin-top:1rem">
      <h3 style="margin:0">Names</h3>
      <button class="primary small" disabled={!culture} onclick={generate}>Generate again</button>
    </div>
    <div class="grid">
      {#each names as m (m.name)}
        <div class="name">
          <button class="pick" title="Copy this name" onclick={() => copy(m.name)}>
            {#if mark(m.gender)}<span class="g" aria-label={m.gender}>{mark(m.gender)}</span>{/if}{copied === m.name ? 'Copied ✓' : m.name}
          </button>
          <button class="small star" class:on={isFav(m.name)} title={isFav(m.name) ? 'Take out of your favourites' : 'Keep this name in your favourites'} aria-pressed={isFav(m.name)} onclick={() => toggleFav(m)}>{isFav(m.name) ? '★' : '☆'}</button>
          <button class="small" title={kind === 'person' ? 'Make a character with this name' : 'Make a place with this name'} onclick={() => make(m.name, kind, m.gender)}>＋ {kind === 'person' ? 'Character' : 'Place'}</button>
        </div>
      {:else}
        <div class="muted">{letter ? `No names of this style start with “${letter}”. Pick another letter.` : culture ? 'No names fit these choices. Try other choices.' : 'No names yet.'}</div>
      {/each}
    </div>

    {#if favs.length}
      <details class="favs">
        <summary>Favourites · {favs.length}</summary>
        {#each favs as f (f.kind + f.name)}
          <div class="name">
            <button class="pick" title="Copy this name" onclick={() => copy(f.name)}>
              {#if mark(f.gender)}<span class="g" aria-label={f.gender}>{mark(f.gender)}</span>{/if}{copied === f.name ? 'Copied ✓' : f.name}<small class="muted"> {f.kind === 'person' ? 'person' : 'place'}</small>
            </button>
            <button class="small" title={f.kind === 'person' ? 'Make a character with this name' : 'Make a place with this name'} onclick={() => make(f.name, f.kind, f.gender)}>＋ {f.kind === 'person' ? 'Character' : 'Place'}</button>
            <button class="small" title="Take out of your favourites" aria-label="Remove {f.name} from favourites" onclick={() => dropFav(f)}>✕</button>
          </div>
        {/each}
      </details>
    {/if}

    <p class="muted" style="font-size:.8rem;margin-bottom:0">
      Click a name to copy it. Names already used for a character, place, faction or item are left out. The names are made up from rules, so some sound odd: press Generate again.
    </p>
  </div>
</div>

<style>
  .box { width: min(660px, 94vw); max-height: 82vh; overflow: auto; }
  .opts { display: grid; grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr)); gap: 0 .8rem; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(16rem, 1fr)); gap: .4rem; }
  .name { display: flex; gap: .3rem; align-items: center; }
  .pick { flex: 1; text-align: left; font-size: 1.05rem; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .g { display: inline-block; width: 1.2rem; color: var(--muted); font-size: .9rem; }
  .star.on { color: var(--accent); }
  .favs { margin-top: 1rem; }
  .favs summary { cursor: pointer; font-weight: 600; margin-bottom: .4rem; }
  .favs .name { margin: .2rem 0; }
</style>
