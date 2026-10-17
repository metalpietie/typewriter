// The texts of the page. Change them here, not in the components.
import nameCard from '../assets/screens/name-card.png';
import board from '../assets/screens/board.png';
import characters from '../assets/screens/characters.png';
import maps from '../assets/screens/maps.png';
import checks from '../assets/screens/checks.png';
import notes from '../assets/screens/notes.png';
import statistics from '../assets/screens/statistics.png';
import bookshelf from '../assets/screens/bookshelf.png';
import sources from '../assets/screens/sources.png';
import exportShot from '../assets/screens/export.png';

export const marquee = ['Chapters & scenes', 'Index-card board', 'Family trees', 'Hand-drawn maps', 'Custom calendars', 'Plot-hole checks', 'Citations & footnotes', 'Word · PDF · ePub', 'Daily backups', 'Vim keys'];

// The books on the shelf. h and w: the size of the spine. r: a small lean. from/to: the cover colours. glow: the light around it.
export const books = [
  { id: 'write', spine: 'Write', icon: '✎', tip: 'the editor', from: '#6d6fff', to: '#3b34b8', glow: '#8082ff', h: 290, w: 68, shot: nameCard,
    title: 'An editor that stays out of your way.', text: 'Chapters and scenes, with word goals for the book, the chapter and the scene.',
    list: ['Point at a name to see who it is', 'Focus mode, typewriter mode and Vim keys', 'History, snapshots and comments'] },
  { id: 'plan', spine: 'Plan', icon: '▦', tip: 'the board', from: '#1fb5c4', to: '#0d5c74', glow: '#2ee6f0', h: 250, shot: board,
    title: 'Move the story around on a board.', text: 'Every scene is an index card. Drag them until the story works.',
    list: ['Tags and story arcs', 'Revision passes', 'Split and merge scenes'] },
  { id: 'characters', spine: 'Characters', icon: '◎', tip: 'people & families', from: '#c04dd8', to: '#5e1f7a', glow: '#d77bff', h: 305, w: 72, shot: characters,
    title: 'Never forget an eye colour again.', text: 'Everyone in your story, with how they relate to each other.',
    list: ['Relationships and family trees', 'Factions, items and a mood board', 'A name generator'] },
  { id: 'maps', spine: 'Maps', icon: '⌖', tip: 'places & paths', from: '#2f9d6e', to: '#124a39', glow: '#4ee0a0', h: 268, r: -5, shot: maps,
    title: 'Draw the map. Follow the road.', text: 'Draw your world and see who is where, on any day of the story.',
    list: ['Regions and routes', 'Travel time between places', 'A timeline with your own calendar'] },
  { id: 'checks', spine: 'Checks', icon: '✓', tip: 'find what to fix', from: '#e0a243', to: '#7a4a12', glow: '#ffc56b', h: 282, w: 60, shot: checks,
    title: 'Catch it before your readers do.', text: 'Typewriter reads your book and points at what to look at again.',
    list: ['Plot holes and loose ends', 'Name spelling and repeated words', 'Style and reading level'] },
  { id: 'notes', spine: 'Notes', icon: '✐', tip: 'ideas, on phone too', from: '#e2507a', to: '#7a1d3c', glow: '#ff6f9c', h: 245, shot: notes,
    title: 'Ideas, wherever you are.', text: 'Keep notes next to your book, and take them with you on your phone.',
    list: ['Notes pages with @ links', 'Android app, synced over Wi-Fi', 'Ideas and quotes for non-fiction'] },
  { id: 'progress', spine: 'Progress', icon: '↗', tip: 'goals & stats', from: '#4a5578', to: '#20263b', glow: '#9aa6d6', h: 296, w: 66, shot: statistics,
    title: 'See the book grow.', text: 'Daily goals, a finish date, and a Pomodoro timer to keep going.',
    list: ['Statistics per day', 'Goal bars on every shelf', 'A finish date that adjusts'] },
];

// The cards under the shelf. size: tall (2 rows), wide (2 of the 3 columns), or normal.
export const cards = [
  { icon: '⌘', title: 'Your bookshelf', text: 'Every project is a shelf. Drag books between them. Covers made for you.', size: 'tall', shot: bookshelf },
  { icon: '❝', title: 'Built for non-fiction too', text: 'Sources, citations, footnotes, a bibliography and an index. The story tools step aside.', size: 'wide', shot: sources },
  { icon: '⇪', title: 'Export anywhere', text: 'Word, PDF, e-book and Markdown, with a cover and front pages.', size: 'normal' },
  { icon: '⌨', title: 'Vim keys', text: 'And a Code font that lines up. For writers who live in the terminal.', size: 'normal' },
  { icon: '⛨', title: 'Private by design', text: 'No account, no cloud. Each project is one file on your computer, with a backup every day.', size: 'wide', shot: exportShot },
  { icon: '✆', title: 'Notes on your phone', text: 'The Android app copies your notes both ways over your home Wi-Fi. No internet in between.', size: 'normal' },
];

export const readers = [
  { title: 'Novelists', text: 'Plan on a board, write in scenes, never lose a thread.' },
  { title: 'World builders', text: 'Lore, maps and timelines with your own calendar.' },
  { title: 'Private writers', text: 'No account and no cloud. One file per project, backed up daily.' },
];
