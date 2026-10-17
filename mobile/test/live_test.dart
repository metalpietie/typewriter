// A test against the real Typewriter server (the Go code), so the phone and the computer are tested together.
// It is skipped unless TYPEWRITER_LIVE is set. See internal/app/syncdev_test.go in the repository, which starts the server.
//
//   set TYPEWRITER_PHONE_DEV=C:\jtmp\dev.json
//   go test -run TestDevServerForPhoneApp -timeout 10m ./internal/app      (in the repository root; leave it running)
//   set TYPEWRITER_LIVE=C:\jtmp\dev.json
//   flutter test test/live_test.dart
import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:typewriter_notes/models.dart';
import 'package:typewriter_notes/sync/api.dart';
import 'package:typewriter_notes/sync/engine.dart';

import 'engine_test.dart' show MemoryStore;

void main() {
  final path = Platform.environment['TYPEWRITER_LIVE'];
  final skip = path == null ? 'set TYPEWRITER_LIVE to the file that the Go dev server writes' : null;

  test('a phone and the real computer', () async {
    final info = (jsonDecode(File(path!).readAsStringSync()) as Map).cast<String, dynamic>();
    const deviceId = 'dddddddd-0000-4000-8000-0000000000aa';

    // looking at the computer first (as when the address is typed): the fingerprint is learned and must match
    final look = ComputerApi(hosts: [info['host'] as String], port: info['port'] as int, fingerprint: '');
    final hello = await look.hello();
    expect(hello['app'], 'Typewriter');
    expect(look.seenFingerprint, info['fingerprint']);

    // a wrong pin is refused
    final wrong = ComputerApi(hosts: [info['host'] as String], port: info['port'] as int, fingerprint: '0' * 64);
    await expectLater(wrong.hello(), throwsA(isA<SyncException>()));

    // a wrong code is refused, the right one adds the phone
    final api = ComputerApi(hosts: [info['host'] as String], port: info['port'] as int, fingerprint: info['fingerprint'] as String);
    await expectLater(api.pair(code: '000000', deviceId: deviceId, name: 'Test phone'), throwsA(isA<SyncException>()));
    final paired = await api.pair(code: info['code'] as String, deviceId: deviceId, name: 'Test phone');
    expect(paired.key, isNotEmpty);
    expect(paired.projectId, (info['project'] as Map)['id']);

    final phone = ComputerApi(hosts: [info['host'] as String], port: info['port'] as int, fingerprint: info['fingerprint'] as String, deviceId: deviceId, key: paired.key);
    final store = MemoryStore();
    store.books[localNotebookId] = Notebook(id: localNotebookId, name: 'On this phone');
    await store.saveNote(Note(sid: '11111111-aaaa-4aaa-8aaa-111111111111', notebook: localNotebookId, title: 'From the phone', text: 'An **idea**\n\n- one\n- two', changed: true, rev: 1));
    final engine = SyncEngine(store);

    // first sync: the phone's note goes up, the computer's notes come down, and the local notebook becomes the project's
    final r1 = await engine.sync(phone, notebookId: localNotebookId);
    expect(r1.sent, 1);
    expect(r1.notebookId, (info['project'] as Map)['id']);
    final notes = await store.notes(r1.notebookId);
    final titles = notes.map((n) => n.title).toSet();
    expect(titles, containsAll(['From the phone', 'Desktop note', 'Linked note']));
    final desktop = notes.firstWhere((n) => n.title == 'Desktop note');
    expect(desktop.text, 'Hello from the **computer**.');
    expect(desktop.editable, true);
    expect(notes.firstWhere((n) => n.title == 'Linked note').editable, false); // links: read-only on the phone
    expect((await store.note('11111111-aaaa-4aaa-8aaa-111111111111'))!.changed, false);

    // second sync: nothing moves
    final r2 = await engine.sync(phone, notebookId: r1.notebookId);
    expect(r2.summary, 'Up to date.');

    // an edit on the phone goes up and is taken
    var mine = (await store.note('11111111-aaaa-4aaa-8aaa-111111111111'))!;
    await store.saveNote(mine.copyWith(text: 'An **idea**, edited', changed: true, rev: mine.rev + 1, updatedAt: DateTime.now().toUtc()));
    final r3 = await engine.sync(phone, notebookId: r1.notebookId);
    expect(r3.sent, 1);
    mine = (await store.note('11111111-aaaa-4aaa-8aaa-111111111111'))!;
    expect(mine.changed, false);
    expect(mine.base, greaterThan(0));

    // an edit of a note with links is not sent (it cannot be edited), and a forced edit is turned down
    final linked = notes.firstWhere((n) => n.title == 'Linked note');
    final forced = await phone.sync(SyncRequest(project: r1.notebookId, since: 0, notes: [linked.copyWith(text: 'Talk to nobody', updatedAt: DateTime.now().toUtc())]));
    expect(forced.rejected, [linked.sid]);

    // a note for another project is not taken
    final other = await phone.sync(SyncRequest(project: 'another-project', since: 0, notes: []));
    expect(other.mismatch, true);

    // a phone with a wrong key gets "no"
    final bad = ComputerApi(hosts: [info['host'] as String], port: info['port'] as int, fingerprint: info['fingerprint'] as String, deviceId: deviceId, key: 'not-the-key');
    await expectLater(bad.sync(SyncRequest(project: '', since: 0, notes: [])), throwsA(isA<SyncException>()));
  }, skip: skip);
}
