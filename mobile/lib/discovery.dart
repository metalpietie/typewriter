import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:nsd/nsd.dart' as nsd;

import 'app.dart';

/// Looks for computers that run Typewriter on the same Wi-Fi: they announce `_typewriter._tcp`.
/// Some routers do not let this through. Then the QR code and the typed address still work.
class Discovery extends ChangeNotifier {
  nsd.Discovery? _d;
  List<Candidate> found = const [];
  bool running = false;
  String? problem;

  Future<void> start() async {
    if (_d != null) return;
    try {
      _d = await nsd.startDiscovery('_typewriter._tcp', ipLookupType: nsd.IpLookupType.v4);
      running = true;
      _d!.addListener(_update);
      _update();
    } catch (e) {
      problem = 'Looking for computers did not start: $e';
      notifyListeners();
    }
  }

  void _update() {
    final out = <Candidate>[];
    for (final s in _d?.services ?? const <nsd.Service>[]) {
      final hosts = <String>[for (final a in s.addresses ?? const <InternetAddress>[]) if (a.type == InternetAddressType.IPv4) a.address];
      if (hosts.isEmpty || s.port == null) continue;
      String fp = '';
      final raw = s.txt?['fp'];
      if (raw != null) fp = utf8.decode(raw, allowMalformed: true);
      out.add(Candidate(name: s.name ?? 'Computer', hosts: hosts, port: s.port!, fingerprint: fp));
    }
    found = out;
    notifyListeners();
  }

  Future<void> stop() async {
    final d = _d;
    _d = null;
    running = false;
    if (d != null) {
      d.removeListener(_update);
      try {
        await nsd.stopDiscovery(d);
      } catch (_) {}
    }
  }

  @override
  void dispose() {
    stop();
    super.dispose();
  }
}
