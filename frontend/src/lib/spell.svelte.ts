import nspell from 'nspell';
import { db } from './db.svelte';
import { look } from './editorlook.svelte';
import { isName, nameWords, notChecked } from './spell-core';

// Spell check for the editor. The dictionaries are the open Hunspell lists (the ones LibreOffice uses), so it works
// offline and the same on every system. The names of the project are known words, and you can add your own.
// The browser's own spell check is off in the editor, so a word is never marked twice.

// the dictionaries are in public/dict (with their licence), loaded only when the spell check is on
const FILES: Record<string, [string, string]> = { 'en-GB': ['/dict/en-GB.aff', '/dict/en-GB.dic'], 'en-US': ['/dict/en-US.aff', '/dict/en-US.dic'] };
const OWN_KEY = 'typewriter-dictionary'; // the words you added: a choice of this computer, like the dictionary

/** version changes when the dictionary, the names or your words change: the editor marks the words again. */
export const spell = $state({ lang: '', ready: false, version: 0 });

let checker: ReturnType<typeof nspell> | null = null;
let loading = '';
const cache = new Map<string, boolean>();
const ignored = new Set<string>(); // "Ignore": until the app closes
let own: string[] = (() => { try { return JSON.parse(localStorage.getItem(OWN_KEY) ?? '[]').filter((w: unknown) => typeof w === 'string'); } catch { return []; } })();

function changed() { cache.clear(); spell.version++; }

/** Load the dictionary that the Look panel says. Called by the editor; it does nothing when that one is loaded. */
export async function useDictionary(lang = look.spell) {
  if (lang === spell.lang || lang === loading) return;
  if (!FILES[lang]) { checker = null; spell.lang = lang; spell.ready = false; changed(); return; }
  loading = lang;
  try {
    const [aff, dic] = await Promise.all(FILES[lang].map((u) => fetch(u).then((r) => r.text())));
    if (loading !== lang) return;
    checker = nspell(aff, dic);
    for (const w of own) checker.add(w);
    spell.lang = lang; spell.ready = true;
    changed();
  } catch {
    spell.ready = false;
  } finally { if (loading === lang) loading = ''; }
}

// The names of the project are known words: characters (and what they are also called), places, factions, items,
// lore notes, glossary words and book titles. They are read again when one of them changes.
let names = new Set<string>();
$effect.root(() => {
  $effect(() => {
    const l = db.lists, list: unknown[] = [];
    for (const c of l.character) list.push(c.name, ...String(c.aliases ?? '').split(','));
    for (const kind of ['location', 'faction', 'item'] as const) for (const it of l[kind]) list.push(it.name);
    for (const t of l.term) list.push(t.word, ...String(t.accepted ?? '').split(','));
    for (const n of l.note) list.push(n.title);
    for (const bk of l.book) list.push(bk.title);
    names = nameWords(list);
    spell.version++; // no cache to clear: the names are checked after the dictionary
  });
});

/** Is this word spelled right (or not worth checking)? before and after are the letters around it. */
export function correct(word: string, before = '', after = ''): boolean {
  if (!checker || notChecked(word, before, after)) return true;
  let ok = cache.get(word);
  if (ok === undefined) {
    ok = checker.correct(word) || (word[0] !== word[0].toLowerCase() && checker.correct(word.toLowerCase()));
    cache.set(word, ok);
  }
  return ok || ignored.has(word.toLowerCase()) || isName(word, names);
}

/** Up to five words it may have been meant to be. */
export function suggest(word: string): string[] {
  if (!checker) return [];
  const c = checker, lower = word.toLowerCase();
  // the words one slip away: two letters the wrong way round (teh), one letter missing (thi), one too many (thiss)
  const score = new Map<string, number>();
  const put = (w: string, n: number) => { if (w.length > 1 && w !== lower && c.correct(w)) score.set(w, Math.max(score.get(w) ?? 0, n)); };
  for (let i = 0; i < lower.length - 1; i++) put(lower.slice(0, i) + lower[i + 1] + lower[i] + lower.slice(i + 2), 5);
  for (let i = 0; i <= lower.length; i++) for (const ch of 'abcdefghijklmnopqrstuvwxyz') put(lower.slice(0, i) + ch + lower.slice(i), i === lower.length ? 4 : 3);
  for (let i = 0; i < lower.length; i++) put(lower.slice(0, i) + lower.slice(i + 1), 2);
  // then what the dictionary suggests, in its order
  const seen = (w: string) => [...score.keys()].some((k) => k.toLowerCase() === w.toLowerCase());
  [...c.suggest(word), ...(lower !== word ? c.suggest(lower) : [])].forEach((w, i) => { if (!seen(w)) score.set(w, 1 - i / 100); });
  // a very common word was most likely meant
  for (const w of score.keys()) if (COMMON.has(w.toLowerCase())) score.set(w, score.get(w)! + 10);
  return [...score.entries()].sort((a, b) => b[1] - a[1]).map(([w]) => w).slice(0, 5);
}

// the most common English words: a slip is most often one of these
const COMMON = new Set(('the of and to a in is it you that he was for on are with as i his they be at one have this from or had by not word but what some we can out other were all there when up use your how said an each she which do their time if will way about many then them write would like so these her long make thing see him two has look more day could go come did number sound no most people my over know water than call first who may down side been now find any new work part take get place made live where after back little only round man year came show every good me give our under name very through just form sentence great think say help low line differ turn cause much mean before move right boy old too same tell does set three want air well also play small end put home read hand port large spell add even land here must big high such follow act why ask men change went light kind off need house picture try us again animal point mother world near build self earth father head stand own page should country found answer school grow study still learn plant cover food sun four between state keep eye never last let thought city tree cross farm hard start might story saw far sea draw left late run while press close night real life few north open seem together next white children begin got walk example ease paper group always music those both mark often letter until mile river car feet care second book carry took science eat room friend began idea fish mountain stop once base hear horse cut sure watch color colour face wood main enough plain girl usual young ready above ever red list though feel talk bird soon body dog family direct pose leave song measure door product black short numeral class wind question happen complete ship area half rock order fire south problem piece told knew pass since top whole king space heard best hour better true during hundred five remember step early hold west ground interest reach fast verb sing listen six table travel less morning ten simple several vowel toward war lay against pattern slow center centre love person money serve appear road map rain rule govern pull cold notice voice unit power town fine certain fly fall lead cry dark machine note wait plan figure star box noun field rest correct able pound done beauty drive stood contain front teach week final gave green oh quick develop ocean warm free minute strong special mind behind clear tail produce fact street inch multiply nothing course stay wheel full force blue object decide surface deep moon island foot system busy test record boat common gold possible plane stead dry wonder laugh thousand ago ran check game shape equate hot miss brought heat snow tire bring yes distant fill east paint language among').split(' '));

/** Add a word to your dictionary (on this computer). */
export function addWord(word: string) {
  if (!own.includes(word)) {
    own = [...own, word];
    try { localStorage.setItem(OWN_KEY, JSON.stringify(own)); } catch { /* it lasts until the app closes */ }
  }
  checker?.add(word);
  changed();
}

/** Stop marking a word until the app closes. */
export function ignoreWord(word: string) { ignored.add(word.toLowerCase()); changed(); }
