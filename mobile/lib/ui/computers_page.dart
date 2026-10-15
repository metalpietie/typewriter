import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../app.dart';
import '../discovery.dart';
import '../models.dart';
import '../sync/api.dart';

/// The computers that this phone can sync with, and how to add one.
class ComputersPage extends StatefulWidget {
  const ComputersPage({super.key, required this.app});
  final AppServices app;

  @override
  State<ComputersPage> createState() => _ComputersPageState();
}

class _ComputersPageState extends State<ComputersPage> {
  final _discovery = Discovery();
  bool _busy = false;
  AppServices get app => widget.app;

  @override
  void initState() {
    super.initState();
    _discovery.start();
  }

  @override
  void dispose() {
    _discovery.dispose();
    super.dispose();
  }

  void _error(Object e) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e is SyncException ? e.message : e.toString()), duration: const Duration(seconds: 6)));
  }

  Future<void> _finish(Computer c) async {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('${c.name} is added. Syncing…')));
    Navigator.of(context).pop(); // back to the notes
    await app.syncNow(computer: c);
  }

  /// Scan the QR code that Typewriter shows.
  Future<void> _scan() async {
    final payload = await Navigator.of(context).push<String>(MaterialPageRoute(builder: (_) => const _ScanPage()));
    if (payload == null) return;
    setState(() => _busy = true);
    try {
      final c = app.fromQr(payload);
      final code = (RegExp(r'"code"\s*:\s*"(\d+)"').firstMatch(payload)?.group(1)) ?? '';
      _finish(await app.pair(c, code));
    } catch (e) {
      _error(e);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  /// A computer that was found on the Wi-Fi, or an address that was typed: look at it, check the fingerprint, then ask for the code.
  Future<void> _addCandidate(Candidate c) async {
    setState(() => _busy = true);
    try {
      final seen = await app.inspect(c);
      if (c.fingerprint.isNotEmpty && !seen.fingerprint.startsWith(c.fingerprint)) throw SyncException('This computer is not the one that was found. Try again.');
      if (!mounted) return;
      final code = await _askCode(seen);
      if (code == null) return;
      _finish(await app.pair(seen, code));
    } catch (e) {
      _error(e);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  String _groups(String fp) => [for (var i = 0; i + 4 <= 16 && i + 4 <= fp.length; i += 4) fp.substring(i, i + 4)].join(' ');

  Future<String?> _askCode(Candidate c) {
    final ctl = TextEditingController();
    return showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Add ${c.name}'),
        content: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Text('In Typewriter, open Export & Backup, Phones, and press Add a phone. Check that this is the same fingerprint, then type the 6-digit code.'),
          const SizedBox(height: 12),
          Text(_groups(c.fingerprint), style: const TextStyle(fontFamily: 'monospace', fontSize: 18, fontWeight: FontWeight.bold)),
          const SizedBox(height: 12),
          TextField(controller: ctl, autofocus: true, keyboardType: TextInputType.number, maxLength: 6, decoration: const InputDecoration(labelText: 'Code')),
        ]),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          FilledButton(onPressed: () => Navigator.pop(ctx, ctl.text.trim()), child: const Text('Add')),
        ],
      ),
    );
  }

  Future<void> _byAddress() async {
    final host = TextEditingController(), port = TextEditingController(text: '47821');
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Add by address'),
        content: Column(mainAxisSize: MainAxisSize.min, children: [
          const Text('Typewriter shows the address under Phones, for example 192.168.1.20:47821.'),
          const SizedBox(height: 8),
          TextField(controller: host, autofocus: true, keyboardType: TextInputType.url, decoration: const InputDecoration(labelText: 'Address')),
          const SizedBox(height: 8),
          TextField(controller: port, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: 'Port')),
        ]),
        actions: [TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')), FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Next'))],
      ),
    );
    if (ok != true || host.text.trim().isEmpty) return;
    await _addCandidate(Candidate(name: host.text.trim(), hosts: [host.text.trim()], port: int.tryParse(port.text.trim()) ?? 47821));
  }

  Future<void> _rename() async {
    final ctl = TextEditingController(text: app.deviceName);
    final name = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Name of this phone'),
        content: TextField(controller: ctl, autofocus: true, decoration: const InputDecoration(helperText: 'Typewriter shows this name next to the notes from this phone.')),
        actions: [TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')), FilledButton(onPressed: () => Navigator.pop(ctx, ctl.text), child: const Text('Save'))],
      ),
    );
    if (name != null) await app.setDeviceName(name);
  }

  @override
  Widget build(BuildContext context) => ListenableBuilder(
        listenable: Listenable.merge([app, _discovery]),
        builder: (context, _) {
          final added = app.computers;
          final nearby = _discovery.found.where((f) => !added.any((a) => a.fingerprint.startsWith(f.fingerprint) && f.fingerprint.isNotEmpty)).toList();
          return Scaffold(
            appBar: AppBar(title: const Text('Computers')),
            body: ListView(padding: const EdgeInsets.all(12), children: [
              if (_busy) const LinearProgressIndicator(),
              ListTile(title: const Text('This phone'), subtitle: Text(app.deviceName), trailing: const Icon(Icons.edit, size: 18), onTap: _rename),
              const Divider(),
              const Padding(padding: EdgeInsets.fromLTRB(16, 8, 16, 4), child: Text('Added computers', style: TextStyle(fontWeight: FontWeight.bold))),
              if (added.isEmpty) const Padding(padding: EdgeInsets.fromLTRB(16, 4, 16, 12), child: Text('None yet.')),
              for (final c in added)
                Card(
                  child: ListTile(
                    leading: const Icon(Icons.computer),
                    title: Text(c.name),
                    subtitle: Text(c.lastSync == null ? 'Not synced yet' : 'Last sync: ${c.lastSync!.toLocal().toString().substring(0, 16)}'),
                    trailing: PopupMenuButton<String>(
                      onSelected: (v) async {
                        if (v == 'sync') {
                          Navigator.of(context).pop();
                          app.syncNow(computer: c);
                        } else {
                          final sure = await showDialog<bool>(
                            context: context,
                            builder: (ctx) => AlertDialog(
                              title: Text('Forget ${c.name}?'),
                              content: const Text('The notes on this phone stay. To sync with it again, add it again. You can also remove this phone in Typewriter.'),
                              actions: [TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')), FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Forget'))],
                            ),
                          );
                          if (sure == true) await app.forget(c);
                        }
                      },
                      itemBuilder: (_) => const [PopupMenuItem(value: 'sync', child: Text('Sync now')), PopupMenuItem(value: 'forget', child: Text('Forget'))],
                    ),
                  ),
                ),
              const Divider(),
              const Padding(padding: EdgeInsets.fromLTRB(16, 8, 16, 4), child: Text('Add a computer', style: TextStyle(fontWeight: FontWeight.bold))),
              const Padding(padding: EdgeInsets.fromLTRB(16, 0, 16, 8), child: Text('On the computer: open Typewriter, then Export & Backup, then Phones. Switch it on, and press Add a phone.')),
              Padding(padding: const EdgeInsets.symmetric(horizontal: 12), child: FilledButton.icon(onPressed: _busy ? null : _scan, icon: const Icon(Icons.qr_code_scanner), label: const Text('Scan the QR code'))),
              const SizedBox(height: 12),
              const Padding(padding: EdgeInsets.fromLTRB(16, 4, 16, 4), child: Text('Found on the Wi-Fi', style: TextStyle(fontWeight: FontWeight.w600))),
              if (_discovery.problem != null) Padding(padding: const EdgeInsets.all(16), child: Text(_discovery.problem!)),
              if (nearby.isEmpty) const Padding(padding: EdgeInsets.fromLTRB(16, 4, 16, 8), child: Text('Nothing yet. Some routers do not let phones find computers. Use the QR code, or add by address.')),
              for (final c in nearby) ListTile(leading: const Icon(Icons.wifi), title: Text(c.name), subtitle: Text('${c.hosts.first}:${c.port}'), onTap: _busy ? null : () => _addCandidate(c)),
              ListTile(leading: const Icon(Icons.keyboard), title: const Text('Add by address'), onTap: _busy ? null : _byAddress),
            ]),
          );
        },
      );
}

class _ScanPage extends StatefulWidget {
  const _ScanPage();
  @override
  State<_ScanPage> createState() => _ScanPageState();
}

class _ScanPageState extends State<_ScanPage> {
  bool _done = false;

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text('Scan the QR code')),
        body: Stack(children: [
          MobileScanner(
            onDetect: (capture) {
              if (_done) return;
              for (final b in capture.barcodes) {
                final v = b.rawValue;
                if (v != null && v.contains('"fp"')) {
                  _done = true;
                  Navigator.of(context).pop(v);
                  return;
                }
              }
            },
            errorBuilder: (context, error) => Center(child: Padding(padding: const EdgeInsets.all(24), child: Text('The camera cannot be used: ${error.errorDetails?.message ?? error.errorCode.name}. You can add the computer by address instead.', textAlign: TextAlign.center))),
          ),
          const Align(alignment: Alignment.bottomCenter, child: Padding(padding: EdgeInsets.all(24), child: Text('Point the camera at the QR code in Typewriter.', style: TextStyle(color: Colors.white, shadows: [Shadow(blurRadius: 6)])))),
        ]),
      );
}
