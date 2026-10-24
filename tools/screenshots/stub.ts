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
  const millStreet = add('location', { name: 'Mill Street', type: 'village', description: 'A cobbled street that smells of bread.', parent: null, tags: '' });
  const book = add('book', { title: 'The Last Letter', synopsis: 'A clockmaker finds a letter that was hidden for fifty years.', order: 1, goal: 60000, chapterGoal: 3000 });
  const c1 = add('chapter', { book, order: 1, title: 'The letter', summary: 'Mara finds the letter and meets Tom.', goal: 0 });
  const c2 = add('chapter', { book, order: 2, title: 'Tom’s story', summary: '', goal: 0 });
  const commentId = next + 3; // the id that the comment below will get
  const s1 = add('scene', { chapter: c1, book, order: 1, title: 'Under the clock', status: 'draft', pov: mara, location: shop, characters: [mara], summary: 'Mara finds the letter.', goal: 0, tags: ['opening'],
    body: `<p>Mara found the letter under <span data-comment="${commentId}" class="cmt">the old clock</span>. The paper was yellow and soft, like cloth. She did not open it yet.</p><p>Outside, the bell of the bakery rang twice. <span data-ref="character:${tom}">Tom</span> was early today.</p><p>She turned the envelope over. There was no name on it, only a drawing of a small brass key.</p>` });
  add('scene', { chapter: c1, book, order: 2, title: 'The shop at night', status: 'idea', pov: mara, location: shop, characters: [mara, tom], summary: 'Tom comes to the door.', goal: 0, body: '<p>The shop was quiet after dark. Every clock ticked a little out of step with the others.</p>' });
  add('scene', { chapter: c2, book, order: 1, title: 'Bread and secrets', status: 'idea', pov: tom, location: null, characters: [tom], summary: '', goal: 0, body: '' });
  const com = add('comment', { scene: s1, quote: 'the old clock', text: 'Who wrote this letter?', resolved: false, at: new Date().toISOString() });
  if (com !== commentId) throw new Error('comment id');
  add('draft', { title: 'What is in the letter?', body: '<p>Idea 1: a map to something under the shop.</p><p>Idea 2: an apology from Tom to Mara’s grandmother.</p><p>What if the key opens one of the clocks?</p>', order: 1 });
  add('draft', { title: 'Names for the bakery', body: '<p>Golden Crust. Two Loaves. The Warm Oven.</p>', order: 2 });
  [0, 1, 2, 3, 4, 6, 7].forEach((n, i) => add('wordday', { date: day(n), words: [420, 380, 510, 300, 650, 340, 480][i] }));
  add('setting', { key: 'dailyGoal', value: 400 });
  // A map of the town for the Maps & Paths picture: places, two regions, and the paths of Mara and Tom.
  const townSvg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1600 1000">
<defs><radialGradient id="p" cx=".5" cy=".45" r=".75"><stop offset="0" stop-color="#efe2c4"/><stop offset=".75" stop-color="#e2cfa6"/><stop offset="1" stop-color="#c9b083"/></radialGradient>
<pattern id="t" width="46" height="40" patternUnits="userSpaceOnUse"><path d="M23 6l11 20H12z" fill="#5f7d4e" opacity=".85"/><path d="M23 14l13 22H10z" fill="#4e6b40" opacity=".9"/></pattern></defs>
<rect width="1600" height="1000" fill="url(#p)"/>
<path d="M1600 640C1450 660 1380 720 1330 800S1250 960 1240 1000H1600Z" fill="#9cc1c9"/>
<path d="M1600 660C1470 680 1400 740 1355 815S1280 960 1272 1000" fill="none" stroke="#7fa8b2" stroke-width="3" stroke-dasharray="10 8"/>
<path d="M0 260C180 300 300 250 430 330S600 520 760 540 1000 470 1120 560 1260 720 1330 800" fill="none" stroke="#86b0bb" stroke-width="34" stroke-linecap="round"/>
<path d="M0 260C180 300 300 250 430 330S600 520 760 540 1000 470 1120 560 1260 720 1330 800" fill="none" stroke="#a8cdd4" stroke-width="18" stroke-linecap="round"/>
<path d="M40 60C200 20 360 70 420 160S360 330 230 360 30 300 20 200Z" fill="url(#t)"/>
<path d="M1180 60c80-30 200-20 280 40s70 160-10 200-220 20-280-40-70-170 10-200z" fill="url(#t)" opacity=".8"/>
<g fill="none" stroke="#a3896a" stroke-width="10" stroke-linecap="round"><path d="M250 980C330 820 520 700 700 660S980 600 1060 430 1200 220 1320 160"/><path d="M700 660C760 760 900 820 1060 830S1260 790 1330 800"/><path d="M640 300C700 420 690 560 700 660"/></g>
<g fill="none" stroke="#e9dcbc" stroke-width="4" stroke-linecap="round" stroke-dasharray="2 10"><path d="M250 980C330 820 520 700 700 660S980 600 1060 430 1200 220 1320 160"/><path d="M700 660C760 760 900 820 1060 830S1260 790 1330 800"/></g>
<g fill="#b98a5a" stroke="#7a5532" stroke-width="2"><rect x="600" y="600" width="34" height="26"/><rect x="650" y="585" width="40" height="30"/><rect x="720" y="610" width="30" height="24"/><rect x="760" y="640" width="36" height="26"/><rect x="620" y="690" width="32" height="26"/><rect x="580" y="650" width="26" height="22"/><rect x="800" y="590" width="28" height="22"/></g>
<g fill="#8a6a4a" stroke="#5c4128" stroke-width="2"><path d="M1040 400h40v-40l-20-26-20 26z"/><circle cx="1060" cy="372" r="8" fill="#efe2c4"/></g>
<g stroke="#7a5532" stroke-width="3" fill="none"><path d="M1028 420v40"/><path d="M1012 440h32"/></g>
<g fill="#6b5a46"><path d="M420 470l30-50 30 50z"/><path d="M460 480l40-64 40 64z"/><path d="M180 560l34-56 34 56z"/><path d="M230 570l26-44 26 44z"/></g>
<g font-family="Georgia, serif" fill="#5a4630" text-anchor="middle"><text x="800" y="90" font-size="58" font-style="italic" letter-spacing="6">Brassford</text>
<text x="1470" y="880" font-size="26" font-style="italic" fill="#3f6a74">Grey Sea</text><text x="900" y="515" font-size="20" font-style="italic" fill="#3f6a74" transform="rotate(-8 900 515)">River Tock</text></g>
<g transform="translate(1440 160)" fill="none" stroke="#5a4630" stroke-width="2"><circle r="44"/><circle r="36" stroke-dasharray="3 5"/><path d="M0-56L9 0 0 56-9 0Z" fill="#5a4630"/><path d="M-56 0L0 9 56 0 0-9Z" fill="#a3896a"/></g>
<g transform="translate(120 900)" fill="#5a4630" font-family="Georgia, serif" font-size="16"><rect width="200" height="8" fill="none" stroke="#5a4630" stroke-width="2"/><rect width="50" height="8"/><rect x="100" width="50" height="8"/><text x="100" y="32" text-anchor="middle">1 mile</text></g>
</svg>`;
  const town = add('map', { name: 'Brassford', image: 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(townSvg), w: 1600, h: 1000, bg: '#2a3a2f', notes: '', scaleDist: 8, scaleUnit: 'mi' });
  const pinAt = (id: number, x: number, y: number) => { const r = rows.find((q) => q.id === id)!; r.data = JSON.stringify({ ...JSON.parse(r.data), map: town, x, y }); };
  pinAt(shop, 0.4, 0.61);
  pinAt(millStreet, 0.5, 0.57);
  const place = (name: string, type: string, description: string, x: number, y: number) => add('location', { name, type, description, parent: null, tags: '', map: town, x, y });
  const tower = place('The clock tower', 'landmark', 'It has been ten minutes slow for fifty years.', 0.66, 0.36);
  const harbour = place('The harbour', 'town', 'Fishing boats, gulls, and the ferry to the islands.', 0.76, 0.71);
  const mill = place('The old mill', 'ruin', 'Closed since the flood. Tom will not go near it.', 0.4, 0.3);
  const cottage = place('Grandmother’s cottage', 'village', 'Where Mara grew up. Empty now.', 0.15, 0.42);
  place('Whisper Wood', 'region', 'Old trees, and a path nobody uses.', 0.14, 0.2);
  add('region', { map: town, name: 'Old Town', description: 'The oldest streets, inside the old wall.', color: '#e0a458',
    points: [{ x: 0.35, y: 0.55 }, { x: 0.55, y: 0.54 }, { x: 0.57, y: 0.8 }, { x: 0.45, y: 0.87 }, { x: 0.34, y: 0.8 }] });
  add('region', { map: town, name: 'Docks', description: '', color: '#2cd5e8', points: [{ x: 0.79, y: 0.75 }, { x: 0.92, y: 0.73 }, { x: 0.94, y: 0.9 }, { x: 0.81, y: 0.93 }] });
  const step = (character: number, order: number, location: number | null, x: number, y: number) => add('pathpoint', { character, map: town, x, y, order, label: '', chapter: null, location, when: '' });
  step(mara, 1, shop, 0.4, 0.61); step(mara, 2, tower, 0.66, 0.36); step(mara, 3, harbour, 0.76, 0.71); step(mara, 4, mill, 0.4, 0.3);
  step(tom, 1, shop, 0.43, 0.58); step(tom, 2, cottage, 0.15, 0.42);
} else {
  add('setting', { key: 'projectType', value: 'nonfiction' });
  const book = add('book', { title: 'Small Habits', synopsis: 'How tiny changes add up.', order: 1, goal: 50000, chapterGoal: 3000 });
  const c1 = add('chapter', { book, order: 1, title: 'Why habits matter', summary: 'Habits shape who we become.', goal: 0, takeaways: 'Small changes add up\nSystems beat goals' });
  add('scene', { chapter: c1, book, order: 1, title: 'The power of one percent', status: 'draft', summary: 'Tiny gains compound.', goal: 0, body: '<p>Getting one percent better each day sounds small. Over a year it adds up to nearly thirty-seven times better.</p>' });
  add('term', { word: 'Compounding', type: 'word', pronunciation: '', meaning: 'Small gains that build on earlier gains, so the total grows faster and faster over time.', accepted: '', notes: '' });
  add('source', { type: 'book', title: 'Atomic Habits', authors: 'Clear, James', year: '2018', container: '', publisher: 'Avery', volume: '', issue: '', pages: '', url: '', accessed: '', edition: '', status: 'read', note: 'The one percent idea is in chapter 1.' });
  add('source', { type: 'article', title: 'How are habits formed', authors: 'Lally, Phillippa\nWardle, Jane', year: '2010', container: 'European Journal of Social Psychology', publisher: '', volume: '40', issue: '6', pages: '998-1009', url: '', accessed: '', edition: '', status: 'to-read', note: '' });
  add('source', { type: 'web', title: 'Sleep and health', authors: 'World Health Organization,', year: '2022', container: 'WHO', publisher: '', volume: '', issue: '', pages: '', url: 'https://who.int/sleep', accessed: '2026-01-02', edition: '', status: 'quoted', note: '' });
}

// The bookshelf: three shelves (projects) with books. Changes stay in memory, so dragging a book can be tried.
type FakeBook = { id: number; title: string; order: number; chapters: number; words: number; seed: number; hasCover: boolean; goalKind: string; goal: number; goalWords: number };
const shelfData: { id: string; name: string; current: boolean; problem: string; group?: string; books: FakeBook[] }[] = [
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
  // a group: two practice novels on one shelf, each with a world of its own
  { id: 'p4', name: 'Glass Harbour', group: 'Practice novels', current: false, problem: '', books: [
    { id: 21, title: 'Glass Harbour', order: 1, chapters: 6, words: 18400, seed: 0, hasCover: false, goalKind: 'book', goal: 50000, goalWords: 18400 },
  ] },
  { id: 'p5', name: 'The Quiet Orchard', group: 'Practice novels', current: false, problem: '', books: [
    { id: 22, title: 'The Quiet Orchard', order: 1, chapters: 2, words: 3900, seed: 0, hasCover: false, goalKind: '', goal: 0, goalWords: 0 },
  ] },
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
  SetProjectGroup: async (id: string, group: string) => { shelfOf(id).group = group.trim(); },
  RenameGroup: async (old: string, name: string) => { for (const x of shelfData) if (x.group === old) x.group = name.trim(); },
  AddWorld: async (group: string, name: string) => { const id = 'p' + fakeId++; shelfData.push({ id, name: name.trim() || 'New book', group, current: false, problem: '', books: [] }); return id; },
  SplitShelf: async (id: string) => {
    const x = shelfOf(id); const group = x.group || x.name; x.group = group;
    const [first, ...rest] = x.books; x.books = first ? [first] : []; if (first) x.name = first.title;
    for (const b of rest) shelfData.push({ id: 'p' + fakeId++, name: b.title, group, current: false, problem: '', books: [{ ...b, order: 1 }] });
  },
  OrderGroup: async (group: string, ids: string[]) => { const at = shelfData.map((x, i) => (x.group === group ? i : -1)).filter((i) => i >= 0); const by = new Map(shelfData.map((x) => [x.id, x])); at.forEach((i, n) => (shelfData[i] = by.get(ids[n])!)); },
  CopyItems: async (_from: string, _to: string, refs: unknown[]) => ({ items: refs.length, links: 0 }),
  RemoveProject: async (id: string) => { const i = shelfData.findIndex((x) => x.id === id); if (i >= 0) shelfData.splice(i, 1); return 'C:/x/removed.db'; },
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
  Version: async () => '1.11.0',
  ChooseDataFolder: async () => '',
  InspectDataFolder: async (p: string) => ({ path: p, same: false, hasData: false }),
  MoveDataFolder: async () => {},
  DataPath: async () => 'C:\\Users\\you\\AppData\\Roaming\\typewriter\\typewriter.db',
  Projects: async () => [{ id: 'default', name: nonfiction ? 'Small Habits' : 'The Last Letter', size: 40960, current: true, group: '' }, ...shelfData.filter((x) => x.id !== 'default').map((x) => ({ id: x.id, name: x.name, size: 20480, current: false, group: x.group ?? '' }))],
  SwitchProject: async () => {}, NewProject: async () => 'p1', RenameProject: async (id: string, name: string) => { shelfOf(id).name = name; },
};
export const Exporter = {};

// Updates: a made-up newer version, so the update window can be seen. localStorage '__update' picks the answer.
export const Updates = {
  CheckForUpdate: async () => {
    await new Promise((r) => setTimeout(r, 600));
    const mode = localStorage.getItem('__update') ?? 'found';
    if (mode === 'error') throw new Error('could not reach GitHub. Check the internet connection, then try again');
    if (mode === 'none') return { current: '1.11.0', available: false, version: '', title: '', notes: '', page: '', canInstall: false, why: '' };
    return {
      current: '1.11.0', available: true, version: '1.11.0', title: 'Version 1.11.0: updates and an installer', page: 'https://github.com/metalpietie/typewriter/releases/tag/v1.11.0',
      canInstall: mode !== 'locked', why: mode === 'locked' ? 'Typewriter is in a folder that it may not change (C:\\Program Files\\Typewriter). Download the new version from the release page instead.' : '',
      notes: [
        '## Version 1.11.0: updates and an installer', '',
        '### New',
        '- **Updates:** Typewriter asks GitHub for a newer version, and updates itself.',
        '- **An installer** for Windows, with no admin prompt.', '',
        '### Fixed', '- Small things.', '',
        '### Notes', '- The installer is not signed, so Windows warns once.',
      ].join('\n'),
    };
  },
  InstallUpdate: async () => { await new Promise((r) => setTimeout(r, 1500)); },
  RestartOnClose: async () => {},
  RestartToUpdate: async () => { await new Promise((r) => setTimeout(r, 800)); },
};

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
