import type { Kind } from './db.svelte';

export type Tab = 'home' | 'draft' | 'write' | 'board' | 'arcs' | 'characters' | 'relationships' | 'world' | 'glossary' | 'factions' | 'items' | 'family' | 'checks' | 'stats' | 'maps' | 'timeline' | 'export' | 'sources' | 'ideas';

/** Shared screen state, so search can jump to an item on another tab. */
export const nav = $state<{ tab: Tab; pending: { kind: Kind; id: number } | null; searchOpen: boolean; focus: boolean; scene: number | null; calendar: boolean; names: '' | 'person' | 'place'; projects: boolean; dataFolder: boolean; projectSettings: boolean; insert: { kind: string; text: string; source: number | null; loc: string } | null }>({
  tab: 'home',
  pending: null,
  searchOpen: false,
  focus: false, // focus mode: only the text is on screen
  scene: null, // a scene the Write tab should open
  calendar: false, // the calendar editor is open
  names: '', // the name generator is open, starting on people or places
  projects: false, // the project list is open
  dataFolder: false, // the data folder window is open
  projectSettings: false, // the project settings window (the type of project) is open
  insert: null, // an idea or quote that the Write tab should put at the end of the open text
});

export function openItem(kind: Kind, id: number) {
  if (kind === 'scene') return openScene(id);
  nav.pending = { kind, id };
  nav.tab = kind === 'character' ? 'characters' : kind === 'faction' ? 'factions' : kind === 'item' ? 'items' : kind === 'event' ? 'timeline' : kind === 'pathpoint' ? 'maps' : kind === 'draft' ? 'draft' : kind === 'term' ? 'glossary' : kind === 'arc' ? 'arcs' : kind === 'source' ? 'sources' : kind === 'idea' ? 'ideas' : 'world';
  nav.searchOpen = false;
}

/** Go to the Write tab and open one scene. */
export function openScene(id: number) {
  nav.scene = id;
  nav.tab = 'write';
  nav.searchOpen = false;
}
