# Typewriter

A desktop app for writing a book or a book series. It keeps your story world in one place: characters, how they relate, places, lore, maps, character routes, and a timeline.

It runs on your computer. Your data stays in a file on your computer, one for each project. No account and no internet needed.

Built with [Wails 3](https://v3.wails.io/) (Go), Svelte, and SQLite.

## Documentation

- **[Tutorial](docs/tutorial.md):** your first hour with Typewriter. Start here if you are new. No experience needed.
- **[Using the app](docs/using-the-app.md):** what every screen does, one by one. Use it to look things up.

## What it does

- **Write:** a book editor with chapters and scenes, a Board of index cards, focus mode, history and snapshots, comments, and a Pomodoro timer.
- **Write fiction or non-fiction:** a project has a type. A non-fiction project hides the story tools and adds sources and citations, footnotes, a bibliography, an index, callout boxes, and ideas and quotes.
- **Know your world:** characters, relationships, a family tree, factions, items, places, lore notes, a glossary, maps you can draw, and a timeline with your own calendar.
- **Check your work:** plot holes, loose ends, spelling of names, repeated words, style, reading level, and loose characters.
- **Notes on your phone:** an Android app ([mobile](mobile/README.md)) copies your Notes pages both ways over your home Wi-Fi. See [the protocol](docs/phone-sync-protocol.md).
- **Share it:** export to Word, PDF, e-book and Markdown, with front and back pages.
- **Stay safe:** your data is in files on your computer (one for each project), with a backup every day and a backup button for a copy of your own.

## Install it

Download the files from the [releases](https://github.com/metalpietie/typewriter/releases) on GitHub.

- **Windows, installed:** run `typewriter-<version>-windows-x64-installer.exe`. It installs for you only, in `%LOCALAPPDATA%\Programs\Typewriter`, with no admin prompt. You can choose a desktop shortcut. A newer installer updates the app in place.
- **Windows, portable:** run `typewriter-<version>-windows-x64.exe` from any folder. It is the same program as the installed one.
- **Linux:** download `typewriter-<version>-linux-x64`, make it runnable (`chmod +x`), and start it.

**Windows warns about the installer.** It is not signed with a bought certificate, so Windows shows "Windows protected your PC". Click **More info**, then **Run anyway**.

The installed and the portable app use the same data folder (see [Your data](#your-data)). Uninstalling (Settings > Apps) removes the program and keeps your writing and your settings.

## Run it from the code

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

The path also shows at the bottom of the left menu. **Use the Backup buttons (see [Export & Backup](docs/using-the-app.md#export--backup)) to protect your work.**

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
- **Remove** (then **Sure?**) takes a project off the list. The file is not deleted. It moves to the folder `projects\removed` in your data folder (normally `%APPDATA%\typewriter`, or the folder you chose with **Change folder…**). Its name gets the date and time, for example `p1760000000000-20261016-221500.db`. To get it back: make a new project, close the app, and replace that project's file in `projects` with the removed file, renamed to the new project's file name. The first project is different: its file is `typewriter.db` in the data folder itself, not in `projects`.
- You cannot remove the project that is open.

Your first project is the old file, `typewriter.db`. New projects are files in `%APPDATA%\typewriter\projects`. The list is in `%APPDATA%\typewriter\projects.json`. The app opens the project you used last.

Backups work on the open project. Each project has its own automatic copies. The first project uses `%APPDATA%\typewriter\backups`. Another project uses `%APPDATA%\typewriter\backups-<id>`, where `<id>` is the project's file name without `.db`.

The app saves as you type. There is no save button.

## Making a release

See [docs/releasing.md](docs/releasing.md). In short: set the version, write `release-notes/<version>.md`, commit, tag and push, then run `wails3 task release`. It builds Windows, Linux and the installer, signs the files, and uploads them to the GitHub release.

## Project layout

| Path | What it holds |
| --- | --- |
| `main.go` | Puts the built screen into the program and starts the app. Nothing else. |
| `internal/app/` | All the Go code of the app, with its tests (one Go package) |
| `internal/app/run.go` | App start-up and window |
| `internal/app/store.go` | SQLite storage service |
| `internal/app/projects.go` | The list of projects, and switching the open data file |
| `internal/app/datafolder.go` | Moving the data folder, and the `datadir.json` pointer |
| `internal/app/version.go` | The app version shown in the menu |
| `internal/app/export.go` | Export: file dialog, Word, and the story bible and manuscript text |
| `internal/app/pdf.go`, `epub.go`, `markdown.go`, `pagesize.go` | The PDF, e-book and Markdown writers, and the page sizes |
| `internal/app/syncserver.go`, `syncnotes.go` | Phone sync |
| `internal/app/updates.go` | Updates from GitHub |
| `mobile/` | The Android app (Typewriter Notes, Flutter) |
| `release-notes/` | The text of each GitHub release |
| `tools/` | The release tool, the screenshot tool, and the GitHub protection script |
| `docs/` | The tutorial and the guide to every screen |
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
