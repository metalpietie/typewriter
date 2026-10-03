import type { Kind } from './db.svelte';

export type Tab = 'write' | 'characters' | 'relationships' | 'world' | 'maps' | 'timeline' | 'export';

/** Shared screen state, so search can jump to an item on another tab. */
export const nav = $state<{ tab: Tab; pending: { kind: Kind; id: number } | null; searchOpen: boolean }>({
  tab: 'write',
  pending: null,
  searchOpen: false,
});

export function openItem(kind: Kind, id: number) {
  nav.pending = { kind, id };
  nav.tab = kind === 'character' ? 'characters' : 'world';
  nav.searchOpen = false;
}
