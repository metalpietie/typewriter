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

The path also shows at the bottom of the left menu. **Use the Backup buttons (see "Export & Backup" below) to protect your work.** One file holds one book universe, and a series fits inside it.

The app saves as you type. There is no save button.

## Using the app

Use the menu on the left to switch screens. Press **Ctrl+K** anywhere to search. The button at the bottom of the menu switches between dark and light theme.

### Write

1. Press **+ Book**, then give the book a title and a place in the series.
2. Press **+ Chapter**. A chapter holds one or more **scenes**. The new chapter starts with Scene 1.
3. Write in the editor. It looks like the page you will export. The toolbar has bold, italic, underline, three heading sizes, lists, quotes, a scene break (`* * *`), and undo/redo. The usual shortcuts work: Ctrl+B, Ctrl+I, Ctrl+U. The word count updates as you type.
4. Reorder by dragging. Drag a chapter or a scene in the left list and drop it where you want it. A line shows the spot. You can also use the ↑ and ↓ buttons in the right panel.

Chapters you wrote before the editor existed open as plain paragraphs. They convert when you edit them.

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

### Characters

Press **+ New** to add a character. Fill in the role, looks, traits, backstory, goals, and notes. Add your own fields, such as "Eye colour". Pick a colour. The same colour is used on the graph and the maps.

The bottom of each character shows their relationships, their chapters, and how many map steps they have.

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
- **Places:** cities, ruins, regions, and so on. A place can sit inside another place.

### Maps & Paths

1. Press **+ Image** to load a map picture, **+ Blank** for a plain canvas, or **✎ Draw a map** to draw one in the app (see below).
2. Use the toolbar above the map:
   - **✋ Move:** drag to pan. Drag pins and steps to move them. Click one to edit it.
   - **📍 Add place:** click the map to drop a pin. Name it in the right panel.
   - **〰 Path:** draw a character's route. Click the map to add numbered steps. Click a pin to snap a step to that place.
3. In the left panel, press **Draw** next to a character to pick whose route you draw. Use the tick boxes to show or hide routes.
4. Click a step to say what happens there. You can add story time, a chapter, and a place.
5. In the right panel, reorder a character's steps with ↑ and ↓.
6. Scroll to zoom. **Fit** shows the whole map.

#### Drawing a map

Press **✎ Draw a map**. Pick a size first (Small, Medium, or Large). The drawing tool opens full screen. Press **← Done** when you finish. Then place pins and draw character paths on it as on any other map. Press **✎ Edit drawing** to open it again.

Tools on the left. The keyboard shortcut is in brackets:

- **Select (V):** click something to select it. Drag to move it. Drag the small squares to reshape a line or a coast. Press Delete to remove it. Ctrl+D copies it.
- **Pan (H):** drag to move the view. Scroll to zoom. Hold Space to pan with any tool.
- **Land (L):** draw around an island or continent. The shape closes by itself, gets a smooth coast, and a light shallow-water edge. Choose *Freehand*, or *Click points* (click each corner, then double-click or press Enter).
- **Terrain brush (B):** paint grass, forest, hills, mountains, desert, snow, swamp, beach, or water. You can set size, softness, strength, and a custom color. Press [ and ] to change the size.
- **Eraser (E):** rubs out terrain paint on the active layer.
- **Lines (R):** rivers (thin at the source, wide at the mouth), roads, trails, and borders.
- **Stamps (S):** mountains, hills, pines, oaks, castles, cities, villages, ports, towers, and caves. Turn on *Scatter* and drag to drop many, such as a forest.
- **Text label (T):** click the map, then type the text on the right. Choose font, size, rotation, letter spacing, and color.

On the right:

- **Layers:** each map has Land, Terrain, Lines, Icons, and Labels layers. Hide, lock, rename, reorder, or delete them. Add more with **+ Layer**. A tool draws on the active layer of its type.
- **Map style:** water, land, and ink colors, with three presets.
- **Undo and redo:** the buttons at the top, or Ctrl+Z and Ctrl+Y. Undo keeps 60 steps while the editor is open.
- **Export PNG:** save the map as a picture at 1× or 2× size.

Changes save by themselves.

Places made here also show in the **World** screen. Use **Unpin** to take a place off a map without deleting it.

### Timeline

- **Story time:** one vertical timeline of events, chapters, and path steps, in order. Press **+ New event** for things like battles or births. You can hide types, or show one character only.
- **Who's where:** a grid. Rows are characters and columns are chapters. ◉ means point of view, ● means present, and a number counts map steps.

The timeline sorts by a **number**, such as a year. "Year 312" is only a label. Fill in the number too. The field is called **Position** on events, and **Timeline position** on chapters and path steps. Items with no number stay off the timeline.

### Search

Press **Ctrl+K**. Type a few letters. Typos are fine: "aldrik" finds "Aldric". It searches characters, places, and lore. Use the arrow keys and **Enter** to open a result. **Esc** closes it.

### Export & Backup

1. Open **Export**.
2. Choose **Manuscript** (a title page, then each chapter on a new page, with your bold, italic, underline, headings, lists and quotes) or **Story bible** (characters, relationships, places, lore, timeline).
3. For a manuscript, pick one book or the whole series.
4. Choose **Word (.docx)** or **PDF**, press **Export…**, and pick where to save.

PDF uses standard fonts. Letters outside Western languages may show as `?`. Use Word for those.

#### Backup and restore

Under the export options, the **Backup** section has two buttons.

- **Back up my data…** saves one `.db` file with everything: books, characters, maps, and drawings. Pick where to save it. The file name has today's date. Keep copies on a USB drive or in cloud storage.
- **Restore from backup…** replaces everything in the app with a backup file. It asks you to confirm first. Before it changes anything, it saves your current data to a safety copy in `%APPDATA%	ypewriterackups`. A file that is not a Typewriter backup is refused and nothing changes.

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
| `export.go` | Word and PDF export |
| `frontend/src/views/` | One Svelte file per screen |
| `frontend/src/lib/` | Shared code: data store, search, theme |
| `frontend/src/lib/mapdraw/` | The map drawing tool: `draw.ts` (drawing and hit tests) and `MapEditor.svelte` |
| `frontend/bindings/` | Generated Go-to-JS bindings. Do not edit. |
| `build/` | Icons and packaging settings. `build/logo.svg` is the logo source. |

After you change a Go method that the frontend calls, regenerate the bindings:

```bash
wails3 generate bindings -ts
```

### How data is stored

All items live in one table, `docs`. Each row has a kind (`character`, `chapter`, `location`, and so on) and its fields as JSON text. New fields need no database change.
