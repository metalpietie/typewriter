<script lang="ts">
  import { onMount, tick as nextTick } from 'svelte';
  import type { Editor } from '@tiptap/core';
  import { db, CHARACTER_COLORS } from './db.svelte';
  import { openItem } from './nav.svelte';
  import { pop } from './motion.svelte';
  import { linkTargets, type Cand } from './linktargets';
  import { titleOf, nameOf, pageHtml, loreText } from './selection-core';

  // The pop-up above selected text: quick actions. It lives with the editor and uses the editor's own selection.
  let { editor, redraw, oncomment, onfind, source = '' }: {
    editor: Editor | undefined;
    redraw: number; // changes whenever the editor changes, so this menu redraws
    oncomment?: () => void;
    onfind?: () => void; // open Find & replace, which starts with the selected text
    source?: string; // where the text is from, for example "Book One, ch. 3, scene 2"
  } = $props();

  let root = $state<HTMLElement>();
  let view = $state<'main' | 'link' | 'note' | 'make' | 'done'>('main');
  let query = $state('');
  let at = $state(0);
  let pressing = $state(false); // the mouse button is down: the selection is still being made
  let hidden = $state(false); // closed with Escape, or the editor lost focus
  let done = $state<{ kind: 'draft' | 'note' | 'character' | 'location' | 'item' | 'faction'; id: number; title: string; what: string } | null>(null);
  let pos = $state<{ x: number; top?: number; bottom?: number }>({ x: 0 });
  let lastKey = '';

  const sel = $derived.by(() => {
    redraw;
    if (!editor) return null;
    const { from, to, empty } = editor.state.selection;
    if (empty) return null;
    const text = editor.state.doc.textBetween(from, to, '\n');
    return text.trim() ? { from, to, text } : null;
  });
  const show = $derived(!!sel && !pressing && !hidden);
  const items = $derived<Cand[]>(view === 'link' ? linkTargets(query) : []);
  const is = (name: string) => { redraw; return !!editor?.isActive(name); };

  // When the menu closes, the next time starts again at the main row. A new selection does too.
  $effect(() => { if (!show) { view = 'main'; done = null; } });
  $effect(() => {
    const key = sel ? sel.from + ':' + sel.to : '';
    if (key !== lastKey) { lastKey = key; view = 'main'; done = null; hidden = false; }
  });

  /** Put the menu above the first line of the selection, or below when there is no room above. */
  function place() {
    if (!editor || !sel) return;
    const a = editor.view.coordsAtPos(sel.from), b = editor.view.coordsAtPos(sel.to);
    const top = Math.min(a.top, b.top);
    const mid = Math.abs(a.top - b.top) < 4 ? (a.left + b.left) / 2 : a.left + 80;
    const x = Math.max(170, Math.min(mid, window.innerWidth - 170));
    pos = top < 80 ? { x, top: Math.max(a.bottom, b.bottom) + 8 } : { x, bottom: window.innerHeight - top + 8 };
  }
  $effect(() => { if (show) { redraw; view; place(); } });

  onMount(() => {
    const down = () => { pressing = true; hidden = false; };
    const up = () => { if (pressing) setTimeout(() => (pressing = false), 0); };
    const scrolled = () => { if (show) place(); };
    const key = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && show) { e.stopPropagation(); hidden = true; editor?.commands.focus(); }
    };
    const dom = editor?.view.dom;
    dom?.addEventListener('mousedown', down);
    window.addEventListener('mouseup', up);
    window.addEventListener('scroll', scrolled, true);
    window.addEventListener('keydown', key, true);
    const onBlur = ({ event }: { event: FocusEvent }) => {
      const to = event?.relatedTarget;
      if (!(to instanceof Node && root?.contains(to))) hidden = true;
    };
    const onFocus = () => { hidden = false; };
    editor?.on('blur', onBlur);
    editor?.on('focus', onFocus);
    return () => {
      dom?.removeEventListener('mousedown', down);
      window.removeEventListener('mouseup', up);
      window.removeEventListener('scroll', scrolled, true);
      window.removeEventListener('keydown', key, true);
      editor?.off('blur', onBlur);
      editor?.off('focus', onFocus);
    };
  });

  const hold = (e: MouseEvent) => { if (!(e.target instanceof HTMLInputElement)) e.preventDefault(); }; // keep the selection while you click

  const mark = (name: 'bold' | 'italic' | 'underline') => {
    const c = editor!.chain().focus();
    (name === 'bold' ? c.toggleBold() : name === 'italic' ? c.toggleItalic() : c.toggleUnderline()).run();
  };

  // ---- link to a character, place, lore note, faction or item ----
  async function openLink() {
    view = 'link';
    query = sel && sel.text.length <= 40 && !sel.text.includes('\n') ? sel.text.trim() : '';
    at = 0;
    await nextTick();
    const input = root?.querySelector<HTMLInputElement>('input.q');
    input?.focus();
    input?.select();
  }
  function linkTo(c: Cand) {
    if (!editor || !sel) return;
    editor.chain().focus().setTextSelection({ from: sel.from, to: sel.to }).setMark('ref', { to: c.kind + ':' + c.id }).setTextSelection(sel.to).run();
  }
  function unlink() {
    if (!editor || !sel) return;
    editor.chain().focus().setTextSelection({ from: sel.from, to: sel.to }).unsetMark('ref').setTextSelection(sel.to).run();
  }
  function linkKeys(e: KeyboardEvent) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      if (items.length) at = (at + (e.key === 'ArrowDown' ? 1 : items.length - 1)) % items.length;
    } else if (e.key === 'Enter') {
      if (items[at]) linkTo(items[at]);
    } else return;
    e.preventDefault();
  }

  // ---- copy the text to a new note ----
  async function toNote(kind: 'draft' | 'note') {
    if (!sel) return;
    const text = sel.text;
    const title = titleOf(text);
    let item;
    if (kind === 'draft') {
      const last = [...db.lists.draft].map((p) => p.order ?? p.id).sort((a, b) => a - b).pop() ?? 0;
      item = await db.save('draft', { title, body: pageHtml(text, source), order: last + 1 });
    } else {
      item = await db.save('note', { title, category: 'Other', body: loreText(text, source), tags: '' });
    }
    done = { kind, id: item.id, title, what: 'Saved' };
    view = 'done';
  }

  // ---- make a new character, place, item or faction from the selected name, and link the text to it ----
  async function makeNew(kind: 'character' | 'location' | 'item' | 'faction') {
    if (!editor || !sel) return;
    const { from, to } = sel;
    const name = nameOf(sel.text);
    if (!name) return;
    const color = CHARACTER_COLORS[db.lists[kind === 'faction' ? 'faction' : 'character'].length % CHARACTER_COLORS.length];
    const fields = kind === 'character'
      ? { name, role: 'supporting', aliases: '', age: '', description: '', traits: '', backstory: '', goals: '', notes: '', fields: [], color }
      : kind === 'location' ? { name, type: 'city', description: '', parent: null, tags: '' }
      : kind === 'item' ? { name, type: 'relic', description: '', history: '', tags: '' }
      : { name, type: 'guild', description: '', ranks: '', hq: null, color };
    const item = await db.save(kind, fields);
    editor.chain().focus().setTextSelection({ from, to }).setMark('ref', { to: kind + ':' + item.id }).run(); // the selection stays, so this message stays
    done = { kind, id: item.id, title: name, what: 'Made a new ' + (kind === 'location' ? 'place' : kind) + ' and linked the text' };
    view = 'done';
  }
  function openNote() {
    if (!done) return;
    db.flush();
    openItem(done.kind, done.id);
  }
</script>

{#if show}
  <div class="selmenu" transition:pop={{ origin: 'center bottom' }} bind:this={root} role="toolbar" aria-label="Quick actions for the selected text" tabindex="-1" onmousedown={hold}
    style:left="{pos.x}px" style:top={pos.top != null ? pos.top + 'px' : null} style:bottom={pos.bottom != null ? pos.bottom + 'px' : null}>
    {#if view === 'main'}
      <div class="row1">
        <button class:active={is('bold')} title="Bold (Ctrl+B)" onclick={() => mark('bold')}><b>B</b></button>
        <button class:active={is('italic')} title="Italic (Ctrl+I)" onclick={() => mark('italic')}><i>I</i></button>
        <button class:active={is('underline')} title="Underline (Ctrl+U)" onclick={() => mark('underline')}><u>U</u></button>
        <span class="sep"></span>
        {#if oncomment}<button title="Comment on this text" onclick={() => oncomment?.()}>💬 Comment</button>{/if}
        <button class:active={is('ref')} title="Link this text to a character, place, lore note, faction or item" onclick={openLink}>@ Link</button>
      </div>
      <div class="row1">
        <button title="Make a new character, place, item or faction from this text" onclick={() => (view = 'make')}>＋ Make new</button>
        <button title="Copy this text to a new note" onclick={() => (view = 'note')}>📝 To a note</button>
        {#if onfind}<button title="Find this text in your book" onclick={() => { hidden = true; onfind?.(); }}>🔍 Find</button>{/if}
      </div>
    {:else if view === 'link'}
      <div class="head">
        <button class="small" title="Back" onclick={() => (view = 'main')}>‹</button>
        <input class="q" bind:value={query} oninput={() => (at = 0)} onkeydown={linkKeys} placeholder="Link to… (a name)" aria-label="Find something to link to" />
      </div>
      {#each items as c, i (c.kind + c.id)}
        <button class="item" class:on={i === at} onclick={() => linkTo(c)}><span>{c.label}</span><small>{c.word}</small></button>
      {/each}
      {#if !items.length}<div class="muted hint">Nothing with that name. Type another name.</div>{/if}
      {#if is('ref')}<button class="item" onclick={unlink}><span>Remove the link</span></button>{/if}
    {:else if view === 'note'}
      <div class="head"><button class="small" title="Back" onclick={() => (view = 'main')}>‹</button><span class="muted">Copy this text to…</span></div>
      <button class="item" onclick={() => toNote('draft')}><span>A page in Notes</span><small>ideas and thoughts</small></button>
      <button class="item" onclick={() => toNote('note')}><span>A lore note</span><small>World tab</small></button>
    {:else if view === 'make'}
      <div class="head"><button class="small" title="Back" onclick={() => (view = 'main')}>‹</button><span class="muted">Make a new… from “{nameOf(sel?.text ?? '')}”</span></div>
      <button class="item" onclick={() => makeNew('character')}><span>Character</span><small>Characters tab</small></button>
      <button class="item" onclick={() => makeNew('location')}><span>Place</span><small>World tab</small></button>
      <button class="item" onclick={() => makeNew('item')}><span>Item</span><small>Items tab</small></button>
      <button class="item" onclick={() => makeNew('faction')}><span>Faction</span><small>Factions tab</small></button>
    {:else if done}
      <div class="head"><span>{done.what}: <b>{done.title}</b></span></div>
      <div class="row1">
        <button class="primary" onclick={openNote}>Open it</button>
        <button onclick={() => (hidden = true)}>Stay here</button>
      </div>
    {/if}
  </div>
{/if}

<style>
  .selmenu {
    position: fixed; z-index: 70; translate: -50% 0; width: max-content; min-width: 14rem; max-width: 94vw;
    background: var(--panel); border: 1px solid var(--line); border-radius: 10px; padding: .3rem;
    box-shadow: 0 10px 30px rgba(0, 0, 0, .4); display: flex; flex-direction: column; gap: .15rem;
  }
  .row1 { display: flex; gap: .15rem; align-items: center; flex-wrap: nowrap; }
  .selmenu button { white-space: nowrap; flex: 0 0 auto; padding: .3rem .6rem; min-width: 0; line-height: 1.3; }
  .selmenu button.active { border-color: var(--accent); }
  .sep { width: 1px; align-self: stretch; background: var(--line); margin: 0 .15rem; }
  .head { display: flex; gap: .3rem; align-items: center; padding: .1rem; }
  .head .q { flex: 1; min-width: 0; padding: .3rem .5rem; }
  .item { display: flex; justify-content: space-between; gap: 1rem; width: 100%; text-align: left; background: none; border: 1px solid transparent; }
  .item.on, .item:hover { background: var(--panel-2); border-color: var(--line); }
  .item small, .hint { color: var(--muted); }
  .hint { padding: .3rem .5rem; font-size: .85rem; }
</style>
