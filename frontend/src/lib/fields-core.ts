// Your own fields on characters, places, factions and items, such as "Magic affinity" or "Rank".
// Each item keeps its fields as fields: [{ k, v }]. A field can also be one that every item of its kind shows
// (a shared field): the project keeps those names in the setting "fieldTemplates". No imports, so it can be tested alone.

export type FieldKind = 'character' | 'location' | 'faction' | 'item';
export const FIELD_KINDS: FieldKind[] = ['character', 'location', 'faction', 'item'];
export type Templates = Record<FieldKind, string[]>;
export type OwnField = { k: string; v: string };
/** A field as the screen shows it. shared: every item of the kind has it. */
export type FieldRow = { k: string; v: string; shared: boolean; at: number }; // at: its place in item.fields, -1 when not there yet

const name = (s: unknown) => String(s ?? '').trim();
const same = (a: string, b: string) => a.trim().toLowerCase() === b.trim().toLowerCase();

/** The saved shared fields, made safe: for each kind a list of names, without empty names or doubles. */
export function cleanTemplates(v: unknown): Templates {
  const src = (v && typeof v === 'object' ? v : {}) as Record<string, unknown>;
  const out = {} as Templates;
  for (const kind of FIELD_KINDS) {
    const list = Array.isArray(src[kind]) ? (src[kind] as unknown[]) : [];
    const seen: string[] = [];
    for (const n of list.map(name)) if (n && !seen.some((s) => same(s, n))) seen.push(n);
    out[kind] = seen.slice(0, 40);
  }
  return out;
}

/** The fields of an item to show: the shared fields first (in their order), then its own other fields. */
export function fieldRows(fields: unknown, shared: string[]): FieldRow[] {
  const own: OwnField[] = Array.isArray(fields) ? fields.map((f: any) => ({ k: name(f?.k), v: String(f?.v ?? '') })) : [];
  const rows: FieldRow[] = shared.map((k) => {
    const at = own.findIndex((f) => same(f.k, k));
    return { k, v: at >= 0 ? own[at].v : '', shared: true, at };
  });
  own.forEach((f, at) => { if (!shared.some((k) => same(k, f.k))) rows.push({ ...f, shared: false, at }); });
  return rows;
}

/** The fields of an item with the value of field k set (a shared field is added to the item when it gets a value). */
export function withValue(fields: unknown, k: string, v: string): OwnField[] {
  const own: OwnField[] = Array.isArray(fields) ? fields.map((f: any) => ({ k: String(f?.k ?? ''), v: String(f?.v ?? '') })) : [];
  const at = own.findIndex((f) => same(f.k, k));
  if (at >= 0) own[at] = { ...own[at], v };
  else own.push({ k, v });
  return own;
}

/** The shared fields with one name added or taken out. */
export function toggleShared(t: Templates, kind: FieldKind, k: string): Templates {
  const list = t[kind];
  const has = list.some((s) => same(s, k));
  return { ...t, [kind]: has ? list.filter((s) => !same(s, k)) : [...list, name(k)].filter(Boolean) };
}
