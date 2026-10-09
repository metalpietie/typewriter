// Text for a note that is made from selected text. No imports, so it can be tested alone.

const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

/** The lines of the selected text, trimmed, without empty lines. */
export const linesOf = (text: string): string[] =>
  String(text ?? '').split(/\r?\n/).map((l) => l.replace(/\s+/g, ' ').trim()).filter(Boolean);

/** A short title from the first words: at most 50 letters, cut at a space, with … when cut. */
export function titleOf(text: string, fallback = 'Note from the text'): string {
  const first = linesOf(text)[0] ?? '';
  if (!first) return fallback;
  if (first.length <= 50) return first;
  const cut = first.slice(0, 50);
  const at = cut.lastIndexOf(' ');
  return (at > 20 ? cut.slice(0, at) : cut).replace(/[\s.,;:!?-]+$/, '') + '…';
}

/** A name for a new character, place, item or faction: the first line, one line, at most 60 letters. */
export function nameOf(text: string): string {
  const first = linesOf(text)[0] ?? '';
  return first.length <= 60 ? first : first.slice(0, 60).trim();
}

/** The text of a page in the Notes tab: one paragraph for each line, then where it came from. */
export function pageHtml(text: string, source = ''): string {
  const body = linesOf(text).map((l) => `<p>${esc(l)}</p>`).join('');
  return body + (source ? `<p><em>From ${esc(source)}</em></p>` : '');
}

/** The text of a lore note, which is plain text. */
export function loreText(text: string, source = ''): string {
  const body = linesOf(text).join('\n\n');
  return source ? `${body}\n\nFrom ${source}` : body;
}
