import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';

import '../models.dart';

/// The messages between the phone and the computer. See docs/phone-sync-protocol.md in the Typewriter repository.
const protocolVersion = 1;

class SyncRequest {
  SyncRequest({required this.project, required this.since, required this.notes});

  /// The project that the phone thinks is open, or '' when it does not know yet.
  final String project;
  final int since;
  final List<Note> notes;

  Map<String, dynamic> toJson() => {
        'project': project,
        'since': since,
        'notes': [
          for (final n in notes) {'sid': n.sid, 'title': n.title, 'text': n.text, 'updatedAt': n.updatedAt.toIso8601String(), 'base': n.base},
        ],
      };
}

class RemoteNote {
  RemoteNote({required this.sid, required this.title, required this.text, required this.seq, required this.editable, required this.from});

  factory RemoteNote.fromJson(Map<String, dynamic> j) => RemoteNote(
        sid: j['sid'] as String,
        title: (j['title'] ?? '') as String,
        text: (j['text'] ?? '') as String,
        seq: (j['seq'] ?? 0) as int,
        editable: (j['editable'] ?? true) as bool,
        from: (j['from'] ?? '') as String,
      );

  final String sid, title, text, from;
  final int seq;
  final bool editable;
}

class SyncResponse {
  SyncResponse({
    required this.projectId,
    required this.projectName,
    required this.mismatch,
    required this.saved,
    required this.rejected,
    required this.changed,
    required this.deleted,
    required this.seq,
  });

  factory SyncResponse.fromJson(Map<String, dynamic> j) {
    final p = (j['project'] ?? {}) as Map<String, dynamic>;
    return SyncResponse(
      projectId: (p['id'] ?? '') as String,
      projectName: (p['name'] ?? '') as String,
      mismatch: (j['mismatch'] ?? false) as bool,
      saved: {for (final s in (j['saved'] ?? []) as List) (s as Map)['sid'] as String: s['seq'] as int},
      rejected: [for (final s in (j['rejected'] ?? []) as List) s as String],
      changed: [for (final c in (j['changed'] ?? []) as List) RemoteNote.fromJson(c as Map<String, dynamic>)],
      deleted: [for (final s in (j['deleted'] ?? []) as List) s as String],
      seq: (j['seq'] ?? 0) as int,
    );
  }

  final String projectId, projectName;
  final bool mismatch;

  /// sid to the seq that the computer gave it.
  final Map<String, int> saved;
  final List<String> rejected;
  final List<RemoteNote> changed;
  final List<String> deleted;
  final int seq;
}

/// What the engine needs from the network. A test gives it a made-up one.
abstract class SyncApi {
  Future<SyncResponse> sync(SyncRequest request);
}

/// Something went wrong that the person can understand: the message says what, in plain words.
class SyncException implements Exception {
  SyncException(this.message);
  final String message;
  @override
  String toString() => message;
}

String fingerprintOf(List<int> der) => sha256.convert(der).toString();

/// Talks to one computer over HTTPS. It accepts only the certificate with the fingerprint that was pinned, so no other computer can answer.
class ComputerApi implements SyncApi {
  ComputerApi({required this.hosts, required this.port, required this.fingerprint, this.deviceId = '', this.key = '', this.timeout = const Duration(seconds: 8)});

  final List<String> hosts;
  final int port;
  final String fingerprint;
  final String deviceId, key;
  final Duration timeout;

  /// The address that worked the last time, so it can be tried first the next time.
  String? lastGoodHost;

  /// When the fingerprint is not known yet (adding a computer by typing a code), this keeps what the computer showed.
  String? seenFingerprint;

  HttpClient _client() {
    final c = HttpClient()..connectionTimeout = hosts.length > 1 && timeout > const Duration(seconds: 3) ? const Duration(seconds: 3) : timeout; // a few addresses: do not wait long for a wrong one
    c.badCertificateCallback = (cert, host, port) {
      final fp = fingerprintOf(cert.der);
      seenFingerprint = fp;
      return fingerprint.isEmpty || fp == fingerprint; // an empty pin means: look, and decide afterwards
    };
    return c;
  }

  Future<Map<String, dynamic>> _call(String method, String path, {Object? body, bool auth = false}) async {
    Object? last;
    final order = [if (lastGoodHost != null && hosts.contains(lastGoodHost)) lastGoodHost!, ...hosts.where((h) => h != lastGoodHost)];
    if (order.isEmpty) throw SyncException('There is no address for this computer.');
    for (final host in order) {
      final client = _client();
      try {
        final req = await client.openUrl(method, Uri(scheme: 'https', host: host, port: port, path: path)).timeout(timeout);
        req.headers.contentType = ContentType.json;
        if (auth) req.headers.set('Authorization', 'Bearer $deviceId:$key');
        if (body != null) req.add(utf8.encode(jsonEncode(body)));
        final resp = await req.close().timeout(timeout);
        final text = await resp.transform(utf8.decoder).join().timeout(timeout);
        lastGoodHost = host;
        Map<String, dynamic> json = {};
        try {
          json = (jsonDecode(text) as Map).cast<String, dynamic>();
        } catch (_) {}
        if (resp.statusCode == 401) throw SyncException('This computer does not know this phone any more. Add it again.');
        if (resp.statusCode == 403) throw SyncException((json['error'] ?? 'The code is wrong or has run out.') as String);
        if (resp.statusCode != 200) throw SyncException((json['error'] ?? 'The computer said: ${resp.statusCode}') as String);
        return json;
      } on SyncException {
        rethrow;
      } on HandshakeException {
        throw SyncException('This is not the computer that was added (its certificate is different).');
      } catch (e) {
        last = e;
      } finally {
        client.close(force: true);
      }
    }
    throw SyncException('Cannot reach the computer. Is Typewriter open, with Phones switched on, and are you on the same Wi-Fi? ($last)');
  }

  Future<Map<String, dynamic>> hello() => _call('GET', '/hello');

  /// Adds this phone to the computer. Returns the key.
  Future<({String key, String computer, String projectId, String projectName})> pair({required String code, required String deviceId, required String name}) async {
    final j = await _call('POST', '/pair', body: {'code': code, 'deviceId': deviceId, 'name': name});
    final p = (j['project'] ?? {}) as Map;
    return (key: j['key'] as String, computer: (j['computer'] ?? '') as String, projectId: (p['id'] ?? '') as String, projectName: (p['name'] ?? '') as String);
  }

  @override
  Future<SyncResponse> sync(SyncRequest request) async => SyncResponse.fromJson(await _call('POST', '/sync', body: request.toJson(), auth: true));
}
