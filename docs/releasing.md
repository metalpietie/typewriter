# Making a release

One command builds Typewriter for Windows and Linux, makes the Windows installer, signs the files, and puts everything in the GitHub release of the version.

## Once, on a new computer

1. Install the tools: Go, Node, Wails 3 (`wails3`), Docker Desktop (for the Linux build), NSIS (`winget install NSIS.NSIS`) and the GitHub CLI (`winget install GitHub.cli`).
2. Log in to GitHub:
   ```bash
   gh auth login
   ```
3. The signing key. Only one key exists for all releases.
   - On the computer that made it, it is in `%APPDATA%\typewriter-release\signing.key`. Copy that file to the same place on the new computer.
   - Only when there is no key at all (the very first time): `go run ./tools/release keygen`. It writes the private key there, and the public key into `internal/app/updatekey.go`. Commit that file.

**Keep a backup of `signing.key`** (a USB stick, a password manager). Never put it in the repository.
- The app installs an update only when it is signed with this key.
- If the key is lost, a new one can be made. Then every installed app refuses the next update, and has to be updated once by hand (download and run the new installer).

## Each release

1. Set the version in `internal/app/version.go` and in `build/config.yml`, then run `wails3 task common:update:build-assets`.
2. Write the release text in `release-notes/<version>.md`. The first line that starts with `## ` becomes the title of the release, for example `## Version 1.11.0: updates`.
3. Commit, make the tag `v<version>`, and push both.
4. Start Docker Desktop.
5. Run:
   ```bash
   wails3 task release
   ```

It stops before uploading when something is wrong: changes that are not committed, no tag, no release text, no key, a key that does not match `internal/app/updatekey.go`, Docker not running, or a build that fails.

To build and sign without uploading (the files go to `bin\release\<version>`):
```bash
wails3 task release -- -no-upload
```
Other options: `-no-build` uses the files that are built already, and `-apk` also uploads the phone app (`bin\typewriter-notes.apk`).

## The files in a release

| File | What it is |
| --- | --- |
| `typewriter-<version>-windows-x64.exe` | The Windows app. It is also the portable version, and the file the updater downloads. |
| `typewriter-<version>-windows-x64-installer.exe` | The Windows installer. It installs for the user who runs it, with no admin prompt. |
| `typewriter-<version>-linux-x64` | The Linux app. |
| `typewriter-notes-<version>.apk` | The phone app (only with `-apk`). |
| `checksums.txt` | The SHA-256 of each file. |
| `*.sig` | The signature of each file that the updater can install. |

Do not rename the files. The updater in the app finds the right file by the words `windows` or `linux` and `x64` in its name.

To check a file by hand:
```bash
go run ./tools/release verify bin/release/1.11.0/typewriter-1.11.0-windows-x64.exe
```

## Windows warns about the installer

The installer is not signed with a bought code-signing certificate. So Windows shows **Windows protected your PC**. Click **More info**, then **Run anyway**. The update inside the app shows no warning.
