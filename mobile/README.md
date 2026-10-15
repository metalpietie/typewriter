# Typewriter Notes (the phone app)

Write ideas and thoughts for your book on your phone. They are saved on the phone, with no internet. When the phone and the computer are on the same Wi-Fi and Typewriter is open, the notes copy both ways with the **Notes** screen of Typewriter.

How the two talk is written in [docs/phone-sync-protocol.md](../docs/phone-sync-protocol.md).

## What it does

- A list of notes for each notebook. A notebook is a project of Typewriter. Search, and a note editor with bold, italic, headings, lists and quotes.
- **No delete on the phone.** A note is deleted in Typewriter, and it goes from the phone at the next sync.
- A note that has links, comments or citations in Typewriter shows here with a lock, and can be read here and changed only in Typewriter.
- **Computers:** find Typewriter on the Wi-Fi, scan its QR code (or type the code, or the address), and sync. The key of each computer is in the phone's secure storage.

## Install it on a phone

1. Build the file (see below), or take `bin/typewriter-notes.apk` if it was made.
2. Copy the `.apk` to the phone and open it. Android asks to allow installs from that app (for example Files): allow it. This is a file for your own use, so it is not in the Play Store.
3. In Typewriter on the computer: **Export & Backup**, **Phones**, **Switch on**, **Add a phone**.
4. In the app: the computer icon, **Scan the QR code**.

## Build and test

Flutter and the Android SDK are needed. In this folder:

```bash
flutter pub get
flutter analyze
flutter test                      # unit tests, database tests, and screen tests
flutter build apk --release       # build\app\outputs\flutter-apk\app-release.apk
```

The release file is signed with the debug key, which is fine for installing it yourself.

### The live test (the phone code against the real Go server)

In the repository root, in one terminal (leave it running):

```bash
set TYPEWRITER_PHONE_DEV=C:\temp\dev.json
go test -run TestDevServerForPhoneApp -timeout 10m .
```

In this folder, in another:

```bash
set TYPEWRITER_LIVE=C:\temp\dev.json
flutter test test/live_test.dart
```

It adds a test phone, syncs both ways, and checks the pinned certificate, a wrong code, a wrong key, notes with links, and another project.

### Two problems on some Windows computers

- **Gradle says "Unable to establish loopback connection".** Java needs a short temp folder path. Set `TEMP` and `TMP` to a short folder, for example `C:\jtmp`, before the build.
- **An Android emulator with a lock screen** shows a black picture to `adb screencap` and refuses to start apps until it is unlocked. Unlock it once by hand.

## Folders

| Folder | What is in it |
| --- | --- |
| `lib/models.dart`, `lib/store.dart` | Notes, notebooks and computers, and the interface of the storage |
| `lib/db.dart` | The database (SQLite) and the key vault |
| `lib/sync/api.dart` | The messages, and the HTTPS client that accepts only the pinned certificate |
| `lib/sync/engine.dart` | One sync: what is sent, and how the answer is applied |
| `lib/simple_text.dart` | The buttons above the keyboard (bold, headings, lists) |
| `lib/app.dart`, `lib/discovery.dart` | The state of the app, adding a computer, finding computers on the Wi-Fi |
| `lib/ui/` | The screens |
| `test/` | The tests |
