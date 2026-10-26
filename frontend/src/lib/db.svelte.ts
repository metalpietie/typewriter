import { Events } from '@wailsio/runtime';
import { Store } from '../../bindings/typewriter/internal/app';
import type { Doc } from '../../bindings/typewriter/internal/app/models.js';
import { WINDOW } from './windows';

export type Kind = 'book' | 'chapter' | 'character' | 'relationship' | 'location' | 'map' | 'pathpoint' | 'note' | 'event' | 'mapdoc' | 'scene' | 'comment' | 'faction' | 'membership' | 'factionrel' | 'item' | 'itemlog' | 'plotignore' | 'wordday' | 'setting' | 'todo' | 'region' | 'draft' | 'arc' | 'arcscene' | 'term' | 'pass' | 'source' | 'idea' | 'syncnote';
export const KINDS: Kind[] = ['book', 'chapter', 'character', 'relationship', 'location', 'map', 'pathpoint', 'note', 'event', 'mapdoc', 'scene', 'comment', 'faction', 'membership', 'factionrel', 'item', 'itemlog', 'plotignore', 'wordday', 'setting', 'todo', 'region', 'draft', 'arc', 'arcscene', 'term', 'pass', 'source', 'idea', 'syncnote'];

// An item is its saved fields plus id, kind and updatedAt.
export type Item = { id: number; kind: Kind; updatedAt: string; [field: string]: any };

function parse(d: Doc): Item {
  let data = {};
  try { data = JSON.parse(d.data); } catch { /* keep empty */ }
  return { ...data, id: d.id, kind: d.kind as Kind, updatedAt: d.updatedAt };
}

class Db {
  lists = $state<Record<Kind, Item[]>>({
    book: [], chapter: [], character: [], relationship: [], location: [], map: [], pathpoint: [], note: [], event: [], mapdoc: [], scene: [], comment: [], faction: [], membership: [], factionrel: [], item: [], itemlog: [], plotignore: [], wordday: [], setting: [], todo: [], region: [], draft: [], arc: [], arcscene: [], term: [], pass: [], source: [], idea: [], syncnote: [],
  });
  ready = $state(false);
  dataPath = $state('');
  project = $state(''); // name of the open project
  projectId = $state(''); // id of the open project
  version = $state(''); // the app version
  folderProblem = $state(''); // set when the chosen data folder could not be opened
  lastBackup = $state(''); // when you last saved a copy with the Backup button, such as 2026-10-04T15:04:05Z. Empty: never
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

  /**
   * Read everything again. quiet: this window follows a change of another window, so it does not tell the others.
   * Otherwise the other windows read everything again too (a delete or an undo can change many items).
   */
  async load(quiet = false) {
    await Promise.all(KINDS.map(async (k) => {
      this.lists[k] = ((await Store.List(k)) ?? []).map(parse);
    }));
    this.dataPath = await Store.DataPath();
    this.version = await Store.Version();
    this.folderProblem = (await Store.DataFolder()).problem;
    this.lastBackup = await Store.LastBackup().catch(() => '');
    const open = ((await Store.Projects()) ?? []).find((p) => p.current);
    this.project = open?.name ?? '';
    this.projectId = open?.id ?? '';
    this.ready = true;
    if (!quiet) Events.Emit('typewriter:reload', { from: WINDOW }).catch(() => {});
  }

  /** The time until which this window expects the data to change under it, because it asked for that (see expectReopen). */
  private expecting = 0;
  /** This window is about to open another project, another folder or a backup: it reloads itself, and the others follow. */
  expectReopen() { this.expecting = Date.now() + 60_000; }
  /** Did this window ask for the change that was just made under the windows? */
  askedForReopen() { return this.expecting > Date.now(); }
  private discarded = false;
  /** Forget the edits that wait, and save nothing more: the data under this window changed (another project). */
  discard() {
    this.discarded = true;
    for (const w of this.timers.values()) clearTimeout(w.timer);
    this.timers.clear();
    this.waiting = 0;
  }

  /** Another window saved an item: take it, unless this window is changing it right now. */
  private follow(kind: Kind, doc: Doc) {
    if (!KINDS.includes(kind) || this.timers.has(kind + doc.id)) return;
    const item = parse(doc);
    const list = this.lists[kind];
    const at = list.findIndex((i) => i.id === item.id);
    if (at >= 0) Object.assign(list[at], item); else list.push(item);
  }
  /** Listen to the other windows (windows.ts). Called once, when the app starts. */
  listen() {
    Events.On('typewriter:doc', (e: any) => {
      const p = e?.data ?? e;
      if (p?.from !== WINDOW && p?.doc) this.follow(p.kind, p.doc);
    });
    Events.On('typewriter:reload', (e: any) => {
      const p = e?.data ?? e;
      if (p?.from !== WINDOW && this.ready) { this.flush(); this.load(true); }
    });
  }

  get(kind: Kind, id: number | null | undefined): Item | undefined {
    return id ? this.lists[kind].find((i) => i.id === id) : undefined;
  }

  /** Create or update an item right now. */
  async save(kind: Kind, item: Partial<Item>): Promise<Item> {
    const { id, kind: _k, updatedAt: _u, ...data } = item;
    this.busy++;
    try {
      const raw = await Store.Save({ id: id ?? 0, kind, data: JSON.stringify(data), updatedAt: '' });
      const saved = parse(raw);
      const list = this.lists[kind];
      const at = list.findIndex((i) => i.id === saved.id);
      if (at >= 0) Object.assign(list[at], saved); else list.push(saved);
      this.failed.delete(kind + saved.id);
      if (this.failed.size === 0) this.saveError = '';
      this.lastSaved = new Date();
      Events.Emit('typewriter:doc', { from: WINDOW, kind, doc: raw }).catch(() => {}); // the other windows take it
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
    if (this.discarded) return;
    for (const [key, w] of [...this.timers]) {
      clearTimeout(w.timer);
      this.timers.delete(key);
      this.waiting--;
      this.save(w.kind, w.item).catch(() => {});
    }
  }

  /**
   * Save every waiting edit now, and wait until the saves are done. Use it before the data file changes under the app
   * (another project opens, a book moves). It stops with an error when something could not be saved, so nothing is lost.
   */
  async settle() {
    this.flush();
    for (let i = 0; i < 100 && this.waiting + this.busy > 0; i++) await new Promise((r) => setTimeout(r, 50));
    if (this.waiting + this.busy > 0) throw new Error('Your last changes are still being saved. Try again in a moment.');
    if (this.saveError) throw new Error(`Some changes could not be saved (${this.saveError}). Fix that first, so nothing is lost.`);
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

// Reading the text of a scene is slow for a long scene, and every key you type asks again for every scene of the book.
// So the answer is kept for each text. A scene that did not change costs almost nothing. The cache is emptied when it grows large.
const MAX_CACHE = 3000;
const textCache = new Map<string, string>();
const countCache = new Map<string, number>();

/** Plain text of a chapter, for counting words. */
export function plainText(body: string): string {
  if (!looksLikeHtml(body)) return body ?? '';
  const hit = textCache.get(body);
  if (hit !== undefined) return hit;
  const spaced = body.replace(/<\/(p|h[1-6]|li|blockquote)>|<br\s*\/?>|<hr\s*\/?>/gi, ' ');
  const text = new DOMParser().parseFromString(spaced, 'text/html').body.textContent ?? '';
  if (textCache.size >= MAX_CACHE) textCache.clear();
  textCache.set(body, text);
  return text;
}

export function wordCount(body: string) {
  const text = body ?? '';
  const hit = countCache.get(text);
  if (hit !== undefined) return hit;
  const n = (plainText(text).match(/\S+/g) ?? []).length;
  if (countCache.size >= MAX_CACHE) countCache.clear();
  countCache.set(text, n);
  return n;
}

export const FACTION_TYPES = ['kingdom', 'empire', 'house', 'guild', 'order', 'army', 'cult', 'company', 'gang', 'other'];
export const FACTION_REL: Record<string, string> = { ally: '#6bbf8a', enemy: '#d9625b', rival: '#d98a5b', rules: '#8f7bf0', neutral: '#9c93a6' };
export const ITEM_TYPES = ['weapon', 'armor', 'relic', 'jewel', 'tool', 'book', 'document', 'vehicle', 'other'];
