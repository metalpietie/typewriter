import 'package:flutter_test/flutter_test.dart';
import 'package:typewriter_notes/models.dart';
import 'package:typewriter_notes/store.dart';
import 'package:typewriter_notes/sync/api.dart';
import 'package:typewriter_notes/sync/engine.dart';

class MemoryStore implements NoteStore {
  final books = <String, Notebook>{};
  final all = <String, Note>{};
  final metas = <String, String>{};

  @override
  Future<List<Notebook>> notebooks() async => books.values.toList();
  @override
  Future<Notebook?> notebook(String id) async => books[id];
  @override
  Future<void> saveNotebook(Notebook n) async => books[n.id] = n;
  @override
  Future<void> mergeNotebook(String from, String into) async {
    for (final n in all.values.where((n) => n.notebook == from).toList()) {
      all[n.sid] = n.copyWith(notebook: into);
    }
    if (from != localNotebookId) books.remove(from);
    if (from == localNotebookId) books.remove(from);
  }

  @override
  Future<void> adoptNotebook(String from, Notebook as) async {
    for (final n in all.values.where((n) => n.notebook == from).toList()) {
      all[n.sid] = n.copyWith(notebook: as.id);
    }
    books.remove(from);
    books[as.id] = as;
  }

  @override
  Future<List<Note>> notes(String notebook, {String query = ''}) async => all.values.where((n) => n.notebook == notebook).toList();
  @override
  Future<Note?> note(String sid) async => all[sid];
  @override
  Future<void> saveNote(Note n) async => all[n.sid] = n;
  @override
  Future<List<Note>> changedNotes(String notebook) async => all.values.where((n) => n.notebook == notebook && n.changed && n.editable).toList();
  @override
  Future<void> deleteNotes(List<String> sids) async {
    for (final s in sids) {
      all.remove(s);
    }
  }

  @override
  Future<T> transaction<T>(Future<T> Function() body) => body();
  @override
  Future<String?> meta(String key) async => metas[key];
  @override
  Future<void> setMeta(String key, String value) async => metas[key] = value;
}

/// A computer that answers with what a test says, and keeps what it was asked.
class FakeApi implements SyncApi {
  FakeApi(this.answers);
  final List<SyncResponse> answers;
  final requests = <SyncRequest>[];
  @override
  Future<SyncResponse> sync(SyncRequest r) async {
    requests.add(r);
    return answers.removeAt(0);
  }
}

SyncResponse answer({
  String project = 'p1',
  String name = 'The Last Letter',
  bool mismatch = false,
  Map<String, int> saved = const {},
  List<String> rejected = const [],
  List<RemoteNote> changed = const [],
  List<String> deleted = const [],
  int seq = 0,
}) =>
    SyncResponse(projectId: project, projectName: name, mismatch: mismatch, saved: saved, rejected: rejected, changed: changed, deleted: deleted, seq: seq);

RemoteNote remote(String sid, {String title = 'T', String text = 'x', int seq = 1, bool editable = true, String from = ''}) =>
    RemoteNote(sid: sid, title: title, text: text, seq: seq, editable: editable, from: from);

Note mine(String sid, {String notebook = 'p1', String text = 'phone text', int base = 0, bool changed = true, int rev = 1, bool editable = true}) =>
    Note(sid: sid, notebook: notebook, title: 'Title $sid', text: text, base: base, changed: changed, rev: rev, editable: editable);

void main() {
  late MemoryStore store;
  late SyncEngine engine;
  setUp(() {
    store = MemoryStore();
    engine = SyncEngine(store);
    store.books['p1'] = Notebook(id: 'p1', name: 'The Last Letter', since: 5);
  });

  test('sends only the changed notes, with the base and since', () async {
    await store.saveNote(mine('a', base: 3));
    await store.saveNote(mine('b', changed: false));
    final api = FakeApi([answer(saved: {'a': 9}, seq: 9)]);
    final r = await engine.sync(api, notebookId: 'p1');
    expect(api.requests.single.project, 'p1');
    expect(api.requests.single.since, 5);
    expect(api.requests.single.notes.map((n) => n.sid), ['a']);
    expect(api.requests.single.toJson()['notes'][0]['base'], 3);
    expect(r.sent, 1);
    expect(r.summary, 'Done: sent 1.');
  });

  test('a saved note gets its new base and is no longer marked', () async {
    await store.saveNote(mine('a'));
    await engine.sync(FakeApi([answer(saved: {'a': 9}, seq: 9)]), notebookId: 'p1');
    final n = (await store.note('a'))!;
    expect(n.base, 9);
    expect(n.changed, false);
    expect((await store.notebook('p1'))!.since, 9);
  });

  test('a note edited again while the sync ran stays marked', () async {
    await store.saveNote(mine('a', rev: 1));
    final api = _EditsDuringSync(store, answer(saved: {'a': 9}, seq: 9));
    await engine.sync(api, notebookId: 'p1');
    final n = (await store.note('a'))!;
    expect(n.base, 9);
    expect(n.changed, true); // the second edit is sent next time
    expect(n.text, 'edited during the sync');
  });

  test('notes from the computer are added and replaced', () async {
    await store.saveNote(mine('old', changed: false, base: 1, text: 'old text'));
    final r = await engine.sync(
        FakeApi([
          answer(changed: [remote('new', text: 'hello', seq: 4, from: 'Pixel 7'), remote('old', text: 'new text', seq: 6)], seq: 6)
        ]),
        notebookId: 'p1');
    expect((await store.note('new'))!.text, 'hello');
    expect((await store.note('new'))!.from, 'Pixel 7');
    expect((await store.note('new'))!.base, 4);
    expect((await store.note('old'))!.text, 'new text');
    expect((await store.note('old'))!.base, 6);
    expect(r.received, 2);
  });

  test('a note that cannot be edited on the phone is kept read-only', () async {
    await store.saveNote(mine('a'));
    await engine.sync(
        FakeApi([
          answer(rejected: ['a'], changed: [remote('a', text: 'computer version', seq: 8, editable: false)], seq: 8)
        ]),
        notebookId: 'p1');
    final n = (await store.note('a'))!;
    expect(n.editable, false);
    expect(n.changed, false);
    expect(n.text, 'computer version');
    expect(await store.changedNotes('p1'), isEmpty); // and it is not sent again
  });

  test('a delete on the computer removes the note', () async {
    await store.saveNote(mine('a', changed: false));
    final r = await engine.sync(FakeApi([answer(deleted: ['a'], seq: 7)]), notebookId: 'p1');
    expect(await store.note('a'), isNull);
    expect(r.removed, 1);
  });

  test('a delete does not remove a note with an edit that is not sent yet', () async {
    await store.saveNote(mine('a', changed: true));
    final api = FakeApi([answer(deleted: ['a'], seq: 7)]);
    // the request is built from the changed notes, so "a" is sent. The computer then says it was deleted: it would have been restored, not deleted.
    // This is the rare case of an old answer: the note is kept.
    await engine.sync(api, notebookId: 'p1');
    expect(await store.note('a'), isNotNull);
  });

  test('an edit that is not sent yet is not overwritten by the computer', () async {
    await store.saveNote(mine('a', changed: false, base: 1));
    final api = _EditsDuringSync(store, answer(changed: [remote('a', text: 'from the computer', seq: 5)], seq: 5), editSid: 'a');
    await engine.sync(api, notebookId: 'p1');
    final n = (await store.note('a'))!;
    expect(n.text, 'edited during the sync');
    expect(n.changed, true);
  });

  test('another project is open: the phone moves to its notebook and asks again', () async {
    await store.saveNote(mine('a')); // stays in the first notebook
    final api = FakeApi([
      answer(project: 'p2', name: 'Second Book', mismatch: true),
      answer(project: 'p2', name: 'Second Book', changed: [remote('b', seq: 2)], seq: 2),
    ]);
    final r = await engine.sync(api, notebookId: 'p1');
    expect(r.switched, true);
    expect(r.notebookId, 'p2');
    expect(api.requests[1].project, 'p2');
    expect(api.requests[1].notes, isEmpty); // the notes of the first notebook are not sent to the other project
    expect((await store.note('b'))!.notebook, 'p2');
    expect((await store.note('a'))!.changed, true); // kept for the day that project is open
    expect((await store.notebook('p2'))!.since, 2);
  });

  test('the local notebook becomes the notebook of the computer project', () async {
    store.books.clear();
    store.books[localNotebookId] = Notebook(id: localNotebookId, name: 'On this phone');
    await store.saveNote(mine('a', notebook: localNotebookId));
    final api = FakeApi([
      answer(project: 'p9', name: 'Fresh Book', saved: {'a': 3}, changed: [remote('c', seq: 2)], seq: 3)
    ]);
    final r = await engine.sync(api, notebookId: localNotebookId);
    expect(api.requests.single.project, '');
    expect(api.requests.single.since, 0);
    expect(r.notebookId, 'p9');
    expect((await store.note('a'))!.notebook, 'p9');
    expect((await store.note('c'))!.notebook, 'p9');
    expect((await store.notebook('p9'))!.name, 'Fresh Book');
    expect((await store.notebook('p9'))!.since, 3);
    expect(await store.notebook(localNotebookId), isNull);
  });

  test('the local notebook joins a notebook that is there already', () async {
    store.books[localNotebookId] = Notebook(id: localNotebookId, name: 'On this phone');
    await store.saveNote(mine('a', notebook: localNotebookId));
    await engine.sync(FakeApi([answer(project: 'p1', saved: {'a': 3}, seq: 3)]), notebookId: localNotebookId);
    expect((await store.note('a'))!.notebook, 'p1');
    expect(await store.notebook(localNotebookId), isNull);
    expect((await store.notebook('p1'))!.since, 3);
  });

  test('a computer that keeps saying mismatch is an error, not a loop', () async {
    final api = FakeApi([answer(project: 'p2', mismatch: true), answer(project: 'p3', mismatch: true)]);
    expect(() => engine.sync(api, notebookId: 'p1'), throwsA(isA<SyncException>()));
  });

  test('when the sync fails, nothing changes', () async {
    await store.saveNote(mine('a'));
    await expectLater(engine.sync(_Failing(), notebookId: 'p1'), throwsA(isA<SyncException>()));
    expect((await store.note('a'))!.changed, true);
    expect((await store.notebook('p1'))!.since, 5);
  });

  test('"Up to date" when nothing moved', () async {
    final r = await engine.sync(FakeApi([answer(seq: 5)]), notebookId: 'p1');
    expect(r.summary, 'Up to date.');
  });

  test('the answer is read from JSON', () {
    final r = SyncResponse.fromJson({
      'project': {'id': 'p1', 'name': 'Book'},
      'mismatch': false,
      'saved': [
        {'sid': 'a', 'seq': 4}
      ],
      'rejected': ['b'],
      'changed': [
        {'sid': 'c', 'title': 'T', 'text': 'x', 'seq': 5, 'updatedAt': '2026-10-14T10:00:00Z', 'editable': false, 'from': 'Pixel 7'}
      ],
      'deleted': ['d'],
      'seq': 5,
    });
    expect(r.saved, {'a': 4});
    expect(r.rejected, ['b']);
    expect(r.changed.single.editable, false);
    expect(r.deleted, ['d']);
    // a null list is an empty list
    final e = SyncResponse.fromJson({
      'project': {'id': 'p', 'name': 'n'},
      'saved': null,
      'changed': null,
      'deleted': null,
      'rejected': null,
      'seq': 1,
    });
    expect(e.saved, isEmpty);
    expect(e.changed, isEmpty);
  });
}

/// Edits a note on the phone while the sync runs: after the request was made, before the answer is applied.
class _EditsDuringSync implements SyncApi {
  _EditsDuringSync(this.store, this.response, {this.editSid});
  final MemoryStore store;
  final SyncResponse response;
  final String? editSid;
  @override
  Future<SyncResponse> sync(SyncRequest r) async {
    final sid = editSid ?? r.notes.first.sid;
    final n = (await store.note(sid)) ?? mine(sid);
    await store.saveNote(n.copyWith(text: 'edited during the sync', changed: true, rev: n.rev + 1));
    return response;
  }
}

class _Failing implements SyncApi {
  @override
  Future<SyncResponse> sync(SyncRequest r) async => throw SyncException('no wifi');
}
