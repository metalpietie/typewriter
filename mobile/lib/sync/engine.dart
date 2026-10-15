import '../models.dart';
import '../store.dart';
import 'api.dart';

/// What one sync did, in numbers, and which notebook is the right one now.
class SyncResult {
  SyncResult({required this.notebookId, required this.projectName, this.sent = 0, this.received = 0, this.removed = 0, this.switched = false});

  /// The notebook that belongs to the project that is open on the computer.
  final String notebookId;
  final String projectName;
  final int sent, received, removed;

  /// The computer has another project open than the notebook that was asked for: the phone moved to that notebook.
  final bool switched;

  String get summary {
    final parts = <String>[
      if (sent > 0) 'sent $sent',
      if (received > 0) 'got $received',
      if (removed > 0) 'removed $removed',
    ];
    return parts.isEmpty ? 'Up to date.' : 'Done: ${parts.join(', ')}.';
  }
}

/// One sync with a computer. The rules are in docs/phone-sync-protocol.md. In short: send what changed here, take what changed there,
/// and remember for each note the seq that the computer gave it. The phone never sends a delete.
class SyncEngine {
  SyncEngine(this.store);
  final NoteStore store;

  Future<SyncResult> sync(SyncApi api, {required String notebookId}) async {
    var nb = await store.notebook(notebookId) ?? Notebook(id: localNotebookId, name: 'On this phone');
    var switched = false;
    late SyncResponse resp;
    late List<Note> sent;
    for (var attempt = 0;; attempt++) {
      sent = await store.changedNotes(nb.id);
      resp = await api.sync(SyncRequest(project: nb.isLocal ? '' : nb.id, since: nb.isLocal ? 0 : nb.since, notes: sent));
      if (!resp.mismatch) break;
      if (attempt > 0 || resp.projectId.isEmpty) throw SyncException('The computer has another project open. Open "${resp.projectName}" there, or switch notebook here.');
      // the computer has another project open: this phone moves to that notebook, and asks again
      nb = await store.notebook(resp.projectId) ?? Notebook(id: resp.projectId, name: resp.projectName);
      await store.saveNotebook(nb.copyWith(name: resp.projectName));
      switched = true;
    }

    final sentRev = {for (final n in sent) n.sid: n.rev};
    var received = 0, removed = 0;
    await store.transaction(() async {
      // the notebook of the project: a local notebook becomes it, or joins it
      if (nb.isLocal) {
        final existing = await store.notebook(resp.projectId);
        if (existing != null) {
          await store.mergeNotebook(localNotebookId, resp.projectId);
          nb = existing;
        } else {
          await store.adoptNotebook(localNotebookId, Notebook(id: resp.projectId, name: resp.projectName));
          nb = Notebook(id: resp.projectId, name: resp.projectName);
        }
      }
      for (final e in resp.saved.entries) {
        final n = await store.note(e.key);
        if (n == null) continue;
        // a note that was edited again while the sync ran stays marked, with the new base
        await store.saveNote(n.copyWith(base: e.value, changed: n.rev != sentRev[e.key]));
      }
      for (final sid in resp.rejected) {
        final n = await store.note(sid);
        if (n != null) await store.saveNote(n.copyWith(changed: false, editable: false));
      }
      for (final c in resp.changed) {
        final local = await store.note(c.sid);
        // an edit that is not sent yet is not overwritten: the next sync finds the clash
        if (local != null && local.changed && sentRev[c.sid] != local.rev) continue;
        received++;
        final at = DateTime.now().toUtc();
        await store.saveNote((local ?? Note(sid: c.sid, notebook: nb.id)).copyWith(
          notebook: nb.id,
          title: c.title,
          text: c.text,
          base: c.seq,
          changed: false,
          editable: c.editable,
          from: c.from,
          updatedAt: local != null && local.title == c.title && local.text == c.text ? local.updatedAt : at,
        ));
      }
      for (final sid in resp.deleted) {
        final n = await store.note(sid);
        if (n != null && !n.changed) {
          await store.deleteNotes([sid]);
          removed++;
        }
      }
      await store.saveNotebook(nb.copyWith(name: resp.projectName.isEmpty ? nb.name : resp.projectName, since: resp.seq));
    });
    return SyncResult(notebookId: nb.id, projectName: resp.projectName, sent: sent.length, received: received, removed: removed, switched: switched);
  }
}
