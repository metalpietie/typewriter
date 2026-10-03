import { Store } from '../../bindings/typewriter';
import type { Doc } from '../../bindings/typewriter/models.js';

export type Kind = 'book' | 'chapter' | 'character' | 'relationship' | 'location' | 'map' | 'pathpoint' | 'note' | 'event';
export const KINDS: Kind[] = ['book', 'chapter', 'character', 'relationship', 'location', 'map', 'pathpoint', 'note', 'event'];

// An item is its saved fields plus id, kind and updatedAt.
export type Item = { id: number; kind: Kind; updatedAt: string; [field: string]: any };

function parse(d: Doc): Item {
  let data = {};
  try { data = JSON.parse(d.data); } catch { /* keep empty */ }
  return { ...data, id: d.id, kind: d.kind as Kind, updatedAt: d.updatedAt };
}

class Db {
  lists = $state<Record<Kind, Item[]>>({
    book: [], chapter: [], character: [], relationship: [], location: [], map: [], pathpoint: [], note: [], event: [],
  });
  ready = $state(false);
  dataPath = $state('');
  private timers = new Map<string, ReturnType<typeof setTimeout>>();

  async load() {
    await Promise.all(KINDS.map(async (k) => {
      this.lists[k] = ((await Store.List(k)) ?? []).map(parse);
    }));
    this.dataPath = await Store.DataPath();
    this.ready = true;
  }

  get(kind: Kind, id: number | null | undefined): Item | undefined {
    return id ? this.lists[kind].find((i) => i.id === id) : undefined;
  }

  /** Create or update an item right now. */
  async save(kind: Kind, item: Partial<Item>): Promise<Item> {
    const { id, kind: _k, updatedAt: _u, ...data } = item;
    const saved = parse(await Store.Save({ id: id ?? 0, kind, data: JSON.stringify(data), updatedAt: '' }));
    const list = this.lists[kind];
    const at = list.findIndex((i) => i.id === saved.id);
    if (at >= 0) Object.assign(list[at], saved); else list.push(saved);
    return this.get(kind, saved.id)!;
  }

  /** Save after a short pause. Use this while the user types. */
  queue(kind: Kind, item: Item) {
    const key = kind + item.id;
    clearTimeout(this.timers.get(key));
    this.timers.set(key, setTimeout(() => { this.timers.delete(key); this.save(kind, item); }, 500));
  }

  async remove(kind: Kind, id: number) {
    await Store.Delete(kind, id);
    await this.load(); // deleting can remove linked items too
  }

  /** Look up a display field of a linked item. */
  name(kind: Kind, id: number | null | undefined, field = 'name') {
    return this.get(kind, id)?.[field] ?? '—';
  }
}

export const db = new Db();

export const REL_TYPES: Record<string, string> = {
  family: '#e0b25f', friend: '#6bbf8a', ally: '#5aa9d9', romance: '#e0689f',
  mentor: '#8f7bf0', rival: '#d98a5b', enemy: '#d9625b', other: '#9c93a6',
};
export const ROLES = ['protagonist', 'antagonist', 'supporting', 'minor'];
export const NOTE_CATEGORIES = ['History', 'Magic', 'Culture', 'Religion', 'Politics', 'Creatures', 'Technology', 'Language', 'Other'];
export const PLACE_TYPES = ['city', 'town', 'village', 'fortress', 'ruin', 'landmark', 'region', 'water', 'other'];

export function wordCount(text: string) {
  return (text.match(/\S+/g) ?? []).length;
}
