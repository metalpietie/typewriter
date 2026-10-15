import 'dart:convert';

import 'package:device_info_plus/device_info_plus.dart';
import 'package:flutter/foundation.dart';
import 'package:uuid/uuid.dart';

import 'db.dart';
import 'models.dart';
import 'sync/api.dart';
import 'sync/engine.dart';

/// A computer that the phone has found or been told about, and that is not added yet.
class Candidate {
  Candidate({required this.name, required this.hosts, required this.port, this.fingerprint = ''});
  final String name;
  final List<String> hosts;
  final int port;

  /// '' when it is not known yet (the person typed the address). Then it is looked at first, and the person checks it.
  final String fingerprint;
}

/// The state of the whole app: the database, the computers, which notebook is open, and the sync.
class AppServices extends ChangeNotifier {
  AppServices._(this.store, this.vault, this.deviceId, this.deviceName, this.computers, this.notebookId);

  final SqliteStore store;
  final KeyVault vault;
  final String deviceId;
  String deviceName;
  List<Computer> computers;

  /// The notebook that the list shows and that new notes go in.
  String notebookId;
  bool syncing = false;

  /// A line to show at the top of the list: what the last sync did, or why it failed.
  String? message;
  bool messageIsError = false;

  /// Goes up when notes change, so the list reads them again.
  int rev = 0;

  static Future<AppServices> create({SqliteStore? store, KeyVault? vault}) async {
    final s = store ?? await SqliteStore.open();
    var id = await s.meta('deviceId');
    if (id == null) {
      id = const Uuid().v4();
      await s.setMeta('deviceId', id);
    }
    var name = await s.meta('deviceName');
    if (name == null) {
      name = await _guessDeviceName();
      await s.setMeta('deviceName', name);
    }
    if (await s.notebook(localNotebookId) == null && (await s.notebooks()).isEmpty) {
      await s.saveNotebook(Notebook(id: localNotebookId, name: 'On this phone'));
    }
    final books = await s.notebooks();
    final last = await s.meta('notebook');
    final current = books.any((b) => b.id == last) ? last! : books.first.id;
    return AppServices._(s, vault ?? KeyVault(), id, name, await s.computers(), current);
  }

  static Future<String> _guessDeviceName() async {
    try {
      final i = await DeviceInfoPlugin().androidInfo;
      final name = '${i.manufacturer} ${i.model}'.trim();
      return name.isEmpty ? 'My phone' : name[0].toUpperCase() + name.substring(1);
    } catch (_) {
      return 'My phone';
    }
  }

  void touch() {
    rev++;
    notifyListeners();
  }

  void say(String text, {bool error = false}) {
    message = text;
    messageIsError = error;
    notifyListeners();
  }

  void clearMessage() {
    message = null;
    notifyListeners();
  }

  Future<Notebook> get notebook async => (await store.notebook(notebookId)) ?? Notebook(id: notebookId, name: 'Notes');

  Future<void> setNotebook(String id) async {
    notebookId = id;
    await store.setMeta('notebook', id);
    touch();
  }

  Future<void> setDeviceName(String name) async {
    deviceName = name.trim().isEmpty ? 'My phone' : name.trim();
    await store.setMeta('deviceName', deviceName);
    notifyListeners();
  }

  /// Looks at a computer that is not added: its name, and the fingerprint of its certificate.
  Future<Candidate> inspect(Candidate c) async {
    final api = ComputerApi(hosts: c.hosts, port: c.port, fingerprint: '', timeout: const Duration(seconds: 5));
    final hello = await api.hello();
    if (hello['app'] != 'Typewriter') throw SyncException('That is not Typewriter.');
    if ((hello['protocol'] ?? 0) != protocolVersion) throw SyncException('This computer and this app do not speak the same version. Update the app or Typewriter.');
    return Candidate(name: (hello['name'] ?? c.name) as String, hosts: [if (api.lastGoodHost != null) api.lastGoodHost!, ...c.hosts.where((h) => h != api.lastGoodHost)], port: c.port, fingerprint: api.seenFingerprint ?? '');
  }

  /// Adds a computer with the code that Typewriter shows. [c.fingerprint] must be the one that the person saw (from the QR code, or checked by eye).
  Future<Computer> pair(Candidate c, String code) async {
    if (c.fingerprint.isEmpty) throw SyncException('The fingerprint of the computer is not known.');
    final api = ComputerApi(hosts: c.hosts, port: c.port, fingerprint: c.fingerprint);
    final res = await api.pair(code: code.trim(), deviceId: deviceId, name: deviceName);
    final computer = Computer(
      id: c.fingerprint,
      name: res.computer.isEmpty ? c.name : res.computer,
      hosts: [if (api.lastGoodHost != null) api.lastGoodHost!, ...c.hosts.where((h) => h != api.lastGoodHost)],
      port: c.port,
      fingerprint: c.fingerprint,
      project: res.projectId,
    );
    await vault.put(computer.id, res.key);
    await store.saveComputer(computer);
    computers = await store.computers();
    notifyListeners();
    return computer;
  }

  /// The QR code of Typewriter holds JSON with everything that is needed.
  Candidate fromQr(String payload) {
    Map<String, dynamic> j;
    try {
      j = (jsonDecode(payload) as Map).cast<String, dynamic>();
    } catch (_) {
      throw SyncException('That is not a Typewriter code.');
    }
    final hosts = [for (final h in (j['hosts'] ?? []) as List) h as String];
    if (j['v'] != protocolVersion) throw SyncException('This code is from another version of Typewriter. Update the app or Typewriter.');
    if (hosts.isEmpty || j['port'] is! int || (j['fp'] ?? '').toString().length != 64 || (j['code'] ?? '').toString().isEmpty) throw SyncException('That is not a Typewriter code.');
    return Candidate(name: (j['name'] ?? 'Computer') as String, hosts: hosts, port: j['port'] as int, fingerprint: j['fp'] as String);
  }

  Future<void> forget(Computer c) async {
    await vault.remove(c.id);
    await store.deleteComputer(c.id);
    computers = await store.computers();
    notifyListeners();
  }

  /// Syncs with a computer: the one given, or the only one that was added, or the last one used.
  Future<SyncResult?> syncNow({Computer? computer, bool quiet = false}) async {
    if (syncing) return null;
    final c = computer ?? (computers.isEmpty ? null : computers.reduce((a, b) => (a.lastSync ?? DateTime(2000)).isAfter(b.lastSync ?? DateTime(2000)) ? a : b));
    if (c == null) {
      if (!quiet) say('Add your computer first.', error: true);
      return null;
    }
    syncing = true;
    if (!quiet) message = null;
    notifyListeners();
    try {
      final key = await vault.get(c.id);
      if (key == null) throw SyncException('This computer needs to be added again.');
      final api = ComputerApi(hosts: c.hosts, port: c.port, fingerprint: c.fingerprint, deviceId: deviceId, key: key, timeout: quiet ? const Duration(seconds: 3) : const Duration(seconds: 10));
      final result = await SyncEngine(store).sync(api, notebookId: notebookId);
      final updated = c.copyWith(hosts: [if (api.lastGoodHost != null) api.lastGoodHost!, ...c.hosts.where((h) => h != api.lastGoodHost)], project: result.notebookId, lastSync: DateTime.now());
      await store.saveComputer(updated);
      computers = await store.computers();
      if (result.notebookId != notebookId) {
        notebookId = result.notebookId;
        await store.setMeta('notebook', notebookId);
      }
      say('${result.projectName.isEmpty ? c.name : result.projectName}: ${result.summary}');
      rev++;
      return result;
    } on SyncException catch (e) {
      if (!quiet) say(e.message, error: true);
      return null;
    } catch (e) {
      if (!quiet) say('The sync did not work: $e', error: true);
      return null;
    } finally {
      syncing = false;
      notifyListeners();
    }
  }
}
