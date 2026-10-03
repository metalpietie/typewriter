import type { Item, Kind } from './db.svelte';

export type Hit = { kind: Kind; id: number; title: string; sub: string; snippet: string; score: number };

type Field = { text: string; weight: number };

function levenshtein(a: string, b: string, max: number): number {
  if (Math.abs(a.length - b.length) > max) return max + 1;
  let prev = Array.from({ length: b.length + 1 }, (_, i) => i);
  for (let i = 1; i <= a.length; i++) {
    const cur = [i];
    let best = i;
    for (let j = 1; j <= b.length; j++) {
      cur[j] = Math.min(prev[j] + 1, cur[j - 1] + 1, prev[j - 1] + (a[i - 1] === b[j - 1] ? 0 : 1));
      best = Math.min(best, cur[j]);
    }
    if (best > max) return max + 1;
    prev = cur;
  }
  return prev[b.length];
}

/** Letters in order, with a penalty for gaps. Returns 0 when it does not match. */
function subsequence(token: string, text: string): number {
  let at = -1, first = -1, gaps = 0;
  for (const ch of token) {
    const next = text.indexOf(ch, at + 1);
    if (next < 0) return 0;
    if (first < 0) first = next; else gaps += next - at - 1;
    at = next;
  }
  const span = at - first + 1;
  if (span > token.length * 2.5) return 0;
  return Math.max(5, 40 - gaps * 3);
}

/** How well one query word matches one field. 0 means no match. */
function tokenScore(token: string, text: string, typos: boolean): number {
  const i = text.indexOf(token);
  if (i >= 0) {
    const wordStart = i === 0 || /[^a-z0-9]/.test(text[i - 1]);
    return 100 - Math.min(i, 40) * 0.5 + (wordStart ? 20 : 0);
  }
  if (!typos) return 0;
  let best = token.length >= 3 ? subsequence(token, text) : 0;
  if (token.length >= 4) {
    const allowed = token.length >= 7 ? 2 : 1;
    for (const w of text.split(/[^a-z0-9]+/)) {
      if (w.length < 3) continue;
      if (levenshtein(token, w, allowed) <= allowed) best = Math.max(best, 60);
    }
  }
  return best;
}

function fieldsOf(kind: Kind, it: Item): Field[] {
  const f = (text: unknown, weight: number): Field => ({ text: String(text ?? '').toLowerCase(), weight });
  if (kind === 'character') {
    return [
      f(it.name, 3), f(it.aliases, 2.5), f(it.role, 1),
      f(it.description, 1), f(it.traits, 1), f(it.backstory, 1), f(it.goals, 1), f(it.notes, 1),
      f((it.fields ?? []).map((x: any) => `${x.k} ${x.v}`).join(' '), 1),
    ];
  }
  if (kind === 'location') return [f(it.name, 3), f(it.tags, 2), f(it.type, 1), f(it.description, 1)];
  return [f(it.title, 3), f(it.tags, 2), f(it.category, 1), f(it.body, 1)];
}

function titleOf(kind: Kind, it: Item) { return kind === 'note' ? it.title : it.name; }
function subOf(kind: Kind, it: Item) { return kind === 'character' ? it.role : kind === 'location' ? `place · ${it.type}` : `lore · ${it.category}`; }
function bodyOf(kind: Kind, it: Item): string {
  return kind === 'character'
    ? [it.description, it.backstory, it.traits, it.goals, it.notes].filter(Boolean).join(' ')
    : (kind === 'location' ? it.description : it.body) ?? '';
}

function snippet(kind: Kind, it: Item, tokens: string[]): string {
  const body = String(bodyOf(kind, it)).replace(/\s+/g, ' ').trim();
  const lower = body.toLowerCase();
  for (const t of tokens) {
    const i = lower.indexOf(t);
    if (i >= 0) {
      const from = Math.max(0, i - 30);
      return (from > 0 ? '…' : '') + body.slice(from, from + 110) + (from + 110 < body.length ? '…' : '');
    }
  }
  return body.slice(0, 110) + (body.length > 110 ? '…' : '');
}

export function search(query: string, lists: Record<Kind, Item[]>, limit = 30): Hit[] {
  const tokens = query.toLowerCase().split(/\s+/).filter(Boolean);
  if (!tokens.length) return [];
  const hits: Hit[] = [];
  const kinds: Kind[] = ['character', 'location', 'note'];
  for (const kind of kinds) {
    for (const it of lists[kind]) {
      const fields = fieldsOf(kind, it);
      let total = 0;
      for (const t of tokens) {
        let best = 0;
        for (const fl of fields) {
          // Typo matching on long text is noisy, so only allow it on short fields.
          const s = tokenScore(t, fl.text, fl.weight > 1 || fl.text.length < 40) * fl.weight;
          if (s > best) best = s;
        }
        if (best === 0) { total = 0; break; } // every word must match something
        total += best;
      }
      if (total > 0) {
        hits.push({ kind, id: it.id, title: titleOf(kind, it) || 'Untitled', sub: subOf(kind, it), snippet: snippet(kind, it, tokens), score: total });
      }
    }
  }
  return hits.sort((a, b) => b.score - a.score).slice(0, limit);
}
