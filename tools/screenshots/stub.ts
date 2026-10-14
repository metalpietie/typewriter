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

export const Store = {
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
  Version: async () => '1.7.1',
  ChooseDataFolder: async () => '',
  InspectDataFolder: async (p: string) => ({ path: p, same: false, hasData: false }),
  MoveDataFolder: async () => {},
  DataPath: async () => 'C:\\Users\\you\\AppData\\Roaming\\typewriter\\typewriter.db',
  Projects: async () => [{ id: 'default', name: nonfiction ? 'Small Habits' : 'The Last Letter', size: 40960, current: true }],
  SwitchProject: async () => {}, NewProject: async () => 'p1', RenameProject: async () => {}, RemoveProject: async () => 'C:/x/removed.db',
};
export const Exporter = {};
