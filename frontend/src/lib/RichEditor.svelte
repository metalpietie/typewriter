<script lang="ts">
  import { onMount } from 'svelte';
  import { Editor } from '@tiptap/core';
  import StarterKit from '@tiptap/starter-kit';
  import Placeholder from '@tiptap/extension-placeholder';
  import SaveIndicator from './SaveIndicator.svelte';

  let { value, onchange, placeholder = 'Start writing…' }: {
    value: string;
    onchange: (html: string) => void;
    placeholder?: string;
  } = $props();

  let host: HTMLElement;
  let editor: Editor | undefined;
  // Bumped on every editor change so the toolbar buttons redraw their active state.
  let tick = $state(0);

  onMount(() => {
    editor = new Editor({
      element: host,
      content: value, // parent remounts this component per chapter, so this is read once
      extensions: [
        StarterKit.configure({ strike: false, code: false, codeBlock: false, heading: { levels: [1, 2, 3] } }),
        Placeholder.configure({ placeholder }),
      ],
      onUpdate: ({ editor }) => { tick++; onchange(editor.isEmpty ? '' : editor.getHTML()); },
      onSelectionUpdate: () => tick++,
    });
    return () => editor?.destroy();
  });

  const is = (name: string, attrs?: object) => { tick; return !!editor?.isActive(name, attrs); };
  const run = () => editor!.chain().focus();
</script>

<div class="rich">
  <div class="rich-bar">
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
    <span class="sep"></span>
    <button title="Undo (Ctrl+Z)" onclick={() => run().undo().run()}>↶</button>
    <button title="Redo (Ctrl+Y)" onclick={() => run().redo().run()}>↷</button>
    <span class="rich-save"><SaveIndicator /></span>
  </div>
  <div class="rich-page" bind:this={host}></div>
</div>

<style>
  .rich { display: flex; flex-direction: column; min-height: 60vh; }
  .rich-bar {
    display: flex; gap: .25rem; align-items: center; flex-wrap: wrap;
    position: sticky; top: -1.2rem; z-index: 5; background: var(--bg);
    padding: .4rem 0; border-bottom: 1px solid var(--line); margin-bottom: .8rem;
  }
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
  .rich-page :global(.ProseMirror p.is-editor-empty:first-child::before) {
    content: attr(data-placeholder); color: var(--muted); float: left; height: 0; pointer-events: none;
  }
</style>
