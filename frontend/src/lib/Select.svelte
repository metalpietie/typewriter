<script lang="ts">
  import { onMount, tick, type Snippet } from 'svelte';
  import { pop } from './motion.svelte';
  import type { HTMLSelectAttributes } from 'svelte/elements';

  // A drop-down that looks like the app. Use it like a <select>: put <option> tags inside,
  // use bind:value, and use onchange. The real <select> stays hidden inside, so the value
  // and the change event work the way they always did.
  let { value = $bindable(), children, class: klass = '', style = '', title, id, disabled = false, onchange, 'aria-label': ariaLabel }: {
    value?: any;
    children?: Snippet;
    class?: string;
    style?: string;
    title?: string;
    id?: string;
    disabled?: boolean;
    onchange?: HTMLSelectAttributes['onchange'];
    'aria-label'?: string;
  } = $props();

  let el: HTMLSelectElement;
  let btn: HTMLButtonElement;
  let menu: HTMLElement | undefined = $state();

  let label = $state('');
  let ver = $state(0); // goes up when the options change
  let open = $state(false);
  let items = $state<{ i: number; text: string; disabled: boolean }[]>([]);
  let active = $state(0);
  let cur = $state(-1); // the picked item while the menu is open
  let pos = $state({ left: 0, minWidth: 0, maxHeight: 300, top: undefined as number | undefined, bottom: undefined as number | undefined });
  let typed = '', typedAt = 0;

  const sync = () => { label = el?.selectedOptions[0]?.textContent?.trim() ?? ''; };
  $effect(() => { value; ver; sync(); });

  onMount(() => {
    sync();
    const watch = new MutationObserver(() => { ver++; if (open) items = read(); });
    watch.observe(el, { childList: true, subtree: true, characterData: true, attributes: true, attributeFilter: ['value', 'disabled'] });
    return () => watch.disconnect();
  });

  const read = () => [...el.options].map((o, i) => ({ i, text: o.textContent?.trim() ?? '', disabled: o.disabled }));

  /** Put the menu under the button, or above it when there is no room below. */
  function place() {
    const r = btn.getBoundingClientRect();
    const below = window.innerHeight - r.bottom - 8, above = r.top - 8;
    const want = Math.min(items.length * 32 + 12, 320);
    const up = below < want && above > below;
    pos = {
      left: Math.max(8, Math.min(r.left, window.innerWidth - Math.max(r.width, 180) - 8)),
      minWidth: r.width,
      maxHeight: Math.max(120, Math.min(320, up ? above : below)),
      top: up ? undefined : r.bottom + 4,
      bottom: up ? window.innerHeight - r.top + 4 : undefined,
    };
  }

  async function show() {
    if (disabled || open) return;
    items = read();
    if (!items.length) return;
    cur = el.selectedIndex;
    active = Math.max(0, items.findIndex((it) => it.i === cur));
    place();
    open = true;
    await tick();
    menu?.querySelector('.on')?.scrollIntoView({ block: 'nearest' });
  }
  const hide = () => { open = false; };

  function choose(n: number) {
    const it = items[n];
    if (!it || it.disabled) return;
    hide();
    btn.focus();
    if (it.i === cur) return;
    el.selectedIndex = it.i;
    el.dispatchEvent(new Event('change', { bubbles: true }));
  }

  /** Move to the next item that can be picked. */
  function step(from: number, dir: 1 | -1) {
    for (let n = from + dir; n >= 0 && n < items.length; n += dir) if (!items[n].disabled) return n;
    return from;
  }
  function goto(n: number) {
    active = n;
    menu?.querySelector('.on')?.scrollIntoView({ block: 'nearest' });
  }

  async function keys(e: KeyboardEvent) {
    if (!open) {
      if (['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(e.key)) { e.preventDefault(); show(); }
      return;
    }
    if (e.key === 'ArrowDown') goto(step(active, 1));
    else if (e.key === 'ArrowUp') goto(step(active, -1));
    else if (e.key === 'Home') goto(step(-1, 1));
    else if (e.key === 'End') goto(step(items.length, -1));
    else if (e.key === 'Enter' || e.key === ' ') choose(active);
    else if (e.key === 'Escape') { e.stopPropagation(); hide(); } // do not also close the window behind
    else if (e.key === 'Tab') hide();
    else if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
      // type the first letters of an item to jump to it
      const now = Date.now();
      typed = now - typedAt > 700 ? e.key.toLowerCase() : typed + e.key.toLowerCase();
      typedAt = now;
      const from = typed.length === 1 ? active + 1 : active;
      const hit = [...items.keys()].map((_, k) => (k + from) % items.length).find((n) => !items[n].disabled && items[n].text.toLowerCase().startsWith(typed));
      if (hit !== undefined) goto(hit);
    } else return;
    e.preventDefault();
    await tick();
  }

  function away(e: Event) {
    const t = e.target as Node;
    if (open && !menu?.contains(t) && !btn.contains(t)) hide();
  }
  const scrolled = (e: Event) => { if (open && !menu?.contains(e.target as Node)) hide(); };

  /** Move the menu to <body>, so no parent can cut it off. */
  function portal(node: HTMLElement) {
    document.body.appendChild(node);
    return { destroy: () => node.remove() };
  }
</script>

<svelte:window onpointerdown={away} onscrollcapture={scrolled} onresize={hide} onblur={hide} />

<select bind:this={el} bind:value {disabled} {onchange} tabindex="-1" aria-hidden="true" style="display: none">{@render children?.()}</select>
<button type="button" bind:this={btn} class="select {klass}" {style} {title} {id} {disabled} aria-label={ariaLabel}
  aria-haspopup="listbox" aria-expanded={open} onclick={() => (open ? hide() : show())} onkeydown={keys} onblur={hide}>
  <span class="txt">{label || ' '}</span>
  <svg class="car" viewBox="0 0 10 6" aria-hidden="true"><path d="M1 1l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" /></svg>
</button>

{#if open}
  <!-- the button keeps the focus, so the keys work. Pressing in the menu must not take it away. -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="menu" role="listbox" tabindex="-1" transition:pop use:portal bind:this={menu} onmousedown={(e) => e.preventDefault()}
    style:left="{pos.left}px" style:min-width="{pos.minWidth}px" style:max-height="{pos.maxHeight}px"
    style:top={pos.top != null ? pos.top + 'px' : null} style:bottom={pos.bottom != null ? pos.bottom + 'px' : null}>
    {#each items as it, n (it.i)}
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_interactive_supports_focus -->
      <div role="option" aria-selected={it.i === cur} class="opt" class:on={n === active} class:sel={it.i === cur} class:dis={it.disabled}
        onpointermove={() => { if (!it.disabled) active = n; }} onclick={() => choose(n)}>
        <span>{it.text || ' '}</span>
        {#if it.i === cur}<svg viewBox="0 0 12 10" aria-hidden="true"><path d="M1 5l3.5 3.5L11 1.5" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" /></svg>{/if}
      </div>
    {/each}
  </div>
{/if}

<style>
  .select {
    display: flex; align-items: center; justify-content: space-between; gap: .5rem;
    width: 100%; text-align: left;
    /* In a row, share the room and shrink like a normal <select>. This must win over ".row > button". */
    flex: 1 1 auto; min-width: 0;
    background: var(--bg); border: 1px solid var(--line); border-radius: 6px; padding: .4rem .55rem;
  }
  .select:hover:not(:disabled) { border-color: color-mix(in srgb, var(--accent) 60%, var(--line)); }
  .select:focus-visible, .select[aria-expanded='true'] { outline: 1px solid var(--accent); border-color: var(--accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 22%, transparent); }
  .txt { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .car { flex: 0 0 auto; width: .6rem; height: .4rem; color: var(--muted); transition: transform var(--m-dur) var(--m-ease); }
  .select[aria-expanded='true'] .car { transform: rotate(180deg); color: var(--accent); }

  .menu {
    position: fixed; z-index: 200; overflow-y: auto; box-sizing: border-box;
    background: var(--panel); border: 1px solid var(--line); border-radius: 8px; padding: .25rem;
    box-shadow: 0 10px 30px rgba(0, 0, 0, .4);
  }
  .opt {
    display: flex; align-items: center; justify-content: space-between; gap: .6rem;
    padding: .35rem .55rem; border-radius: 6px; cursor: pointer; white-space: nowrap;
  }
  .opt span { overflow: hidden; text-overflow: ellipsis; }
  .opt.on { background: var(--panel-2); }
  .opt.sel { color: var(--accent); }
  .opt.dis { opacity: .4; cursor: default; }
  .opt svg { flex: 0 0 auto; width: .75rem; height: .65rem; }
</style>
