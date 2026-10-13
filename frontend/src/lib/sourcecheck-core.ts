// Checks for the sources of a non-fiction book. No imports, so it can be tested alone.
// A text is the html of one section. A claim is marked "needs a source" in the editor.

export type Piece = { id: number; label: string; body: string };

export type Unsourced = { id: number; label: string; text: string };
export type DeadCite = { id: number; label: string; source: number; times: number };

const tagless = (s: string): string => s.replace(/<[^>]*>/g, '').replace(/&nbsp;/g, ' ').replace(/&amp;/g, '&').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/\s+/g, ' ').trim();

/** The blocks of a text: one for each paragraph, list item, heading or quote. A claim and its citation are in the same block. */
function blocksOf(html: string): string[] {
  return html.split(/<\/(?:p|li|h[1-6]|blockquote|div)>/i);
}

/** Claims that are marked "needs a source" and have no citation after them in the same paragraph. */
export function unsourced(pieces: Piece[]): Unsourced[] {
  const out: Unsourced[] = [];
  for (const p of pieces) {
    for (const block of blocksOf(p.body)) {
      const re = /<span\b[^>]*\bdata-needs\b[^>]*>([\s\S]*?)<\/span>/gi;
      let m: RegExpExecArray | null;
      while ((m = re.exec(block))) {
        if (block.indexOf('data-cite', m.index) >= 0) continue; // a citation follows
        const text = tagless(m[1]);
        if (text) out.push({ id: p.id, label: p.label, text: text.length > 140 ? text.slice(0, 140) + '…' : text });
      }
    }
  }
  return out;
}

/** The ids of the sources that the texts cite, with the number of times. */
export function citeCounts(pieces: Piece[]): Map<number, number> {
  const counts = new Map<number, number>();
  for (const p of pieces) for (const m of p.body.matchAll(/data-cite="(\d+)"/g)) counts.set(+m[1], (counts.get(+m[1]) ?? 0) + 1);
  return counts;
}

/** The ids of the sources that no text cites. */
export function neverCited(sourceIds: number[], pieces: Piece[]): number[] {
  const used = citeCounts(pieces);
  return sourceIds.filter((id) => !used.has(id));
}

/** Citations of a source that is gone. */
export function deadCites(sourceIds: number[], pieces: Piece[]): DeadCite[] {
  const known = new Set(sourceIds);
  const out: DeadCite[] = [];
  for (const p of pieces) {
    const here = new Map<number, number>();
    for (const m of p.body.matchAll(/data-cite="(\d+)"/g)) if (!known.has(+m[1])) here.set(+m[1], (here.get(+m[1]) ?? 0) + 1);
    for (const [source, times] of here) out.push({ id: p.id, label: p.label, source, times });
  }
  return out;
}
