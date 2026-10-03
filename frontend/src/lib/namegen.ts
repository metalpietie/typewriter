/** Name pieces for each style. A name is a few pieces glued together. */
export type Style = {
  key: string;
  label: string;
  starts: string[];
  mids: string[];
  ends: string[]; // the last piece of a person name
  placeEnds: string[]; // the last piece of a place name
};

export const STYLES: Style[] = [
  {
    key: 'elvish', label: 'Elvish',
    starts: ['ae', 'al', 'ar', 'el', 'eir', 'fae', 'gal', 'il', 'ith', 'lor', 'mi', 'nae', 'sil', 'thal', 'vael', 'yl', 'cele', 'lue'],
    mids: ['la', 'li', 'lo', 'na', 'ri', 'the', 'wen', 'va', 'ra', 'el', 'an', 'ia'],
    ends: ['dan', 'dor', 'iel', 'ion', 'las', 'lyn', 'nor', 'riel', 'wyn', 'thir', 'ys', 'ael', 'eth', 'dil', 'wen', 'ra'],
    placeEnds: ['dell', 'vale', 'lorien', 'wood', 'thil', 'indor', 'alas', 'mar', 'riel', 'dor'],
  },
  {
    key: 'dwarvish', label: 'Dwarvish',
    starts: ['bor', 'bar', 'dur', 'dwal', 'gim', 'gron', 'har', 'kaz', 'mor', 'nor', 'thor', 'thrain', 'bruen', 'dain', 'khor', 'bal'],
    mids: ['a', 'i', 'o', 'in', 'ar', 'un', 'ur'],
    ends: ['in', 'ur', 'ak', 'dar', 'grim', 'li', 'nar', 'rik', 'din', 'gar', 'bok', 'mund', 'dal', 'ir'],
    placeEnds: ['hold', 'deep', 'delve', 'gard', 'forge', 'heim', 'mine', 'dur', 'hall', 'peak'],
  },
  {
    key: 'norse', label: 'Norse',
    starts: ['ag', 'ar', 'bjor', 'eir', 'frey', 'gun', 'hal', 'ing', 'leif', 'ragn', 'sig', 'sten', 'thor', 'ulf', 'yng', 'ast', 'sven'],
    mids: ['a', 'i', 'u', 'e'],
    ends: ['ar', 'gard', 'hild', 'ild', 'mund', 'rik', 'run', 'stein', 'vald', 'vard', 'dis', 'ulf', 'bjorn', 'frid', 'vi'],
    placeEnds: ['heim', 'fjord', 'vik', 'by', 'gard', 'stad', 'holm', 'ness', 'dal', 'borg'],
  },
  {
    key: 'medieval', label: 'Old English',
    starts: ['al', 'ed', 'ethel', 'god', 'hen', 'hugh', 'ing', 'os', 'wil', 'beo', 'cad', 'ald', 'wulf', 'ger', 'ead', 'leof'],
    mids: ['a', 'e', 'i', 'er', 'el'],
    ends: ['bert', 'ric', 'ward', 'win', 'wulf', 'mund', 'ard', 'ton', 'red', 'gar', 'ing', 'fred', 'helm', 'hild', 'wyn', 'lin'],
    placeEnds: ['ford', 'ton', 'ham', 'bury', 'wick', 'shire', 'mouth', 'minster', 'field', 'haven', 'stead', 'moor'],
  },
  {
    key: 'desert', label: 'Desert',
    starts: ['am', 'az', 'fa', 'ha', 'id', 'ja', 'ka', 'mal', 'nas', 'rash', 'sa', 'ta', 'zah', 'khal', 'yas', 'ben'],
    mids: ['i', 'a', 'ir', 'im', 'al', 'ush'],
    ends: ['id', 'im', 'ra', 'ah', 'an', 'ir', 'eem', 'sa', 'ar', 'il', 'ud', 'een', 'mir', 'dar'],
    placeEnds: ['abad', 'stan', 'dar', 'sah', 'kesh', 'rim', 'sar', 'bek', 'ra', 'zim'],
  },
  {
    key: 'harsh', label: 'Harsh (orcs, raiders)',
    starts: ['gru', 'kra', 'mog', 'naz', 'og', 'rag', 'skar', 'thok', 'uzg', 'zug', 'gor', 'drak', 'ruk', 'bol'],
    mids: ['g', 'k', 'z', 'ak', 'og', 'ug', 'ra'],
    ends: ['gash', 'nak', 'ruk', 'bag', 'zog', 'tar', 'mok', 'gul', 'dush', 'kar', 'ash', 'thrak'],
    placeEnds: ['gor', 'mar', 'rock', 'pit', 'nar', 'hold', 'gash', 'dun', 'krag', 'maw'],
  },
];

export const CUSTOM = 'custom';

const pick = <T>(a: T[]): T => a[Math.floor(Math.random() * a.length)];
const vowel = (c: string) => 'aeiouy'.includes(c);

/** Glue two pieces. Repeated letters and clashing vowels are smoothed out. */
function glue(a: string, b: string): string {
  if (!a) return b;
  const x = a[a.length - 1], y = b[0];
  if (x === y) return a + b.slice(1);
  if (vowel(x) && vowel(y)) return a.slice(0, -1) + b;
  return a + b;
}

const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

/** One name of the given style. */
export function makeName(style: Style, kind: 'person' | 'place'): string {
  for (let tries = 0; tries < 30; tries++) {
    const n = Math.random();
    const first = pick(style.starts);
    const last = pick(kind === 'person' ? style.ends : style.placeEnds);
    if (first === last) continue; // "Windwind"
    let s = first;
    if (style.mids.length && n < (kind === 'person' ? 0.4 : 0.25)) s = glue(s, pick(style.mids));
    s = glue(s, last);
    if (s.length >= 4 && s.length <= (kind === 'person' ? 11 : 13) && !/(.)\1\1/.test(s)) return cap(s);
  }
  return cap(glue(pick(style.starts), pick(kind === 'person' ? style.ends : style.placeEnds)));
}

/** A style built from the writer's own words (one per line, or separated by commas or spaces). */
export function customStyle(text: string): Style | null {
  const words = [...new Set(text.toLowerCase().split(/[\s,;]+/).map((w) => w.replace(/[^\p{L}'-]/gu, '')).filter((w) => w.length >= 2))];
  if (words.length < 3) return null;
  return { key: CUSTOM, label: 'My own words', starts: words, mids: [], ends: words, placeEnds: words };
}

/** Several different names. Names in `skip` (lower case) are left out. */
export function makeNames(style: Style, kind: 'person' | 'place', count: number, skip: Set<string>): string[] {
  const out: string[] = [];
  const seen = new Set(skip);
  for (let i = 0; i < count * 40 && out.length < count; i++) {
    const n = makeName(style, kind);
    if (!seen.has(n.toLowerCase())) {
      seen.add(n.toLowerCase());
      out.push(n);
    }
  }
  return out;
}
