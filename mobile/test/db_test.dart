import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';
import 'package:typewriter_notes/db.dart';
import 'package:typewriter_notes/models.dart';

void main() {
  sqfliteFfiInit();
  databaseFactory = databaseFactoryFfi;

  late SqliteStore store;
  setUp(() async {
    // a new file for each test: an in-memory database would be shared by all of them
    final dir = Directory.systemTemp.createTempSync('notes_test');
    store = await SqliteStore.open(path: '${dir.path}/notes.db');
    addTearDown(() async {
      await store.close();
      try {
        dir.deleteSync(recursive: true);
      } catch (_) {} // the system cleans the temp folder later
    });
  });

  Note n(String sid, {String nb = 'p1', String title = 'T', String text = 'x', bool changed = false, bool editable = true, int rev = 0, int base = 0}) =>
      Note(sid: sid, notebook: nb, title: title, text: text, changed: changed, editable: editable, rev: rev, base: base, from: 'Pixel 7');

  test('a note is kept and read back whole', () async {
    final at = DateTime.utc(2026, 10, 14, 9, 30);
    await store.saveNote(Note(sid: 'a', notebook: 'p1', title: 'Title', text: 'Body\n\n- one', updatedAt: at, base: 7, changed: true, rev: 3, editable: false, from: 'Pixel 7'));
    final r = (await store.note('a'))!;
    expect((r.title, r.text, r.base, r.changed, r.rev, r.editable, r.from), ('Title', 'Body\n\n- one', 7, true, 3, false, 'Pixel 7'));
    expect(r.updatedAt, at);
    expect(await store.note('missing'), isNull);
  });

  test('saving again replaces the note', () async {
    await store.saveNote(n('a', title: 'One'));
    await store.saveNote(n('a', title: 'Two'));
    expect((await store.notes('p1')).map((x) => x.title), ['Two']);
  });

  test('notes are listed for one notebook, newest edit first', () async {
    await store.saveNote(Note(sid: 'old', notebook: 'p1', updatedAt: DateTime.utc(2026, 1, 1)));
    await store.saveNote(Note(sid: 'new', notebook: 'p1', updatedAt: DateTime.utc(2026, 6, 1)));
    await store.saveNote(Note(sid: 'other', notebook: 'p2'));
    expect((await store.notes('p1')).map((x) => x.sid), ['new', 'old']);
  });

  test('search finds words in the title and the text, and treats % and _ as plain letters', () async {
    await store.saveNote(n('a', title: 'Clock shop', text: 'brass key'));
    await store.saveNote(n('b', title: 'Other', text: '100% sure'));
    await store.saveNote(n('c', title: 'snake_case', text: 'x'));
    expect((await store.notes('p1', query: 'BRASS')).map((x) => x.sid), ['a']);
    expect((await store.notes('p1', query: 'clock')).map((x) => x.sid), ['a']);
    expect((await store.notes('p1', query: '100%')).map((x) => x.sid), ['b']);
    expect((await store.notes('p1', query: 'e_c')).map((x) => x.sid), ['c']);
    expect(await store.notes('p1', query: '%'), hasLength(1)); // only the note that has a % in it
  });

  test('changed notes: marked, and allowed to be edited here', () async {
    await store.saveNote(n('a', changed: true));
    await store.saveNote(n('b', changed: false));
    await store.saveNote(n('c', changed: true, editable: false));
    await store.saveNote(n('d', nb: 'p2', changed: true));
    expect((await store.changedNotes('p1')).map((x) => x.sid), ['a']);
  });

  test('notebooks: save, read, order by name', () async {
    await store.saveNotebook(Notebook(id: 'p2', name: 'banana', since: 4));
    await store.saveNotebook(Notebook(id: 'p1', name: 'Apple'));
    expect((await store.notebooks()).map((b) => b.name), ['Apple', 'banana']);
    expect((await store.notebook('p2'))!.since, 4);
    await store.saveNotebook(Notebook(id: 'p2', name: 'banana', since: 9));
    expect((await store.notebook('p2'))!.since, 9);
  });

  test('the local notebook is adopted by a project, and notes go with it', () async {
    await store.saveNotebook(Notebook(id: localNotebookId, name: 'On this phone'));
    await store.saveNote(n('a', nb: localNotebookId));
    await store.adoptNotebook(localNotebookId, Notebook(id: 'p9', name: 'Fresh', since: 3));
    expect(await store.notebook(localNotebookId), isNull);
    expect((await store.note('a'))!.notebook, 'p9');
    expect((await store.notebook('p9'))!.since, 3);
  });

  test('a notebook merges into another', () async {
    await store.saveNotebook(Notebook(id: localNotebookId, name: 'On this phone'));
    await store.saveNotebook(Notebook(id: 'p1', name: 'Book'));
    await store.saveNote(n('a', nb: localNotebookId));
    await store.mergeNotebook(localNotebookId, 'p1');
    expect((await store.note('a'))!.notebook, 'p1');
    expect(await store.notebook(localNotebookId), isNull);
  });

  test('a transaction keeps all of its work, or none of it', () async {
    await store.transaction(() async {
      await store.saveNote(n('a'));
      await store.saveNote(n('b'));
    });
    expect(await store.notes('p1'), hasLength(2));
    await expectLater(
        store.transaction(() async {
          await store.saveNote(n('c'));
          throw StateError('stop');
        }),
        throwsStateError);
    expect(await store.note('c'), isNull);
    // after a failed transaction the store still works
    await store.saveNote(n('d'));
    expect(await store.note('d'), isNotNull);
  });

  test('deleting removes notes', () async {
    await store.saveNote(n('a'));
    await store.saveNote(n('b'));
    await store.deleteNotes(['a']);
    expect((await store.notes('p1')).map((x) => x.sid), ['b']);
  });

  test('meta values and computers', () async {
    expect(await store.meta('deviceId'), isNull);
    await store.setMeta('deviceId', 'abc');
    await store.setMeta('deviceId', 'def');
    expect(await store.meta('deviceId'), 'def');
    await store.saveComputer(Computer(id: 'fp1', name: 'Desk', hosts: ['192.168.0.2', '10.0.0.5'], port: 47821, fingerprint: 'fp1', project: 'p1', lastSync: DateTime.utc(2026, 10, 14)));
    final c = (await store.computers()).single;
    expect((c.name, c.port, c.project, c.lastSync), ('Desk', 47821, 'p1', DateTime.utc(2026, 10, 14)));
    expect(c.hosts, ['192.168.0.2', '10.0.0.5']);
    await store.deleteComputer('fp1');
    expect(await store.computers(), isEmpty);
  });
}
