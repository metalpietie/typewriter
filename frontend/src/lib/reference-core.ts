// The reference pane: something you pin next to the editor and read while you write. No imports, so it can be tested alone.

export type RefKind = 'scene' | 'draft' | 'character' | 'location' | 'note';

export const REF_KINDS: [RefKind, string][] = [
  ['scene', 'Scene'], ['draft', 'Notes page'], ['character', 'Character'], ['location', 'Place'], ['note', 'Lore note'],
];

export type RefPick = { kind: RefKind; id: number | null; open: boolean };

export const DEFAULT_PICK: RefPick = { kind: 'scene', id: null, open: false };

/** Whatever was saved, made safe. */
export function cleanPick(raw: unknown): RefPick {
  const r = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>;
  const kind = REF_KINDS.some(([k]) => k === r.kind) ? (r.kind as RefKind) : 'scene';
  const id = typeof r.id === 'number' && Number.isFinite(r.id) && r.id > 0 ? r.id : null;
  return { kind, id, open: r.open === true };
}

/** The kinds of item that an @ link can point at, as a reference kind. Other kinds give null. */
export function refKindOf(kind: string): RefKind | null {
  return kind === 'scene' || kind === 'draft' || kind === 'character' || kind === 'location' || kind === 'note' ? kind : null;
}

export type Field = { label: string; text: string };

const text = (v: unknown): string => (v === null || v === undefined ? '' : String(v).trim());
const add = (out: Field[], label: string, v: unknown) => { const t = text(v); if (t) out.push({ label, text: t }); };

/** The filled-in parts of a character, to read: role, names, description, traits and so on, then the custom fields. */
export function characterFields(c: Record<string, any>): Field[] {
  const out: Field[] = [];
  add(out, 'Role', c.role);
  add(out, 'Also called', c.aliases);
  add(out, 'Age', c.age);
  add(out, 'Looks and description', c.description);
  add(out, 'Personality', c.traits);
  add(out, 'Backstory', c.backstory);
  add(out, 'Goals and fears', c.goals);
  add(out, 'Notes', c.notes);
  for (const f of Array.isArray(c.fields) ? c.fields : []) add(out, text(f?.k) || 'Field', f?.v);
  return out;
}

/** The filled-in parts of a place. parent is the name of the place it sits in, if any. */
export function placeFields(l: Record<string, any>, parent = ''): Field[] {
  const out: Field[] = [];
  add(out, 'Type', l.type);
  add(out, 'Inside', parent);
  add(out, 'Description', l.description);
  return out;
}

/** The filled-in parts of a lore note. The text is plain text. */
export function loreFields(n: Record<string, any>): Field[] {
  const out: Field[] = [];
  add(out, 'Category', n.category);
  add(out, 'Tags', n.tags);
  add(out, 'Details', n.body);
  return out;
}
