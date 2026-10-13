// Sources and citations: the rules for the short form that the editor shows. No imports, so it can be tested alone.
// The exported files are written by the Go code (bibliography.go), which uses the same rules for names and pages.

export const SOURCE_TYPES: [string, string][] = [
  ['book', 'Book'], ['article', 'Article'], ['web', 'Web page'], ['report', 'Report'], ['interview', 'Interview'], ['other', 'Other'],
];
export const SOURCE_STATUS: [string, string][] = [['to-read', 'To read'], ['read', 'Read'], ['quoted', 'Quoted']];
export const CITE_STYLES: [string, string][] = [
  ['authordate', 'Author and year, for example (Smith, 2020, p. 45)'],
  ['numbered', 'Numbers, for example [1]'],
  ['notes', 'A footnote for each citation'],
];

export type SourceLike = Record<string, unknown>;

const text = (v: unknown): string => String(v ?? '').trim();

/** A name is "Last, First" or "First Last". A name that ends with a comma is a group, such as "World Health Organization,". */
export function splitName(a: string): { last: string; first: string; group: boolean } {
  a = a.trim();
  if (a.endsWith(',')) return { last: a.slice(0, -1).trim(), first: '', group: true };
  const comma = a.indexOf(',');
  if (comma >= 0) return { last: a.slice(0, comma).trim(), first: a.slice(comma + 1).trim(), group: false };
  const words = a.split(/\s+/).filter(Boolean);
  if (words.length === 0) return { last: '', first: '', group: false };
  if (words.length === 1) return { last: words[0], first: '', group: false };
  return { last: words[words.length - 1], first: words.slice(0, -1).join(' '), group: false };
}

/** The authors, one on each line. */
export const authorLines = (authors: unknown): string[] => text(authors).split(/\r?\n/).map((l) => l.trim()).filter(Boolean);

/** The short form of the authors: "Clear", "Smith and Jones", "Lally et al.". */
export function lastNames(authors: unknown): string {
  const names = authorLines(authors).map((a) => splitName(a).last).filter(Boolean);
  if (names.length === 0) return '';
  if (names.length === 1) return names[0];
  if (names.length === 2) return names[0] + ' and ' + names[1];
  return names[0] + ' et al.';
}

/** The first four words of a title. */
export function shortTitle(title: unknown): string {
  const w = text(title).split(/\s+/).filter(Boolean);
  return w.length > 4 ? w.slice(0, 4).join(' ') + '…' : w.join(' ');
}

/** What the writer typed for a place in the source ("45", "45-47", "ch. 3") as words: "p. 45", "pp. 45–47", "ch. 3". */
export function location(loc: unknown): string {
  const l = text(loc);
  if (!l) return '';
  if (!/^[\d\-–, ]+$/.test(l)) return l;
  if (/[-–,]/.test(l)) return 'pp. ' + l.replace(/-/g, '–').replace(/ /g, '');
  return 'p. ' + l;
}

/** A citation as the editor shows it: (Clear, 2018, p. 45). The exported file uses the style that the writer chose. */
export function citeLabel(src: SourceLike | undefined, loc: unknown): string {
  if (!src) return '(missing source)';
  const who = lastNames(src.authors) || shortTitle(src.title) || 'source';
  const year = text(src.year) || 'n.d.';
  const l = location(loc);
  return '(' + [who, year, l].filter(Boolean).join(', ') + ')';
}

/** A line to pick a source from a list: "Clear (2018): Atomic Habits". */
export function sourceLine(src: SourceLike): string {
  const who = lastNames(src.authors);
  const year = text(src.year);
  const head = [who, year ? '(' + year + ')' : ''].filter(Boolean).join(' ');
  return [head, text(src.title)].filter(Boolean).join(': ') || 'Untitled source';
}

/** The ids of the sources that a text cites. */
export function citedIds(html: string): number[] {
  return [...html.matchAll(/data-cite="(\d+)"/g)].map((m) => +m[1]);
}

/** The text with every citation of one source taken out. */
export function stripCitations(html: string, id: number): string {
  return html.replace(new RegExp(`<span\\b[^>]*\\bdata-cite="${id}"[^>]*>\\s*</span>`, 'g'), '');
}

/** The saved fields of a source, made safe: text fields are single-line text, the type is one that exists. */
export function cleanSource(raw: Record<string, unknown>): Record<string, string> {
  const line = (v: unknown, max = 300) => text(v).replace(/\s+/g, ' ').slice(0, max);
  return {
    type: SOURCE_TYPES.some(([k]) => k === raw.type) ? String(raw.type) : 'other',
    title: line(raw.title), authors: authorLines(raw.authors).map((a) => line(a, 120)).join('\n'), year: line(raw.year, 12),
    container: line(raw.container), publisher: line(raw.publisher), volume: line(raw.volume, 20), issue: line(raw.issue, 20),
    pages: line(raw.pages, 40), url: line(raw.url, 500), accessed: line(raw.accessed, 40), edition: line(raw.edition, 40),
    status: SOURCE_STATUS.some(([k]) => k === raw.status) ? String(raw.status) : 'to-read', note: text(raw.note).slice(0, 5000),
  };
}
