<script lang="ts">
  import { onMount } from 'svelte';
  import { Editor, Mark, mergeAttributes } from '@tiptap/core';
  import StarterKit from '@tiptap/starter-kit';
  import Placeholder from '@tiptap/extension-placeholder';
  import SaveIndicator from './SaveIndicator.svelte';
  import { db } from './db.svelte';
  import { LINK_KINDS, labelOf, type LinkKind } from './backlinks';

  let { value, onchange, placeholder = 'Start writing…', bare = false, oncomment, onfocuscomment, onopenref }: {
    value: string;
    onchange: (html: string) => void;
    placeholder?: string;
    bare?: boolean; // hide the toolbar (focus mode)
    /** Called with the selected text. Returns the id of the new comment, or null to cancel. */
    oncomment?: (quote: string) => Promise<number | null>;
    /** Called when the cursor moves into or out of commented text. */
    onfocuscomment?: (id: number | null) => void;
    /** Called when an @ link is Ctrl+clicked. */
    onopenref?: (kind: LinkKind, id: number) => void;
  } = $props();

  let host: HTMLElement;
  let editor: Editor | undefined;
  // Bumped on every editor change so the toolbar buttons redraw their active state.
  let tick = $state(0);

  // A comment is a mark on the text. It saves as <span data-comment="id">; the note itself is its own item.
  const Comment = Mark.create({
    name: 'comment',
    inclusive: false,
    excludes: '', // comments may overlap
    addAttributes() {
      return { id: { default: null, parseHTML: (el) => Number(el.getAttribute('data-comment')), renderHTML: (a) => ({ 'data-comment': String(a.id) }) } };
    },
    parseHTML() { return [{ tag: 'span[data-comment]' }]; },
    renderHTML({ HTMLAttributes }) { return ['span', mergeAttributes({ class: 'cmt' }, HTMLAttributes), 0]; },
  });

  // An @ link is a mark on the name. It saves as <span data-ref="character:12">.
  const Ref = Mark.create({
    name: 'ref',
    inclusive: false,
    addAttributes() {
      return { to: { default: null, parseHTML: (el) => el.getAttribute('data-ref'), renderHTML: (a) => ({ 'data-ref': a.to }) } };
    },
    parseHTML() { return [{ tag: 'span[data-ref]' }]; },
    renderHTML({ HTMLAttributes }) { return ['span', mergeAttributes({ class: 'ref' }, HTMLAttributes), 0]; },
  });

  // The pop-up that opens when you type @.
  type Cand = { kind: LinkKind; id: number; label: string; word: string; also: string };
  let sug = $state<{ from: number; to: number; x: number; top?: number; bottom?: number; items: Cand[]; at: number } | null>(null);

  /** Where the pop-up goes: under the cursor, or above it when there is no room below. */
  function place(pos: number, count: number) {
    const c = editor!.view.coordsAtPos(pos);
    const height = Math.min(count * 32 + 12, 290);
    const below = window.innerHeight - c.bottom - 8;
    const up = below < height && c.top > below;
    return {
      x: Math.max(8, Math.min(c.left, window.innerWidth - 270)),
      top: up ? undefined : c.bottom + 4,
      bottom: up ? window.innerHeight - c.top + 4 : undefined,
    };
  }

  /** The page scrolled: keep the pop-up beside the cursor. */
  function follow() {
    if (sug && editor) Object.assign(sug, place(editor.state.selection.from, sug.items.length));
  }

  function candidates(q: string): Cand[] {
    const query = q.toLowerCase();
    const all: Cand[] = [];
    for (const { kind, word } of LINK_KINDS) {
      for (const it of db.lists[kind]) {
        const label = labelOf(kind, it);
        if (label) all.push({ kind, id: it.id, label, word, also: String(it.aliases ?? '').toLowerCase() });
      }
    }
    if (!query) return all.slice(0, 8);
    const score = (c: Cand) => {
      const l = c.label.toLowerCase();
      if (l.startsWith(query)) return 0;
      if (l.split(/\s+/).some((w) => w.startsWith(query))) return 1;
      if (c.also.includes(query)) return 2;
      return l.includes(query) ? 3 : -1;
    };
    return all.map((c) => [score(c), c] as const).filter(([n]) => n >= 0).sort((a, b) => a[0] - b[0]).slice(0, 8).map(([, c]) => c);
  }

  /** Open, move or close the pop-up, depending on the text before the cursor. */
  function checkSuggest() {
    if (!editor || !onopenref) { sug = null; return; }
    const { selection } = editor.state;
    if (!selection.empty) { sug = null; return; }
    const before = selection.$from.parent.textBetween(0, selection.$from.parentOffset, undefined, '￼');
    const m = /(?:^|[\s(“"—])@([\p{L}\p{N}'’-]*(?: [\p{L}\p{N}'’-]*)?)$/u.exec(before);
    if (!m) { sug = null; return; }
    const items = candidates(m[1]);
    if (!items.length) { sug = null; return; }
    const keep = sug && sug.from === selection.from - m[1].length - 1 ? sug.at : 0;
    sug = { from: selection.from - m[1].length - 1, to: selection.from, ...place(selection.from, items.length), items, at: Math.min(keep, items.length - 1) };
  }

  function pick(c: Cand) {
    if (!editor || !sug) return;
    const { from, to } = sug;
    sug = null;
    editor.chain().focus().insertContentAt({ from, to }, [
      { type: 'text', text: c.label, marks: [{ type: 'ref', attrs: { to: c.kind + ':' + c.id } }] },
      { type: 'text', text: ' ' },
    ]).run();
  }

  function onKeys(e: KeyboardEvent): boolean {
    if (!sug) return false;
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      sug.at = (sug.at + (e.key === 'ArrowDown' ? 1 : sug.items.length - 1)) % sug.items.length;
    } else if (e.key === 'Enter' || e.key === 'Tab') {
      pick(sug.items[sug.at]);
    } else if (e.key === 'Escape') {
      e.stopPropagation(); // do not also leave focus mode
      sug = null;
    } else return false;
    e.preventDefault();
    return true;
  }

  /** Where the text of one comment sits in the document. */
  function rangeOf(id: number): { from: number; to: number } | null {
    let from = -1, to = -1;
    editor?.state.doc.descendants((node, pos) => {
      if (node.isText && node.marks.some((m) => m.type.name === 'comment' && m.attrs.id === id)) {
        if (from < 0) from = pos;
        to = pos + node.nodeSize;
      }
    });
    return from < 0 ? null : { from, to };
  }

  /** Select the text a comment belongs to, and scroll to it. */
  export function showComment(id: number) {
    const r = rangeOf(id);
    if (r && editor) editor.chain().focus().setTextSelection(r).scrollIntoView().run();
  }

  /** Take a comment off the text. */
  export function removeComment(id: number) {
    const r = rangeOf(id);
    if (!r || !editor) return;
    const { state, view } = editor;
    view.dispatch(state.tr.removeMark(r.from, r.to, state.schema.marks.comment.create({ id })));
  }

  const noSelection = () => { tick; return !editor || editor.state.selection.empty; };

  async function addComment() {
    if (!editor || !oncomment) return;
    const { from, to } = editor.state.selection;
    if (from === to) return;
    const id = await oncomment(editor.state.doc.textBetween(from, to, ' '));
    if (id) editor.chain().focus().setTextSelection({ from, to }).setMark('comment', { id }).setTextSelection(to).run();
  }

  onMount(() => {
    editor = new Editor({
      element: host,
      content: value, // parent remounts this component per chapter, so this is read once
      extensions: [
        StarterKit.configure({ strike: false, code: false, codeBlock: false, heading: { levels: [1, 2, 3] } }),
        Placeholder.configure({ placeholder }),
        Comment,
        Ref,
      ],
      editorProps: {
        handleKeyDown: (_view, e) => onKeys(e),
        handleDOMEvents: {
          click: (_view, e) => {
            const el = (e.target as HTMLElement).closest?.('[data-ref]');
            if (!el || !(e.ctrlKey || e.metaKey) || !onopenref) return false;
            const [kind, id] = el.getAttribute('data-ref')!.split(':');
            onopenref(kind as LinkKind, +id);
            return true;
          },
        },
      },
      onUpdate: ({ editor }) => { tick++; onchange(editor.isEmpty ? '' : editor.getHTML()); checkSuggest(); },
      onSelectionUpdate: ({ editor }) => { tick++; onfocuscomment?.(editor.getAttributes('comment').id ?? null); checkSuggest(); },
      onBlur: () => setTimeout(() => (sug = null), 150),
    });
    window.addEventListener('scroll', follow, true); // capture: the page scrolls inside a panel
    return () => { window.removeEventListener('scroll', follow, true); editor?.destroy(); };
  });

  const is = (name: string, attrs?: object) => { tick; return !!editor?.isActive(name, attrs); };
  const run = () => editor!.chain().focus();
</script>

<div class="rich">
  <div class="rich-bar" hidden={bare}>
    <button class:active={is('bold')} title="Bold (Ctrl+B)" onclick={() => run().toggleBold().run()}><b>B</b></button>
    <button class:active={is('italic')} title="Italic (Ctrl+I)" onclick={() => run().toggleItalic().run()}><i>I</i></button>
    <button class:active={is('underline')} title="Underline (Ctrl+U)" onclick={() => run().toggleUnderline().run()}><u>U</u></button>
    <span class="sep"></span>
    <button class:active={is('heading', { level: 1 })} title="Big heading" onclick={() => run().toggleHeading({ level: 1 }).run()}>H1</button>
    <button class:active={is('heading', { level: 2 })} title="Heading" onclick={() => run().toggleHeading({ level: 2 }).run()}>H2</button>
    <button class:active={is('heading', { level: 3 })} title="Small heading" onclick={() => run().toggleHeading({ level: 3 }).run()}>H3</button>
    <span class="sep"></span>
    <button class:active={is('bulletList')} title="Bullet list" onclick={() => run().toggleBulletList().run()}>• List</button>
    <button class:active={is('orderedList')} title="Numbered list" onclick={() => run().toggleOrderedList().run()}>1. List</button>
    <button class:active={is('blockquote')} title="Quote" onclick={() => run().toggleBlockquote().run()}>❝</button>
    <button title="Scene break" onclick={() => run().setHorizontalRule().run()}>―</button>
    {#if onopenref}
      <button title="Link to a character, place or lore note (type @)" onclick={() => run().insertContent('@').run()}>@</button>
    {/if}
    {#if oncomment}
      <button title="Comment on the selected text" disabled={noSelection()} onclick={addComment}>💬</button>
    {/if}
    <span class="sep"></span>
    <button title="Undo (Ctrl+Z)" onclick={() => run().undo().run()}>↶</button>
    <button title="Redo (Ctrl+Y)" onclick={() => run().redo().run()}>↷</button>
    <span class="rich-save"><SaveIndicator /></span>
  </div>
  <div class="rich-page" bind:this={host}></div>
</div>

{#if sug}
  <div class="sug" style:left="{sug.x}px" style:top={sug.top != null ? sug.top + 'px' : null} style:bottom={sug.bottom != null ? sug.bottom + 'px' : null}>
    {#each sug.items as c, i (c.kind + c.id)}
      <button class:on={i === sug.at} onmousedown={(e) => { e.preventDefault(); pick(c); }}>
        <span>{c.label}</span><small>{c.word}</small>
      </button>
    {/each}
  </div>
{/if}

<style>
  .rich { display: flex; flex-direction: column; min-height: 60vh; }
  .rich-bar {
    display: flex; gap: .25rem; align-items: center; flex-wrap: wrap;
    position: sticky; top: -1.2rem; z-index: 5; background: var(--bg);
    padding: .4rem 0; border-bottom: 1px solid var(--line); margin-bottom: .8rem;
  }
  .rich-bar[hidden] { display: none; }
  .rich-bar button { padding: .2rem .55rem; min-width: 2rem; }
  .rich-save { margin-left: auto; }
  .sep { width: 1px; height: 1.4rem; background: var(--line); margin: 0 .3rem; }
  .rich-page { flex: 1; }

  .rich-page :global(.ProseMirror) {
    outline: none; min-height: 55vh; padding-bottom: 30vh;
    font-family: Georgia, 'Times New Roman', serif; font-size: 1.1rem; line-height: 1.8;
  }
  .rich-page :global(.ProseMirror p) { margin: 0 0 .9em; }
  .rich-page :global(.ProseMirror h1) { font-size: 1.9rem; margin: 1.4em 0 .6em; }
  .rich-page :global(.ProseMirror h2) { font-size: 1.5rem; margin: 1.2em 0 .5em; }
  .rich-page :global(.ProseMirror h3) { font-size: 1.2rem; margin: 1em 0 .4em; }
  .rich-page :global(.ProseMirror blockquote) {
    margin: 1em 0; padding-left: 1rem; border-left: 3px solid var(--accent); color: var(--muted); font-style: italic;
  }
  .rich-page :global(.ProseMirror ul), .rich-page :global(.ProseMirror ol) { padding-left: 1.6rem; margin: 0 0 .9em; }
  .rich-page :global(.ProseMirror hr) { border: 0; text-align: center; margin: 1.5em 0; }
  .rich-page :global(.ProseMirror hr::after) { content: '* * *'; color: var(--muted); letter-spacing: .5em; }
  .rich-page :global(.ProseMirror .cmt) { background: color-mix(in srgb, var(--accent-2) 28%, transparent); border-bottom: 2px solid var(--accent-2); cursor: pointer; }
  .rich-page :global(.ProseMirror .ref) { color: var(--accent); border-bottom: 1px dotted var(--accent); }
  .sug { position: fixed; z-index: 60; width: 260px; background: var(--panel); border: 1px solid var(--line); border-radius: 8px; padding: .25rem; box-shadow: 0 10px 30px rgba(0, 0, 0, .4); }
  .sug button { display: flex; justify-content: space-between; width: 100%; text-align: left; background: none; border: 1px solid transparent; padding: .3rem .5rem; }
  .sug button.on { background: var(--panel-2); border-color: var(--line); }
  .sug small { color: var(--muted); }
  .rich-page :global(.ProseMirror p.is-editor-empty:first-child::before) {
    content: attr(data-placeholder); color: var(--muted); float: left; height: 0; pointer-events: none;
  }
</style>
