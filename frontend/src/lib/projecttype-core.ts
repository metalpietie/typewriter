// The type of a project (fiction, non-fiction or mixed): which tools show, and which words the screens use.
// No imports, so it can be tested alone. The state (what is saved) is in projecttype.svelte.ts.

export type ProjectType = 'fiction' | 'nonfiction' | 'mixed';

export const TYPES: [ProjectType, string, string][] = [
  ['fiction', 'Fiction', 'A story: characters, places, maps and a timeline.'],
  ['nonfiction', 'Non-fiction', 'A how-to, self-help, history or other book about real things: sections, sources and key points.'],
  ['mixed', 'Mixed', 'Both: every tool shows. Use this for a memoir or a story with real research behind it.'],
];

/** Every tool that can be hidden. The tabs that are always shown (Dashboard, Notes, Write, Board, Checks, Statistics, Export) are not here. */
export type Tool = 'characters' | 'relationships' | 'family' | 'factions' | 'world' | 'items' | 'maps' | 'timeline' | 'arcs' | 'names' | 'glossary' | 'sources' | 'ideas';

export const TOOLS: [Tool, string][] = [
  ['characters', 'Characters'], ['relationships', 'Relationships'], ['family', 'Family tree'], ['factions', 'Factions'],
  ['world', 'World (lore notes and places)'], ['items', 'Items'], ['maps', 'Maps & Paths'], ['timeline', 'Timeline'],
  ['arcs', 'Story arcs'], ['names', 'Name generator'], ['glossary', 'Glossary'], ['sources', 'Sources'], ['ideas', 'Ideas and quotes'],
];

const STORY: Tool[] = ['characters', 'relationships', 'family', 'factions', 'world', 'items', 'maps', 'timeline', 'arcs', 'names'];

/** The tools that are on in each type, before any tick is changed. */
export const PRESETS: Record<ProjectType, Tool[]> = {
  fiction: [...STORY, 'glossary'],
  nonfiction: ['glossary', 'sources', 'ideas'],
  mixed: [...STORY, 'glossary', 'sources', 'ideas'],
};

export type Ticks = Partial<Record<Tool, boolean>>;

const isTool = (v: string): v is Tool => TOOLS.some(([t]) => t === v);

/** Whatever was saved, made safe: a known type, and fiction when nothing was saved. */
export function cleanType(v: unknown): ProjectType {
  return TYPES.some(([t]) => t === v) ? (v as ProjectType) : 'fiction';
}

/** The ticks that differ from the preset. Anything that is not a known tool with a true or false is dropped. */
export function cleanTicks(v: unknown): Ticks {
  const out: Ticks = {};
  if (!v || typeof v !== 'object' || Array.isArray(v)) return out;
  for (const [k, on] of Object.entries(v as Record<string, unknown>)) if (isTool(k) && typeof on === 'boolean') out[k] = on;
  return out;
}

/** Is this tool on? The tick of the writer wins over the preset. */
export function toolOn(type: ProjectType, ticks: Ticks, tool: Tool): boolean {
  const t = ticks[tool];
  return typeof t === 'boolean' ? t : PRESETS[type].includes(tool);
}

/** The new ticks after one tool is switched: only a difference from the preset is kept. */
export function withTick(type: ProjectType, ticks: Ticks, tool: Tool, on: boolean): Ticks {
  const next = { ...ticks };
  if (PRESETS[type].includes(tool) === on) delete next[tool]; else next[tool] = on;
  return next;
}

/** The tool that owns a tab, or null for a tab that is always shown. */
export function toolOfTab(tab: string): Tool | null {
  return isTool(tab) ? tab : null;
}

/** The tool that owns a kind of item, or null when the kind is always there. */
export function toolOfKind(kind: string): Tool | null {
  switch (kind) {
    case 'character': return 'characters';
    case 'relationship': return 'relationships';
    case 'faction': case 'membership': case 'factionrel': return 'factions';
    case 'location': case 'note': return 'world';
    case 'item': case 'itemlog': return 'items';
    case 'map': case 'pathpoint': case 'mapdoc': case 'region': return 'maps';
    case 'event': return 'timeline';
    case 'arc': case 'arcscene': return 'arcs';
    case 'term': return 'glossary';
    case 'source': return 'sources';
    case 'idea': return 'ideas';
    default: return null;
  }
}

export type MenuGroup = { title?: string; items: [string, string][] };

const LABEL: Record<string, string> = {
  home: 'Dashboard', draft: 'Notes', write: 'Write', board: 'Board', arcs: 'Story arcs', characters: 'Characters', relationships: 'Relationships',
  family: 'Family tree', factions: 'Factions', world: 'World', glossary: 'Glossary', items: 'Items', maps: 'Maps & Paths', timeline: 'Timeline',
  sources: 'Sources', ideas: 'Ideas and quotes', checks: 'Checks', stats: 'Statistics', export: 'Export & Backup',
};

/** The menu for a project: the tabs that are on, in groups. A group with no tab is left out. */
export function menuFor(type: ProjectType, ticks: Ticks): MenuGroup[] {
  const nf = type === 'nonfiction';
  const on = (t: string) => { const tool = toolOfTab(t); return !tool || toolOn(type, ticks, tool); };
  const pick = (ids: string[]): [string, string][] => ids.filter(on).map((id) => [id, id === 'board' ? word(type, 'Board') : LABEL[id]]);
  const groups: MenuGroup[] = [
    { items: [['home', LABEL.home]] },
    { title: word(type, 'Story'), items: pick(['draft', 'write', 'board', 'arcs']) },
    { title: 'People', items: pick(['characters', 'relationships', 'family', 'factions']) },
    { title: 'Worldbuilding', items: pick(nf ? ['world', 'items', 'maps', 'timeline'] : ['world', 'glossary', 'items', 'maps', 'timeline']) },
    { title: 'Research', items: pick(nf ? ['glossary', 'sources', 'ideas'] : ['sources', 'ideas']) },
    { title: 'Review', items: pick(['checks', 'stats', 'export']) },
  ];
  return groups.filter((g) => g.items.length > 0);
}

/** Every tab that is on. A screen that is hidden is not opened. */
export function tabsOn(type: ProjectType, ticks: Ticks): string[] {
  return menuFor(type, ticks).flatMap((g) => g.items.map(([id]) => id));
}

// The words. Fiction and mixed use the story words. Keys are in lower case; a key that starts with a capital gives a capital back.
const WORDS: Record<'story' | 'nonfiction', Record<string, string>> = {
  story: {
    scene: 'scene', scenes: 'scenes', story: 'story', board: 'board', summary: 'summary', chaptersummary: 'summary', mood: 'mood board',
    summaryhint: 'What happens in this scene?', chapterhint: 'What happens in this chapter?',
  },
  nonfiction: {
    scene: 'section', scenes: 'sections', story: 'book', board: 'outline board', summary: 'key point', chaptersummary: 'chapter promise', mood: 'ideas and pictures',
    summaryhint: 'What is the key point? What should the reader take away?', chapterhint: 'What will this chapter give the reader?',
  },
};

/** A word for the type of project. word('Scene') gives 'Scene' or 'Section'. An unknown key comes back as it was. */
export function word(type: ProjectType, key: string): string {
  const w = WORDS[type === 'nonfiction' ? 'nonfiction' : 'story'][key.toLowerCase()];
  if (w === undefined) return key;
  const first = key.charAt(0);
  return first !== first.toLowerCase() ? w.charAt(0).toUpperCase() + w.slice(1) : w;
}

/** Does the Section panel show the story-world fields (point of view, place, time, cast, arcs)? */
export const usesStoryPanel = (type: ProjectType): boolean => type !== 'nonfiction';
