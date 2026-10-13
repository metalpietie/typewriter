<script lang="ts">
  import ConfirmButton from './ConfirmButton.svelte';
  import { word } from './projecttype.svelte';
  import { shrinkPicture } from './picture';
  import { listMood, addMood, saveMood, deleteMood, type Mood } from './mood';
  import { MAX_CARDS, MAX_TEXT, whyNot } from './mood-core';

  // Pictures and short notes for one character, place or scene. Drop pictures here, paste them, or choose files.
  let { owner }: { owner: number } = $props();

  let cards = $state<Mood[]>([]);
  let open = $state(false);
  let busy = $state(false);
  let message = $state('');
  let big = $state<Mood | null>(null);
  let over = $state(false);
  let fileBox: HTMLInputElement;
  let seen = 0; // which owner the list belongs to, so a slow answer for an old owner is ignored

  $effect(() => {
    const mine = ++seen, id = owner;
    cards = []; open = false; message = '';
    listMood(id).then((l) => { if (mine === seen) { cards = l; open = l.length > 0; } }).catch((e) => { if (mine === seen) message = String(e?.message ?? e); });
  });

  async function addFiles(files: Iterable<File>) {
    busy = true; message = '';
    try {
      for (const f of files) {
        if (cards.length >= MAX_CARDS) { message = `A board holds ${MAX_CARDS} cards. Delete one first.`; break; }
        const why = whyNot(f);
        if (why) { message = why; continue; }
        try { cards = [...cards, await addMood(owner, cards, { image: await shrinkPicture(f, { size: 900 }) })]; }
        catch { message = `${f.name} could not be read as a picture.`; }
      }
      open = true;
    } finally { busy = false; }
  }
  async function addNote() {
    if (cards.length >= MAX_CARDS) { message = `A board holds ${MAX_CARDS} cards. Delete one first.`; return; }
    try { cards = [...cards, await addMood(owner, cards, { text: '' })]; open = true; } catch (e: any) { message = String(e?.message ?? e); }
  }
  async function remove(c: Mood) {
    cards = cards.filter((x) => x.id !== c.id);
    await deleteMood(c.id).catch((e) => (message = String(e?.message ?? e)));
  }
  const keep = (c: Mood) => saveMood(c).catch((e) => (message = String(e?.message ?? e)));

  const onPick = (e: Event) => { const el = e.currentTarget as HTMLInputElement; addFiles([...(el.files ?? [])]); el.value = ''; };
  const onDrop = (e: DragEvent) => { e.preventDefault(); over = false; addFiles([...(e.dataTransfer?.files ?? [])]); };
  function onPaste(e: ClipboardEvent) {
    if (e.target instanceof HTMLTextAreaElement) return; // pasting text into a note is normal
    const files = [...(e.clipboardData?.files ?? [])].filter((f) => f.type.startsWith('image/'));
    if (files.length) { e.preventDefault(); addFiles(files); }
  }
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape' && big) { e.stopPropagation(); big = null; } }} />

<details class="mood" bind:open>
  <summary>{word('Mood')}{cards.length ? ' · ' + cards.length : ''}</summary>
  <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
  <div class="drop" class:over role="group" aria-label={word('Mood')} tabindex="0"
    ondragover={(e) => { e.preventDefault(); over = true; }} ondragleave={() => (over = false)} ondrop={onDrop} onpaste={onPaste}>
    <div class="grid">
      {#each cards as c (c.id)}
        <div class="card">
          {#if c.image}<button class="pic" title="Show it larger" onclick={() => (big = c)}><img src={c.image} alt={c.text || 'Mood board picture'} /></button>{/if}
          <textarea rows={c.image ? 2 : 4} maxlength={MAX_TEXT} bind:value={c.text} onchange={() => keep(c)} placeholder={c.image ? 'A short note…' : 'A note: a feeling, a smell, a line…'}></textarea>
          <div class="del"><ConfirmButton onconfirm={() => remove(c)} /></div>
        </div>
      {/each}
    </div>
    {#if !cards.length}<p class="muted hint">Drop pictures here, paste one (Ctrl+V), or use the buttons. A card can also be a short note.</p>{/if}
    <div class="row" style="justify-content:flex-start;gap:.4rem">
      <button class="small" disabled={busy} onclick={() => fileBox.click()}>{busy ? 'Adding…' : '+ Pictures'}</button>
      <button class="small" onclick={addNote}>+ Note</button>
    </div>
    <input bind:this={fileBox} type="file" accept="image/*" multiple hidden onchange={onPick} />
    {#if message}<p class="msg" role="status">{message}</p>{/if}
  </div>
</details>

{#if big}
  <div class="light" role="presentation" onclick={() => (big = null)}>
    <img src={big.image} alt={big.text || 'Mood board picture'} />
    {#if big.text}<p>{big.text}</p>{/if}
  </div>
{/if}

<style>
  .mood { margin-top: 1rem; }
  summary { cursor: pointer; font-weight: 600; margin-bottom: .4rem; }
  .drop { padding: .4rem; border: 1px dashed transparent; border-radius: 8px; }
  .drop.over { border-color: var(--accent); background: color-mix(in srgb, var(--accent) 8%, transparent); }
  .drop:focus-visible { outline: 1px solid var(--accent); }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: .5rem; margin-bottom: .5rem; }
  .card { position: relative; display: flex; flex-direction: column; gap: .25rem; background: var(--panel-2); border: 1px solid var(--line); border-radius: 8px; padding: .3rem; }
  .pic { padding: 0; border: 0; background: none; line-height: 0; cursor: zoom-in; }
  .pic img { width: 100%; height: 110px; object-fit: cover; border-radius: 6px; }
  .card textarea { min-height: 0; font-size: .8rem; resize: vertical; }
  .del { text-align: right; }
  .hint, .msg { font-size: .85rem; margin: .3rem 0; }
  .msg { color: var(--danger); }
  .light { position: fixed; inset: 0; z-index: 90; background: rgba(0, 0, 0, .82); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: .6rem; padding: 2rem; cursor: zoom-out; }
  .light img { max-width: 92vw; max-height: 80vh; border-radius: 8px; }
  .light p { color: #fff; max-width: 40rem; text-align: center; margin: 0; white-space: pre-wrap; }
</style>
