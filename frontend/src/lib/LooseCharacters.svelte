<script lang="ts">
  import Select from './Select.svelte';
  import { db } from './db.svelte';
  import { openItem, openScene } from './nav.svelte';
  import { scenesOf, sceneLabel } from './scenes';
  import { mentionedIn } from './mentions';
  import { analyze, type Finding, type ScenePresence } from './loose-core';

  // Characters who are loose ends. This reads the text of every scene, so the Checks tab shows it only while this tab is open.
  const GAP_KEY = 'typewriter-loose-gap';
  let scope = $state<number | 'all'>('all');
  let gap = $state(5);
  try { const v = Number(localStorage.getItem(GAP_KEY)); if (Number.isFinite(v) && v >= 2) gap = Math.min(50, Math.floor(v)); } catch { /* storage may be blocked */ }
  function setGap(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    const v = Math.min(50, Math.max(2, Math.floor(Number(input.value)) || 5));
    gap = v; input.value = String(v);
    try { localStorage.setItem(GAP_KEY, String(v)); } catch { /* it lasts until the app closes */ }
  }

  let showFine = $state(false);

  const books = $derived([...db.lists.book].sort((a, b) => a.order - b.order));
  // the chapters we look at, in reading order. A chapter is a place in this list.
  const chapters = $derived(
    books.filter((b) => scope === 'all' || b.id === scope)
      .flatMap((b) => db.lists.chapter.filter((c) => c.book === b.id).sort((x, y) => x.order - y.order).map((c) => ({ c, book: b }))),
  );
  const label = (i: number) => {
    const x = chapters[i];
    if (!x) return 'a chapter that is gone';
    return (scope === 'all' && books.length > 1 ? x.book.title + ', ' : '') + 'chapter ' + x.c.order;
  };

  /** Everyone in each scene: the point of view, the cast, and every name in the text. */
  const presence = $derived.by(() => {
    const out: (ScenePresence & { sceneId: number })[] = [];
    chapters.forEach(({ c }, i) => {
      for (const s of scenesOf(c.id)) {
        const ids = new Set<number>([...(s.pov ? [s.pov] : []), ...(s.characters ?? []), ...mentionedIn(s).map((m) => m.id)]);
        out.push({ chapter: i, ids: [...ids].filter((id) => db.get('character', id)), sceneId: s.id });
      }
    });
    return out;
  });

  const hasDied = (c: any) => c.died !== null && c.died !== undefined && String(c.died).trim() !== '' && Number.isFinite(Number(c.died));
  const chars = $derived(
    [...db.lists.character].sort((a, b) => String(a.name).localeCompare(String(b.name)))
      .map((c) => ({ id: c.id as number, name: String(c.name ?? ''), died: hasDied(c), ignore: c.looseOk === true })),
  );
  const result = $derived(analyze(chars, presence, chapters.length, gap));
  const sceneWithCharacter = (id: number) => presence.find((p) => p.ids.includes(id))?.sceneId ?? null;
  const lastSceneOf = (id: number) => [...presence].reverse().find((p) => p.ids.includes(id))?.sceneId ?? null;

  const total = $derived(result.none.length + result.once.length + result.quiet.length);
  const writtenAny = $derived(presence.length > 0);
  const who = (id: number) => db.get('character', id);
  const plural = (n: number, one: string) => `${n} ${one}${n === 1 ? '' : 's'}`;

  const fine = (id: number, ok: boolean) => { const c = who(id); if (c) { c.looseOk = ok; db.save('character', c); } };
  const sceneName = (id: number | null) => { const s = db.get('scene', id); return s ? sceneLabel(s) : ''; };
</script>

<p class="muted">
  Finds characters who may be loose ends: in no scene, in one scene only, or gone for many chapters.
  A character is in a scene when they are the point of view, in the cast, or named in the text.
  A character who has a death date is not counted as quiet at the end of the book.
  Press <b>It is fine</b> if it is on purpose.
</p>

<div class="row" style="justify-content:flex-start;margin-bottom:.8rem;flex-wrap:wrap">
  <label class="fit" for="lscope" style="margin:0">Look in</label>
  <Select id="lscope" style="flex:0 1 16rem" bind:value={scope}>
    <option value="all">All books</option>
    {#each books as b}<option value={b.id}>{b.title}</option>{/each}
  </Select>
  <label class="fit" for="lgap" style="margin:0">Quiet after</label>
  <input id="lgap" class="gapbox" type="number" min="2" max="50" value={gap} onchange={setGap} onblur={setGap} />
  <span class="fit muted">chapters</span>
</div>

{#if !chars.length}
  <div class="empty">No characters yet. Add some in the Characters tab.</div>
{:else if !writtenAny}
  <div class="empty">Nothing to check yet. Write some scenes first.</div>
{:else}
  <p class="muted">{plural(total, 'character')} to look at, out of {plural(chars.length, 'character')}, in {plural(chapters.length, 'chapter')}.</p>

  {#if result.none.length}
    <div class="card">
      <h3>In no scene · {result.none.length}</h3>
      {#each result.none as f (f.id)}
        {@render person(f, 'is not in any scene.')}
      {/each}
    </div>
  {/if}

  {#if result.once.length}
    <div class="card">
      <h3>In one scene only · {result.once.length}</h3>
      {#each result.once as f (f.id)}
        {@render person(f, 'is only in ' + label(f.chapters[0]) + '.', sceneWithCharacter(f.id))}
      {/each}
    </div>
  {/if}

  {#if result.quiet.length}
    <div class="card">
      <h3>Gone quiet · {result.quiet.length}</h3>
      {#each result.quiet as f (f.id)}
        {@render person(
          f,
          f.gapTo !== undefined
            ? `is in ${label(f.gapFrom!)}, then not again until ${label(f.gapTo)}: ${plural(f.gap!, 'chapter')} without them.`
            : `was last seen in ${label(f.gapFrom!)}. After that, ${plural(f.gap!, 'chapter')} without them.`,
          lastSceneOf(f.id),
        )}
      {/each}
    </div>
  {/if}

  {#if !total}
    <div class="empty">No loose characters. Everyone appears, and nobody is gone for long.</div>
  {/if}

  {#if result.ignored.length}
    <button class="small" style="margin-top:1rem" onclick={() => (showFine = !showFine)}>
      {showFine ? 'Hide' : 'Show'} {result.ignored.length} marked “it is fine”
    </button>
    {#if showFine}
      <div class="card">
        {#each result.ignored as f (f.id)}
          <div class="gap done">
            <span class="t"><span class="dot" style:background={who(f.id)?.color}></span> <b>{who(f.id)?.name}</b></span>
            <button class="small" onclick={() => fine(f.id, false)}>Bring back</button>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
{/if}

{#snippet person(f: Finding, text: string, sceneId: number | null = null)}
  {@const c = who(f.id)}
  <div class="gap">
    <button class="row-link" onclick={() => openItem('character', f.id)} title="Open the character">
      <span class="dot" style:background={c?.color}></span>
      <span class="t"><b>{c?.name}</b> {text}{f.scenes > 1 ? ` (${plural(f.scenes, 'scene')})` : ''}</span>
    </button>
    {#if sceneId}<button class="small" title={sceneName(sceneId)} onclick={() => openScene(sceneId)}>Open scene</button>{/if}
    <button class="small" onclick={() => fine(f.id, true)}>It is fine</button>
  </div>
{/snippet}

<style>
  .row-link { display: flex; align-items: center; gap: .5rem; width: 100%; text-align: left; background: none; border: 1px solid transparent; padding: .3rem .4rem; }
  .row-link:hover { background: var(--panel-2); border-color: var(--line); }
  .row-link .t { flex: 1; min-width: 0; }
  .gap { display: flex; align-items: center; gap: .5rem; }
  .gap .row-link { flex: 1; width: auto; }
  .gap.done { opacity: .65; padding: .3rem .4rem; }
  .gap.done .t { flex: 1; }
  .gapbox { width: 4.5rem; flex: 0 0 auto; padding: .3rem .5rem; }
</style>
