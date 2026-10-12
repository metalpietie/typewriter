// The name generator. No imports, so it can be tested alone.
//
// A person name is built from pieces of a culture: a start, sometimes a middle, and an end.
// The end decides the gender: each culture has its own female, male and shared endings.
// A full name adds a family name, a byname ("the Grey") or a home ("of Ashford").
// A place name is built from the same pieces. It can be a town, a kingdom, a wild place or an inn.

export type Gender = 'any' | 'female' | 'male';
export type Form = 'first' | 'family' | 'byname' | 'home';
export type Length = 'any' | 'short' | 'long';
export type PlaceKind = 'settlement' | 'realm' | 'wilds' | 'inn';

/** How the family name of a culture is made. */
export type Family =
  | { kind: 'compound'; a: string[]; b: string[] } // Silver + moon. An empty piece in b means a alone.
  | { kind: 'father'; mode: 'suffix' | 'word' | 'prefix'; m: string[]; f: string[] } // Ulfsson, ibn Karim, MacBran
  | { kind: 'stem'; stems?: string[]; m: string[]; f: string[] }; // Volkov and Volkova, Aurelius and Aurelia

export type Culture = {
  key: string;
  label: string;
  starts: string[];
  mids: string[];
  endsM: string[];
  endsF: string[];
  endsN: string[]; // endings that suit anyone
  family: Family | null;
  bynames: string[]; // added to the shared list
  placeEnds: string[];
  maxCluster?: number; // most consonants in a row (3 if not set)
  gendered: boolean;
};

const compound = (a: string[], b: string[]): Family => ({ kind: 'compound', a, b });

export const CULTURES: Culture[] = [
  {
    key: 'elvish', label: 'Elvish', gendered: true,
    starts: ['ae', 'al', 'ar', 'el', 'eir', 'fae', 'gal', 'il', 'ith', 'lor', 'mi', 'nae', 'sil', 'thal', 'vael', 'yl', 'cele', 'lue', 'aer', 'cael', 'elen', 'fen', 'ior', 'lir', 'mal', 'nim', 'quel', 'sael', 'tir', 'uri', 'vir'],
    mids: ['la', 'li', 'lo', 'na', 'ri', 'the', 'wen', 'va', 'ra', 'el', 'an', 'ia', 'ni', 'sa', 'lin', 'dre'],
    endsM: ['dan', 'dor', 'ion', 'las', 'nor', 'thir', 'ael', 'dil', 'rion', 'vin', 'rod', 'mir', 'thas', 'lion'],
    endsF: ['iel', 'wyn', 'riel', 'lyn', 'ara', 'ia', 'ys', 'wen', 'ril', 'thea', 'lia', 'vyn', 'ina', 'essa'],
    endsN: ['eth', 'yr', 'is', 'an', 'ien'],
    family: compound(['silver', 'star', 'moon', 'dawn', 'night', 'sun', 'leaf', 'wind', 'dusk', 'glimmer', 'swift', 'bright', 'amber', 'dew', 'white'], ['moon', 'leaf', 'song', 'wind', 'blade', 'star', 'fall', 'shade', 'whisper', 'bough', 'brook', 'flower', 'light', 'glade', 'dancer']),
    bynames: ['Starsinger', 'Moonwalker', 'of the Silver Leaf', 'Dawnbringer'],
    placeEnds: ['dell', 'vale', 'lorien', 'wood', 'thil', 'indor', 'alas', 'mar', 'riel', 'dor', 'thalas', 'sylva'],
  },
  {
    key: 'dwarvish', label: 'Dwarvish', gendered: true, maxCluster: 4,
    starts: ['bor', 'bar', 'dur', 'dwal', 'gim', 'gron', 'har', 'kaz', 'mor', 'nor', 'thor', 'thrain', 'bruen', 'dain', 'khor', 'bal', 'dor', 'fun', 'glo', 'ori', 'bif', 'bom', 'nal', 'rur', 'tho'],
    mids: ['a', 'i', 'o', 'in', 'ar', 'un', 'ur'],
    endsM: ['in', 'ur', 'ak', 'dar', 'grim', 'li', 'nar', 'rik', 'din', 'gar', 'bok', 'mund', 'dal', 'ir', 'bur', 'rim', 'thi', 'ok'],
    endsF: ['a', 'dis', 'hild', 'ra', 'la', 'ni', 'dra', 'rid', 'na', 'dri', 'gunn', 'bera', 'ild'],
    endsN: ['in', 'i', 'ur'],
    family: compound(['iron', 'stone', 'deep', 'black', 'gold', 'anvil', 'hammer', 'rock', 'steel', 'thunder', 'coal', 'copper', 'granite', 'bronze'], ['beard', 'fist', 'hammer', 'forge', 'shield', 'helm', 'axe', 'breaker', 'delver', 'hand', 'born', 'heart', 'mantle', 'hold']),
    bynames: ['Oathkeeper', 'Stonehelm', 'the Stubborn', 'Deepdelver'],
    placeEnds: ['hold', 'deep', 'delve', 'gard', 'forge', 'heim', 'mine', 'dur', 'hall', 'peak'],
  },
  {
    key: 'norse', label: 'Norse', gendered: true,
    starts: ['ag', 'ar', 'bjor', 'eir', 'frey', 'gun', 'hal', 'ing', 'leif', 'ragn', 'sig', 'sten', 'thor', 'ulf', 'yng', 'ast', 'sven', 'ask', 'bor', 'dag', 'ey', 'gud', 'hak', 'her', 'kol', 'orm', 'rol', 'tor', 'vig'],
    mids: ['a', 'i', 'u', 'e', 'o'],
    endsM: ['ar', 'gard', 'mund', 'rik', 'stein', 'vald', 'vard', 'ulf', 'bjorn', 'finn', 'mar', 'leif', 'geir'],
    endsF: ['hild', 'ild', 'run', 'dis', 'frid', 'vi', 'gerd', 'laug', 'veig', 'rid', 'unn', 'borg', 'a'],
    endsN: ['i', 'ey'],
    family: { kind: 'father', mode: 'suffix', m: ['sson'], f: ['sdottir'] },
    bynames: ['Bloodaxe', 'Ironside', 'Fairhair', 'Forkbeard', 'the Boneless', 'Skullsplitter', 'Shieldbreaker'],
    placeEnds: ['heim', 'fjord', 'vik', 'by', 'gard', 'stad', 'holm', 'ness', 'dal', 'borg'],
  },
  {
    key: 'medieval', label: 'Old English', gendered: true,
    starts: ['al', 'ed', 'ethel', 'god', 'hen', 'hugh', 'ing', 'os', 'wil', 'beo', 'cad', 'ald', 'wulf', 'ger', 'ead', 'leof', 'aelf', 'cuth', 'dun', 'hild', 'win', 'ber', 'ro', 'hor'],
    mids: ['a', 'e', 'i', 'er', 'el'],
    endsM: ['bert', 'ric', 'ward', 'win', 'wulf', 'mund', 'ard', 'ton', 'red', 'gar', 'ing', 'fred', 'helm', 'ald', 'ulf'],
    endsF: ['hild', 'wyn', 'lin', 'a', 'gyth', 'burh', 'flaed', 'swith', 'thryth', 'wen', 'ild', 'ie', 'ette'],
    endsN: ['ric', 'win', 'ing'],
    family: compound(['ash', 'black', 'brook', 'cart', 'church', 'cole', 'fletch', 'green', 'hill', 'mill', 'north', 'under', 'wood', 'west', 'thatch', 'wright', 'smith', 'baker', 'thorn', 'stone', 'fair', 'hale'], ['', '', 'ford', 'ton', 'worth', 'by', 'ley', 'man', 'er', 'field', 'well', 'wood']),
    bynames: ['the Bold', 'Lackland', 'the Elder', 'the Unready', 'Redhand'],
    placeEnds: ['ford', 'ton', 'ham', 'bury', 'wick', 'shire', 'mouth', 'minster', 'field', 'haven', 'stead', 'moor'],
  },
  {
    key: 'celtic', label: 'Celtic', gendered: true,
    starts: ['bran', 'bri', 'cael', 'cian', 'dar', 'ev', 'fin', 'gwyn', 'iun', 'kel', 'lleu', 'mael', 'mor', 'nia', 'oen', 'rhi', 'sean', 'tris', 'ulla', 'aine', 'bre', 'cad', 'dun', 'eil', 'fer', 'gar', 'her', 'ker'],
    mids: ['a', 'e', 'i', 'o', 'an', 'en', 'in', 'wy'],
    endsM: ['dan', 'gus', 'lan', 'mor', 'ric', 'ach', 'och', 'aid', 'an', 'ydd', 'win', 'ghan', 'ell', 'dric'],
    endsF: ['wen', 'wyn', 'ia', 'ine', 'een', 'ona', 'eth', 'a', 'ys', 'gan', 'ain', 'ina', 'la'],
    endsN: ['an', 'in', 'wy'],
    family: { kind: 'father', mode: 'prefix', m: ['Mac', 'Mc', "O'", 'Ap '], f: ['Nic', "Ní ", 'Ferch '] },
    bynames: ['the Red', 'Longhand', 'of the Mists', 'Silverhand', 'Stonewall'],
    placeEnds: ['dun', 'glen', 'loch', 'tir', 'carn', 'ross', 'ness', 'aber', 'brae', 'mor', 'wick'],
  },
  {
    key: 'slavic', label: 'Slavic', gendered: true,
    starts: ['bo', 'bra', 'dra', 'gos', 'ja', 'kaz', 'lub', 'mi', 'mir', 'ni', 'ra', 'ro', 'sta', 'svet', 'tom', 'vla', 'vol', 'zbi', 'zlat', 'bog', 'dob', 'mar', 'rad', 'vese', 'bor', 'cher'],
    mids: ['a', 'i', 'o', 'e', 'ko', 'mi', 'sla'],
    endsM: ['slav', 'mir', 'mil', 'bor', 'gor', 'dan', 'imir', 'yan', 'ek', 'ik', 'vit', 'raz', 'nik', 'ko', 'dag'],
    endsF: ['slava', 'mila', 'ka', 'ina', 'na', 'ya', 'ela', 'ia', 'mira', 'dara', 'isa', 'ena', 'ushka', 'ja'],
    endsN: ['sha', 'ya'],
    family: { kind: 'stem', m: ['ov', 'ev', 'sky', 'ovic'], f: ['ova', 'eva', 'skaya', 'ovic'] },
    bynames: ['the Grey Wolf', 'Ironhand', 'the Brave', 'Firebird'],
    placeEnds: ['grad', 'sk', 'ov', 'gorod', 'pol', 'vo', 'nitsa', 'ska', 'dvor'],
  },
  {
    key: 'latin', label: 'Roman', gendered: true,
    starts: ['aem', 'aug', 'brut', 'cas', 'cla', 'dec', 'fab', 'fel', 'gai', 'hel', 'iul', 'lu', 'mar', 'nu', 'oct', 'pub', 'qui', 'rom', 'sab', 'tit', 'val', 'ver', 'cor', 'dom', 'flav'],
    mids: ['e', 'i', 'o', 'u', 'el', 'an', 'er'],
    endsM: ['us', 'ius', 'ianus', 'inus', 'ulus', 'ix', 'o', 'ax', 'cus', 'tus', 'lus', 'nus'],
    endsF: ['a', 'ia', 'illa', 'ina', 'ula', 'ena', 'essa', 'lia', 'tia', 'na', 'la'],
    endsN: [],
    family: { kind: 'stem', stems: ['cornel', 'aurel', 'jul', 'claud', 'flav', 'val', 'fab', 'pomp', 'tull', 'sext', 'luc', 'oct', 'ner', 'vit', 'porc', 'anton', 'sept', 'dec', 'crass', 'ful'], m: ['ius', 'anus', 'inus'], f: ['ia', 'ana', 'ina'] },
    bynames: ['the Elder', 'the Younger', 'Magnus', 'Felix', 'Africanus', 'Imperator'],
    placeEnds: ['ium', 'ia', 'polis', 'ona', 'ae', 'um', 'ora', 'ana', 'ensis'],
  },
  {
    key: 'greek', label: 'Greek', gendered: true,
    starts: ['ald', 'ari', 'cal', 'dem', 'eur', 'hec', 'her', 'kal', 'lys', 'mel', 'nik', 'ore', 'pel', 'phil', 'sar', 'thal', 'xen', 'zen', 'ant', 'dio', 'eph', 'hip', 'isa', 'kyr', 'myr', 'pan', 'theo'],
    mids: ['o', 'a', 'e', 'i', 'ste', 'ro', 'ni'],
    endsM: ['os', 'es', 'ion', 'on', 'ides', 'as', 'eus', 'ander', 'ippos', 'crates', 'medes', 'ias'],
    endsF: ['a', 'e', 'ia', 'ope', 'ene', 'is', 'ona', 'ippe', 'ina', 'ea', 'issa'],
    endsN: [],
    family: { kind: 'stem', stems: ['alex', 'demetr', 'nikol', 'kost', 'pap', 'anast', 'georg', 'theod', 'dim', 'vas', 'chris', 'pan', 'sot', 'stav', 'kyr'], m: ['opoulos', 'akis', 'idis'], f: ['opoulou', 'aki', 'idou'] },
    bynames: ['the Wise', 'Lionheart', 'the Younger', 'Swiftfoot'],
    placeEnds: ['polis', 'ios', 'ea', 'ope', 'nos', 'ium', 'ona', 'thos', 'ia'],
  },
  {
    key: 'desert', label: 'Desert', gendered: true,
    starts: ['am', 'az', 'fa', 'ha', 'id', 'ja', 'ka', 'mal', 'nas', 'rash', 'sa', 'ta', 'zah', 'khal', 'yas', 'ben', 'bas', 'dar', 'far', 'jam', 'kar', 'lat', 'mar', 'nur', 'sal', 'tar', 'yus', 'zul'],
    mids: ['i', 'a', 'ir', 'im', 'al', 'ush', 'ad', 'am'],
    endsM: ['id', 'im', 'ir', 'mir', 'dar', 'ud', 'ar', 'an', 'il', 'eem', 'ush', 'ik', 'ad', 'zar', 'bar'],
    endsF: ['ra', 'ah', 'sa', 'een', 'ya', 'ima', 'ina', 'la', 'ira', 'na', 'ia', 'eh', 'ifa'],
    endsN: ['an', 'ir'],
    family: { kind: 'father', mode: 'word', m: ['ibn'], f: ['bint'] },
    bynames: ['the Wanderer', 'of the Dunes', 'the Generous', 'Sandwalker', 'the Just'],
    placeEnds: ['abad', 'stan', 'dar', 'sah', 'kesh', 'rim', 'sar', 'bek', 'ra', 'zim'],
  },
  {
    key: 'halfling', label: 'Halfling (cosy)', gendered: true,
    starts: ['bil', 'bun', 'daf', 'fro', 'ger', 'hol', 'lob', 'mer', 'mil', 'ned', 'pip', 'ros', 'sam', 'tob', 'wil', 'bell', 'cam', 'don', 'eg', 'fal', 'gri', 'hob', 'lil', 'mar', 'nol'],
    mids: ['a', 'e', 'i', 'o', 'ly', 'bo'],
    endsM: ['bo', 'do', 'go', 'by', 'ric', 'mund', 'bin', 'bert', 'rin', 'mo', 'wise', 'fast'],
    endsF: ['a', 'bell', 'ie', 'ia', 'lie', 'ina', 'ly', 'la', 'ose', 'ette'],
    endsN: ['y', 'ie'],
    family: compound(['under', 'green', 'good', 'brand', 'bur', 'took', 'proud', 'long', 'hill', 'brook', 'tea', 'apple', 'honey', 'sweet', 'bram'], ['hill', 'foot', 'barrel', 'bottom', 'body', 'bush', 'wick', 'berry', 'ford', 'bank', 'burrow', 'cake', 'comb', 'field', 'dale']),
    bynames: ['the Second', 'Goodfellow', 'the Elder', 'Lightfoot'],
    placeEnds: ['ton', 'ing', 'by', 'wick', 'hollow', 'dale', 'bury', 'ford', 'field', 'combe'],
  },
  {
    key: 'fae', label: 'Fae', gendered: true,
    starts: ['aur', 'bri', 'cae', 'dew', 'eld', 'fi', 'gl', 'lun', 'mab', 'nym', 'ob', 'pix', 'ros', 'sere', 'tit', 'wil', 'thist', 'brie', 'fey', 'gos', 'lil'],
    mids: ['a', 'e', 'i', 'o', 'y', 'ly', 'mi', 'ri'],
    endsM: ['on', 'in', 'wick', 'thorn', 'rin', 'pin', 'bryn', 'vyn', 'el', 'ric'],
    endsF: ['ia', 'ella', 'la', 'ine', 'wyn', 'belle', 'ina', 'ora', 'elle', 'ry'],
    endsN: ['wing', 'fly', 'sprite', 'y', 'ie'],
    family: compound(['moon', 'thistle', 'dew', 'gossamer', 'willow', 'briar', 'twilight', 'petal', 'mist', 'star', 'honey', 'fern', 'glim'], ['wing', 'whisper', 'weave', 'thorn', 'dance', 'song', 'shade', 'glow', 'spark', 'bloom', 'mantle', 'light']),
    bynames: ['the Fair', 'of the Hollow Hill', 'Thornweaver', 'Moonlit'],
    placeEnds: ['glen', 'hollow', 'dell', 'ring', 'bower', 'mere', 'glade', 'wyn', 'thorn', 'nook'],
  },
  {
    key: 'harsh', label: 'Harsh (orcs, raiders)', gendered: true, maxCluster: 4,
    starts: ['gru', 'kra', 'mog', 'naz', 'og', 'rag', 'skar', 'thok', 'uzg', 'zug', 'gor', 'drak', 'ruk', 'bol', 'ghar', 'hurk', 'mur', 'snag', 'urz', 'vol', 'yag'],
    mids: ['g', 'k', 'z', 'ak', 'og', 'ug', 'ra'],
    endsM: ['gash', 'nak', 'ruk', 'bag', 'zog', 'tar', 'mok', 'gul', 'dush', 'kar', 'ash', 'thrak', 'mash', 'bul'],
    endsF: ['ga', 'ka', 'za', 'ush', 'ra', 'ba', 'dra', 'sha', 'ggra', 'kha', 'ula'],
    endsN: ['ash', 'ug', 'uk'],
    family: compound(['skull', 'bone', 'blood', 'iron', 'black', 'gut', 'death', 'grim', 'ash', 'storm'], ['crusher', 'splitter', 'fang', 'jaw', 'render', 'eater', 'claw', 'bane', 'hide', 'fist']),
    bynames: ['the Brutal', 'Eye-Gouger', 'the Unburnt', 'Skullcleaver'],
    placeEnds: ['gor', 'mar', 'rock', 'pit', 'nar', 'hold', 'gash', 'dun', 'krag', 'maw'],
  },
];

export const CUSTOM = 'custom';

/** Names that fit anyone, in any culture. Each culture adds its own. */
const BYNAMES = [
  'the Bold', 'the Grey', 'the Wise', 'the Younger', 'the Fair', 'the Quiet', 'the Red', 'the Swift', 'the Silent', 'the Just',
  'the Lame', 'the Unlucky', 'the Wanderer', 'the Elder', 'Stormborn', 'Oathkeeper', 'Ironhand', 'Longshanks', 'Blackthorn',
  'Nightwalker', 'Brightblade', 'Wolfsbane', 'Greycloak', 'Halfhand', 'the Merciful', 'the Lucky', 'the Kind', 'Lightbringer',
];

export const PLACE_KINDS: [PlaceKind, string][] = [
  ['settlement', 'Town or village'], ['realm', 'Kingdom or land'], ['wilds', 'Wild place (wood, river, mountain)'], ['inn', 'Inn or tavern'],
];
export const GENDERS: [Gender, string][] = [['any', 'Any'], ['female', 'Female'], ['male', 'Male']];
export const FORMS: [Form, string][] = [
  ['first', 'First name only'], ['family', 'First and family name'], ['byname', 'First name and a byname (the Grey)'], ['home', 'First name and a home (of Ashford)'],
];
export const LENGTHS: [Length, string][] = [['any', 'Any length'], ['short', 'Short (up to 6 letters)'], ['long', 'Long (8 letters or more)']];

export type Options = {
  kind: 'person' | 'place';
  culture: Culture;
  gender?: Gender;
  form?: Form;
  length?: Length;
  letter?: string; // the first letter of the name, or "" for any
  place?: PlaceKind;
  rand?: () => number; // random numbers from 0 to 1. The tests give a fixed one.
};
/** A name, and the gender it was made for. */
export type Made = { name: string; gender: 'female' | 'male' | '' };

const vowel = (c: string) => 'aeiouy'.includes(c);
const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);
const capWords = (s: string) => s.split(' ').map(cap).join(' ');

/** Glue two pieces. Repeated letters and clashing vowels are smoothed out. */
export function glue(a: string, b: string): string {
  if (!a || !b) return a + b;
  const x = a[a.length - 1], y = b[0];
  if (x === y) return a + b.slice(1);
  if (vowel(x) && vowel(y)) return a.slice(0, -1) + b;
  return a + b;
}

/** A name that is easy to say: no three of one letter, no long runs of consonants, some vowel. */
export function sayable(s: string, maxCluster = 3): boolean {
  const t = s.toLowerCase();
  // no three of one letter, no three vowels in a row, no piece said twice ("Jakoko"), at least one vowel
  if (/(.)\1\1/.test(t) || /[aeiouy]{3,}/.test(t) || /(.{2,3})\1/.test(t) || !/[aeiouy]/.test(t)) return false;
  return !new RegExp(`[^aeiouy'\\- ]{${maxCluster + 1},}`).test(t);
}

type R = () => number;
const pick = <T,>(r: R, a: T[]): T => a[Math.floor(r() * a.length) % a.length];
const chance = (r: R, p: number) => r() < p;

/** One person name of one gender. Null when no name fits after many tries. */
function firstName(c: Culture, g: 'female' | 'male', o: Options, r: R): string | null {
  const length = o.length ?? 'any';
  const letter = (o.letter ?? '').toLowerCase();
  const starts = letter ? c.starts.filter((s) => s.startsWith(letter)) : c.starts;
  if (!starts.length) return null;
  const own = g === 'female' ? c.endsF : c.endsM;
  for (let tries = 0; tries < 60; tries++) {
    const ends = c.endsN.length && chance(r, 0.22) ? c.endsN : own;
    const start = pick(r, starts), end = pick(r, ends);
    if (start === end) continue; // "Windwind"
    let s = start;
    if (c.mids.length && chance(r, length === 'long' ? 0.7 : length === 'short' ? 0 : 0.3)) s = glue(s, pick(r, c.mids));
    s = glue(s, end);
    const max = length === 'short' ? 6 : 11;
    const min = length === 'long' ? 8 : 3;
    if (s.length >= min && s.length <= max && sayable(s, c.maxCluster ?? 3)) return cap(s);
  }
  return null;
}

function familyName(c: Culture, g: 'female' | 'male', r: R): string | null {
  const f = c.family;
  if (!f) return null;
  if (f.kind === 'compound') {
    for (let i = 0; i < 20; i++) {
      const a = pick(r, f.a), b = pick(r, f.b);
      if (a === b) continue;
      return cap(b ? glue(a, b) : a);
    }
    return null;
  }
  if (f.kind === 'father') {
    const base = firstName(c, 'male', { kind: 'person', culture: c, length: 'any' }, r);
    if (!base) return null;
    let mark = pick(r, g === 'female' ? f.f : f.m);
    if (f.mode === 'suffix') {
      if (base.toLowerCase().endsWith('s') && mark.startsWith('s')) mark = mark.slice(1);
      return base + mark;
    }
    return f.mode === 'word' ? `${mark} ${base}` : mark + cap(base); // "ibn Karim", "MacBran"
  }
  // stem: a short start of a name, and a suffix for each gender
  const stem = f.stems ? pick(r, f.stems) : (firstName(c, 'male', { kind: 'person', culture: c, length: 'short' }, r) ?? pick(r, c.starts));
  const suffix = pick(r, g === 'female' ? f.f : f.m);
  return cap(glue(stem.toLowerCase(), suffix));
}

const FEATURES = ['Wood', 'Forest', 'River', 'Fen', 'Pass', 'Marsh', 'Falls', 'Bay', 'Isle', 'Hills', 'Moors'];
const WILD_A = ['Whisper', 'Raven', 'Ash', 'Frost', 'Thorn', 'Grey', 'Silver', 'Hollow', 'Mist', 'Bramble', 'Elder', 'Storm', 'Wolf', 'Briar', 'Shadow', 'Amber'];
const WILD_B = ['wood', 'fell', 'mere', 'moor', 'vale', 'fen', 'peak', 'crag', 'water', 'holt', 'wold', 'combe', 'ridge', 'marsh'];
const INN_A = ['Gilded', 'Rusty', 'Drunken', 'Laughing', 'Sleeping', 'Silver', 'Prancing', 'Broken', 'Weary', 'Golden', 'Crooked', 'Blind', 'Merry', 'Black', 'Red', 'Grey', 'Howling', 'Lucky', 'Wandering'];
const INN_B = ['Stag', 'Dragon', 'Pony', 'Boar', 'Goose', 'Lantern', 'Anchor', 'Crown', 'Raven', 'Tankard', 'Fox', 'Wheel', 'Bell', 'Barrel', 'Hound', 'Owl', 'Kettle', 'Harp', 'Oak', 'Badger', 'Swan', 'Lamb'];

/** The base of a place name, in the style of the culture. Null when none fits. */
function placeBase(c: Culture, o: Options, r: R): string | null {
  const letter = (o.letter ?? '').toLowerCase();
  const starts = letter ? c.starts.filter((s) => s.startsWith(letter)) : c.starts;
  if (!starts.length) return null;
  for (let tries = 0; tries < 60; tries++) {
    const start = pick(r, starts), end = pick(r, c.placeEnds);
    if (start === end) continue;
    let s = start;
    if (c.mids.length && chance(r, 0.25)) s = glue(s, pick(r, c.mids));
    s = glue(s, end);
    if (s.length >= 4 && s.length <= 13 && sayable(s, c.maxCluster ?? 3)) return cap(s);
  }
  return null;
}

function placeName(c: Culture, o: Options, r: R): string | null {
  const kind = o.place ?? 'settlement';
  if (kind === 'inn') {
    const letter = (o.letter ?? '').toUpperCase();
    for (let i = 0; i < 40; i++) {
      const a = pick(r, INN_A), b = pick(r, INN_B), b2 = pick(r, INN_B);
      const n = Math.floor(r() * 5);
      const s = n <= 1 ? `The ${a} ${b}` : n === 2 ? `The ${b} and ${b2}` : n === 3 ? `The ${b}'s Head` : `The ${b} and ${pick(r, ['Crown', 'Anchor', 'Plough', 'Key', 'Barrel'])}`;
      if (n === 2 && b === b2) continue;
      const first = s.replace(/^The /, '')[0];
      if (!letter || first === letter) return s;
    }
    return null;
  }
  const base = placeBase(c, o, r);
  if (!base) return null;
  if (kind === 'settlement') return base;
  if (kind === 'realm') {
    const n = Math.floor(r() * 10);
    return n < 4 ? base : n < 6 ? `Kingdom of ${base}` : n === 6 ? `The ${base} Marches` : n === 7 ? `${base} Reach` : n === 8 ? `The Free Lands of ${base}` : `Empire of ${base}`;
  }
  // wild places: half in the style of the culture, half plain English
  if (chance(r, 0.5)) {
    const n = Math.floor(r() * 5);
    return n === 0 ? `Mount ${base}` : n === 1 ? `Lake ${base}` : n === 2 ? `The ${base} ${pick(r, ['Hills', 'Moors', 'Woods'])}` : `${base} ${pick(r, FEATURES)}`;
  }
  const letter = (o.letter ?? '').toUpperCase();
  for (let i = 0; i < 30; i++) {
    const a = pick(r, WILD_A);
    if (letter && a[0] !== letter) continue;
    return `${a}${pick(r, WILD_B)}`;
  }
  return `${base} ${pick(r, FEATURES)}`;
}

/** One name. Null when none fits the choices, for example no name of this style starts with that letter. */
export function makeOne(o: Options): Made | null {
  const r = o.rand ?? Math.random;
  const c = o.culture;
  if (o.kind === 'place') {
    const name = placeName(c, o, r);
    return name ? { name, gender: '' } : null;
  }
  const want = o.gender ?? 'any';
  const g: 'female' | 'male' = want === 'any' || !c.gendered ? (chance(r, 0.5) ? 'female' : 'male') : want;
  const first = firstName(c, g, o, r);
  if (!first) return null;
  const gender = c.gendered ? g : '';
  const form = o.form ?? 'first';
  if (form === 'family') {
    // the family name must not start like the first name ("Fabilla Fabana")
    for (let i = 0; i < 6; i++) {
      const family = familyName(c, g, r);
      if (family && !family.toLowerCase().startsWith(first.toLowerCase().slice(0, 3))) return { name: `${first} ${family}`, gender };
    }
    return { name: first, gender };
  }
  if (form === 'byname') {
    const by = pick(r, [...BYNAMES, ...c.bynames]);
    return { name: `${first} ${by}`, gender };
  }
  if (form === 'home') {
    const home = placeBase(c, { ...o, letter: '' }, r);
    return { name: home ? `${first} of ${home}` : first, gender };
  }
  return { name: first, gender };
}

/** A culture built from the writer's own words (one per line, or separated by commas or spaces). No gender. */
export function customCulture(text: string): Culture | null {
  const words = [...new Set(text.toLowerCase().split(/[\s,;]+/).map((w) => w.replace(/[^\p{L}'-]/gu, '')).filter((w) => w.length >= 2))];
  if (words.length < 3) return null;
  return { key: CUSTOM, label: 'My own words', gendered: false, starts: words, mids: [], endsM: words, endsF: words, endsN: [], family: compound(words, words), bynames: [], placeEnds: words };
}

/** Several different names. A name is left out when its lower case form is in `skip`, or when its first name is. */
export function makeNames(o: Options, count: number, skip: Set<string>): Made[] {
  const out: Made[] = [];
  const seen = new Set(skip);
  for (let i = 0; i < count * 40 && out.length < count; i++) {
    const m = makeOne(o);
    if (!m) { if (i > 80) break; continue; } // nothing fits the choices
    const key = m.name.toLowerCase();
    const head = key.split(' ')[0];
    if (seen.has(key) || (o.kind === 'person' && seen.has(head))) continue;
    seen.add(key);
    out.push(m);
  }
  return out;
}

/** The letters that some name of this culture can start with. */
export const lettersOf = (c: Culture): string[] => [...new Set(c.starts.map((s) => s[0].toUpperCase()))].sort();
