import 'dart:async';

import 'package:flutter/material.dart';

import '../app.dart';
import '../models.dart';
import '../simple_text.dart';

/// One note. It is saved as you type. There is no delete here: a note is deleted in Typewriter.
class EditorPage extends StatefulWidget {
  const EditorPage({super.key, required this.app, required this.sid, this.isNew = false});
  final AppServices app;
  final String sid;
  final bool isNew;

  @override
  State<EditorPage> createState() => _EditorPageState();
}

class _EditorPageState extends State<EditorPage> {
  final _title = TextEditingController();
  final _body = TextEditingController();
  final _bodyFocus = FocusNode();
  Note? _note;
  Timer? _timer;
  bool _loaded = false;

  AppServices get app => widget.app;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final n = await app.store.note(widget.sid) ?? Note(sid: widget.sid, notebook: app.notebookId, changed: false);
    _title.text = n.title;
    _body.text = n.text;
    setState(() {
      _note = n;
      _loaded = true;
    });
  }

  void _changed() {
    _timer?.cancel();
    _timer = Timer(const Duration(milliseconds: 500), _save);
  }

  Future<void> _save() async {
    _timer?.cancel();
    final n = _note;
    if (n == null || !n.editable) return;
    if (_title.text == n.title && _body.text == n.text) return;
    if (_title.text.trim().isEmpty && _body.text.trim().isEmpty && !(await app.store.note(n.sid) != null)) return; // an empty new note is not kept
    final saved = n.copyWith(title: _title.text, text: _body.text, updatedAt: DateTime.now().toUtc(), changed: true, rev: n.rev + 1);
    await app.store.saveNote(saved);
    _note = saved;
    app.touch();
  }

  @override
  void dispose() {
    _timer?.cancel();
    _save(); // the last words are not lost when the page closes
    _title.dispose();
    _body.dispose();
    _bodyFocus.dispose();
    super.dispose();
  }

  void _wrap(String mark) => _apply((t, s, e) => toggleWrap(t, s, e, mark));
  void _lines(LineMark k) => _apply((t, s, e) => toggleLines(t, s, e, k));

  void _apply(Edit Function(String, int, int) f) {
    final sel = _body.selection;
    final start = sel.isValid ? sel.start : _body.text.length;
    final end = sel.isValid ? sel.end : _body.text.length;
    final e = f(_body.text, start, end);
    _body.value = TextEditingValue(text: e.text, selection: TextSelection(baseOffset: e.start, extentOffset: e.end));
    _bodyFocus.requestFocus();
    _changed();
  }

  @override
  Widget build(BuildContext context) {
    final n = _note;
    final readOnly = n != null && !n.editable;
    return Scaffold(
      appBar: AppBar(title: Text(widget.isNew && (n?.title.isEmpty ?? true) ? 'New note' : 'Note'), actions: [
        if (n != null && n.from.isNotEmpty) Padding(padding: const EdgeInsets.only(right: 12), child: Center(child: Text('From ${n.from}', style: TextStyle(color: Theme.of(context).colorScheme.onSurfaceVariant)))),
      ]),
      body: !_loaded
          ? const SizedBox()
          : Column(children: [
              if (readOnly)
                MaterialBanner(
                  leading: const Icon(Icons.lock_outline),
                  content: const Text('This note has links, comments or citations. You can read it here. To change it, open it in Typewriter.'),
                  actions: const [SizedBox()],
                ),
              Expanded(
                child: ListView(padding: const EdgeInsets.fromLTRB(16, 8, 16, 16), children: [
                  readOnly
                      ? SelectableText(n.title.isEmpty ? 'Untitled' : n.title, style: Theme.of(context).textTheme.headlineSmall)
                      : TextField(
                          controller: _title,
                          textCapitalization: TextCapitalization.sentences,
                          style: Theme.of(context).textTheme.headlineSmall,
                          decoration: const InputDecoration(hintText: 'Title', border: InputBorder.none),
                          onChanged: (_) => _changed(),
                          textInputAction: TextInputAction.next,
                          onSubmitted: (_) => _bodyFocus.requestFocus(),
                        ),
                  const SizedBox(height: 8),
                  readOnly
                      ? SelectableText(n.text, style: const TextStyle(fontSize: 16, height: 1.5))
                      : TextField(
                          controller: _body,
                          focusNode: _bodyFocus,
                          autofocus: widget.isNew,
                          maxLines: null,
                          minLines: 12,
                          keyboardType: TextInputType.multiline,
                          textCapitalization: TextCapitalization.sentences,
                          style: const TextStyle(fontSize: 16, height: 1.5),
                          decoration: const InputDecoration(hintText: 'Write your idea…', border: InputBorder.none),
                          onChanged: (_) => _changed(),
                        ),
                ]),
              ),
              if (!readOnly)
                Material(
                  color: Theme.of(context).colorScheme.surfaceContainerHighest,
                  child: SafeArea(
                    top: false,
                    child: SingleChildScrollView(
                      scrollDirection: Axis.horizontal,
                      child: Row(children: [
                        IconButton(tooltip: 'Bold', icon: const Icon(Icons.format_bold), onPressed: () => _wrap('**')),
                        IconButton(tooltip: 'Italic', icon: const Icon(Icons.format_italic), onPressed: () => _wrap('*')),
                        IconButton(tooltip: 'Heading', icon: const Icon(Icons.title), onPressed: () => _lines(LineMark.heading)),
                        IconButton(tooltip: 'Bullet list', icon: const Icon(Icons.format_list_bulleted), onPressed: () => _lines(LineMark.bullet)),
                        IconButton(tooltip: 'Numbered list', icon: const Icon(Icons.format_list_numbered), onPressed: () => _lines(LineMark.numbered)),
                        IconButton(tooltip: 'Quote', icon: const Icon(Icons.format_quote), onPressed: () => _lines(LineMark.quote)),
                      ]),
                    ),
                  ),
                ),
            ]),
    );
  }
}
