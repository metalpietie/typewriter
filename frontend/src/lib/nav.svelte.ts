import type { Kind } from './db.svelte';

export type Tab = 'home' | 'write' | 'board' | 'characters' | 'relationships' | 'world' | 'factions' | 'items' | 'family' | 'checks' | 'stats' | 'maps' | 'timeline' | 'export';

/** Shared screen state, so search can jump to an item on another tab. */
export const nav = $state<{ tab: Tab; pending: { kind: Kind; id: number } | null; searchOpen: boolean; focus: boolean; scene: number | null; calendar: boolean; names: '' | 'person' | 'place'; projects: boolean; dataFolder: boolean }>({
  tab: 'home',
  pending: null,
  searchOpen: false,
  focus: false, // focus mode: only the text is on screen
  scene: null, // a scene the Write tab should open
  calendar: false, // the calendar editor is open
  names: '', // the name generator is open, starting on people or places
  projects: false, // the project list is open
  dataFolder: false, // the data folder window is open
});

export function openItem(kind: Kind, id: number) {
  nav.pending = { kind, id };
  nav.tab = kind === 'character' ? 'characters' : kind === 'faction' ? 'factions' : kind === 'item' ? 'items' : kind === 'event' ? 'timeline' : kind === 'pathpoint' ? 'maps' : 'world';
  nav.searchOpen = false;
}

/** Go to the Write tab and open one scene. */
export function openScene(id: number) {
  nav.scene = id;
  nav.tab = 'write';
  nav.searchOpen = false;
}
