# Typewriter

A desktop app for writing a book or a book series. It keeps your story world in one place: characters, how they relate, places, lore, maps, character routes, and a timeline.

It runs on your computer. Your data stays in one file. No account and no internet needed.

Built with [Wails 3](https://v3.wails.io/) (Go), Svelte, and SQLite.

## Run it

Build once, then start the app:

```bash
wails3 build
bin\typewriter.exe
```

For live editing while you change the code:

```bash
wails3 dev
```

### Linux

You can build the Linux app on Windows with Docker. This works because the build runs inside a Linux container.

1. Install Docker Desktop and start it.
2. Build the image once (about 800 MB):

   ```bash
   wails3 task setup:docker
   ```

3. Build the Linux app:

   ```bash
   wails3 task linux:build
   ```

   The program is `bin/typewriter`. It is for 64-bit Linux (amd64). The Windows program is `bin\typewriter.exe`, but the Linux build can remove it. Run `wails3 build` again to get the Windows program back.

On the Linux computer:

1. Install the window parts. On Debian or Ubuntu: `sudo apt install libgtk-4-1 libwebkitgtk-6.0-4`. On Fedora: `sudo dnf install gtk4 webkitgtk6.0`. On Arch: `sudo pacman -S gtk4 webkitgtk-6.0`.
2. Copy `typewriter` over, then run `chmod +x typewriter` and `./typewriter`.
3. Your data is in `~/.config/typewriter` (the file `typewriter.db`, the `projects` folder, and the `backups` folders). To move your book from Windows, close the app, then copy the `typewriter.db` file from `%APPDATA%\typewriter` to that folder.

On Linux, **Open the backup folder** uses `xdg-open`. To build packages (.deb, .rpm, AppImage), see `build/linux` and `wails3 task linux:package`.

If you build on a Linux computer instead, install Go, Node.js, `gcc`, `libgtk-4-dev` and `libwebkitgtk-6.0-dev`, then run `wails3 build`.

### Requirements for building

- Go 1.25 or newer
- Node.js and npm
- The Wails 3 CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
- On a fresh clone, run `cd frontend && npm install` first.

## Your data

Everything lives in one SQLite file:

```
%APPDATA%\typewriter\typewriter.db
```

The path also shows at the bottom of the left menu. **Use the Backup buttons (see "Export & Backup" below) to protect your work.**

### Where your data is kept

The bottom of the left menu shows the data file in use and the app version, for example "Typewriter 1.0.0".

To keep your data in another folder, press **Change folder…** under the data file. Then press **Choose another folder…** and pick a folder.

- **A folder with no Typewriter data:** the app asks to **Copy and use**. It copies all your projects and backups there, then uses the copy. The old files stay where they are. Delete them yourself when you are sure the copy is good.
- **A folder that already has Typewriter data:** the app asks to **Use that data**. It shows that data. Your current data stays where it is. Nothing is overwritten.
- **Go back to the normal folder** (it shows when you use another folder) switches back. It uses whatever data is in the normal folder. If you copied your data away and then kept writing, the normal folder has the older copy.
- The app reloads after a change.
- Pick a normal folder on a local drive. Do not pick a folder that Dropbox, OneDrive or Google Drive keeps in sync while the app is open. They can damage the data file. Use the Backup buttons for a second copy.
- If the folder is on a drive that is not plugged in when the app starts, the app opens the normal folder, and a red **Data folder problem** button shows in the menu. Plug the drive in and restart the app to go back to your folder.

The choice is saved in `datadir.json` in the normal folder (`%APPDATA%\typewriter` on Windows, `~/.config/typewriter` on Linux). Delete that file to go back to the normal folder by hand.

### Projects

A **project** is one book or one series. Each project has its own data file, so the characters, places, maps and text of one project never mix with another.

- The button at the top of the left menu shows the open project. Click it to see the list.
- **Open** switches to another project. The app reloads and shows that project.
- **New project** makes an empty project and opens it.
- **Rename** changes the name.
- **Remove** (then **Sure?**) takes a project off the list. The file is not deleted. It moves to `%APPDATA%\typewriter\projects\removed`. To get it back, move the file to `%APPDATA%\typewriter\projects` and ask for help adding it to `projects.json`.
- You cannot remove the project that is open.

Your first project is the old file, `typewriter.db`. New projects are files in `%APPDATA%\typewriter\projects`. The list is in `%APPDATA%\typewriter\projects.json`. The app opens the project you used last.

Backups work on the open project. Each project has its own automatic copies. The first project uses `%APPDATA%\typewriter\backups`. Another project uses `%APPDATA%\typewriter\backups-<id>`, where `<id>` is the project's file name without `.db`.

The app saves as you type. There is no save button.

## Using the app

Use the menu on the left to switch screens. Press **Ctrl+K** anywhere to search. The menu has headings (Story, People, Worldbuilding, Review) to keep the screens in groups. The outlined button at the bottom of the menu switches between dark and light theme.

Most screens have a list on the left. Drag its right edge to make it wider or narrower. You can also click the edge and press the left or right arrow key. Double-click the edge to go back to the normal width. The app remembers the width.

### Draft

The **Draft** tab is a rough notepad for your story idea. Nothing here is part of the book.

- Press **+ New** to add a page. Give it a title, such as "Idea", "Ending" or "What if".
- Write in the same editor as the Write tab: bold, italic, headings, lists, and `@` links.
- Use the **↑** and **↓** buttons to put the pages in order.
- The search box above the list looks at the title and the text of every page.
- Draft pages are not counted in your word counts, checks or exports.


### Write

The Write tab reopens the book, chapter, and scene you had open last. This also works after you close the app.

1. Press **+ Book**, then give the book a title and a place in the series.
2. Press **+ Chapter**. A chapter holds one or more **scenes**. The new chapter starts with Scene 1.
3. Write in the editor. It looks like the page you will export. The toolbar has bold, italic, underline, three heading sizes, lists, quotes, a scene break (`* * *`), and undo/redo. The usual shortcuts work: Ctrl+B, Ctrl+I, Ctrl+U. The word count updates as you type.
4. Reorder by dragging. Drag a chapter or a scene in the left list and drop it where you want it. A line shows the spot. You can also use the ↑ and ↓ buttons in the right panel.

Chapters you wrote before the editor existed open as plain paragraphs. They convert when you edit them.

#### Finish date

In the book panel of the Write tab, set a **Book goal (words)** and a **Finish by** date. The app then says how many words to write each day to finish on time.

- Today and the finish day both count as days to write.
- It compares the number with what you wrote on a normal day: the average of the 7 days before today. It says **On track**, or **Behind** with the day you would finish at that pace.
- It counts all your writing, not one book, so with several books the pace is for your whole writing.
- When the goal is reached, it says so. When the date has passed, it says how long ago, and you can pick a new date.
- Books with a finish date also show a **Finish line** card on the Dashboard.

#### Scenes

A scene is a part of a chapter with its own place, point of view, time, and cast. Use scenes when a chapter jumps around.

- In the left list, the open chapter shows its scenes. Press **+ Scene** to add one. Click a scene to write it.
- The **Scene** panel on the right holds the status (idea, draft, revised, final), point of view, place, story time, a timeline number, who is in it, notes and plan, and a word goal.
- A scene with only notes and no text is a plan. Fill in the text later.
- **Drag a scene** up or down to reorder it. Drop it on another chapter's name to move it to the end of that chapter. **Move to chapter** in the panel does the same.
- The **Chapter** panel holds the chapter goal and summary, and shows everyone who appears in any of its scenes.
- The scene title is for you. It does not print.
- Older chapters that had their own text, cast, and place are converted once, when the app starts. The text becomes Scene 1 of that chapter.

Scenes also show up elsewhere: each scene is its own row on the **Timeline**, **Who's where** counts a character as present in a chapter if they are in any scene, a character's page lists their scenes, and a map path step can name a chapter and a scene.

When you export, the scenes of a chapter are joined with a scene break (`* * *`). The first paragraph of each scene has no indent.

#### Look of the page

Press **Look** above the editor to change how the text looks. The page changes while you click. The choice belongs to this computer, not to the project, so it is not in your backups.

- **Font**: Serif, Book, Sans, or Typewriter.
- **Text size** and **Line spacing**.
- **Page width**: Narrow, Medium, or Wide. **Auto** is the old way: the whole page, and a narrow column in focus mode. The exports do not change.
- **Typewriter mode**: the line you write on stays a little above the middle of the screen, and the page scrolls as you type. The first lines of a scene also sit in the middle.
- **Focus line**: every paragraph but the one you write in is dimmed.
- **Back to normal** puts all choices back.

Focus mode uses the same choices.

#### Tags on scenes

In the **Scene** panel (right side of the Write tab), under the story time, type a tag and press **Enter** or **,** to add it. Examples: flashback, dream, fight. Tags are lower case. The tags you used on other scenes show below as small buttons: press one to add it here. Press **✕** on a tag to take it off.

- When at least one scene has a tag, the left side of the Write tab shows **Show scenes tagged**. Pick a tag to list only those scenes. Chapters without such a scene are dimmed, and each chapter says how many scenes it has with the tag. Pick **All scenes** to see everything again.
- The **Board** has the same list at the top, and each card shows its tags.
- Tags are only for you. They do not print in exports.

#### Focus mode

Press **F11** (or **Ctrl+Shift+F**, or the **Focus mode** button) to hide every menu and panel. You see only the text. The window goes full screen. A small word count and save status sit in the corner. Press **Esc** to leave.

#### Find and replace

Press **Ctrl+H** (or the **Find & replace** button). Type the text to find. Every scene that holds it is listed, with the words around each match.

- **Match case** and **Whole word** narrow the search. **All books** searches the whole series.
- Click a scene name to jump to it.
- **Replace here** changes one scene. **Replace all** changes every scene in the list.
- Bold and italic stay as they are. A match that runs across a style change (half bold, half plain) is not found.
- Before each change, the old text of that scene is saved in History.

#### History

Press **History** to see old copies of the open scene. Click one to read it. **Go back to this** puts it back. The text you had just before is saved first, so you can undo going back.

- The app keeps a copy of the old text when you start writing, at most once every 10 minutes per scene. It keeps the last 40 of these.
- **Save a copy now** makes a copy that you name. The app never removes these by itself.
- Deleting a scene deletes its copies. If you press **Undo** on the delete message, they come back.
- **What changed** (next to **Read**) shows the words that changed, word by word. Words in green with an underline were added. Words in red with a line through were cut.
  - **This copy → now** compares the copy you picked with the scene as it is today.
  - **Copy before → this copy** compares it with the copy just older than it. The oldest copy has nothing to compare with.
  - A very large rewrite (more than about 1,500 changed words) shows all old words as cut and all new words as added.

#### Comments

Select some text in a scene and press **💬** in the toolbar. The text is marked, and a note card opens in the **Comments** panel at the top of the right side. Type your note.

- Click the quote on a card to jump to its text. Click the marked text to find its card.
- **Resolve** hides the mark on the text and moves the card to the "resolved" list. **Reopen** brings it back.
- **Delete** removes the card and the mark.
- If you delete the commented text, the card stays and says "(text deleted)".
- The scene list shows 💬 and the number of open comments.
- Comments never print in Word or PDF.

#### @ links and backlinks

Type **@** in a scene (or press **@** in the toolbar). A list opens with your characters, places, and lore notes. Keep typing to narrow it. Use the arrow keys and **Enter** (or click) to pick one.

- The name goes into the text as a link, shown in the accent colour with a dotted line. It prints as normal text.
- **Ctrl+click** a link to open that character, place, or note.
- The link stays if you rename the item later. The text does not change by itself.
- A tagged character counts as in the scene for the name check.
- On every character, place, and lore note, **Appears in** lists the scenes that tag it with @ or write its name. Click a row to open that scene.

#### Name check

If a scene names a character who is not in its cast (and is not its point of view), the Scene panel shows a warning and the scene list shows ⚠.

- It looks for the full name, the first name, and each name in "Also called". Whole words only, so "Aldric" does not match "Aldrich".
- **Add to cast** puts the character in the scene.
- **Not here** means they are only mentioned. The warning stays away for that scene.

### Board

The **Board** tab shows your book as index cards. Each chapter is a column. Each scene is a card.

- A card shows the scene title, a summary, a status, the point of view, the place, and the word count. Type in the summary box to change it. Use the status list to change the status.
- **Drag the grip ⠿** on a card (top right) to move it up or down in its chapter, or into another chapter. Only the grip starts a drag, so you can click and type on the card as normal. No text is selected while you drag. A line shows where it will land. Drop it on the empty part of a column to put it at the end.
- **Drag the grip ⠿** left of a chapter's title to move the whole chapter left or right. The chapter numbers update.
- Hold a card near the left or right edge to scroll the board.
- **Click a card's title** to open that scene in the Write tab.
- **+ Scene** adds a scene to a chapter and opens it. **+ Chapter** adds a column.
- **Colour by status** or **Colour by point of view** sets the colour bar on the left of each card. A point of view uses the character's colour.
- Pick the book at the top. The Board starts on the book you wrote in last.

Changes on the Board are the same changes as in the Write tab. The new order shows there, and in your exports.

### Story arcs

A story arc is one plot thread, such as "The crown" or "Ana and Bram". The **Story arcs** tab shows every thread across a book.

- Press **+ New arc**. Give it a name and a colour. Add notes about how it starts, turns and ends.
- Tag the scenes that belong to the arc. You can do it in two ways: tick the arc in the right-hand panel of a scene in the Write tab, or use **Add a scene…** on the arc's page.
- For each scene you can say what it does for the arc: **starts**, **turns**, **ends**, or just **touches** it.
- **All arcs** is the overview. It has one row for each arc and one column for each chapter. A dot is a scene. The letters **S**, **T** and **E** mark a start, a turn and an end. A line shows how long the arc runs. Click a dot to write that scene.
- **To check** lists hints: no scene marked as the start or the end, a start that is not the first scene, scenes after the end, or 3 or more chapters in a row with nothing for the arc.
- Cards on the Board show a small coloured dot for each arc of the scene.
- Deleting an arc only removes the tags. It never deletes a scene. You can undo it for a short time.

### Characters

Press **+ New** to add a character. Fill in the role, looks, traits, backstory, goals, and notes. Add your own fields, such as "Eye colour". Pick a colour. The same colour is used on the graph and the maps.

Press **Add portrait** to give a character a picture. The app cuts the middle square out of your picture and shrinks it to 480 px, so the data file stays small. The portrait shows in the character list. Use **Remove** to take it away.

**Born** and **Died** are dates, picked with your own calendar if you set one up (or plain year numbers if not). They are the same timeline numbers that scenes and events use. With a birth date the app can tell how old a character is on any day:

- Under **Age on a date**, pick a day to see the age. A character is "not born yet" before the birth date, and "dead" after the death date.
- The same card lists the scenes and events the character is in, with the age at each one.
- In the Write tab, the cast list of a scene shows each age ("age 23") when the scene has a date. The same goes for the people in a Timeline event.
- A red note means the date cannot be true. The **Unfinished** list in the Checks tab also lists these: a scene or event that has someone in it before their birth or after their death.
- The age is counted in whole years. It goes up on the birthday (the same month and day).
- The old **Age** field is now only a note, for example "about thirty". A birth date counts as an age, so the Unfinished list stops asking for one.

The bottom of each character shows their relationships, their chapters, and how many map steps they have.

### Name generator

Press **Name generator…** on the **Characters** screen (people) or on the **Places** list in the **World** screen (places).

- Pick a **style**: Elvish, Dwarvish, Norse, Old English, Desert, or Harsh. Each style builds names from its own sound pieces. Places get endings such as "-ford", "-heim" or "-hold".
- **My own words** builds names from words you give it, for example `frost, ember, hollow, stone` gives "Emberfrost" and "Stonefrost". Your words are saved.
- The panel shows 12 names. Press **Generate again** for more. Click a name to copy it.
- **＋ Character** or **＋ Place** makes a new item with that name and opens it.
- Names already used by a character (or its aliases), place, faction, or item are left out.

The names are made up from rules. Some sound odd, so press **Generate again**.

### Relationships

This screen draws a graph of how characters connect.

1. Under **New link**, pick two people and a type, then press **Link**.
2. Pick a link in the list to edit its label, start time, details, and direction. Tick **One-way** to add an arrow.
3. Use **Show types** to hide link types you do not need.
4. Click a person to focus on their links. Click empty space to clear.
5. Drag a person to move them. They stay where you drop them.
6. Scroll to zoom. Drag empty space to pan. **Re-arrange** lays the graph out again.

### World

Two lists live here:

- **Lore notes:** history, magic, culture, religion, politics, and more. Filter by category and tags.
- **Places:** cities, ruins, regions, and so on. A place can sit inside another place. Press **Add picture** to give a place a picture. It is shrunk to 480 px on the long side.

### Glossary

The **Glossary** tab holds the made-up words of your world: a greeting, a curse, a title, a unit of time.

- Press **+ New**. Write the word, its type, how it is said, what it means, and notes.
- **Other spellings that are fine** is a comma separated list. The spelling check leaves these alone.
- Under **Spelling** you see words in your scenes that are almost this word, for example "ketah" when your word is "ketha". Press **Change all** to fix it in every scene. The old text goes to History. Press **It is fine** to keep the other spelling.
- **Used in the text** lists the scenes that use the word. Click one to write it.
- A red mark in the list shows a word that has another spelling in the text.
- The same hints show in the **Names** list of the Checks tab.
- The Story bible export has a Glossary section, in alphabetical order.

### Factions

A faction is any group: a kingdom, guild, order, house, army, cult.

- Set a type, a colour, a headquarters (one of your places), and the **ranks**, one per line, highest first.
- **Members:** add characters and give each a rank. Members are listed highest rank first. A character can be in many factions.
- **Allies and enemies:** link a faction to another one as ally, enemy, rival, neutral, or “rules” (this faction rules the other).
- A character's page lists their factions and ranks.
- You can tag a faction with **@** in a scene. **Appears in** shows where it is named.

### Items

An item is a sword, a crown, a letter, anything that moves through your story.

- Fill in the type, a description, and its history and powers.
- **Journey:** each step says who has the item and where it is, and optionally when and in which scene. A new step starts as a copy of the last one, so you only change what moved. Use ↑ ↓ to put steps in order.
- **Where it is now** comes from the last step.
- A character's page lists the items they hold now.
- Items can be tagged with **@** in a scene.

### Family tree

The tree is drawn from the “family” links of the Relationships tab. Each link needs a role:

- **A is a parent of B:** B hangs under A. Put the parent as person A.
- **A and B are partners:** they sit side by side and share their children.
- **A and B are siblings:** use this when you do not know the parents. If the parents are linked, siblings show up from them.

You can add links right on the Family tree tab. A family link with no role appears in a list on the left, where you pick its role. Pick **Family of …** to show one family only. Drag to move, scroll to zoom, click a person for details.

### Maps & Paths

1. Press **+ Image** to load a map picture, **+ Blank** for a plain canvas, or **✎ Draw a map** to draw one in the app (see below).
2. Use the toolbar above the map:
   - **✋ Move:** drag to pan. Drag pins and steps to move them. Click one to edit it.
   - **📍 Add place:** click the map to drop a pin. Name it in the right panel.
   - **⬠ Region:** draw a border. Click the map to place corners. Click the first corner, or press **Finish** (or Enter), to close it. Backspace removes the last corner. Esc cancels.
   - **〰 Path:** draw a character's route. Click the map to add numbered steps. Click a pin to snap a step to that place.
3. In the left panel, press **Draw** next to a character to pick whose route you draw. Use the tick boxes to show or hide routes.
4. Click a step to say what happens there. You can add story time, a chapter, and a place.
5. In the right panel, reorder a character's steps with ↑ and ↓.
6. Scroll to zoom. **Fit** shows the whole map.

#### Regions

A region is a shape on one map, such as a kingdom or a forest. Click a region in **Move** mode to select it. In the right panel you can:

- Give it a name, a colour and a description. The name shows on the map.
- Drag a corner to move it. Click the outline to add a corner. Click a corner, then press **Remove corner**.
- See the places whose pins lie inside it. When you select a pin, the panel names the region it is in.
- Hide all regions with the **Regions** tick box in the tool bar.

Regions are saved with the map. Deleting a map deletes its regions.

#### Travel time

Open **Scale and travel** in the left panel and type how wide the whole map is, for example `500` `km` (miles, leagues, or any word you like). The app then knows the distance between any two spots.

- **Ways to travel:** the app adds On foot (30 a day), Horse (60), and Cart (25). Change the names and numbers to fit your world and your unit. Add more with **+ Way to travel**.
- **📏 Measure:** click two points, or two pins, to see the distance and the days for each way to travel.
- **Paths:** the right panel shows the distance and days between each pair of steps, and for the whole path. Pick the way to travel above the list. When both steps have a timeline number, it also shows how many days the story gives.
- **Check:** if a character must be at two spots on a map and the story time between them is too short, even for your fastest way to travel, the **Checks** tab shows a plot hole. See *Checks* below.

Distances are straight lines, so real roads are longer. Leave a margin.

#### Who is where

When characters have steps, scenes, or events with a timeline number and a pinned place on the map, a **time bar** shows at the bottom of the map.

- Drag the slider, or press **▶** to play. Everyone moves along their route at once. Each person is a coloured circle with a name, and a thick line shows the way they have come.
- A person is not shown before their first stop. After their last stop they stay there, faded (**Show last seen**). Turn that off to hide them.
- Between two stops on the same map, a person is on the straight line between them.
- The right panel lists everyone at that moment: "is at", "is travelling from … to …", or "last seen at". People on another map show the map name. Click one to go to that map.
- Use the tick boxes under **Character paths** to choose who shows.
- With a calendar set, the time shows as a date.

#### Drawing a map

Press **✎ Draw a map**. Pick a size first (Small, Medium, or Large). The drawing tool opens full screen. Press **← Done** when you finish. Then place pins and draw character paths on it as on any other map. Press **✎ Edit drawing** to open it again.

**Draw on a picture you loaded:** pick a map that has a picture, then press **✎ Draw on this picture**. The picture stays at the bottom, and the layers sit on top. Use lines, labels, stamps, terrain paint, and the grid to mark it up. Tick *Show the original picture* on the right to hide it while you work. **Remove drawing, keep picture** brings back the picture alone and deletes the drawing. A picture over 3200 pixels wide or tall is made a little smaller.

Tools on the left. The keyboard shortcut is in brackets:

- **Select (V):** click something to select it. Drag to move it. Drag the small squares to reshape a line or a coast. Press Delete to remove it. Ctrl+D copies it.
- **Pan (H):** drag to move the view. Scroll to zoom. Hold Space to pan with any tool.
- **Land (L):** draw around an island or continent. The shape closes by itself, gets a smooth coast, and a light shallow-water edge. Choose *Freehand*, or *Click points* (click each corner, then double-click or press Enter).
- **Terrain brush (B):** paint grass, forest, hills, mountains, desert, snow, swamp, beach, or water. You can set size, softness, strength, texture, and a custom color. Press [ and ] to change the size. A texture (stipple, hatching, cross hatching, waves, grass tufts, or forest) adds small marks inside the paint, and the marks line up from stroke to stroke.
- **Eraser (E):** rubs out terrain paint on the active layer.
- **Lines (R):** rivers (thin at the source, wide at the mouth), roads, trails, and borders.
- **Stamps (S):** mountains, hills, pines, oaks, castles, cities, villages, ports, towers, and caves. Turn on *Scatter* and drag to drop many, such as a forest.
- **Text label (T):** click the map, then type the text on the right. Choose font, size, rotation, letter spacing, and color. Choose **Along a curve**, then draw a line: the text follows it, which suits rivers, coasts, and mountain ranges. Drag the squares to reshape the curve.

On the right:

- **Layers:** each map has Land, Terrain, Lines, Icons, and Labels layers. Hide, lock, rename, reorder, or delete them. Add more with **+ Layer**. A tool draws on the active layer of its type.
- **Map style:** water, land, and ink colors, with three presets. You can also give the land and the water a texture.
- **Grid:** a square or hexagon grid on top of the map, for travel and battle maps. Set the size, strength, and color. If the map has a scale (see *Travel time*), the panel says how far one square or hexagon is. The grid shows in the exported PNG and in the map picture.
- **Undo and redo:** the buttons at the top, or Ctrl+Z and Ctrl+Y. Undo keeps 60 steps while the editor is open.
- **Export PNG:** save the map as a picture at 1× or 2× size.

Changes save by themselves.

Places made here also show in the **World** screen. Use **Unpin** to take a place off a map without deleting it.

### Timeline

- **Story time:** one vertical timeline of events, chapters, and path steps, in order. Press **+ New event** for things like battles or births. You can hide types, or show one character only.
- **Who's where:** a grid. Rows are characters and columns are chapters. ◉ means point of view, ● means present, and a number counts map steps.

The timeline sorts by a **number**, such as a year. "Year 312" is only a label. Fill in the number too. The field is called **Position** on events, and **Timeline position** on chapters and path steps. Items with no number stay off the timeline.

**Your own calendar:** press **Calendar…** on the Timeline screen (or **Set up your own calendar** next to any date field). Add your months and how many days each has. You can also add weekday names, names for years that repeat (such as animals), and text before or after the year ("Year 312 AE"). **Use a standard calendar** gives you a starting point to change.

With a calendar, the date fields on scenes, events, and path steps change. You pick a **year**, a **month**, and a **day**. Leave the month empty for a whole year. The app saves the timeline number for you (the year plus a part for the day). It also fills the label, for example "Tuesday, 14 Frostmoon, Year 312 AE". If you write your own label, the app keeps it. Lists and the timeline show the label, or the date in your calendar when there is no label.

Notes: there are no leap years. If you change the months later, numbers you already saved stay the same, so a date can read differently. The plot hole check works on the numbers, so days now have their own numbers.

### Dashboard

The **Dashboard** is the first screen. It shows:

- today's words, your daily goal, your streak, and a **Continue writing** button that opens the scene you changed last,
- a **Time for a backup** card when your last backup is old (see *Backup reminder* below),
- a **Finish line** card for each book that has a finish date (words a day, and whether you are on track),
- what needs attention: open plot holes, unfinished things, and open comments (click to go to the Checks tab),
- your 8 most recent changes (scenes, characters, places, lore notes, factions, items, events). Click one to open it,
- a **to-do** list. Type a task and press Enter. Tick it when done. **Clear done** removes the ticked ones.

### Statistics and daily goal

The **Statistics** tab shows your progress.

- **Daily writing goal:** set the words you want to write each day. A bar shows today. The **streak** counts days in a row that reached the goal. Today does not break the streak until the day ends. With no goal, any writing counts.
- **Words per day:** a bar chart for 14, 30 or 90 days. Green bars reached the goal.
- **Words per chapter** and **words per character** (all scenes they are in, and scenes told through them).
- **Scenes by stage:** idea, draft, revised, final.

Words are counted while you type in the Write tab. Each day keeps one tally. Cut words are taken off that day, but a day never shows less than 0. Text you paste in counts as written. The Write tab shows today's count under the text and in focus mode.

### Checks

The **Checks** tab has five lists: **Plot holes**, **Unfinished**, **Names**, **Repeats** and **Style**. A red number next to the tab name shows how many plot holes are open.

**Unfinished** lists things that look incomplete: characters with no age (a birth date counts), description, backstory, relationships or scene; places on no map or with no description; scenes with no point of view, place, timeline number, summary or text; a scene or event dated before someone in it was born or after they died; factions with no members or headquarters; items with no description or journey; events with no number or place; empty lore notes. Click a row to open it. Press **Skip** if it is fine as it is. Skipped rows go to a list where you can bring them back.

**Names** finds a name that is spelled two ways. It reads your scenes. It flags a capitalised word that is almost a name from your Characters (including "also called" names), Places, Factions or Items, for example "Aldrick" when you have "Aldric". It also flags a rare word that is almost a name you use often in the text, even if that name is not in your data. It also flags a word that is almost one of your Glossary words, for example "ketah" when you have "ketha".

- Each row shows the word, the spelling it probably should be, and where it is. Click a place to open that scene.
- **Change all to “…”** fixes it in every scene. Your bold, italic and other styles stay. The old text of each scene is saved, so **History** in that scene can bring it back.
- **Skip** hides a row that is fine, for example a different person with a close name. Skipped rows go to a list where you can bring them back.
- Words that also appear in lower case in your text are left alone, so ordinary words at the start of a sentence are not flagged. Names with 3 letters or fewer are not checked. This is a guess from spelling, so a few rows may be wrong. Use **Skip** for those.
- It does not check ordinary spelling. Use the spell check of the window for that.

**Repeats** helps you tighten your prose. It reads your scenes. Pick one book or all books, then pick a part:

- **Words close together:** a word of 4 or more letters used again within 20 words, for example "the cold wind … the cold hills". Each scene is one card. The repeated words are highlighted.
- **Same sentence start:** 3 or more sentences in a row that begin with the same word ("He ran. He jumped. He fell.").
- **Most used words:** the 20 words you use most, with a bar and a count per 1,000 words. A word used a lot is not always wrong. Look for words that surprise you. A word shows here after 5 uses.

Everyday words (the, and, said, were, and so on) are left out. Names from your Characters, Places, Factions and Items are left out too. The word list is English. Click a row to open that scene. Press **Skip** if a repeat is on purpose. Skipped rows go to a list where you can bring them back.

**Style** shows how your prose is built. It reads your scenes. Pick one book or all books, then pick a part:

- **Dialogue:** the share of words that are inside quotation marks, for the whole text and for each chapter, with a bar. There is no right number. Click a chapter to open its first scene. Curly quotes (“ ”) and straight quotes (") both work. A quote that is not closed ends with its paragraph.
- **Long sentences:** sentences longer than 30, 40, 50 or 60 words (you pick; the start is 40). The longest come first. Press **Skip** if a long sentence is on purpose. It stays skipped when you change other parts of the scene.
- **Weak words:** three lists with a count and a number per 1,000 words. Click a word to see each place it is used.
  - *Filter words* put a step between the reader and the scene: felt, saw, heard, noticed, realized, wondered, seemed, watched, knew, thought.
  - *Empty words* add little: very, really, just, quite, rather, actually, basically, suddenly, literally, somewhat.
  - *Words that end in -ly* are often adverbs. Words that are not adverbs, such as "only", "family" and "lonely", are left out.

These are hints about style, not mistakes. The word lists are English. A sentence ends at a full stop, a question mark, an exclamation mark or "…" when a capital letter follows. "Mr." and "Dr." do not end a sentence.

**Plot holes** is the next list.

**Plot holes** looks for one person in two places at the same moment of story time. It compares:

- scenes (timeline number, place, point of view, and cast),
- timeline events (position, place, and who is involved),
- map path steps (timeline position, and a place or a spot on the map),
- item steps that name a scene (it also finds one item with two owners at once).

**Travel** is a third kind of problem. It needs a map scale (see *Travel time* above). For each character, the check takes each pair of stops that follow each other in story time on the same map. It is a plot hole when the distance, at your fastest way to travel, needs more days than the story gives. It uses your calendar's year length, or 365 days, to turn the timeline numbers into days. It does not look at travel between different maps.

How to read it:

- Only things with **exactly the same number** are compared. If one number covers a whole year, use decimals for the days: 312.1, 312.2.
- A place inside another place is fine. Being in the market and in the city that holds it is not a problem.
- Two map spots very close together count as the same spot.
- Click a row to open that scene, event, path step, or item.
- If it is not a real problem, press **Not a problem**. It moves to a list you can open and bring back. If you change the scenes later and the clash changes shape, it shows up again.
- The page says how many scenes it could not check, because they have no number or no place.

### Search

Press **Ctrl+K**. Type a few letters. Typos are fine: "aldrik" finds "Aldric". It searches everything:

- the text and summaries of your **scenes** (a hit opens that scene in the Write tab),
- your **Draft** pages,
- characters, places, lore, factions and items,
- **Glossary** words and their meanings,
- **Story arcs** and their notes.

Every result shows what kind it is. Typos are only allowed in short text such as names and titles. Inside the long text of a scene, a word must be spelled as it is written. Use the arrow keys and **Enter** to open a result. **Esc** closes it.

### Export & Backup

1. Open **Export**.
2. Choose **Manuscript** (a title page, then each chapter on a new page, with your bold, italic, underline, headings, lists and quotes) or **Story bible** (characters, relationships, places, glossary, factions, items, lore, timeline).
3. For a manuscript, pick one book or the whole series.
4. Type your **author name** if you want it on the title page. The app remembers it.
5. Leave **Table of contents** ticked if you want one after the title page. The app remembers your choice.
6. Choose a file type, press **Export…**, and pick where to save.

| File type | Use it for |
| --- | --- |
| **Word (.docx)** | Editing in Word, or sending to an editor. You can pick a page size. |
| **PDF (.pdf)** | Reading or printing. You can pick a page size, and turn on **Print layout**. |
| **E-book (.epub)** | E-readers and phone reading apps. |
| **Markdown (.md)** | Plain text with simple marks, for other writing tools or for keeping in version control. |

**Page sizes** (Word and PDF): Letter, A4, A5, US trade book (6 × 9 in), digest (5.5 × 8.5 in), and pocket book (5 × 8 in). The small book sizes use smaller margins and 11 pt type.

**Print layout** (PDF only) makes a file that is ready for a print shop:

- the inside margin (next to the binding) is 6 mm wider than the outside margin,
- page numbers are in the outer bottom corners,
- every chapter starts on a right-hand page, and the app adds an empty page when it needs one,
- the title page and the empty pages have no number.

Check the page count with your printer. Some printers want a multiple of 4 pages or more binding space, and the app does not add pages at the end.

**E-book notes:** each chapter is one page file, and the e-book has a contents list (your e-reader's own list, and a Contents page after the title page when **Table of contents** is ticked). It does not have a cover image. Your bold, italic, underline, quotes, lists and scene breaks are kept. The language is set to English. Test it in an e-reader app before you publish. It has not been run through the official EPUB checker tool.

**Markdown notes:** the book title is a `#` heading and each chapter is `##`. Bold and italic become `**` and `*`. Underline becomes `<u>`. A scene break is `* * *`.

PDF uses standard fonts. Letters outside Western languages may show as `?`. Use Word or the e-book for those.

**Table of contents** comes after the title page and lists every chapter. A series with several books lists each book, with its chapters under it. The Story bible lists its sections.

- **PDF:** each line has the page number, with dots between. You can click a line to jump to the chapter. In print layout the numbers count the empty pages too.
- **Word:** each line is a link to the chapter. Word does not tell the app which page a chapter lands on, so the lines have no page numbers.
- **E-book:** a Contents page with a link for each chapter.
- **Markdown:** a list of links to the chapter headings.

#### Backup and restore

Under the export options, the **Backup** section has two buttons.

- **Back up my data…** saves one `.db` file with everything: books, characters, maps, and drawings. Pick where to save it. The file name has today's date. Keep copies on a USB drive or in cloud storage.
- **Restore from backup…** replaces everything in the app with a backup file. It asks you to confirm first. Before it changes anything, it saves your current data to a safety copy in `%APPDATA%\typewriter\backups`. A file that is not a Typewriter backup is refused and nothing changes.

#### Backup reminder

The automatic copies stay in the app's folder. If the computer breaks, they are gone too. So the app reminds you to save a copy somewhere else, such as a USB drive.

- The app remembers the last time you pressed **Back up my data…**. Each project has its own date. The Backup section shows it ("Last backup: 3 days ago").
- When that was a week ago (or you never did, and you have written at least 100 words), the **Dashboard** shows a **Time for a backup** card. Press **Back up now…**, pick a place, and it is saved. The card says where. **Remind me tomorrow** hides the card for the rest of the day.
- **Remind me to back up** (in the Backup section) sets how long to wait: 3 days, a week, 2 weeks, a month, or never. A week is the start.
- Restoring a backup or moving the data folder does not reset the date. A new project has no date until you back it up.

#### Automatic backups

You do not have to press anything. The first time you open the app each day, it saves a copy of your data in `%APPDATA%\typewriter\backups`. The file is called `auto-YYYY-MM-DD.db`. The app keeps the last 7 days and deletes older ones. If the app stays open past midnight, it makes the new copy within half an hour.

- A copy is your data as it was when that day began. A mistake you make today is not in today's copy.
- The **Automatic backups** list on the Export & Backup tab shows each copy. **Restore** (then **Sure?**) puts that copy back. Your current data is saved to a safety copy first. The app keeps the last 5 safety copies.
- **Open the backup folder** shows the files. You can copy them to a USB drive or the cloud.
- If the app has no data yet, it makes no copy. An empty copy never replaces a good one.

## Making a release

1. Change the version in two places: `Version` in `version.go`, and `version` in `build/config.yml`. A test fails if they differ.
2. Run `wails3 task common:update:build-assets`, so the Windows and Linux files use the new version.
3. Build: `wails3 build` (Windows) and `wails3 task linux:build` (Linux, needs Docker). Build Windows last, because the Linux build can remove `bin\typewriter.exe`.
4. Commit, then tag and push: `git tag -a v1.0.1 -m "Typewriter 1.0.1"`, then `git push && git push origin v1.0.1`.

## Deleting things

Delete buttons ask twice. The first click turns the button red and says "Sure?". Click again within three seconds to confirm.

After a delete, a bar appears at the bottom for 12 seconds. Press **Undo** to bring everything back, including linked items. You can also press **Ctrl+Z** when you are not typing in a text box.

Deleting removes the things that depend on it:

| You delete | Also removed |
| --- | --- |
| A character | Their relationships and map steps |
| A map | Its path steps. The places stay, unpinned. |
| A chapter | Its scenes |
| A book | Its chapters and scenes |

## Project layout

| Path | What it holds |
| --- | --- |
| `main.go` | App start-up and window |
| `store.go` | SQLite storage service |
| `projects.go` | The list of projects, and switching the open data file |
| `datafolder.go` | Moving the data folder, and the `datadir.json` pointer |
| `version.go` | The app version shown in the menu |
| `export.go` | Export: file dialog, Word, and the story bible and manuscript text |
| `pdf.go`, `epub.go`, `markdown.go`, `pagesize.go` | The PDF, e-book and Markdown writers, and the page sizes |
| `frontend/src/views/` | One Svelte file per screen |
| `frontend/src/lib/` | Shared code: data store, search, theme, calendar (`calendarmath.ts` holds the date maths) |
| `frontend/src/lib/mapdraw/` | The map drawing tool: `draw.ts` (drawing and hit tests) and `MapEditor.svelte` |
| `frontend/bindings/` | Generated Go-to-JS bindings. Do not edit. |
| `build/` | Icons and packaging settings. `build/logo.svg` is the logo source. |

After you change a Go method that the frontend calls, regenerate the bindings:

```bash
wails3 generate bindings -clean=true -ts -i
```

### How data is stored

All items live in one table, `docs`. Each row has a kind (`character`, `chapter`, `location`, and so on) and its fields as JSON text. New fields need no database change.
