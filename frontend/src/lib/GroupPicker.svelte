<script lang="ts">
  import { tick } from 'svelte';
  import { pop } from './motion.svelte';

  // Picks the group of a project: one of the groups there are, a new one (type its name), or none.
  // It looks like Select, but you can type in it. onchange gets the group ('' for none) when it changes.
  let { value = '', groups = [], disabled = false, id, title, onchange }: {
    value?: string; groups?: string[]; disabled?: boolean; id?: string; title?: string; onchange: (group: string) => void;
  } = $props();

  type Opt = { group: string; text: string; kind: 'none' | 'group' | 'new' };
  let text = $state(''); // filled from value by the effect below
  let open = $state(false);
  let active = $state(0);
  let input: HTMLInputElement;
  let menu: HTMLElement | undefined = $state();
  let pos = $state({ left: 0, width: 0, top: 0 as number | undefined, bottom: undefined as number | undefined, maxHeight: 260 });
  $effect(() => { if (!open) text = value; });
  const listId = $derived((id ?? 'group') + '-list'); // shows the saved group when the menu is closed

  const opts = $derived.by((): Opt[] => {
    const q = text.trim().toLowerCase();
    const list: Opt[] = [{ group: '', text: 'No group (a shelf of its own)', kind: 'none' }];
    const typed = q && q !== value.trim().toLowerCase();
    for (const g of groups) if (!typed || g.toLowerCase().includes(q)) list.push({ group: g, text: g, kind: 'group' });
    if (q && !groups.some((g) => g.toLowerCase() === q)) list.push({ group: text.trim(), text: `New group “${text.trim()}”`, kind: 'new' });
    return list;
  });

  function place() {
    const r = input.getBoundingClientRect();
    const below = window.innerHeight - r.bottom - 8, above = r.top - 8;
    const up = below < 160 && above > below;
    pos = { left: r.left, width: r.width, top: up ? undefined : r.bottom + 4, bottom: up ? window.innerHeight - r.top + 4 : undefined, maxHeight: Math.max(120, Math.min(260, up ? above : below)) };
  }
  async function show() {
    if (disabled || open) return;
    place();
    open = true;
    const at = opts.findIndex((o) => o.group === value);
    active = at < 0 ? 0 : at;
    await tick();
    menu?.querySelector('.on')?.scrollIntoView({ block: 'nearest' });
  }
  function pick(o: Opt | undefined) {
    open = false;
    if (!o) { text = value; return; }
    text = o.group;
    if (o.group !== value.trim()) onchange(o.group);
  }
  /** The typed name counts when the field is left: a group that is there, a new group, or none when it is empty. */
  function leave() {
    if (!open) return;
    const t = text.trim();
    pick(opts.find((o) => o.group.toLowerCase() === t.toLowerCase()) ?? (t ? { group: t, text: t, kind: 'new' } : opts[0]));
  }
  function keys(e: KeyboardEvent) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      if (!open) { show(); return; }
      active = (active + (e.key === 'ArrowDown' ? 1 : opts.length - 1)) % opts.length;
      tick().then(() => menu?.querySelector('.on')?.scrollIntoView({ block: 'nearest' }));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (open) pick(opts[active]); else show();
    } else if (e.key === 'Escape' && open) {
      e.stopPropagation(); // do not also close the window behind
      open = false; text = value;
    }
  }
  function typing() {
    if (!open) show();
    active = Math.min(active, opts.length - 1);
    const t = text.trim().toLowerCase();
    const best = opts.findIndex((o) => o.kind !== 'none' && o.group.toLowerCase().startsWith(t));
    if (t && best >= 0) active = best;
  }
  function portal(node: HTMLElement) {
    document.body.appendChild(node);
    return { destroy: () => node.remove() };
  }
</script>

<svelte:window onresize={() => { if (open) place(); }} onscrollcapture={(e) => { if (open && !menu?.contains(e.target as Node)) place(); }} />

<div class="field" class:open>
  <input bind:this={input} {id} {title} {disabled} bind:value={text} placeholder="No group (a shelf of its own)" autocomplete="off"
    role="combobox" aria-expanded={open} aria-controls={listId} aria-autocomplete="list"
    onfocus={show} onclick={show} oninput={typing} onkeydown={keys} onblur={leave} />
  <svg class="car" viewBox="0 0 10 6" aria-hidden="true"><path d="M1 1l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" /></svg>
</div>

{#if open}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="menu" id={listId} role="listbox" tabindex="-1" transition:pop use:portal bind:this={menu} onmousedown={(e) => e.preventDefault()}
    style:left="{pos.left}px" style:width="{pos.width}px" style:max-height="{pos.maxHeight}px"
    style:top={pos.top != null ? pos.top + 'px' : null} style:bottom={pos.bottom != null ? pos.bottom + 'px' : null}>
    {#each opts as o, n (o.kind + o.group)}
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_interactive_supports_focus -->
      <div role="option" aria-selected={o.group === value} class="opt" class:on={n === active} class:sel={o.group === value} class:muted={o.kind !== 'group'}
        onpointermove={() => (active = n)} onclick={() => pick(o)}>
        <span>{o.text}</span>
        {#if o.group === value}<svg viewBox="0 0 12 10" aria-hidden="true"><path d="M1 5l3.5 3.5L11 1.5" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" /></svg>{/if}
      </div>
    {/each}
  </div>
{/if}

<style>
  .field { position: relative; flex: 1 1 auto; min-width: 0; display: flex; }
  .field input {
    width: 100%; box-sizing: border-box; padding: .3rem 1.6rem .3rem .5rem; font-size: .85rem;
    background: var(--bg); border: 1px solid var(--line); border-radius: 6px;
  }
  .field input:hover:not(:disabled) { border-color: color-mix(in srgb, var(--accent) 60%, var(--line)); }
  .field.open input, .field input:focus-visible { outline: 1px solid var(--accent); border-color: var(--accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 22%, transparent); }
  .car { position: absolute; right: .55rem; top: 50%; translate: 0 -50%; width: .6rem; height: .4rem; color: var(--muted); pointer-events: none; transition: transform var(--m-dur) var(--m-ease); }
  .field.open .car { transform: rotate(180deg); color: var(--accent); }

  .menu {
    position: fixed; z-index: 200; overflow-y: auto; box-sizing: border-box;
    background: var(--panel); border: 1px solid var(--line); border-radius: 8px; padding: .25rem;
    box-shadow: 0 10px 30px rgba(0, 0, 0, .4); font-size: .85rem;
  }
  .opt { display: flex; align-items: center; justify-content: space-between; gap: .6rem; padding: .35rem .55rem; border-radius: 6px; cursor: pointer; white-space: nowrap; }
  .opt span { overflow: hidden; text-overflow: ellipsis; }
  .opt.on { background: var(--panel-2); }
  .opt.sel { color: var(--accent); }
  .opt.muted:not(.sel) { color: var(--muted); }
  .opt svg { flex: 0 0 auto; width: .75rem; height: .65rem; }
</style>
