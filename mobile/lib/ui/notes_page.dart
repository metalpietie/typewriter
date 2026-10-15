import 'package:flutter/material.dart';
import 'package:uuid/uuid.dart';

import '../app.dart';
import '../models.dart';
import 'computers_page.dart';
import 'editor_page.dart';

/// The list of notes of one notebook. A notebook is a project of Typewriter.
class NotesPage extends StatefulWidget {
  const NotesPage({super.key, required this.app});
  final AppServices app;

  @override
  State<NotesPage> createState() => _NotesPageState();
}

class _NotesPageState extends State<NotesPage> {
  final _search = TextEditingController();
  bool _searching = false;

  AppServices get app => widget.app;

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  Future<void> _newNote() async {
    final sid = const Uuid().v4();
    await Navigator.of(context).push(MaterialPageRoute(builder: (_) => EditorPage(app: app, sid: sid, isNew: true)));
  }

  Future<void> _pickNotebook() async {
    final books = await app.store.notebooks();
    if (!mounted) return;
    final id = await showModalBottomSheet<String>(
      context: context,
      builder: (ctx) => SafeArea(
        child: ListView(shrinkWrap: true, children: [
          const ListTile(title: Text('Notebooks', style: TextStyle(fontWeight: FontWeight.bold)), subtitle: Text('A notebook is a project in Typewriter.')),
          for (final b in books)
            ListTile(leading: Icon(b.id == app.notebookId ? Icons.radio_button_checked : Icons.radio_button_off), title: Text(b.name), onTap: () => Navigator.pop(ctx, b.id)),
          if (!books.any((b) => b.isLocal))
            ListTile(leading: const Icon(Icons.add), title: const Text('A notebook on this phone only'), onTap: () => Navigator.pop(ctx, localNotebookId)),
        ]),
      ),
    );
    if (id == null) return;
    if (id == localNotebookId && await app.store.notebook(id) == null) await app.store.saveNotebook(Notebook(id: localNotebookId, name: 'On this phone'));
    await app.setNotebook(id);
  }

  @override
  Widget build(BuildContext context) => ListenableBuilder(
        listenable: app,
        builder: (context, _) => Scaffold(
          appBar: AppBar(
            title: _searching
                ? TextField(controller: _search, autofocus: true, decoration: const InputDecoration(hintText: 'Search the notes', border: InputBorder.none), onChanged: (_) => setState(() {}))
                : FutureBuilder<Notebook>(
                    future: app.notebook,
                    key: ValueKey('nb${app.notebookId}${app.rev}'),
                    builder: (context, s) => InkWell(
                      onTap: _pickNotebook,
                      child: Row(mainAxisSize: MainAxisSize.min, children: [Flexible(child: Text(s.data?.name ?? 'Notes', overflow: TextOverflow.ellipsis)), const Icon(Icons.arrow_drop_down)]),
                    ),
                  ),
            actions: [
              IconButton(
                tooltip: _searching ? 'Close the search' : 'Search',
                icon: Icon(_searching ? Icons.close : Icons.search),
                onPressed: () => setState(() {
                  _searching = !_searching;
                  _search.clear();
                }),
              ),
              app.syncing
                  ? const Padding(padding: EdgeInsets.all(14), child: SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2)))
                  : IconButton(
                      tooltip: 'Sync with the computer',
                      icon: const Icon(Icons.sync),
                      onPressed: () => app.computers.isEmpty ? Navigator.of(context).push(MaterialPageRoute(builder: (_) => ComputersPage(app: app))) : app.syncNow(),
                    ),
              IconButton(tooltip: 'Computers', icon: const Icon(Icons.computer), onPressed: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => ComputersPage(app: app)))),
            ],
          ),
          floatingActionButton: FloatingActionButton.extended(onPressed: _newNote, icon: const Icon(Icons.edit), label: const Text('New note')),
          body: Column(children: [
            if (app.message != null)
              MaterialBanner(
                content: Text(app.message!),
                backgroundColor: app.messageIsError ? Theme.of(context).colorScheme.errorContainer : Theme.of(context).colorScheme.secondaryContainer,
                actions: [TextButton(onPressed: app.clearMessage, child: const Text('OK'))],
              ),
            Expanded(
              child: FutureBuilder<List<Note>>(
                key: ValueKey('${app.notebookId}/${app.rev}/${_search.text}'),
                future: app.store.notes(app.notebookId, query: _search.text),
                builder: (context, s) {
                  final notes = s.data ?? const <Note>[];
                  if (s.connectionState != ConnectionState.done && notes.isEmpty) return const SizedBox();
                  if (notes.isEmpty) {
                    return Center(
                      child: Padding(
                        padding: const EdgeInsets.all(32),
                        child: Text(
                          _search.text.isNotEmpty
                              ? 'No note has that in it.'
                              : app.computers.isEmpty
                                  ? 'No notes yet. Press "New note" to write one.\n\nTo copy notes with Typewriter, press the computer icon above and add your computer.'
                                  : 'No notes yet. Press "New note" to write one, or the sync button to get your notes from Typewriter.',
                          textAlign: TextAlign.center,
                        ),
                      ),
                    );
                  }
                  return ListView.separated(
                    padding: const EdgeInsets.fromLTRB(12, 8, 12, 96),
                    itemCount: notes.length,
                    separatorBuilder: (context, index) => const SizedBox(height: 6),
                    itemBuilder: (context, i) => _NoteCard(note: notes[i], onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => EditorPage(app: app, sid: notes[i].sid)))),
                  );
                },
              ),
            ),
          ]),
        ),
      );
}

class _NoteCard extends StatelessWidget {
  const _NoteCard({required this.note, required this.onTap});
  final Note note;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).colorScheme.onSurfaceVariant;
    return Card(
      margin: EdgeInsets.zero,
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Row(children: [
              Expanded(child: Text(note.title.isEmpty ? 'Untitled' : note.title, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 16))),
              if (!note.editable) Icon(Icons.lock_outline, size: 16, color: muted),
              if (note.changed) Padding(padding: const EdgeInsets.only(left: 6), child: Icon(Icons.circle, size: 8, color: Theme.of(context).colorScheme.primary)),
            ]),
            if (note.preview.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 4), child: Text(note.preview, maxLines: 2, overflow: TextOverflow.ellipsis, style: TextStyle(color: muted))),
          ]),
        ),
      ),
    );
  }
}
