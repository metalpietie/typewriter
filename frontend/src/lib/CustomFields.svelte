<script lang="ts">
  import { db, type Item } from './db.svelte';
  import { cleanTemplates, fieldRows, toggleShared, withValue, type FieldKind } from './fields-core';

  // Your own fields on a character, place, faction or item (fields-core.ts). ★ makes a field one that every item of
  // this kind has; the names of those are kept with the project.
  let { kind, item, save }: { kind: FieldKind; item: Item; save: () => void } = $props();

  const NOUN: Record<FieldKind, string> = { character: 'character', location: 'place', faction: 'faction', item: 'item' };
  const row = $derived(db.lists.setting.find((s) => s.key === 'fieldTemplates'));
  const templates = $derived(cleanTemplates(row?.value));
  const rows = $derived(fieldRows(item.fields, templates[kind]));

  function setValue(k: string, v: string) { item.fields = withValue(item.fields, k, v); save(); }
  function setName(at: number, k: string) {
    const list = [...(item.fields ?? [])];
    list[at] = { ...list[at], k };
    item.fields = list; save();
  }
  function drop(at: number) { item.fields = (item.fields ?? []).filter((_: unknown, n: number) => n !== at); save(); }
  function add() { item.fields = [...(item.fields ?? []), { k: '', v: '' }]; save(); }
  async function share(k: string) {
    if (!k.trim()) return;
    await db.save('setting', { ...(row ?? {}), key: 'fieldTemplates', value: toggleShared(templates, kind, k) });
  }
</script>

<h3 style="margin-top:1.2rem">Your own fields</h3>
{#each rows as f (f.shared ? 'shared:' + f.k : 'own:' + f.at)}
  <div class="row field">
    {#if f.shared}
      <span class="name" title="Every {NOUN[kind]} has this field">{f.k}</span>
    {:else}
      <input placeholder="Field (e.g. Eye colour)" value={f.k} oninput={(e) => setName(f.at, e.currentTarget.value)} />
    {/if}
    <input placeholder="Value" value={f.v} oninput={(e) => setValue(f.k, e.currentTarget.value)} />
    <button class="small star" class:on={f.shared} disabled={!f.k.trim()}
      title={f.shared ? `Every ${NOUN[kind]} has this field. Click: only the ones with a value keep it.` : `Give every ${NOUN[kind]} this field`}
      onclick={() => share(f.k)}>{f.shared ? '★' : '☆'}</button>
    {#if !f.shared}<button class="small" title="Remove this field" onclick={() => drop(f.at)}>✕</button>{/if}
  </div>
{/each}
<button class="small" onclick={add}>+ Add field</button>
<span class="muted hint">☆ gives every {NOUN[kind]} the field.</span>

<style>
  .field { margin-bottom: .3rem; align-items: center; }
  .field .name { flex: 1; padding: .3rem .5rem; font-size: .9rem; color: var(--text); border: 1px dashed var(--line); border-radius: 6px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .field .star { flex: 0 0 auto; color: var(--muted); }
  .field .star.on { color: var(--accent); }
  .hint { font-size: .78rem; margin-left: .5rem; }
</style>
