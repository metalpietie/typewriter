import { db } from './db.svelte';
import { cleanTicks, cleanType, menuFor, tabsOn, toolOfKind, toolOn, usesStoryPanel, withTick, word as wordFor, type MenuGroup, type ProjectType, type Tool } from './projecttype-core';

// The type of the open project and the tools that are on. It is saved as two settings of the project, so it goes with backups.
// No setting means fiction, so every older project looks as it always did.

const row = (key: string) => db.lists.setting.find((s) => s.key === key);

export const projectType = (): ProjectType => cleanType(row('projectType')?.value);
const ticks = () => cleanTicks(row('projectTools')?.value);

export const isNonfiction = (): boolean => projectType() === 'nonfiction';
/** Is this tool on? */
export const has = (tool: Tool): boolean => toolOn(projectType(), ticks(), tool);
/** Is this kind of item shown (search, links, lists)? A kind with no owner tool is always shown. */
export const hasKind = (kind: string): boolean => { const t = toolOfKind(kind); return !t || has(t); };
/** The word for this type: word('scene') is 'scene' or 'section'. */
export const word = (key: string): string => wordFor(projectType(), key);
export const menu = (): MenuGroup[] => menuFor(projectType(), ticks());
export const tabOn = (tab: string): boolean => tabsOn(projectType(), ticks()).includes(tab);
export const storyPanel = (): boolean => usesStoryPanel(projectType());

async function put(key: string, value: unknown) {
  const old = row(key);
  await db.save('setting', { ...(old ?? {}), key, value });
}

/** Change the type. The ticks go back to the preset of the new type. Nothing is deleted. */
export async function setType(type: ProjectType) {
  if (type === projectType()) return;
  await put('projectType', type);
  await put('projectTools', {});
}

/** Switch one tool on or off. */
export async function setTool(tool: Tool, on: boolean) {
  await put('projectTools', withTick(projectType(), ticks(), tool, on));
}
