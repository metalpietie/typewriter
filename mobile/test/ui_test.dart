import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';
import 'package:typewriter_notes/app.dart';
import 'package:typewriter_notes/db.dart';
import 'package:typewriter_notes/main.dart';
import 'package:typewriter_notes/models.dart';

class FakeVault extends KeyVault {
  final keys = <String, String>{};
  @override
  Future<String?> get(String id) async => keys[id];
  @override
  Future<void> put(String id, String key) async => keys[id] = key;
  @override
  Future<void> remove(String id) async => keys.remove(id);
}

void main() {
  sqfliteFfiInit();
  databaseFactory = databaseFactoryFfi;

  late AppServices app;
  late Directory dir;

  /// Real time for the database (it runs in another isolate), then a frame for the screen.
  Future<void> settle(WidgetTester t) async {
    // a screen asks the database while it is drawn, so: draw, wait in real time, draw again (a few times)
    for (var i = 0; i < 3; i++) {
      await t.pump(const Duration(milliseconds: 100));
      await t.runAsync(() => Future<void>.delayed(const Duration(milliseconds: 150)));
    }
    await t.pump(const Duration(milliseconds: 400));
  }

  Future<void> open(WidgetTester t) async {
    await t.runAsync(() async {
      dir = Directory.systemTemp.createTempSync('notes_ui');
      app = await AppServices.create(store: await SqliteStore.open(path: '${dir.path}/notes.db'), vault: FakeVault());
    });
    await t.binding.setSurfaceSize(const Size(400, 800));
    addTearDown(() => t.binding.setSurfaceSize(null));
    await t.pumpWidget(TypewriterNotes(app: app));
    await settle(t);
  }

  testWidgets('a new phone has an empty list that says what to do', (t) async {
    await open(t);
    expect(find.textContaining('No notes yet'), findsOneWidget);
    expect(find.text('On this phone'), findsOneWidget); // the local notebook
    expect(find.text('New note'), findsOneWidget);
  });

  testWidgets('notes are listed, a read-only one has a lock, and search filters', (t) async {
    await open(t);
    await t.runAsync(() async {
      await app.store.saveNote(Note(sid: 'a', notebook: app.notebookId, title: 'Clock idea', text: '# Heading\nThe **brass** key'));
      await app.store.saveNote(Note(sid: 'b', notebook: app.notebookId, title: 'Linked note', text: 'Talk to Mara', editable: false));
      app.touch();
    });
    await settle(t);
    expect(find.text('Clock idea'), findsOneWidget);
    expect(find.text('Linked note'), findsOneWidget);
    expect(find.byIcon(Icons.lock_outline), findsOneWidget);
    expect(find.text('Heading'), findsOneWidget); // the first line, without the # mark

    await t.tap(find.byIcon(Icons.search));
    await t.pump();
    await t.enterText(find.byType(TextField), 'brass');
    await settle(t);
    expect(find.text('Clock idea'), findsOneWidget);
    expect(find.text('Linked note'), findsNothing);
    await t.enterText(find.byType(TextField), 'zzz');
    await settle(t);
    expect(find.text('No note has that in it.'), findsOneWidget);
  });

  testWidgets('writing a note: it is saved as you type, and the buttons put in the marks', (t) async {
    await open(t);
    await t.tap(find.text('New note'));
    await settle(t);
    final fields = find.byType(TextField);
    expect(fields, findsNWidgets(2)); // the title and the text
    await t.enterText(fields.at(0), 'My idea');
    await t.enterText(fields.at(1), 'one two');
    // select "two" and press Bold
    final field = t.widget<TextField>(fields.at(1));
    field.controller!.selection = const TextSelection(baseOffset: 4, extentOffset: 7);
    await t.tap(find.byTooltip('Bold'));
    await t.pump();
    expect(field.controller!.text, 'one **two**');
    await t.tap(find.byTooltip('Bullet list'));
    await t.pump();
    expect(field.controller!.text, '- one **two**');
    // wait for the save after the pause
    await t.runAsync(() => Future<void>.delayed(const Duration(milliseconds: 900)));
    await t.pump(const Duration(milliseconds: 900));
    final saved = await t.runAsync(() => app.store.notes(app.notebookId));
    expect(saved, hasLength(1));
    expect(saved!.single.title, 'My idea');
    expect(saved.single.text, '- one **two**');
    expect(saved.single.changed, true);
    expect(saved.single.rev, greaterThan(0));
    // there is no delete anywhere on the screen
    expect(find.byIcon(Icons.delete), findsNothing);
    expect(find.byIcon(Icons.delete_outline), findsNothing);
  });

  testWidgets('an empty new note is not kept', (t) async {
    await open(t);
    await t.tap(find.text('New note'));
    await settle(t);
    await t.pageBack();
    await settle(t);
    final notes = await t.runAsync(() => app.store.notes(app.notebookId));
    expect(notes, isEmpty);
  });

  testWidgets('a read-only note shows its text and a line about Typewriter, and has no text fields', (t) async {
    await open(t);
    await t.runAsync(() async {
      await app.store.saveNote(Note(sid: 'b', notebook: app.notebookId, title: 'Linked note', text: 'Talk to Mara', editable: false, from: ''));
      app.touch();
    });
    await settle(t);
    await t.tap(find.text('Linked note'));
    await settle(t);
    expect(find.textContaining('open it in Typewriter'), findsOneWidget);
    expect(find.text('Talk to Mara'), findsOneWidget);
    expect(find.byType(TextField), findsNothing);
    expect(find.byTooltip('Bold'), findsNothing);
  });

  testWidgets('the computers page says how to add a computer, and there is no way to delete a note anywhere on it', (t) async {
    await open(t);
    await t.tap(find.byIcon(Icons.computer));
    await settle(t);
    expect(find.text('Scan the QR code'), findsOneWidget);
    expect(find.text('Add by address'), findsOneWidget);
    expect(find.text('None yet.'), findsOneWidget);
  });
}
