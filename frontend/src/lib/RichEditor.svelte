<script lang="ts">
  import { has, word } from './projecttype.svelte';
  import { onMount, tick as nextTick } from 'svelte';
  import { Editor, Extension, Mark, Node, mergeAttributes } from '@tiptap/core';
  import Select from './Select.svelte';
  import { citeLabel, cleanSource, sourceLine } from './cite-core';
  import { BOXES, CAPTIONS } from './boxes-core';
  import { Plugin, PluginKey } from '@tiptap/pm/state';
  import { Decoration, DecorationSet } from '@tiptap/pm/view';
  import { DOMSerializer } from '@tiptap/pm/model';
  import StarterKit from '@tiptap/starter-kit';
  import Placeholder from '@tiptap/extension-placeholder';
  import SaveIndicator from './SaveIndicator.svelte';
  import { db } from './db.svelte';
  import { type LinkKind } from './backlinks';
  import { linkTargets, type Cand } from './linktargets';
  import SelectionMenu from './SelectionMenu.svelte';
  import { pop } from './motion.svelte';
  import { look } from './editorlook.svelte';
  import { lookVars } from './editorlook-core';

  let { value, onchange, placeholder = 'Start writing…', bare = false, source = '', onfind, onbeside, oncomment, onfocuscomment, onopenref }: {
    value: string;
    onchange: (html: string) => void;
    placeholder?: string;
    bare?: boolean; // hide the toolbar (focus mode)
    /** Where this text is from, for notes made from selected text. */
    source?: string;
    /** Called when an @ link is Alt+clicked: show that item beside the text. */
    onbeside?: (kind: LinkKind, id: number) => void;
    /** Called to open Find & replace for the selected text. */
    onfind?: () => void;
    /** Called with the selected text. Returns the id of the new comment, or null to cancel. */
    oncomment?: (quote: string) => Promise<number | null>;
    /** Called when the cursor moves into or out of commented text. */
    onfocuscomment?: (id: number | null) => void;
    /** Called when an @ link is Ctrl+clicked. */
    onopenref?: (kind: LinkKind, id: number) => void;
  } = $props();

  let host: HTMLElement;
  let editor = $state.raw<Editor | undefined>();
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

  // A citation is a small piece inside the text. It saves as <span data-cite="12" data-loc="45"></span>: the source and the place in it.
  // The editor shows it short, for example (Clear, 2018, p. 45). The exported book writes it in the style that the writer chose.
  const Cite = Node.create({
    name: 'cite',
    group: 'inline',
    inline: true,
    atom: true,
    addAttributes() {
      return {
        source: { default: null, parseHTML: (el) => Number(el.getAttribute('data-cite')), renderHTML: (a) => ({ 'data-cite': String(a.source) }) },
        loc: { default: '', parseHTML: (el) => el.getAttribute('data-loc') ?? '', renderHTML: (a) => (a.loc ? { 'data-loc': String(a.loc) } : {}) },
      };
    },
    parseHTML() { return [{ tag: 'span[data-cite]' }]; },
    renderHTML({ HTMLAttributes }) { return ['span', mergeAttributes({ class: 'cite' }, HTMLAttributes)]; },
    addNodeView() {
      return ({ node }) => {
        const dom = document.createElement('span');
        dom.className = 'cite';
        dom.setAttribute('data-cite', String(node.attrs.source));
        dom.contentEditable = 'false';
        dom.textContent = citeLabel(db.get('source', node.attrs.source), node.attrs.loc);
        return { dom };
      };
    },
  });

  // A footnote is a small piece inside the text that holds its own short text. It saves as <span data-footnote="the text"></span>.
  // The number comes from a CSS counter, so it is always right, in the order of the text.
  const Footnote = Node.create({
    name: 'footnote',
    group: 'inline',
    inline: true,
    atom: true,
    addAttributes() {
      return { text: { default: '', parseHTML: (el) => el.getAttribute('data-footnote') ?? '', renderHTML: (a) => ({ 'data-footnote': String(a.text ?? '') }) } };
    },
    parseHTML() { return [{ tag: 'span[data-footnote]' }]; },
    renderHTML({ HTMLAttributes }) { return ['span', mergeAttributes({ class: 'fn' }, HTMLAttributes)]; },
  });

  // A callout box holds a few paragraphs or lists under a coloured bar: a key idea, a story, an example, an exercise, a warning, a quote.
  // It saves as <div data-callout="exercise">…</div>.
  const Callout = Node.create({
    name: 'callout',
    group: 'block',
    content: '(paragraph | caption | bulletList | orderedList | blockquote | heading)+',
    defining: true,
    addAttributes() {
      return { kind: { default: 'idea', parseHTML: (el) => el.getAttribute('data-callout') || 'idea', renderHTML: (a) => ({ 'data-callout': String(a.kind) }) } };
    },
    parseHTML() { return [{ tag: 'div[data-callout]' }]; },
    renderHTML({ HTMLAttributes }) { return ['div', mergeAttributes({ class: 'callout' }, HTMLAttributes), 0]; },
  });

  // A caption sits under a figure or a table. The export numbers it by chapter: Figure 3.2. It saves as <p data-caption="figure">.
  const Caption = Node.create({
    name: 'caption',
    group: 'block',
    content: 'inline*',
    // The paragraph must stay the first block, because the editor makes it when it needs an empty block (an empty page, a line after a box).
    addAttributes() {
      return { kind: { default: 'figure', parseHTML: (el) => el.getAttribute('data-caption') || 'figure', renderHTML: (a) => ({ 'data-caption': String(a.kind) }) } };
    },
    parseHTML() { return [{ tag: 'p[data-caption]', priority: 100 }]; }, // before the plain paragraph, which also reads a <p>
    renderHTML({ HTMLAttributes }) { return ['p', mergeAttributes({ class: 'caption' }, HTMLAttributes), 0]; },
  });

  // A term for the index at the end of the book. It saves as <span data-index="habit loop">.
  const IndexTerm = Mark.create({
    name: 'indexTerm',
    inclusive: false,
    addAttributes() {
      return { term: { default: '', parseHTML: (el) => el.getAttribute('data-index') ?? '', renderHTML: (a) => ({ 'data-index': String(a.term ?? '') }) } };
    },
    parseHTML() { return [{ tag: 'span[data-index]' }]; },
    renderHTML({ HTMLAttributes }) { return ['span', mergeAttributes({ class: 'idx' }, HTMLAttributes), 0]; },
  });

  // A claim that still needs a source. The Checks tab lists the ones that have no citation after them. It saves as <span data-needs="1">.
  const NeedsSource = Mark.create({
    name: 'needsSource',
    inclusive: false,
    parseHTML() { return [{ tag: 'span[data-needs]' }]; },
    renderHTML({ HTMLAttributes }) { return ['span', mergeAttributes({ class: 'needs', 'data-needs': '1' }, HTMLAttributes), 0]; },
  });

  // Focus line: every text block but the one with the cursor gets the class "dim". The page styles it only when the choice is on.
  const FocusLine = Extension.create({
    name: 'focusLine',
    addProseMirrorPlugins() {
      return [new Plugin({
        key: new PluginKey('focusLine'),
        props: {
          decorations(state) {
            if (!look.focusLine) return null;
            const from = state.selection.$from;
            const here = from.parent.isTextblock ? from.before() : -1;
            const out: Decoration[] = [];
            state.doc.descendants((node, pos) => {
              if (!node.isTextblock) return true;
              if (pos !== here) out.push(Decoration.node(pos, pos + node.nodeSize, { class: 'dim' }));
              return false;
            });
            return DecorationSet.create(state.doc, out);
          },
        },
      })];
    },
  });
  // draw the dimming again when the choice is switched
  $effect(() => { look.focusLine; if (editor) editor.view.dispatch(editor.state.tr); });

  /** The box that scrolls the page: the nearest parent that can scroll. */
  function scroller(el: HTMLElement | null): HTMLElement | null {
    for (let p = el?.parentElement; p; p = p.parentElement) {
      const o = getComputedStyle(p).overflowY;
      if ((o === 'auto' || o === 'scroll') && p.scrollHeight > p.clientHeight) return p;
    }
    return null;
  }
  /** Typewriter mode: scroll so the line you write on is a little above the middle of the box. */
  function keepCentred() {
    if (!look.typewriter || !editor || !editor.state.selection.empty) return;
    // the editor has already drawn the change, so the line can be measured now
    const box = scroller(host);
    if (!box) return;
    const line = editor.view.coordsAtPos(editor.state.selection.head);
    const r = box.getBoundingClientRect();
    const delta = line.top - (r.top + r.height * 0.45);
    if (Math.abs(delta) > 3) box.scrollTop += delta;
  }
  const NAV_KEYS = new Set(['ArrowUp', 'ArrowDown', 'PageUp', 'PageDown', 'Home', 'End']);

  // The pop-up that opens when you type @.
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

  const candidates = (q: string): Cand[] => linkTargets(q);

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

  /** The text before the cursor and the text after it, as HTML. Used to split a scene in two. */
  export function splitAtCursor(): { before: string; after: string } | null {
    if (!editor) return null;
    const { doc, selection, schema } = editor.state;
    const ser = DOMSerializer.fromSchema(schema);
    const html = (node: typeof doc) => { const div = document.createElement('div'); div.appendChild(ser.serializeFragment(node.content)); return div.innerHTML; };
    return { before: html(doc.cut(0, selection.from)), after: html(doc.cut(selection.from)) };
  }

  const noSelection = () => { tick; return !editor || editor.state.selection.empty; };

  async function addComment() {
    if (!editor || !oncomment) return;
    const { from, to } = editor.state.selection;
    if (from === to) return;
    const id = await oncomment(editor.state.doc.textBetween(from, to, ' '));
    if (id) editor.chain().focus().setTextSelection({ from, to }).setMark('comment', { id }).setTextSelection(to).run();
  }

  // ---- footnotes and citations: a small box next to the piece, to write or change it ----
  type Piece = { kind: 'fn' | 'cite'; pos: number | null; at: number; x: number; top?: number; bottom?: number; text: string; source: string; loc: string; title: string; authors: string; year: string };
  let piece = $state<Piece | null>(null);
  let pieceInput = $state<HTMLElement>();
  const sourceList = $derived([...db.lists.source].sort((a, b) => sourceLine(a).toLowerCase().localeCompare(sourceLine(b).toLowerCase())));

  function showPiece(kind: Piece['kind'], pos: number | null, at: number, attrs: { text?: string; source?: number | null; loc?: string } = {}) {
    if (!editor) return;
    const spot = place(at, 8);
    piece = { kind, pos, at, ...spot, text: attrs.text ?? '', source: attrs.source ? String(attrs.source) : sourceList[0] ? String(sourceList[0].id) : 'new', loc: attrs.loc ?? '', title: '', authors: '', year: '' };
    sug = null;
    nextTick().then(() => pieceInput?.focus());
  }
  /** A footnote or a citation that is already in the text was clicked. */
  function openPiece(pos: number) {
    const node = editor?.state.doc.nodeAt(pos);
    if (!node) return;
    if (node.type.name === 'footnote') showPiece('fn', pos, pos, { text: node.attrs.text });
    else if (node.type.name === 'cite') showPiece('cite', pos, pos, { source: node.attrs.source, loc: node.attrs.loc });
  }
  /** A new footnote at the end of the selection, or at the cursor. */
  export function addFootnote() {
    if (!editor) return;
    const at = editor.state.selection.to;
    editor.chain().focus().insertContentAt(at, { type: 'footnote', attrs: { text: '' } }).run();
    showPiece('fn', at, at);
  }
  /** A new citation at the end of the selection, or at the cursor. */
  export function addCitation() {
    if (!editor) return;
    showPiece('cite', null, editor.state.selection.to);
  }
  function setFootnote(text: string) {
    if (!editor || !piece || piece.pos == null) return;
    piece.text = text;
    editor.view.dispatch(editor.state.tr.setNodeMarkup(piece.pos, undefined, { text }));
  }
  async function applyCitation() {
    if (!editor || !piece) return;
    let id = Number(piece.source);
    if (piece.source === 'new') {
      if (!piece.title.trim()) return;
      const made = await db.save('source', { ...cleanSource({ type: 'book', title: piece.title, authors: piece.authors, year: piece.year }) });
      id = made.id;
    }
    const attrs = { source: id, loc: piece.loc.trim() };
    if (piece.pos != null) editor.view.dispatch(editor.state.tr.setNodeMarkup(piece.pos, undefined, attrs));
    else editor.chain().focus().insertContentAt(piece.at, { type: 'cite', attrs }).run();
    closePiece();
  }
  function removePiece() {
    if (!editor || !piece || piece.pos == null) { closePiece(); return; }
    editor.view.dispatch(editor.state.tr.delete(piece.pos, piece.pos + 1));
    piece = null;
    editor.commands.focus();
  }
  function closePiece() {
    // a footnote that was added and left empty is not kept
    if (piece?.kind === 'fn' && piece.pos != null && !piece.text.trim() && editor?.state.doc.nodeAt(piece.pos)?.type.name === 'footnote') editor.view.dispatch(editor.state.tr.delete(piece.pos, piece.pos + 1));
    piece = null;
    editor?.commands.focus();
  }
  // ---- callout boxes and captions ----
  const boxKind = $derived.by(() => { tick; return editor?.isActive('callout') ? String(editor.getAttributes('callout').kind) : ''; });
  const captionKind = $derived.by(() => { tick; return editor?.isActive('caption') ? String(editor.getAttributes('caption').kind) : ''; });
  function chooseBox(kind: string) {
    if (!editor) return;
    const c = editor.chain().focus();
    if (!kind) c.lift('callout').run();
    else if (editor.isActive('callout')) c.updateAttributes('callout', { kind }).run();
    else c.wrapIn('callout', { kind }).run();
  }
  function chooseCaption(kind: string) {
    if (!editor) return;
    const c = editor.chain().focus();
    if (!kind) c.setNode('paragraph').run();
    else c.setNode('caption', { kind }).run();
  }
  /** An idea, a quote or a fact from the Ideas tab, put at the end of the text, with its citation. */
  export function insertIdea(idea: { kind: string; text: string; source?: number | null; loc?: string }) {
    if (!editor) return;
    const inline: any[] = [{ type: 'text', text: idea.text }];
    if (idea.source) inline.push({ type: 'text', text: ' ' }, { type: 'cite', attrs: { source: idea.source, loc: idea.loc ?? '' } });
    const para = { type: 'paragraph', content: inline };
    const node = idea.kind === 'quote' ? { type: 'blockquote', content: [para] } : para;
    editor.chain().focus('end').insertContentAt(editor.state.doc.content.size, node).run();
  }
  /** Add html at the end of the text. */
  export function atEnd(html: string) {
    editor?.chain().focus('end').insertContentAt(editor.state.doc.content.size, html).run();
  }

  function hotkeys(e: KeyboardEvent): boolean {
    if ((e.ctrlKey || e.metaKey) && e.altKey && e.key.toLowerCase() === 'f') { addFootnote(); e.preventDefault(); return true; }
    return false;
  }

  /** A caption with no text is never wanted. Older text may have one at the end, so take them out when the text opens. */
  function dropEmptyCaptions() {
    if (!editor) return;
    const gone: [number, number][] = [];
    editor.state.doc.descendants((node, pos) => {
      if (node.type.name === 'caption' && node.content.size === 0) gone.push([pos, pos + node.nodeSize]);
    });
    if (!gone.length) return;
    const tr = editor.state.tr;
    for (const [from, to] of gone.reverse()) tr.delete(from, to);
    if (tr.doc.childCount === 0) return;
    editor.view.dispatch(tr.setMeta('addToHistory', false));
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
        Cite,
        Footnote,
        Callout,
        Caption,
        IndexTerm,
        NeedsSource,
        FocusLine,
      ],
      editorProps: {
        handleKeyDown: (_view, e) => hotkeys(e) || onKeys(e),
        handleDOMEvents: {
          click: (view, e) => {
            const piece = (e.target as HTMLElement).closest?.('.fn, .cite');
            if (piece) { openPiece(view.posAtDOM(piece, 0)); return true; }
            const el = (e.target as HTMLElement).closest?.('[data-ref]');
            if (!el) return false;
            const [kind, id] = el.getAttribute('data-ref')!.split(':');
            if (e.altKey && onbeside) { onbeside(kind as LinkKind, +id); return true; }
            if (!(e.ctrlKey || e.metaKey) || !onopenref) return false;
            onopenref(kind as LinkKind, +id);
            return true;
          },
        },
      },
      onUpdate: ({ editor }) => { tick++; onchange(editor.isEmpty ? '' : editor.getHTML()); checkSuggest(); keepCentred(); },
      onSelectionUpdate: ({ editor }) => { tick++; onfocuscomment?.(editor.getAttributes('comment').id ?? null); checkSuggest(); },
      onBlur: () => setTimeout(() => (sug = null), 150),
    });
    dropEmptyCaptions();
    window.addEventListener('scroll', follow, true); // capture: the page scrolls inside a panel
    const onKeyUp = (e: KeyboardEvent) => { if (NAV_KEYS.has(e.key)) keepCentred(); };
    host.addEventListener('keyup', onKeyUp);
    return () => { window.removeEventListener('scroll', follow, true); host.removeEventListener('keyup', onKeyUp); editor?.destroy(); };
  });

  const is = (name: string, attrs?: object) => { tick; return !!editor?.isActive(name, attrs); };
  const run = () => editor!.chain().focus();
</script>

<div class="rich" class:autowidth={look.width === 'auto'} class:typewriter={look.typewriter} class:focusline={look.focusLine} style={lookVars(look)}>
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
    <button title="{word('Scene')} break" onclick={() => run().setHorizontalRule().run()}>―</button>
    <button title="Footnote (Ctrl+Alt+F): a note at the foot of the page, or at the end of the chapter" onclick={addFootnote}>¹ Footnote</button>
    {#if has('sources')}<button title="Cite a source" onclick={addCitation}>❞ Cite</button>{/if}
    {#if has('sources') || has('ideas')}
      <span class="sep"></span>
      <Select aria-label="Callout box" style="flex:0 0 7.5rem" value={boxKind} onchange={(e) => chooseBox(e.currentTarget.value)}>
        <option value="">Box…</option>
        {#each BOXES as [k, name]}<option value={k}>{name}</option>{/each}
        {#if boxKind}<option value="">Take the box off</option>{/if}
      </Select>
      <Select aria-label="Caption" style="flex:0 0 7.5rem" value={captionKind} onchange={(e) => chooseCaption(e.currentTarget.value)}>
        <option value="">Caption…</option>
        {#each CAPTIONS as [k, name]}<option value={k}>{name}</option>{/each}
        {#if captionKind}<option value="">Plain text</option>{/if}
      </Select>
    {/if}
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

{#if editor}
  <SelectionMenu {editor} redraw={tick} {source} {onfind} oncomment={oncomment ? addComment : undefined} oncite={addCitation} onfootnote={addFootnote} />
{/if}

{#if piece}
  <div class="piece" transition:pop style:left="{piece.x}px" style:top={piece.top != null ? piece.top + 'px' : null} style:bottom={piece.bottom != null ? piece.bottom + 'px' : null}
    role="dialog" aria-label={piece.kind === 'fn' ? 'Footnote' : 'Citation'} tabindex="-1"
    onkeydown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); closePiece(); } }}>
    {#if piece.kind === 'fn'}
      <label for="fn-text">Footnote</label>
      <textarea id="fn-text" bind:this={pieceInput} rows="3" value={piece.text} oninput={(e) => setFootnote(e.currentTarget.value)} placeholder="The text of the note"></textarea>
      <div class="pbtns"><button class="small primary" onclick={closePiece}>Done</button>{#if piece.pos != null}<button class="small danger" onclick={removePiece}>Remove</button>{/if}</div>
    {:else}
      <label for="cite-src">Source</label>
      <Select id="cite-src" bind:value={piece.source}>
        {#each sourceList as s (s.id)}<option value={String(s.id)}>{sourceLine(s)}</option>{/each}
        <option value="new">+ A new source…</option>
      </Select>
      {#if piece.source === 'new'}
        <input bind:this={pieceInput} placeholder="Title" bind:value={piece.title} aria-label="Title of the new source" />
        <input placeholder="Author, as Last, First" bind:value={piece.authors} aria-label="Author" />
        <input placeholder="Year" bind:value={piece.year} aria-label="Year" />
      {/if}
      <label for="cite-loc">Page or place <small class="muted">(optional)</small></label>
      <input id="cite-loc" bind:value={piece.loc} placeholder="45 or 45-47 or ch. 3" onkeydown={(e) => { if (e.key === 'Enter') applyCitation(); }} />
      <div class="pbtns">
        <button class="small primary" disabled={piece.source === 'new' && !piece.title.trim()} onclick={applyCitation}>{piece.pos != null ? 'Update' : 'Insert'}</button>
        {#if piece.pos != null}<button class="small danger" onclick={removePiece}>Remove</button>{/if}
        <button class="small" onclick={closePiece}>Cancel</button>
      </div>
    {/if}
  </div>
{/if}

{#if sug}
  <div class="sug" transition:pop style:left="{sug.x}px" style:top={sug.top != null ? sug.top + 'px' : null} style:bottom={sug.bottom != null ? sug.bottom + 'px' : null}>
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
  .rich-page { flex: 1; width: 100%; max-width: var(--ed-width, none); margin: 0 auto; }

  .rich-page :global(.ProseMirror) {
    outline: none; min-height: 55vh; padding-bottom: 30vh;
    font-family: var(--ed-font); font-size: var(--ed-size); line-height: var(--ed-lh);
  }
  /* typewriter mode: room above and below, so the first and last lines can reach the middle */
  .typewriter .rich-page :global(.ProseMirror) { padding-top: 30vh; padding-bottom: 50vh; }
  .rich-page :global(.ProseMirror .dim) { transition: opacity .2s; }
  .focusline .rich-page :global(.ProseMirror .dim) { opacity: .3; }
  .rich-page :global(.ProseMirror p) { margin: 0 0 .9em; }
  .rich-page :global(.ProseMirror h1) { font-size: 1.9rem; margin: 1.4em 0 .6em; }
  .rich-page :global(.ProseMirror h2) { font-size: 1.5rem; margin: 1.2em 0 .5em; }
  .rich-page :global(.ProseMirror h3) { font-size: 1.2rem; margin: 1em 0 .4em; }
  .rich-page :global(.ProseMirror blockquote) {
    margin: 1em 0; padding-left: 1rem; border-left: 3px solid var(--accent); color: var(--muted); font-style: italic;
  }
  .rich-page :global(.ProseMirror ul), .rich-page :global(.ProseMirror ol) { padding-left: 1.6rem; margin: 0 0 .9em; }
  /* the editor puts a paragraph inside each list item, so it must not keep the paragraph gap */
  .rich-page :global(.ProseMirror li > p) { margin: 0; }
  .rich-page :global(.ProseMirror li + li) { margin-top: .25em; }
  .rich-page :global(.ProseMirror hr) { border: 0; text-align: center; margin: 1.5em 0; }
  .rich-page :global(.ProseMirror hr::after) { content: '* * *'; color: var(--muted); letter-spacing: .5em; }
  .rich-page :global(.ProseMirror .cmt) { background: color-mix(in srgb, var(--accent-2) 28%, transparent); border-bottom: 2px solid var(--accent-2); cursor: pointer; }
  .rich-page :global(.ProseMirror .ref) { color: var(--accent); border-bottom: 1px dotted var(--accent); }
  .sug { position: fixed; z-index: 60; width: 260px; background: var(--panel); border: 1px solid var(--line); border-radius: 8px; padding: .25rem; box-shadow: 0 10px 30px rgba(0, 0, 0, .4); }
  .sug button { display: flex; justify-content: space-between; width: 100%; text-align: left; background: none; border: 1px solid transparent; padding: .3rem .5rem; }
  .sug button.on { background: var(--panel-2); border-color: var(--line); }
  .sug small { color: var(--muted); }
  .piece { position: fixed; z-index: 61; width: 300px; background: var(--panel); border: 1px solid var(--line); border-radius: 8px; padding: .6rem; box-shadow: 0 10px 30px rgba(0, 0, 0, .4); display: grid; gap: .3rem; }
  .piece label { margin: 0; }
  .pbtns { display: flex; gap: .3rem; margin-top: .3rem; }
  .rich-page :global(.ProseMirror) { counter-reset: fn; }
  .rich-page :global(.ProseMirror .fn) { cursor: pointer; color: var(--accent); }
  .rich-page :global(.ProseMirror .fn::before) { counter-increment: fn; content: counter(fn); font-size: .7em; vertical-align: super; font-weight: 600; padding: 0 .1em; }
  .rich-page :global(.ProseMirror .callout) { border-left: 4px solid var(--accent-2); background: color-mix(in srgb, var(--accent-2) 10%, transparent); padding: .2rem .9rem; margin: 1em 0; border-radius: 0 6px 6px 0; }
  .rich-page :global(.ProseMirror .callout::before) { content: 'Key idea'; display: block; font-size: .75em; font-weight: 700; letter-spacing: .04em; text-transform: uppercase; color: var(--accent-2); margin-top: .3rem; }
  .rich-page :global(.ProseMirror .callout[data-callout='story']) { border-color: #8b5cf6; background: color-mix(in srgb, #8b5cf6 10%, transparent); }
  .rich-page :global(.ProseMirror .callout[data-callout='story']::before) { content: 'Story'; color: #8b5cf6; }
  .rich-page :global(.ProseMirror .callout[data-callout='example']) { border-color: #10b981; background: color-mix(in srgb, #10b981 10%, transparent); }
  .rich-page :global(.ProseMirror .callout[data-callout='example']::before) { content: 'Example'; color: #10b981; }
  .rich-page :global(.ProseMirror .callout[data-callout='exercise']) { border-color: #f59e0b; background: color-mix(in srgb, #f59e0b 10%, transparent); }
  .rich-page :global(.ProseMirror .callout[data-callout='exercise']::before) { content: 'Exercise'; color: #f59e0b; }
  .rich-page :global(.ProseMirror .callout[data-callout='warning']) { border-color: #ef4444; background: color-mix(in srgb, #ef4444 10%, transparent); }
  .rich-page :global(.ProseMirror .callout[data-callout='warning']::before) { content: 'Warning'; color: #ef4444; }
  .rich-page :global(.ProseMirror .callout[data-callout='quote']) { border-color: #6b7280; background: color-mix(in srgb, #6b7280 10%, transparent); }
  .rich-page :global(.ProseMirror .callout[data-callout='quote']::before) { content: 'Quote'; color: #6b7280; }
  .rich-page :global(.ProseMirror .caption) { text-align: center; font-style: italic; font-size: .92em; color: var(--muted); }
  .rich-page :global(.ProseMirror .caption::before) { content: 'Figure: '; font-weight: 700; }
  .rich-page :global(.ProseMirror .caption[data-caption='table']::before) { content: 'Table: '; }
  .rich-page :global(.ProseMirror .caption[data-caption='chart']::before) { content: 'Chart: '; }
  .rich-page :global(.ProseMirror .idx) { border-bottom: 2px dotted var(--accent); }
  .rich-page :global(.ProseMirror .needs) { background: color-mix(in srgb, var(--danger) 18%, transparent); border-bottom: 2px wavy var(--danger); }
  .rich-page :global(.ProseMirror .cite) { cursor: pointer; color: var(--accent-2); font-size: .92em; white-space: nowrap; border-bottom: 1px dotted var(--accent-2); }
  .rich-page :global(.ProseMirror p.is-editor-empty:first-child::before) {
    content: attr(data-placeholder); color: var(--muted); float: left; height: 0; pointer-events: none;
  }
</style>
