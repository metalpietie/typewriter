# Phone sync protocol

This page is for the phone app. It says exactly how a phone talks to Typewriter. The code is in `internal/app/syncserver.go` (the server) and `internal/app/syncnotes.go` (the rules for notes). The tests `internal/app/syncserver_test.go` and `internal/app/syncnotes_test.go` are a working example of a phone.

Protocol version: **1**. Everything is JSON over HTTPS.

## Finding the computer

- **By name:** Typewriter announces the service `_typewriter._tcp` (mDNS, Bonjour) on the local network. The TXT records are `v=1` and `fp=<the first 16 letters of the certificate fingerprint>`. The port is in the service record. It is `47821` when that port is free, otherwise any free port.
- **By QR code:** the QR code holds JSON, see "Adding a phone".
- **By hand:** the address and port are shown in Typewriter, on the Export & Backup screen, under **Phones**.

## The certificate

The computer makes its own self-signed certificate (ECDSA P-256) the first time sync is switched on, and keeps it. The phone must **not** use the system's certificate check. It must keep the **SHA-256 fingerprint of the certificate (the DER bytes), in hex**, and accept only a server whose certificate has that fingerprint. The QR code carries the fingerprint, so the phone has it from the start. Adding a phone with the typed code (no QR code) can only learn the fingerprint from the first connection, so the app should show it and ask the user to check it.

## Adding a phone (pairing)

In Typewriter: **Phones**, **Add a phone**. The QR code holds:

```json
{ "v": 1, "name": "DESKTOP-WRITER", "hosts": ["192.168.1.20"], "port": 47821, "fp": "<64 hex letters>", "code": "482913" }
```

The phone sends:

```
POST /pair
{ "code": "482913", "deviceId": "<a uuid that the phone made and keeps>", "name": "Pixel 7" }
```

The answer is `200` with:

```json
{ "key": "<a secret key>", "computer": "DESKTOP-WRITER", "project": { "id": "...", "name": "..." }, "protocol": 1 }
```

The phone keeps the key in secure storage. The code works **once**, for **5 minutes**, and dies after **5 wrong tries** (`403`). Adding the same `deviceId` again gives a new key, and the old key stops working. The computer keeps only a fingerprint of the key.

## Every other request

Header: `Authorization: Bearer <deviceId>:<key>`. A wrong or missing key gives `401`. A phone that was removed in Typewriter gets `401`.

## GET /hello

No key needed. Answer: `{ "name": "DESKTOP-WRITER", "protocol": 1, "app": "Typewriter" }`. It says nothing about the notes. If `protocol` is not the one the app knows, the app tells the user to update.

## POST /sync

One sync. The body is at most 10 MB.

```json
{
  "project": "<the project id that the phone thinks is open>",
  "since": 41,
  "notes": [
    { "sid": "<uuid>", "title": "Idea", "text": "A **big** idea", "updatedAt": "2026-10-14T10:00:00Z", "base": 0 }
  ]
}
```

- `project`: if it is not the project that is open in Typewriter, the answer has `"mismatch": true` and **nothing is changed**. The phone then uses the project in the answer (a phone keeps one notebook for each project).
- `since`: the `seq` from the last answer for this project (0 the first time).
- `notes`: only the notes that are **new or changed on the phone** since the last sync. **The phone never sends a delete, and has no way to delete.** `sid` is a uuid that the phone makes for a new note and never changes. `base` is the `seq` that the phone last got for that note (0 for a note made on the phone). `text` is the simple form, below. A note with the wrong `sid` form, or a `text` over 200,000 letters, is turned down (`rejected`). A title is cut to 300 letters.

Answer (`200`):

```json
{
  "project": { "id": "...", "name": "..." },
  "mismatch": false,
  "saved":   [ { "sid": "...", "seq": 42 } ],
  "rejected": [ "<sid>" ],
  "changed": [ { "sid": "...", "title": "...", "text": "...", "seq": 43, "updatedAt": "2026-10-14T10:00:00Z", "editable": true, "from": "Pixel 7" } ],
  "deleted": [ "<sid>" ],
  "seq": 43
}
```

What the phone does with it:

1. `saved`: store the new `seq` as that note's `base`, and clear its "changed" mark.
2. `rejected`: the note may not be edited on the phone (see "editable"). Clear the mark and show the computer's version, which comes in `changed`.
3. `changed`: new or edited on the computer. **Replace** the note on the phone (or add it). Store `seq` as `base`. If `editable` is `false`, show it **read-only**, with "Edit this note on the computer".
4. `deleted`: **remove** these notes from the phone, for good.
5. Store the top-level `seq` as `since` for the next sync.
6. Do all this in one database transaction, and clear the "changed" marks only after the whole answer is saved. If the connection drops, nothing is lost: send the same notes again. Sending a note twice is safe.

### What the computer does with a note from the phone

| Case | Result |
| --- | --- |
| A new `sid` | A new page in Notes, marked as from this phone. |
| The computer did not change the note since `base` | The phone's version replaces the page. |
| Both changed it | The newest (by `updatedAt`) stays as the note. The other is kept as a new page, "Title (copy from Pixel 7)" or "Title (earlier version)". The phone gets both back in `changed`. |
| The computer deleted it, the phone edited it | The phone's edit becomes a new page, "Title (restored from Pixel 7)", under the same `sid`. |
| The page has links, comments, citations, footnotes, boxes, underline or other things that the simple form cannot keep | Turned down. Shown on the phone as read-only. |
| The same note again (the answer was lost) | Nothing happens. It counts as saved. |

Time is used only to say which side of a clash is newer. What changed is decided by the `seq` counter, so the clocks of the phone and the computer do not matter.

## The simple form of a note

```
# Heading, ## Heading, ### Heading     one line
- an item                              a bullet list, one item on each line
1. an item                            a numbered list
> a quote                              a quote, one paragraph on each line
**bold** and *italic*                  inside any line
a blank line                           between blocks
```

A `*` or a `\` in the text is written `\*` and `\\`. A line of plain text that starts like a mark (`- `, `# `, `> `, `1. `) starts with a `\`. A single new line inside a block is a line break. A `**` or `*` with no partner is plain text.

## DELETE /pair

Not built yet. A phone is removed in Typewriter. (Reserved: the phone forgetting itself.)

## Security, short

No internet. Only devices on the same network can reach the computer, and only while **Phones** is switched on. Encrypted, with the certificate pinned. A key for each phone, which the user can remove. The one-time code. Every note is checked (size, text only) before it is saved.
