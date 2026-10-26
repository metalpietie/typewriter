import { Windows } from '../../bindings/typewriter/internal/app';

// More windows of the app, for a second screen (windows.go). Each window is the whole app; the first one is "main".
// A window that is not the main one opens on the screen in its address (?window=…&tab=…), without the bookshelf.

const params = new URLSearchParams(location.search);

/** The name of this window: "main", or "window-2" and so on. */
export const WINDOW = params.get('window') || 'main';
export const isSecondWindow = WINDOW !== 'main';
/** The screen a second window opens on. */
export const windowTab = params.get('tab');

/** Open a new window on a screen. title is what the window is called, such as "Maps & Paths". */
export function openWindow(tab: string, title: string) {
  return Windows.OpenWindow(tab, title);
}
