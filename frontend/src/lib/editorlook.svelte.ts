import { cleanLook, DEFAULT_LOOK, type Look } from './editorlook-core';

// How the writing page looks. It is a choice of this computer, like the colour theme, so it is not in the project data.
const KEY = 'typewriter-editor-look';

function load(): Look {
  try { return cleanLook(JSON.parse(localStorage.getItem(KEY) ?? 'null')); } catch { return { ...DEFAULT_LOOK }; }
}

export const look = $state<Look>(load());

export function setLook(patch: Partial<Look>) {
  Object.assign(look, cleanLook({ ...look, ...patch }));
  try { localStorage.setItem(KEY, JSON.stringify({ ...look })); } catch { /* the choice lasts until the app closes */ }
}

export const resetLook = () => setLook({ ...DEFAULT_LOOK });
