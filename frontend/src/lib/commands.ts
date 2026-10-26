import { nav, closeShelf, type Tab } from './nav.svelte';
import { has, hasKind, menu, word } from './projecttype.svelte';
import { look, setLook } from './editorlook.svelte';
import { pomo, start as startPomodoro, pause as pausePomodoro } from './pomodoro.svelte';
import { checkForUpdates } from './updates.svelte';
import { LORE_TEMPLATES } from './loretemplates-core';

// The command menu: Ctrl+Shift+P, or > in the search box. Each command is one thing the app can do from the keyboard.
// A command for a screen goes there and leaves its name in nav.command; the screen does the work (see takeCommand).

export type Command = { id: string; label: string; group: string; keys?: string; run: () => void };

/** Go to a screen and ask it to do something (the screen reads it with takeCommand). */
function on(tab: Tab, name: string) {
  return () => { if (nav.shelf) closeShelf(); nav.tab = tab; nav.command = tab + ':' + name; };
}
const go = (tab: Tab) => () => { if (nav.shelf) closeShelf(); nav.tab = tab; };
const app = (name: string) => () => { nav.command = 'app:' + name; };

/** The commands that make sense in the open project. */
export function commands(): Command[] {
  const out: Command[] = [];
  const add = (c: Command | false) => { if (c) out.push(c); };
  const scene = word('scene');

  // making things
  add({ id: 'new-scene', group: 'New', label: `New ${scene} in this chapter`, run: on('write', 'new-scene') });
  add({ id: 'new-chapter', group: 'New', label: 'New chapter', run: on('write', 'new-chapter') });
  add({ id: 'new-note-page', group: 'New', label: 'New Notes page', run: on('draft', 'new') });
  add(hasKind('character') && { id: 'new-character', group: 'New', label: 'New character', run: on('characters', 'new') });
  add(has('world') && { id: 'new-place', group: 'New', label: 'New place', run: on('world', 'new-place') });
  add(has('world') && { id: 'new-lore', group: 'New', label: 'New lore note', run: on('world', 'new-note') });
  if (has('world')) for (const t of LORE_TEMPLATES) add({ id: 'template-' + t.id, group: 'New', label: `New lore note: ${t.name.toLowerCase()} (template)`, run: on('world', 'template-' + t.id) });
  add(has('factions') && { id: 'new-faction', group: 'New', label: 'New faction', run: on('factions', 'new') });
  add(has('items') && { id: 'new-item', group: 'New', label: 'New item', run: on('items', 'new') });
  add(has('glossary') && { id: 'new-term', group: 'New', label: 'New glossary word', run: on('glossary', 'new') });
  add(has('arcs') && { id: 'new-arc', group: 'New', label: 'New story arc', run: on('arcs', 'new') });
  add(has('sources') && { id: 'new-source', group: 'New', label: 'New source', run: on('sources', 'new') });
  add(has('ideas') && { id: 'new-idea', group: 'New', label: 'New idea or quote', run: on('ideas', 'new') });
  add(has('timeline') && { id: 'new-event', group: 'New', label: 'New timeline event', run: on('timeline', 'new') });

  // the text
  add({ id: 'split', group: 'Write', label: `Split the ${scene} at the cursor`, run: on('write', 'split') });
  add({ id: 'merge', group: 'Write', label: `Merge the ${scene} with the next one`, run: on('write', 'merge') });
  add({ id: 'find', group: 'Write', label: 'Find and replace', keys: 'Ctrl+H', run: on('write', 'find') });
  add({ id: 'history', group: 'Write', label: `History of this ${scene}`, run: on('write', 'history') });
  add({ id: 'snapshots', group: 'Write', label: 'Snapshots of the book', run: on('write', 'snapshots') });
  add({ id: 'focus', group: 'Write', label: nav.focus ? 'Leave focus mode' : 'Focus mode', keys: 'F11', run: on('write', 'focus') });
  add({ id: 'vim', group: 'Write', label: look.vim ? 'Vim keys: switch off' : 'Vim keys: switch on', run: () => setLook({ vim: !look.vim }) });
  add({ id: 'typewriter', group: 'Write', label: look.typewriter ? 'Typewriter mode: switch off' : 'Typewriter mode: switch on', run: () => setLook({ typewriter: !look.typewriter }) });
  add({ id: 'focusline', group: 'Write', label: look.focusLine ? 'Focus line: switch off' : 'Focus line: switch on', run: () => setLook({ focusLine: !look.focusLine }) });
  add({ id: 'pomodoro', group: 'Write', label: pomo.running ? 'Pause the Pomodoro timer' : 'Start the Pomodoro timer', run: () => (pomo.running ? pausePomodoro() : startPomodoro()) });

  // going places: every screen in the menu
  for (const g of menu()) for (const [id, label] of g.items) add({ id: 'go-' + id, group: 'Go to', label, run: go(id as Tab) });
  add({ id: 'go-shelf', group: 'Go to', label: 'Bookshelf', run: () => (nav.shelf = true) });

  // the app
  add(has('names') && { id: 'names', group: 'App', label: 'Name generator', run: () => (nav.names = 'person') });
  add(has('timeline') && { id: 'calendar', group: 'App', label: 'Your own calendar', run: () => (nav.calendar = true) });
  add({ id: 'projects', group: 'App', label: 'Manage shelves (projects)', run: () => (nav.projects = true) });
  add({ id: 'project-settings', group: 'App', label: 'Project type and tools', run: () => (nav.projectSettings = true) });
  add({ id: 'window', group: 'App', label: 'Open this screen in a new window', run: app('window') });
  add({ id: 'window-ref', group: 'App', label: 'Open the reference in a new window', run: app('window-ref') });
  add({ id: 'theme', group: 'App', label: 'Switch light or dark theme', run: app('theme') });
  add({ id: 'motion', group: 'App', label: 'Change the animations', run: app('motion') });
  add({ id: 'backup', group: 'App', label: 'Back up now', run: on('export', 'backup') });
  add({ id: 'updates', group: 'App', label: 'Check for updates', run: () => checkForUpdates() });
  return out;
}

/**
 * Does this screen have the command `name` waiting? Then it is taken (call it in an $effect). A command names its screen
 * (scope, the open tab unless told), so a screen that is just closing does not take a command for the next one.
 */
export function takeCommand(name: string, scope: string = nav.tab): boolean {
  if (nav.command !== scope + ':' + name) return false;
  nav.command = null;
  return true;
}

/**
 * The commands that match what was typed, best first. Every word must be found in the label or the group,
 * as a start of a word or anywhere.
 */
export function matchCommands(list: Command[], query: string): Command[] {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  if (!words.length) return list;
  const scored: [Command, number][] = [];
  for (const c of list) {
    const text = (c.label + ' ' + c.group).toLowerCase();
    let score = 0;
    for (const w of words) {
      const at = text.indexOf(w);
      if (at < 0) { score = -1; break; }
      score += at === 0 ? 3 : /\W/.test(text[at - 1]) ? 2 : 1;
    }
    if (score >= 0) scored.push([c, score]);
  }
  return scored.sort((a, b) => b[1] - a[1]).map(([c]) => c);
}
