import { Store } from '../../bindings/typewriter';
import type { Doc } from '../../bindings/typewriter/models.js';

export type Kind = 'book' | 'chapter' | 'character' | 'relationship' | 'location' | 'map' | 'pathpoint' | 'note' | 'event' | 'mapdoc' | 'scene' | 'comment' | 'faction' | 'membership' | 'factionrel' | 'item' | 'itemlog' | 'plotignore' | 'wordday' | 'setting' | 'todo' | 'region';
export const KINDS: Kind[] = ['book', 'chapter', 'character', 'relationship', 'location', 'map', 'pathpoint', 'note', 'event', 'mapdoc', 'scene', 'comment', 'faction', 'membership', 'factionrel', 'item', 'itemlog', 'plotignore', 'wordday', 'setting', 'todo', 'region'];

// An item is its saved fields plus id, kind and updatedAt.
export type Item = { id: number; kind: Kind; updatedAt: string; [field: string]: any };

function parse(d: Doc): Item {
  let data = {};
  try { data = JSON.parse(d.data); } catch { /* keep empty */ }
  return { ...data, id: d.id, kind: d.kind as Kind, updatedAt: d.updatedAt };
}

class Db {
  lists = $state<Record<Kind, Item[]>>({
    book: [], chapter: [], character: [], relationship: [], location: [], map: [], pathpoint: [], note: [], event: [], mapdoc: [], scene: [], comment: [], faction: [], membership: [], factionrel: [], item: [], itemlog: [], plotignore: [], wordday: [], setting: [], todo: [], region: [],
  });
  ready = $state(false);
  dataPath = $state('');
  private timers = new Map<string, { timer: ReturnType<typeof setTimeout>; kind: Kind; item: Item }>();

  // Save progress, shown by the save indicator.
  waiting = $state(0); // edits waiting for the short pause to end
  busy = $state(0); // saves running now
  lastSaved = $state<Date | null>(null);
  saveError = $state('');
  private failed = new Map<string, { kind: Kind; item: Item }>();

  /** What the save indicator should say. */
  get saveState(): 'saved' | 'saving' | 'error' {
    if (this.saveError) return 'error';
    return this.waiting + this.busy > 0 ? 'saving' : 'saved';
  }

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
    this.busy++;
    try {
      const saved = parse(await Store.Save({ id: id ?? 0, kind, data: JSON.stringify(data), updatedAt: '' }));
      const list = this.lists[kind];
      const at = list.findIndex((i) => i.id === saved.id);
      if (at >= 0) Object.assign(list[at], saved); else list.push(saved);
      this.failed.delete(kind + saved.id);
      if (this.failed.size === 0) this.saveError = '';
      this.lastSaved = new Date();
      return this.get(kind, saved.id)!;
    } catch (e: any) {
      this.saveError = e?.message ?? String(e);
      if (id) this.failed.set(kind + id, { kind, item: item as Item });
      throw e;
    } finally {
      this.busy--;
    }
  }

  /** Try again to save everything that failed. */
  async retry() {
    for (const { kind, item } of [...this.failed.values()]) await this.save(kind, item).catch(() => {});
  }

  /** Save every waiting edit now, for example when the window closes. */
  flush() {
    for (const [key, w] of [...this.timers]) {
      clearTimeout(w.timer);
      this.timers.delete(key);
      this.waiting--;
      this.save(w.kind, w.item).catch(() => {});
    }
  }

  /** Save after a short pause. Use this while the user types. */
  queue(kind: Kind, item: Item) {
    const key = kind + item.id;
    const old = this.timers.get(key);
    if (old) clearTimeout(old.timer); else this.waiting++;
    const timer = setTimeout(() => {
      this.timers.delete(key);
      this.save(kind, item).catch(() => {}); // the error is shown by the save indicator
      this.waiting--;
    }, 500);
    this.timers.set(key, { timer, kind, item });
  }

  /** What the last delete removed, kept for a short time so it can be undone. */
  undoable = $state<{ label: string; docs: Doc[] } | null>(null);
  private undoTimer: ReturnType<typeof setTimeout> | undefined;

  async remove(kind: Kind, id: number, undo = true) {
    const it = this.get(kind, id);
    const docs = (await Store.Delete(kind, id)) ?? [];
    await this.load(); // deleting can remove linked items too
    if (!docs.length || !undo) return;
    const name = it?.name ?? it?.title ?? (kind === 'pathpoint' ? 'path step' : kind);
    const extra = docs.filter((d) => d.kind !== kind || d.id !== id).length;
    this.undoable = { label: `Deleted ${kind === 'pathpoint' ? '' : kind + ' '}“${name}”${extra ? ` and ${extra} linked item${extra === 1 ? '' : 's'}` : ''}`, docs };
    clearTimeout(this.undoTimer);
    this.undoTimer = setTimeout(() => (this.undoable = null), 12000);
  }

  async undo() {
    const u = this.undoable;
    if (!u) return;
    this.undoable = null;
    clearTimeout(this.undoTimer);
    await Store.Restore(u.docs);
    await this.load();
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
export const CHARACTER_COLORS = ['#2cd5e8', '#e0b25f', '#6bbf8a', '#5aa9d9', '#e0689f', '#8f6bd1', '#d98a5b', '#d9625b', '#b5c95b'];
export const ROLES = ['protagonist', 'antagonist', 'supporting', 'minor'];
export const NOTE_CATEGORIES = ['History', 'Magic', 'Culture', 'Religion', 'Politics', 'Creatures', 'Technology', 'Language', 'Other'];
export const PLACE_TYPES = ['city', 'town', 'village', 'fortress', 'ruin', 'landmark', 'region', 'water', 'other'];

const looksLikeHtml = (s: string) => s.trimStart().startsWith('<');

/** Chapter text is HTML now. Older chapters are plain text, so accept both. */
export function toHtml(body: string): string {
  if (!body || looksLikeHtml(body)) return body || '';
  const esc = (t: string) => t.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  return body
    .split(/\r?\n/)
    .filter((l) => l.trim())
    .map((l) => `<p>${esc(l.trim())}</p>`)
    .join('');
}

/** Plain text of a chapter, for counting words. */
export function plainText(body: string): string {
  if (!looksLikeHtml(body)) return body ?? '';
  const spaced = body.replace(/<\/(p|h[1-6]|li|blockquote)>|<br\s*\/?>|<hr\s*\/?>/gi, ' ');
  return new DOMParser().parseFromString(spaced, 'text/html').body.textContent ?? '';
}

export function wordCount(body: string) {
  return (plainText(body).match(/\S+/g) ?? []).length;
}

export const FACTION_TYPES = ['kingdom', 'empire', 'house', 'guild', 'order', 'army', 'cult', 'company', 'gang', 'other'];
export const FACTION_REL: Record<string, string> = { ally: '#6bbf8a', enemy: '#d9625b', rival: '#d98a5b', rules: '#8f7bf0', neutral: '#9c93a6' };
export const ITEM_TYPES = ['weapon', 'armor', 'relic', 'jewel', 'tool', 'book', 'document', 'vehicle', 'other'];
