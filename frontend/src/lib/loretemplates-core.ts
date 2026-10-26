// Lore templates: a ready outline for a new lore note, with the questions that are worth an answer.
// The note is plain text: each part is a heading in capitals, then its questions. Write your answers under them,
// and delete the questions you do not need. No imports, so it can be tested alone.

export type LoreTemplate = { id: string; name: string; category: string; title: string; parts: [string, string[]][] };

export const LORE_TEMPLATES: LoreTemplate[] = [
  {
    id: 'magic', name: 'Magic system', category: 'Magic', title: 'New magic system', parts: [
      ['Source', ['Where does the power come from: a person, a place, a god, a substance, the world itself?', 'Is it limited? Can it run out?']],
      ['Who can use it', ['Is it born, learned, given or bought?', 'How rare is it? How are users found and trained?']],
      ['How it works', ['What does a user do: words, gestures, objects, will?', 'What can it do well? What can it never do?']],
      ['Cost and limits', ['What does it cost the user: energy, health, memory, years, money?', 'What goes wrong when it fails or is pushed too far?']],
      ['Rules the reader learns', ['Which rules does the reader learn early, so a later solution feels earned?', 'Which rule will be broken, and what does it cost?']],
      ['Society', ['Is magic legal, feared, worshipped or controlled? By whom?', 'How has it changed war, medicine, travel and daily life?']],
      ['History', ['When and how was it found or lost?', 'What is the worst thing magic has done?']],
    ],
  },
  {
    id: 'religion', name: 'Religion', category: 'Religion', title: 'New religion', parts: [
      ['Belief', ['What, or who, do the faithful believe in?', 'How did the world begin, and how will it end?']],
      ['Gods and holy figures', ['One god, many, none? Their names, domains and moods.', 'Are they real in this world? Do they answer?']],
      ['Practice', ['Prayers, rites, feasts, fasts. What happens daily, yearly, once in a life?', 'What is holy, and what is forbidden?']],
      ['Clergy and power', ['Who leads? How are they chosen?', 'How much power does the faith have over rulers and law?']],
      ['Afterlife and morals', ['What happens after death?', 'What makes a person good or wicked?']],
      ['Splits and enemies', ['Which sects, heresies or rival faiths exist?', 'What did the faithful fight over?']],
      ['In the story', ['Which character believes, doubts or fights it?', 'Which places, objects or days are holy?']],
    ],
  },
  {
    id: 'culture', name: 'Culture or people', category: 'Culture', title: 'New culture', parts: [
      ['Who they are', ['Where do they live, and how many are they?', 'What do they call themselves, and what do others call them?']],
      ['Daily life', ['Food, clothes, homes, work.', 'What does an ordinary day look like?']],
      ['Values', ['What do they admire? What shames them?', 'How do they treat strangers, the old, children, the poor?']],
      ['Family and customs', ['Marriage, birth, coming of age, death.', 'Greetings, gifts, taboos, manners.']],
      ['Arts and language', ['Music, stories, crafts, sports.', 'How do they speak? Sayings, curses, names.']],
      ['Neighbours', ['Who are their friends and enemies?', 'What do outsiders get wrong about them?']],
    ],
  },
  {
    id: 'government', name: 'Government or kingdom', category: 'Politics', title: 'New realm', parts: [
      ['Who rules', ['A king, a council, a church, a guild, the strongest?', 'How does a ruler get power, and how is it passed on?']],
      ['Law', ['Who makes the laws, and who judges?', 'Which crimes are the worst, and how are they punished?']],
      ['Power and money', ['Taxes, trade, armies. Who pays, who profits?', 'Which families or groups hold power behind the throne?']],
      ['The land', ['Borders, regions, the capital.', 'Which parts are loyal, and which are restless?']],
      ['Friends and enemies', ['Allies, rivals, old wars.', 'What treaty or grudge could start a war?']],
      ['Trouble', ['What is the realm afraid of?', 'What is about to change?']],
    ],
  },
  {
    id: 'history', name: 'Historical event or era', category: 'History', title: 'New event or era', parts: [
      ['When and where', ['When did it happen, and how long did it last?', 'Where did it happen?']],
      ['What happened', ['The causes, the turning point, the end.', 'Who were the winners and the losers?']],
      ['People', ['Who led, who fought, who suffered?', 'Who is remembered, and who is forgotten?']],
      ['After', ['What changed because of it?', 'What ruins, laws, songs or grudges are left today?']],
      ['The truth', ['What do the history books say?', 'What really happened, and who hides it?']],
    ],
  },
  {
    id: 'creature', name: 'Creature or species', category: 'Creatures', title: 'New creature', parts: [
      ['Looks', ['Size, shape, colour, sounds, smell.', 'How do you tell young from old, male from female?']],
      ['Where it lives', ['Which lands, climates or depths?', 'How common is it?']],
      ['How it lives', ['What does it eat? What eats it?', 'Alone or in groups? How does it breed and raise its young?']],
      ['Mind', ['How clever is it? Can it speak or be tamed?', 'What does it want?']],
      ['Danger and use', ['How dangerous is it, and how do people fight or avoid it?', 'Is it hunted, farmed, ridden or worshipped? What are its parts worth?']],
      ['Stories', ['What do people believe about it that is false?', 'What is true that few know?']],
    ],
  },
  {
    id: 'technology', name: 'Technology or invention', category: 'Technology', title: 'New invention', parts: [
      ['What it is', ['What does it do, and how does it work (as much as the reader needs)?', 'What does it look like?']],
      ['Who made it', ['Who invented it, when, and why?', 'Who controls how it is made?']],
      ['Who has it', ['Is it common, rare, secret, banned?', 'What does it cost?']],
      ['Limits', ['What can go wrong? What can it not do?', 'What does it need: fuel, parts, skill?']],
      ['Change', ['How has it changed war, work, travel or the rich and the poor?', 'Who fears it, and who wants it?']],
    ],
  },
  {
    id: 'language', name: 'Language', category: 'Language', title: 'New language', parts: [
      ['Who speaks it', ['Which peoples or places? How many speakers?', 'Is it rising, dying, sacred or secret?']],
      ['Sound', ['Which sounds are common? Which never appear?', 'Soft or harsh, fast or slow? Give a few sample words.']],
      ['Names', ['How are people and places named? Patterns, endings, titles.', 'Add the words you invent to the Glossary.']],
      ['Writing', ['Is it written? How, and by whom?', 'What is written down, and what is only spoken?']],
      ['Use in the story', ['When do characters switch to it, and why?', 'What do others think of people who speak it?']],
    ],
  },
  {
    id: 'trade', name: 'Economy and trade', category: 'Other', title: 'New trade', parts: [
      ['Money', ['Coins, barter, credit? What are they called, and what do they buy?', 'Who mints the money?']],
      ['Goods', ['What is made, grown, mined here? What is scarce?', 'What is the most valuable thing in this world?']],
      ['Routes', ['How do goods travel: roads, rivers, sea, magic?', 'Which routes are dangerous, and who guards them?']],
      ['Players', ['Merchants, guilds, banks, smugglers.', 'Who gets rich, and who is in debt?']],
      ['Trouble', ['Shortages, tariffs, piracy, a crash?', 'What would happen if one route closed?']],
    ],
  },
];

/** The text of a new note from a template: each part as a heading in capitals, with its questions under it. */
export function templateBody(t: LoreTemplate): string {
  return t.parts.map(([head, asks]) => `${head.toUpperCase()}\n${asks.map((a) => '- ' + a).join('\n')}\n`).join('\n');
}
