<script lang="ts">
  import { veil, dialog } from './motion.svelte';
  import Select from './Select.svelte';
  import { db, CHARACTER_COLORS, type Item } from './db.svelte';
  import { nav, openItem } from './nav.svelte';
  import { CUSTOM, STYLES, customStyle, makeNames, type Style } from './namegen';

  const COUNT = 12;
  let kind = $state<'person' | 'place'>(nav.names === 'place' ? 'place' : 'person');
  let styleKey = $state(STYLES[0].key);
  let names = $state<string[]>([]);
  let copied = $state('');
  let words = $state('');
  let row = $state<Item | null>(null);

  const style = $derived<Style | null>(styleKey === CUSTOM ? customStyle(words) : STYLES.find((s) => s.key === styleKey) ?? null);

  // names already in the book: new names never repeat them
  const used = $derived.by(() => {
    const out = new Set<string>();
    const add = (v: unknown) => { for (const p of String(v ?? '').split(',')) if (p.trim()) out.add(p.trim().toLowerCase()); };
    for (const c of db.lists.character) { add(c.name); add(c.aliases); }
    for (const k of ['location', 'faction', 'item'] as const) for (const i of db.lists[k]) add(i.name);
    return out;
  });

  function generate() {
    names = style ? makeNames(style, kind, COUNT, used) : [];
  }

  $effect(() => {
    // the word list is saved with the other settings
    row = db.lists.setting.find((s) => s.key === 'nameWords') ?? null;
    if (row && !words) words = row.value ?? '';
  });
  async function saveWords() {
    if (row) { row.value = words; db.queue('setting', row); return; }
    row = await db.save('setting', { key: 'nameWords', value: words });
  }

  // make a first set of names when the panel opens, and again when the choices change
  $effect(() => { kind; style; generate(); });

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

  async function make(name: string) {
    db.flush();
    nav.names = '';
    if (kind === 'person') {
      const c = await db.save('character', {
        name, role: 'supporting', aliases: '', age: '', description: '',
        traits: '', backstory: '', goals: '', notes: '', fields: [],
        color: CHARACTER_COLORS[db.lists.character.length % CHARACTER_COLORS.length],
      });
      openItem('character', c.id);
    } else {
      const l = await db.save('location', { name, type: 'city', description: '', parent: null, tags: '' });
      openItem('location', l.id);
    }
  }
  const close = () => { db.flush(); nav.names = ''; };
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
    <Select id="style" bind:value={styleKey}>
      {#each STYLES as s}<option value={s.key}>{s.label}</option>{/each}
      <option value={CUSTOM}>My own words…</option>
    </Select>

    {#if styleKey === CUSTOM}
      <label for="words">Your words (at least 3, separated by spaces, commas or new lines). Names are built from them.</label>
      <textarea id="words" style="min-height:4.5rem" bind:value={words} oninput={saveWords} placeholder="frost, ember, hollow, stone, wind"></textarea>
      {#if !style}<p class="muted" style="margin:.4rem 0 0">Add at least 3 words to make names.</p>{/if}
    {/if}

    <div class="head" style="margin-top:1rem">
      <h3 style="margin:0">Names</h3>
      <button class="primary small" disabled={!style} onclick={generate}>Generate again</button>
    </div>
    <div class="grid">
      {#each names as n (n)}
        <div class="name">
          <button class="pick" title="Copy this name" onclick={() => copy(n)}>{copied === n ? 'Copied ✓' : n}</button>
          <button class="small" title={kind === 'person' ? 'Make a character with this name' : 'Make a place with this name'} onclick={() => make(n)}>＋ {kind === 'person' ? 'Character' : 'Place'}</button>
        </div>
      {:else}
        <div class="muted">No names yet.</div>
      {/each}
    </div>
    <p class="muted" style="font-size:.8rem;margin-bottom:0">
      Click a name to copy it. Names already used for a character, place, faction or item are left out. The names are made up from rules, so some sound odd: press Generate again.
    </p>
  </div>
</div>

<style>
  .box { width: min(560px, 94vw); max-height: 80vh; overflow: auto; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(15rem, 1fr)); gap: .4rem; }
  .name { display: flex; gap: .4rem; align-items: center; }
  .pick { flex: 1; text-align: left; font-size: 1.05rem; }
</style>
