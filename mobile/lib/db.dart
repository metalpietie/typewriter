import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:path/path.dart' as p;
import 'package:sqflite/sqflite.dart';

import 'models.dart';
import 'store.dart';

/// The notes, kept in a database on the phone. They are there with no network.
class SqliteStore implements NoteStore {
  SqliteStore._(this._db);
  final Database _db;
  Transaction? _txn;

  static Future<SqliteStore> open({String? path}) async {
    final file = path ?? p.join(await getDatabasesPath(), 'typewriter_notes.db');
    final db = await openDatabase(file, version: 1, onCreate: (db, _) async {
      await db.execute('CREATE TABLE notebooks (id TEXT PRIMARY KEY, name TEXT NOT NULL, since INTEGER NOT NULL DEFAULT 0)');
      await db.execute('''CREATE TABLE notes (
        sid TEXT PRIMARY KEY, notebook TEXT NOT NULL, title TEXT NOT NULL DEFAULT '', text TEXT NOT NULL DEFAULT '',
        updated_at TEXT NOT NULL, base INTEGER NOT NULL DEFAULT 0, changed INTEGER NOT NULL DEFAULT 0,
        rev INTEGER NOT NULL DEFAULT 0, editable INTEGER NOT NULL DEFAULT 1, from_who TEXT NOT NULL DEFAULT '')''');
      await db.execute('CREATE INDEX notes_notebook ON notes(notebook)');
      await db.execute('CREATE TABLE meta (k TEXT PRIMARY KEY, v TEXT NOT NULL)');
      await db.execute('''CREATE TABLE computers (
        id TEXT PRIMARY KEY, name TEXT NOT NULL, hosts TEXT NOT NULL, port INTEGER NOT NULL, fingerprint TEXT NOT NULL,
        project TEXT NOT NULL DEFAULT '', last_sync TEXT)''');
    });
    return SqliteStore._(db);
  }

  Future<void> close() => _db.close();

  DatabaseExecutor get _x => _txn ?? _db;

  Notebook _book(Map<String, Object?> r) => Notebook(id: r['id'] as String, name: r['name'] as String, since: r['since'] as int);
  Note _note(Map<String, Object?> r) => Note(
        sid: r['sid'] as String,
        notebook: r['notebook'] as String,
        title: r['title'] as String,
        text: r['text'] as String,
        updatedAt: DateTime.parse(r['updated_at'] as String),
        base: r['base'] as int,
        changed: (r['changed'] as int) == 1,
        rev: r['rev'] as int,
        editable: (r['editable'] as int) == 1,
        from: r['from_who'] as String,
      );

  @override
  Future<List<Notebook>> notebooks() async => [for (final r in await _x.query('notebooks', orderBy: 'name COLLATE NOCASE')) _book(r)];

  @override
  Future<Notebook?> notebook(String id) async {
    final r = await _x.query('notebooks', where: 'id = ?', whereArgs: [id]);
    return r.isEmpty ? null : _book(r.first);
  }

  @override
  Future<void> saveNotebook(Notebook n) => _x.insert('notebooks', {'id': n.id, 'name': n.name, 'since': n.since}, conflictAlgorithm: ConflictAlgorithm.replace);

  @override
  Future<void> mergeNotebook(String from, String into) async {
    await _x.update('notes', {'notebook': into}, where: 'notebook = ?', whereArgs: [from]);
    await _x.delete('notebooks', where: 'id = ?', whereArgs: [from]);
  }

  @override
  Future<void> adoptNotebook(String from, Notebook as) async {
    await _x.update('notes', {'notebook': as.id}, where: 'notebook = ?', whereArgs: [from]);
    await _x.delete('notebooks', where: 'id = ?', whereArgs: [from]);
    await saveNotebook(as);
  }

  @override
  Future<List<Note>> notes(String notebook, {String query = ''}) async {
    final q = query.trim();
    final rows = q.isEmpty
        ? await _x.query('notes', where: 'notebook = ?', whereArgs: [notebook], orderBy: 'updated_at DESC')
        : await _x.query('notes',
            where: "notebook = ? AND (title LIKE ? ESCAPE '\\' OR text LIKE ? ESCAPE '\\')",
            whereArgs: [notebook, '%${_like(q)}%', '%${_like(q)}%'],
            orderBy: 'updated_at DESC');
    return [for (final r in rows) _note(r)];
  }

  String _like(String s) => s.replaceAll('\\', '\\\\').replaceAll('%', '\\%').replaceAll('_', '\\_');

  @override
  Future<Note?> note(String sid) async {
    final r = await _x.query('notes', where: 'sid = ?', whereArgs: [sid]);
    return r.isEmpty ? null : _note(r.first);
  }

  @override
  Future<void> saveNote(Note n) => _x.insert(
      'notes',
      {
        'sid': n.sid,
        'notebook': n.notebook,
        'title': n.title,
        'text': n.text,
        'updated_at': n.updatedAt.toUtc().toIso8601String(),
        'base': n.base,
        'changed': n.changed ? 1 : 0,
        'rev': n.rev,
        'editable': n.editable ? 1 : 0,
        'from_who': n.from,
      },
      conflictAlgorithm: ConflictAlgorithm.replace);

  @override
  Future<List<Note>> changedNotes(String notebook) async =>
      [for (final r in await _x.query('notes', where: 'notebook = ? AND changed = 1 AND editable = 1', whereArgs: [notebook])) _note(r)];

  @override
  Future<void> deleteNotes(List<String> sids) async {
    for (final s in sids) {
      await _x.delete('notes', where: 'sid = ?', whereArgs: [s]);
    }
  }

  @override
  Future<T> transaction<T>(Future<T> Function() body) => _db.transaction((txn) async {
        _txn = txn;
        try {
          return await body();
        } finally {
          _txn = null;
        }
      });

  @override
  Future<String?> meta(String key) async {
    final r = await _x.query('meta', where: 'k = ?', whereArgs: [key]);
    return r.isEmpty ? null : r.first['v'] as String;
  }

  @override
  Future<void> setMeta(String key, String value) => _x.insert('meta', {'k': key, 'v': value}, conflictAlgorithm: ConflictAlgorithm.replace);

  // ---- computers that were added ----

  Future<List<Computer>> computers() async => [
        for (final r in await _x.query('computers', orderBy: 'name COLLATE NOCASE'))
          Computer(
            id: r['id'] as String,
            name: r['name'] as String,
            hosts: [for (final h in jsonDecode(r['hosts'] as String) as List) h as String],
            port: r['port'] as int,
            fingerprint: r['fingerprint'] as String,
            project: r['project'] as String,
            lastSync: r['last_sync'] == null ? null : DateTime.parse(r['last_sync'] as String),
          ),
      ];

  Future<void> saveComputer(Computer c) => _x.insert(
      'computers',
      {
        'id': c.id,
        'name': c.name,
        'hosts': jsonEncode(c.hosts),
        'port': c.port,
        'fingerprint': c.fingerprint,
        'project': c.project,
        'last_sync': c.lastSync?.toUtc().toIso8601String(),
      },
      conflictAlgorithm: ConflictAlgorithm.replace);

  Future<void> deleteComputer(String id) => _x.delete('computers', where: 'id = ?', whereArgs: [id]);
}

/// The key of each computer, in the phone's secure storage. The database does not hold keys.
class KeyVault {
  KeyVault([FlutterSecureStorage? storage]) : _s = storage ?? const FlutterSecureStorage();
  final FlutterSecureStorage _s;

  Future<String?> get(String computerId) => _s.read(key: 'key_$computerId');
  Future<void> put(String computerId, String key) => _s.write(key: 'key_$computerId', value: key);
  Future<void> remove(String computerId) => _s.delete(key: 'key_$computerId');
}
