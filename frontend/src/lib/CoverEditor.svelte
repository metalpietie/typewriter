<script lang="ts">
  import Select from './Select.svelte';
  import ConfirmButton from './ConfirmButton.svelte';
  import { db } from './db.svelte';
  import { TEMPLATES, DEFAULT_SPEC, cleanSpec, type Spec } from './cover-core';
  import { drawCover, getCover, loadImage, pictureToCover, pictureToPhoto, removeCover, saveCover, type CoverDoc } from './cover';

  // The cover of one book: a picture you choose, or a simple cover made from a template.
  // It is the first page of the Word file, the PDF and the e-book.
  let { bookId, author = '' }: { bookId: number | null; author?: string } = $props();

  const book = $derived(db.get('book', bookId));
  let doc = $state<CoverDoc | null>(null);
  let making = $state(false);
  let spec = $state<Spec>({ ...DEFAULT_SPEC });
  let photo = $state('');
  let message = $state('');
  let busy = $state(false);
  let canvas = $state<HTMLCanvasElement>();
  let pic: HTMLInputElement;
  let photoBox: HTMLInputElement;
  let seen = 0; // which book the cover belongs to, so a slow answer for an old book is ignored

  $effect(() => {
    const mine = ++seen, id = bookId;
    doc = null; making = false; message = '';
    if (id) getCover(id).then((d) => { if (mine === seen) doc = d; }).catch((e) => { if (mine === seen) message = String(e?.message ?? e); });
  });

  // draw the preview again when a choice changes
  $effect(() => {
    if (!making || !canvas) return;
    const s = { ...spec }, url = photo, c = canvas;
    let stale = false;
    (url ? loadImage(url).catch(() => null) : Promise.resolve(null)).then((img) => { if (!stale) drawCover(c, s, img); });
    return () => { stale = true; };
  });

  function startMaking() {
    const old = doc?.kind === 'made' ? doc.spec : undefined;
    spec = cleanSpec(old, { title: String(book?.title ?? ''), author: author.trim() });
    photo = doc?.kind === 'made' ? doc.photo ?? '' : '';
    making = true; message = '';
  }

  async function choose(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file || !bookId) return;
    if (!file.type.startsWith('image/')) { message = `${file.name} is not a picture.`; return; }
    busy = true; message = '';
    try { doc = await saveCover({ id: doc?.id, book: bookId, image: await pictureToCover(file), kind: 'picture' }); making = false; }
    catch (err: any) { message = err?.message ?? 'That picture could not be read.'; }
    finally { busy = false; }
  }

  async function chooseBackground(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file) return;
    if (!file.type.startsWith('image/')) { message = `${file.name} is not a picture.`; return; }
    try { photo = await pictureToPhoto(file); message = ''; } catch { message = 'That picture could not be read.'; }
  }

  async function useIt() {
    if (!bookId || !canvas) return;
    busy = true; message = '';
    try {
      doc = await saveCover({ id: doc?.id, book: bookId, image: canvas.toDataURL('image/jpeg', 0.9), kind: 'made', spec: cleanSpec(spec), photo: photo || undefined });
      making = false;
    } catch (err: any) { message = err?.message ?? String(err); } finally { busy = false; }
  }

  async function remove() {
    if (!doc?.id) return;
    const id = doc.id;
    doc = null;
    await removeCover(id).catch((e) => (message = String(e?.message ?? e)));
  }
</script>

<details class="pages cover">
  <summary><b>Cover</b> <span class="muted">(optional)</span></summary>
  {#if !bookId}
    <p class="muted">Add a book first. Then you can give it a cover.</p>
  {:else}
    <p class="muted" style="margin:.4rem 0">
      The cover of “{book?.title ?? 'this book'}” is the first page of the Word file, the PDF and the e-book. A Markdown file has no cover, and neither does a print-ready PDF (the printer wants the cover as its own file).
    </p>

    {#if !making}
      <div class="show">
        {#if doc}<img class="thumb2" src={doc.image} alt="The cover" />{:else}<div class="none">No cover yet</div>{/if}
        <div class="btns">
          <button class="small" disabled={busy} onclick={() => pic.click()}>{busy ? 'Working…' : 'Choose a picture…'}</button>
          <button class="small" disabled={busy} onclick={startMaking}>{doc?.kind === 'made' ? 'Change the cover…' : 'Make a cover…'}</button>
          {#if doc}<ConfirmButton label="Remove" onconfirm={remove} />{/if}
        </div>
      </div>
      <input bind:this={pic} type="file" accept="image/*" hidden onchange={choose} />
    {:else}
      <div class="make">
        <div class="ctl">
          <label for="cv-t">Style</label>
          <Select id="cv-t" bind:value={spec.template}>{#each TEMPLATES as [k, name]}<option value={k}>{name}</option>{/each}</Select>
          <label for="cv-title">Title</label>
          <input id="cv-title" bind:value={spec.title} maxlength="120" />
          <label for="cv-sub">Subtitle (optional)</label>
          <input id="cv-sub" bind:value={spec.subtitle} maxlength="120" />
          <label for="cv-author">Author</label>
          <input id="cv-author" bind:value={spec.author} maxlength="80" />
          <div class="colors">
            <div><label for="cv-bg">Background</label><input id="cv-bg" type="color" bind:value={spec.bg} /></div>
            <div><label for="cv-fg">Text</label><input id="cv-fg" type="color" bind:value={spec.fg} /></div>
            <div><label for="cv-ac">Accent</label><input id="cv-ac" type="color" bind:value={spec.accent} /></div>
          </div>
          <label for="cv-photo">Background picture (optional)</label>
          <div class="row" style="justify-content:flex-start;gap:.4rem">
            <button id="cv-photo" class="small" onclick={() => photoBox.click()}>{photo ? 'Change picture…' : 'Choose a picture…'}</button>
            {#if photo}<button class="small" onclick={() => (photo = '')}>Remove picture</button>{/if}
          </div>
          <input bind:this={photoBox} type="file" accept="image/*" hidden onchange={chooseBackground} />
          {#if photo}
            <label for="cv-dim">Darken the picture: {Math.round(spec.dim * 100)}%</label>
            <input id="cv-dim" type="range" min="0" max="0.8" step="0.05" bind:value={spec.dim} />
          {/if}
          <div class="row" style="justify-content:flex-start;gap:.4rem;margin-top:.8rem">
            <button class="primary small" disabled={busy || !spec.title.trim()} onclick={useIt}>{busy ? 'Saving…' : 'Use this cover'}</button>
            <button class="small" onclick={() => (making = false)}>Cancel</button>
          </div>
        </div>
        <canvas class="prev" bind:this={canvas} width="1600" height="2400" aria-label="Preview of the cover"></canvas>
      </div>
    {/if}
    {#if message}<p class="msg" role="status">{message}</p>{/if}
  {/if}
</details>

<style>
  .cover { margin-top: 1rem; }
  summary { cursor: pointer; }
  .show { display: flex; gap: 1rem; align-items: flex-start; flex-wrap: wrap; }
  .thumb2 { height: 200px; width: auto; border: 1px solid var(--line); border-radius: 4px; }
  .none { height: 200px; width: 133px; border: 1px dashed var(--line); border-radius: 4px; display: flex; align-items: center; justify-content: center; color: var(--muted); font-size: .8rem; text-align: center; }
  .btns { display: flex; flex-direction: column; gap: .4rem; align-items: flex-start; }
  .make { display: grid; grid-template-columns: minmax(0, 1fr) 200px; gap: 1rem; align-items: start; }
  .prev { width: 200px; height: 300px; border: 1px solid var(--line); border-radius: 4px; position: sticky; top: 0; }
  .colors { display: flex; gap: .8rem; }
  .colors input { width: 3rem; }
  .msg { color: var(--danger); font-size: .85rem; margin: .4rem 0 0; }
  @media (max-width: 520px) { .make { grid-template-columns: 1fr; } }
</style>
