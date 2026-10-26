import type { Kind } from './db.svelte';
import { isSecondWindow, windowTab } from './windows';

export type Tab = 'reference' | 'home' | 'draft' | 'write' | 'board' | 'arcs' | 'characters' | 'relationships' | 'world' | 'glossary' | 'factions' | 'items' | 'family' | 'checks' | 'stats' | 'maps' | 'timeline' | 'export' | 'sources' | 'ideas';

// The bookshelf shows when the app starts. After the app reloads itself (a project was opened), it does not show again.
const SEEN = 'typewriter.shelfSeen', START = 'typewriter.startTab';
function fromSession(key: string): string | null { try { return sessionStorage.getItem(key); } catch { return null; } }
function startTab(): Tab {
  if (isSecondWindow && windowTab) return windowTab as Tab; // a second window opens on the screen it was opened for
  const t = fromSession(START);
  try { sessionStorage.removeItem(START); } catch { /* fine */ }
  return t === 'write' ? 'write' : 'home';
}

/** Where the Write tab keeps the book, chapter and scene that were open last. Each project has its own. */
export const lastWriteKey = (projectId: string) => 'typewriter.write.' + projectId;

/** Close the bookshelf. tab: the screen to show next. */
export function closeShelf(tab?: Tab) {
  nav.shelf = false;
  if (tab) nav.tab = tab;
  try { sessionStorage.setItem(SEEN, '1'); } catch { /* fine */ }
}

/** Remember that the next load (after a reload) should not show the bookshelf, and which screen it opens on. */
export function reloadInto(tab: Tab) {
  try { sessionStorage.setItem(SEEN, '1'); sessionStorage.setItem(START, tab); } catch { /* fine */ }
}

/** Shared screen state, so search can jump to an item on another tab. */
export const nav = $state<{ tab: Tab; shelf: boolean; book: number | null; pending: { kind: Kind; id: number } | null; searchOpen: boolean; focus: boolean; scene: number | null; calendar: boolean; names: '' | 'person' | 'place'; projects: boolean; dataFolder: boolean; projectSettings: boolean; copy: { kind: string; id: number } | null; command: string | null; commands: boolean; insert: { kind: string; text: string; source: number | null; loc: string } | null }>({
  tab: startTab(),
  shelf: !isSecondWindow && fromSession(SEEN) == null, // the bookshelf is on screen (when the app starts)
  book: null, // a book the Write tab should open
  pending: null,
  searchOpen: false,
  focus: false, // focus mode: only the text is on screen
  scene: null, // a scene the Write tab should open
  calendar: false, // the calendar editor is open
  names: '', // the name generator is open, starting on people or places
  projects: false, // the project list is open
  dataFolder: false, // the data folder window is open
  projectSettings: false, // the project settings window (the type of project) is open
  copy: null, // the window that copies world items to another shelf is open, with this item ticked ({ kind: '', id: 0 } for none)
  command: null, // a command from the command menu that a screen should do (commands.ts)
  commands: false, // the search window opens on the commands (Ctrl+Shift+P)
  insert: null, // an idea or quote that the Write tab should put at the end of the open text
});

export function openItem(kind: Kind, id: number) {
  if (kind === 'scene') return openScene(id);
  if (nav.shelf) closeShelf(); // a search result picked on the bookshelf opens in front of it
  nav.pending = { kind, id };
  nav.tab = kind === 'character' ? 'characters' : kind === 'faction' ? 'factions' : kind === 'item' ? 'items' : kind === 'event' ? 'timeline' : kind === 'pathpoint' ? 'maps' : kind === 'draft' ? 'draft' : kind === 'term' ? 'glossary' : kind === 'arc' ? 'arcs' : kind === 'source' ? 'sources' : kind === 'idea' ? 'ideas' : 'world';
  nav.searchOpen = false;
}

/** Go to the Write tab and open one scene. */
export function openScene(id: number) {
  if (nav.shelf) closeShelf();
  nav.scene = id;
  nav.tab = 'write';
  nav.searchOpen = false;
}
