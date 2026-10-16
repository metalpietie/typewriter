// A stand-in for the Go backend, with a small demo book, so the app can run in a plain browser for the tutorial screenshots.
// It is used only by tools/screenshots. The app itself never loads it.

const rows: any[] = [];
let next = 1;
const add = (kind: string, data: any): number => { rows.push({ id: next, kind, data: JSON.stringify(data), updatedAt: 'x' }); return next++; };
const day = (n: number) => { const d = new Date(); d.setDate(d.getDate() - n); return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0'); };
const nonfiction = localStorage.getItem('__nf') === '1';

if (!nonfiction) {
  const mara = add('character', { name: 'Mara Finch', role: 'protagonist', aliases: 'Mara', age: '29', description: 'A clockmaker with ink-stained fingers and a habit of humming while she works.', traits: 'curious, stubborn, kind', backstory: 'She took over the shop when her grandmother died.', goals: 'Find out who wrote the letter, and why it was hidden.', notes: '', fields: [], color: '#8082ff' });
  const tom = add('character', { name: 'Old Tom', role: 'supporting', aliases: '', age: '71', description: 'The baker next door. He knows more than he says.', traits: 'gentle, secretive', backstory: '', goals: 'Keep an old promise.', notes: '', fields: [], color: '#e0a458' });
  const shop = add('location', { name: 'The clock shop', type: 'village', description: 'A narrow shop on Mill Street. Every clock in it shows a different time.', parent: null, tags: 'shop, home' });
  add('location', { name: 'Mill Street', type: 'village', description: 'A cobbled street that smells of bread.', parent: null, tags: '' });
  const book = add('book', { title: 'The Last Letter', synopsis: 'A clockmaker finds a letter that was hidden for fifty years.', order: 1, goal: 60000, chapterGoal: 3000 });
  const c1 = add('chapter', { book, order: 1, title: 'The letter', summary: 'Mara finds the letter and meets Tom.', goal: 0 });
  const c2 = add('chapter', { book, order: 2, title: 'Tom’s story', summary: '', goal: 0 });
  const commentId = next + 3; // the id that the comment below will get
  const s1 = add('scene', { chapter: c1, book, order: 1, title: 'Under the clock', status: 'draft', pov: mara, location: shop, characters: [mara], summary: 'Mara finds the letter.', goal: 0, tags: ['opening'],
    body: `<p>Mara found the letter under <span data-comment="${commentId}" class="cmt">the old clock</span>. The paper was yellow and soft, like cloth. She did not open it yet.</p><p>Outside, the bell of the bakery rang twice. Tom was early today.</p><p>She turned the envelope over. There was no name on it, only a drawing of a small brass key.</p>` });
  add('scene', { chapter: c1, book, order: 2, title: 'The shop at night', status: 'idea', pov: mara, location: shop, characters: [mara, tom], summary: 'Tom comes to the door.', goal: 0, body: '<p>The shop was quiet after dark. Every clock ticked a little out of step with the others.</p>' });
  add('scene', { chapter: c2, book, order: 1, title: 'Bread and secrets', status: 'idea', pov: tom, location: null, characters: [tom], summary: '', goal: 0, body: '' });
  const com = add('comment', { scene: s1, quote: 'the old clock', text: 'Who wrote this letter?', resolved: false, at: new Date().toISOString() });
  if (com !== commentId) throw new Error('comment id');
  add('draft', { title: 'What is in the letter?', body: '<p>Idea 1: a map to something under the shop.</p><p>Idea 2: an apology from Tom to Mara’s grandmother.</p><p>What if the key opens one of the clocks?</p>', order: 1 });
  add('draft', { title: 'Names for the bakery', body: '<p>Golden Crust. Two Loaves. The Warm Oven.</p>', order: 2 });
  [0, 1, 2, 3, 4, 6, 7].forEach((n, i) => add('wordday', { date: day(n), words: [420, 380, 510, 300, 650, 340, 480][i] }));
  add('setting', { key: 'dailyGoal', value: 400 });
} else {
  add('setting', { key: 'projectType', value: 'nonfiction' });
  const book = add('book', { title: 'Small Habits', synopsis: 'How tiny changes add up.', order: 1, goal: 50000, chapterGoal: 3000 });
  const c1 = add('chapter', { book, order: 1, title: 'Why habits matter', summary: 'Habits shape who we become.', goal: 0, takeaways: 'Small changes add up\nSystems beat goals' });
  add('scene', { chapter: c1, book, order: 1, title: 'The power of one percent', status: 'draft', summary: 'Tiny gains compound.', goal: 0, body: '<p>Getting one percent better each day sounds small. Over a year it adds up to nearly thirty-seven times better.</p>' });
  add('source', { type: 'book', title: 'Atomic Habits', authors: 'Clear, James', year: '2018', container: '', publisher: 'Avery', volume: '', issue: '', pages: '', url: '', accessed: '', edition: '', status: 'read', note: 'The one percent idea is in chapter 1.' });
  add('source', { type: 'article', title: 'How are habits formed', authors: 'Lally, Phillippa\nWardle, Jane', year: '2010', container: 'European Journal of Social Psychology', publisher: '', volume: '40', issue: '6', pages: '998-1009', url: '', accessed: '', edition: '', status: 'to-read', note: '' });
  add('source', { type: 'web', title: 'Sleep and health', authors: 'World Health Organization,', year: '2022', container: 'WHO', publisher: '', volume: '', issue: '', pages: '', url: 'https://who.int/sleep', accessed: '2026-01-02', edition: '', status: 'quoted', note: '' });
}

// The bookshelf: three shelves (projects) with books. Changes stay in memory, so dragging a book can be tried.
type FakeBook = { id: number; title: string; order: number; chapters: number; words: number; seed: number; hasCover: boolean; goalKind: string; goal: number; goalWords: number };
const shelfData: { id: string; name: string; current: boolean; problem: string; books: FakeBook[] }[] = [
  { id: 'default', name: nonfiction ? 'Small Habits' : 'The Clockmaker series', current: true, problem: '', books: [
    { id: rows.find((r) => r.kind === 'book')!.id, title: nonfiction ? 'Small Habits' : 'The Last Letter', order: 1, chapters: 2, words: 412, seed: 0, hasCover: false, goalKind: 'book', goal: 60000, goalWords: 412 },
    { id: 901, title: 'Winter Clocks', order: 2, chapters: 14, words: 61250, seed: 0, hasCover: false, goalKind: 'chapter', goal: 70000, goalWords: 61250 },
    { id: 902, title: 'The Brass Key and the Very Long Night of Whispering Gears', order: 3, chapters: 3, words: 8800, seed: 0, hasCover: false, goalKind: '', goal: 0, goalWords: 0 },
  ] },
  { id: 'p2', name: 'Short stories', current: false, problem: '', books: [
    { id: 11, title: 'Salt', order: 1, chapters: 5, words: 12040, seed: 0, hasCover: false, goalKind: 'scene', goal: 20000, goalWords: 12040 },
    { id: 12, title: 'Ten Small Doors', order: 2, chapters: 10, words: 33100, seed: 0, hasCover: false, goalKind: '', goal: 0, goalWords: 0 },
  ] },
  { id: 'p3', name: 'Ideas for later', current: false, problem: '', books: [] },
];
let fakeId = 1000;
const shelfOf = (id: string) => shelfData.find((s) => s.id === id)!;
const bookOf = (p: string, id: number) => shelfOf(p).books.find((b) => b.id === id)!;
const shelfStub = {
  Shelves: async () => JSON.parse(JSON.stringify(shelfData)),
  BookCover: async () => '',
  SetBookSeed: async (p: string, id: number, seed: number) => { bookOf(p, id).seed = seed; },
  SetBookOrder: async (p: string, ids: number[]) => { const s = shelfOf(p); s.books = ids.map((id, i) => ({ ...s.books.find((b) => b.id === id)!, order: i + 1 })); },
  AddBook: async (p: string, title: string) => { const id = fakeId++; const s = shelfOf(p); s.books.push({ id, title: title.trim() || 'New book', order: s.books.length + 1, chapters: 0, words: 0, seed: 0, hasCover: false, goalKind: '', goal: 0, goalWords: 0 }); return id; },
  AddProject: async (name: string) => { const id = 'p' + fakeId++; shelfData.push({ id, name, current: false, problem: '', books: [] }); return id; },
  MoveBook: async (from: string, id: number, to: string) => {
    await new Promise((r) => setTimeout(r, 400));
    const a = shelfOf(from), b = shelfOf(to), i = a.books.findIndex((x) => x.id === id);
    const [book] = a.books.splice(i, 1); const nid = fakeId++;
    b.books.push({ ...book, id: nid, order: b.books.length + 1 }); return nid;
  },
};

export const Store = {
  ...shelfStub,
  List: async (k: string) => rows.filter((r) => r.kind === k).map((r) => ({ ...r })),
  ListBy: async (k: string, f: string, id: number) => rows.filter((r) => r.kind === k && JSON.parse(r.data)[f] === id).map((r) => ({ ...r })).reverse(),
  Save: async (d: any) => { if (!d.id) { d.id = next++; rows.push({ ...d }); } else { rows.find((r) => r.id === d.id).data = d.data; } return { ...d, updatedAt: 'y' }; },
  Delete: async (_k: string, id: number) => { const i = rows.findIndex((r) => r.id === id); return i < 0 ? [] : rows.splice(i, 1); },
  LastBackup: async () => { const d = new Date(); d.setDate(d.getDate() - 3); return d.toISOString(); },
  Backup: async () => 'C:/usb/typewriter-backup.db',
  Restore: async () => {},
  AutoBackups: async () => [{ name: 'auto-2026-10-13.db', date: '2026-10-13', size: 183296 }, { name: 'auto-2026-10-12.db', date: '2026-10-12', size: 180224 }],
  RestoreAuto: async () => ({ restored: true, items: 5, safetyCopy: 'x' }),
  OpenBackupFolder: async () => {},
  DataFolder: async () => ({ path: 'C:/data', default: 'C:/data', isDefault: true, problem: '' }),
  Version: async () => '1.9.0',
  ChooseDataFolder: async () => '',
  InspectDataFolder: async (p: string) => ({ path: p, same: false, hasData: false }),
  MoveDataFolder: async () => {},
  DataPath: async () => 'C:\\Users\\you\\AppData\\Roaming\\typewriter\\typewriter.db',
  Projects: async () => [{ id: 'default', name: nonfiction ? 'Small Habits' : 'The Last Letter', size: 40960, current: true }],
  SwitchProject: async () => {}, NewProject: async () => 'p1', RenameProject: async (id: string, name: string) => { shelfOf(id).name = name; }, RemoveProject: async () => 'C:/x/removed.db',
};
export const Exporter = {};

// The phone sync section, with a made-up computer and one added phone.
let syncOn = false;
const phones: any[] = [{ id: 'aaaaaaaa-0000-4000-8000-000000000001', name: 'Pixel 7', added: '2026-10-10T10:00:00Z', last: new Date(Date.now() - 3600e3).toISOString() }];
const syncStatus = () => ({ on: syncOn, port: syncOn ? 47821 : 0, addresses: syncOn ? ['192.168.1.20'] : [], devices: phones, project: { id: 'default', name: nonfiction ? 'Small Habits' : 'The Last Letter' }, computer: 'DESKTOP-WRITER', fingerprint: 'ab'.repeat(32), announced: true });
const fakeQR = 'data:image/svg+xml;utf8,' + encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 21 21" shape-rendering="crispEdges"><rect width="21" height="21" fill="#fff"/>' + Array.from({ length: 21 * 21 }, (_, i) => { const x = i % 21, y = Math.floor(i / 21); const corner = (x < 7 && y < 7) || (x > 13 && y < 7) || (x < 7 && y > 13); return corner || ((x * 7 + y * 13 + x * y) % 5 < 2) ? `<rect x="${x}" y="${y}" width="1" height="1"/>` : ''; }).join('') + '</svg>');
export const PhoneSync = {
  Status: async () => syncStatus(),
  Start: async () => { syncOn = true; return syncStatus(); },
  Stop: async () => { syncOn = false; return syncStatus(); },
  StartPairing: async () => ({ code: '482913', expires: Math.floor(Date.now() / 1000) + 290, qr: fakeQR, payload: '{}' }),
  CancelPairing: async () => {},
  RemoveDevice: async (id: string) => { const i = phones.findIndex((p) => p.id === id); if (i >= 0) phones.splice(i, 1); },
};
