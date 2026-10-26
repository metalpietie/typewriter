// Spell check: the parts that need no dictionary. Find the words of a text, leave out what is not worth checking,
// and know the names of the project. No imports, so it can be tested alone.

export type WordAt = { start: number; end: number; word: string };

const WORD = /\p{L}[\p{L}\p{M}'’]*/gu;

/** The words of a text, with their places. A curly apostrophe counts as a straight one; one at the end is not part of the word. */
export function wordsOf(text: string): WordAt[] {
  const out: WordAt[] = [];
  for (const m of text.matchAll(WORD)) {
    let w = m[0].replace(/’/g, "'");
    while (w.endsWith("'")) w = w.slice(0, -1);
    if (w) out.push({ start: m.index!, end: m.index! + w.length, word: w });
  }
  return out;
}

/** A word that is not checked: one letter, all capitals (an abbreviation such as NASA), or a mix of letters and digits. */
export function notChecked(word: string, before = '', after = ''): boolean {
  if (word.length < 2) return true;
  if (word.length > 1 && word === word.toUpperCase() && word !== word.toLowerCase()) return true;
  if (/\d/.test(before.slice(-1)) || /\d/.test(after.charAt(0))) return true; // 3rd, 10km
  return false;
}

/** The words of the names in a project (characters, places, glossary words and so on), in small letters. */
export function nameWords(names: unknown[]): Set<string> {
  const out = new Set<string>();
  for (const n of names) {
    for (const w of wordsOf(String(n ?? ''))) {
      const lw = w.word.toLowerCase();
      out.add(lw);
      if (lw.endsWith("'s")) out.add(lw.slice(0, -2));
    }
  }
  return out;
}

/** Is a word one of the names? A possessive ("Aeryn's") counts too. */
export function isName(word: string, names: Set<string>): boolean {
  const w = word.toLowerCase();
  return names.has(w) || (w.endsWith("'s") && names.has(w.slice(0, -2))) || (w.endsWith("s'") && names.has(w.slice(0, -1)));
}

/** A replacement keeps the capital of the word it replaces ("Teh" becomes "The"). */
export function sameCase(word: string, suggestion: string): string {
  if (word && word[0] === word[0].toUpperCase() && word[0] !== word[0].toLowerCase()) return suggestion.charAt(0).toUpperCase() + suggestion.slice(1);
  return suggestion;
}
