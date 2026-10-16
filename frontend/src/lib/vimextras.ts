import { Extension } from '@tiptap/core';
import { Plugin, TextSelection } from '@tiptap/pm/state';
import type { EditorView } from '@tiptap/pm/view';
import { changeLine, changeRange, deleteChar, deleteLine, deleteRange, moveDown, moveLeft, moveToFirstNonWhitespace, vimPluginKey, yankLine, yankRange, type CommandContext, type VimState } from 'tiptap-extension-vim';
import { textObject } from './textobjects-core';

// What the Vim add-on (tiptap-extension-vim) does not have, and that a writer needs:
//  - text objects: viw, ciw, daw, yi", ci(, dis, dip ... (d, c, y and v, then i or a, then the object)
//  - Ctrl+d and Ctrl+u (half a page), Ctrl+f and Ctrl+b (a page)
//  - the page follows the cursor when it moves with j, k, gg, G and so on
//  - Enter, Backspace, Delete and Tab do what they do in Vim, and never change the text outside the insert mode
//  - an Esc that the Vim keys use does not also reach the page (it would leave focus mode)
//  - keys typed with AltGr ({ [ ~ on many keyboards) reach the Vim keys
// It runs before the add-on, and it only takes the keys that it knows. Everything else goes on to the add-on.

/** Keys that are only a modifier, or the first half of a typed character. */
const MODIFIERS = new Set(['Shift', 'Control', 'Alt', 'AltGraph', 'Meta', 'CapsLock', 'NumLock', 'Dead', 'Process', 'Unidentified']);

/** The nearest box that scrolls. When there is none, the window scrolls. */
function scrollBox(el: HTMLElement): HTMLElement | null {
  for (let p = el.parentElement; p; p = p.parentElement) {
    const o = getComputedStyle(p).overflowY;
    if ((o === 'auto' || o === 'scroll') && p.scrollHeight > p.clientHeight) return p;
  }
  return null;
}

/** Bars that stay on top of the text while it scrolls (the toolbar, the word count at the bottom, the Vim label). */
const COVERS = '.rich-bar:not([hidden]), .statusbar, .vimmode';

/**
 * The part of the scrolling box where the text can be read, as top and bottom in window coordinates.
 * The bars that stay on top of the text are taken off, so a line is not hidden behind them.
 */
function visible(box: HTMLElement | null): { top: number; bottom: number } {
  const r = box ? box.getBoundingClientRect() : { top: 0, bottom: window.innerHeight };
  let top = Math.max(0, r.top), bottom = Math.min(window.innerHeight, r.bottom);
  const middle = (top + bottom) / 2;
  for (const el of (box ?? document).querySelectorAll<HTMLElement>(COVERS)) {
    const b = el.getBoundingClientRect();
    if (!b.height || b.bottom <= top || b.top >= bottom) continue;
    if (b.top >= middle) bottom = Math.min(bottom, b.top);   // a bar at the bottom
    else top = Math.max(top, b.bottom);                       // a bar at the top
  }
  return { top, bottom };
}

/** Move the page so the line of the cursor is on screen, with some room around it. */
export function showCursor(view: EditorView, margin = 28) {
  const box = scrollBox(view.dom);
  const head = view.state.selection.head;
  let c: { top: number; bottom: number };
  try { c = view.coordsAtPos(head); } catch { return; }
  const { top, bottom } = visible(box);
  let by = 0;
  if (c.top < top + margin) by = c.top - (top + margin);
  else if (c.bottom > bottom - margin) by = c.bottom - (bottom - margin);
  if (!by) return;
  if (box) box.scrollTop += by; else window.scrollBy(0, by);
}

/** The cursor of the normal mode sits on a character, so it cannot be after the last one. */
function onChar(doc: import('@tiptap/pm/model').Node, pos: number): number {
  const p = Math.max(0, Math.min(pos, doc.content.size));
  const $p = doc.resolve(p);
  if (!$p.parent.isTextblock) return TextSelection.near($p).from;
  const len = $p.parent.content.size;
  return len === 0 ? $p.start() : Math.min(p, $p.start() + len - 1);
}

/** The Vim commands of the add-on need a context. This makes one, the same way as the add-on does. */
function context(view: EditorView, editor: any): CommandContext {
  const updateVimState = (updates: Partial<VimState>) => {
    view.dispatch(view.state.tr.setMeta(vimPluginKey, updates));
    const s = vimPluginKey.getState(view.state);
    if (s && editor.storage.vim) Object.assign(editor.storage.vim, { mode: s.mode, keyBuffer: s.keyBuffer, count: s.count });
    if (updates.mode) view.dom.setAttribute('data-vim-mode', updates.mode);
  };
  return {
    editor, view, state: view.state, dispatch: (tr) => view.dispatch(tr),
    vimState: vimPluginKey.getState(view.state)!, updateVimState,
    resetBuffer: () => updateVimState({ keyBuffer: '', count: null, pendingOperator: null, operatorCount: null, pendingFind: null }),
  };
}

export const VimExtras = Extension.create({
  name: 'vimExtras',
  priority: 1000, // before the add-on (100), so these keys are seen first

  addProseMirrorPlugins() {
    const editor = this.editor;
    let wait: 'i' | 'a' | null = null; // i or a was pressed: the object key comes next

    /** Do an operator (or select, in the visual mode) on a text object. Returns false when there is no such object. */
    function applyObject(view: EditorView, vs: VimState, kind: 'i' | 'a', key: string): boolean {
      const visual = vs.mode === 'visual' || vs.mode === 'visual-line';
      const at = visual ? (vs.visualHead ?? view.state.selection.head) : view.state.selection.from;
      const $p = view.state.doc.resolve(at);
      if (!$p.parent.isTextblock) return false;
      const block = $p.parent;
      const count = (vs.operatorCount ?? 1) * (vs.count ?? 1);
      const ctx = context(view, editor);
      const op = visual ? 'v' : vs.pendingOperator;

      if (key === 'p') { // a paragraph is a whole block
        if (!block.content.size && op !== 'd') return false;
        if (op === 'd') deleteLine(ctx, count);
        else if (op === 'c') changeLine(ctx);
        else if (op === 'y') yankLine(ctx, count);
        else {
          view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, $p.start(), $p.end())));
          ctx.updateVimState({ mode: 'visual-line', visualAnchor: $p.start(), visualHead: $p.end() });
          ctx.resetBuffer();
        }
        return true;
      }

      // the text of the block, one character for each inline thing that is not text, so places match:
      // a line break (Shift+Enter) is white space, a citation or a footnote is a mark
      const text = block.textBetween(0, block.content.size, '', (leaf) => (leaf.type.name === 'hardBreak' ? '\n' : '\ufffc'));
      const r = textObject(text, at - $p.start(), kind, key, count);
      if (!r) return false;
      const from = $p.start() + r[0], to = $p.start() + r[1];
      if (op === 'd') deleteRange(ctx, from, to, false);
      else if (op === 'c') changeRange(ctx, from, to);
      else if (op === 'y') {
        yankRange(ctx, from, to);
        view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, onChar(view.state.doc, from))));
      } else if (to > from) { // the visual mode: the object becomes the selection
        view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, from, to)));
        ctx.updateVimState({ mode: 'visual', visualAnchor: from, visualHead: to - 1 });
        ctx.resetBuffer();
      } else ctx.resetBuffer();
      return true;
    }

    /**
     * Ctrl+d, Ctrl+u (half a page) and Ctrl+f, Ctrl+b (a page), as in Vim: the page scrolls, and the cursor stays on the same
     * row of the screen, so it moves the same number of lines. At the end of the text the cursor goes on to the last line.
     */
    function page(view: EditorView, vs: VimState, down: boolean, fraction: number) {
      const box = scrollBox(view.dom);
      const vis = visible(box);
      const delta = (down ? 1 : -1) * (vis.bottom - vis.top) * fraction;
      const visual = vs.mode === 'visual' || vs.mode === 'visual-line';
      const doc = view.state.doc;
      const from = visual ? (vs.visualHead ?? view.state.selection.head) : view.state.selection.from;
      let c: { left: number; top: number; bottom: number };
      try { c = view.coordsAtPos(from); } catch { return; }
      const before = box ? box.scrollTop : window.scrollY;
      if (box) box.scrollTop += delta; else window.scrollBy(0, delta);
      const scrolled = (box ? box.scrollTop : window.scrollY) - before;
      // the same row of the screen, and further when the page could not scroll all the way
      const y = Math.max(vis.top + 1, Math.min(vis.bottom - 1, (c.top + c.bottom) / 2 + (delta - scrolled)));
      const hit = view.posAtCoords({ left: c.left, top: y });
      let pos = hit ? hit.pos : down ? doc.content.size : 0;
      pos = onChar(doc, TextSelection.near(doc.resolve(Math.max(0, Math.min(pos, doc.content.size)))).from);
      const ctx = context(view, editor);
      if (vs.mode === 'visual') {
        const anchor = vs.visualAnchor ?? view.state.selection.anchor;
        const sel = pos >= anchor ? TextSelection.create(doc, anchor, Math.min(doc.resolve(pos).end(), pos + 1)) : TextSelection.create(doc, Math.min(doc.resolve(anchor).end(), anchor + 1), pos);
        view.dispatch(view.state.tr.setSelection(sel));
        ctx.updateVimState({ desiredColumn: null, visualHead: pos });
      } else if (vs.mode === 'visual-line') {
        const anchor = vs.visualAnchor ?? view.state.selection.anchor;
        const $a = doc.resolve(anchor), $t = doc.resolve(pos);
        const sel = $t.start() >= $a.start() ? TextSelection.create(doc, $a.start(), $t.end()) : TextSelection.create(doc, $a.end(), $t.start());
        view.dispatch(view.state.tr.setSelection(sel));
        ctx.updateVimState({ desiredColumn: null, visualHead: pos });
      } else {
        view.dispatch(view.state.tr.setSelection(TextSelection.create(doc, pos)));
        ctx.updateVimState({ desiredColumn: null });
      }
      ctx.resetBuffer(); // a d or c that waited is dropped, as in Vim
    }

    /**
     * Enter, Backspace, Delete and Tab outside the insert mode. Without this, the editor's own keys would split, join or
     * delete text. In the normal mode they do what Vim does: Enter goes to the start of the next line, Backspace goes left,
     * Delete deletes the character. In the visual mode Delete deletes the selection. Anything else does nothing.
     */
    function editKey(view: EditorView, vs: VimState, key: string) {
      const ctx = context(view, editor);
      const visual = vs.mode === 'visual' || vs.mode === 'visual-line';
      const count = vs.count ?? 1;
      if (vs.pendingOperator || vs.pendingFind) { ctx.resetBuffer(); return; }
      if (visual) {
        if (key === 'Delete') {
          const { from, to } = view.state.selection;
          deleteRange(ctx, from, to, vs.mode === 'visual-line');
          ctx.updateVimState({ mode: 'normal', visualAnchor: null, visualHead: null });
        }
        return;
      }
      const state = view.state, at = state.selection.from;
      if (key === 'Enter') {
        const down = moveDown(view, state, at, count, null).pos;
        view.dispatch(state.tr.setSelection(TextSelection.create(state.doc, onChar(state.doc, moveToFirstNonWhitespace(state, down)))));
        ctx.updateVimState({ desiredColumn: null });
        ctx.resetBuffer();
      } else if (key === 'Backspace') {
        view.dispatch(state.tr.setSelection(TextSelection.create(state.doc, moveLeft(state, at, count))));
        ctx.resetBuffer();
      } else if (key === 'Delete') {
        deleteChar(ctx, count);
      } else ctx.resetBuffer();
    }

    return [new Plugin({
      props: {
        handleKeyDown(view, e) {
          const vs = vimPluginKey.getState(view.state);
          if (!vs) return false;
          const visual = vs.mode === 'visual' || vs.mode === 'visual-line';
          if (e.key === 'Escape') {
            // An Esc that ends the insert or visual mode, or drops a command, is for the Vim keys only. It must not also
            // reach the page, where it would leave focus mode. Esc in the normal mode with nothing waiting goes on as usual.
            if (vs.mode !== 'normal' || vs.keyBuffer || vs.pendingOperator || vs.pendingFind || wait) e.stopPropagation();
            wait = null;
            return false;
          }
          if (vs.mode === 'insert') { wait = null; return false; }
          if (e.metaKey) return false;
          if (e.altKey) {
            // AltGr is Ctrl+Alt on Windows. A character typed with it ({ [ ~ on many keyboards) is given to the Vim keys
            // as a plain key, because the add-on ignores every key that has Alt.
            if (!e.ctrlKey || e.key.length !== 1) return false;
            const plain = new KeyboardEvent('keydown', { key: e.key });
            const done = view.someProp('handleKeyDown', (f) => f(view, plain));
            if (done) e.preventDefault();
            return !!done;
          }
          // A key like " or $ is typed with Shift, and the browser sends the Shift key first. The add-on takes that as a key that
          // means nothing, and drops the command that waits (d$, f(, vi" ...). So a modifier key is not passed on.
          if (MODIFIERS.has(e.key)) return true;

          if (e.ctrlKey) {
            if (e.shiftKey) return false; // Ctrl+Shift+F and the like are the app's own keys
            const k = e.key.toLowerCase();
            if (k !== 'd' && k !== 'u' && k !== 'f' && k !== 'b') return false;
            if (k === 'b' && visual) return false; // in the visual mode Ctrl+B makes the selection bold
            e.preventDefault();
            wait = null;
            page(view, vs, k === 'd' || k === 'f', k === 'd' || k === 'u' ? 0.5 : 0.9);
            return true;
          }

          if (e.key === 'Enter' || e.key === 'Backspace' || e.key === 'Delete' || e.key === 'Tab') {
            e.preventDefault();
            wait = null;
            editKey(view, vs, e.key);
            return true;
          }

          if (wait) { // the key after i or a
            const kind = wait;
            wait = null;
            e.preventDefault();
            const ok = e.key.length === 1 && applyObject(view, vs, kind, e.key);
            if (!ok) context(view, editor).resetBuffer(); // not an object: drop the command, as Vim does
            return true;
          }

          const operator = vs.pendingOperator === 'd' || vs.pendingOperator === 'c' || vs.pendingOperator === 'y';
          if ((e.key === 'i' || e.key === 'a') && (operator || visual) && !vs.pendingFind) {
            e.preventDefault();
            wait = e.key;
            context(view, editor).updateVimState({ keyBuffer: (vs.keyBuffer || '') + e.key });
            return true;
          }
          return false;
        },
      },
      // the page follows the cursor when it moves in the normal and visual modes
      view: () => ({
        update(view, prev) {
          const vs = vimPluginKey.getState(view.state);
          if (!vs || vs.mode === 'insert' || prev.selection.eq(view.state.selection)) return;
          showCursor(view);
        },
      }),
    })];
  },
});
