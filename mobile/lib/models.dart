// What the phone keeps. Notes and notebooks are in a database on the phone. A computer that was added is a small record, and its key is in secure storage.

/// The notebook for notes that were made before the phone met a computer. When the phone syncs, it becomes the notebook of the computer's open project.
const localNotebookId = 'local';

class Notebook {
  Notebook({required this.id, required this.name, this.since = 0});

  /// The id of the project on the computer, or [localNotebookId].
  final String id;
  final String name;

  /// The seq that the computer last gave for this notebook.
  final int since;

  bool get isLocal => id == localNotebookId;

  Notebook copyWith({String? id, String? name, int? since}) => Notebook(id: id ?? this.id, name: name ?? this.name, since: since ?? this.since);
}

class Note {
  Note({
    required this.sid,
    required this.notebook,
    this.title = '',
    this.text = '',
    DateTime? updatedAt,
    this.base = 0,
    this.changed = false,
    this.rev = 0,
    this.editable = true,
    this.from = '',
  }) : updatedAt = (updatedAt ?? DateTime.now()).toUtc();

  /// A uuid that never changes. The computer uses the same one.
  final String sid;
  final String notebook;
  final String title;

  /// The simple form: paragraphs, headings, **bold**, *italic*, lists and quotes.
  final String text;
  final DateTime updatedAt;

  /// The seq that the computer last gave for this note. 0 for a note made on the phone.
  final int base;

  /// Made or edited on the phone since the last sync: the next sync sends it.
  final bool changed;

  /// Goes up with every edit on the phone. A sync uses it to see that you edited a note while it ran.
  final int rev;

  /// False for a note with links, comments or citations. It can be read here, and changed only on the computer.
  final bool editable;

  /// The phone that first made the note, as the computer says. Empty for a note made on the computer.
  final String from;

  Note copyWith({String? notebook, String? title, String? text, DateTime? updatedAt, int? base, bool? changed, int? rev, bool? editable, String? from}) => Note(
        sid: sid,
        notebook: notebook ?? this.notebook,
        title: title ?? this.title,
        text: text ?? this.text,
        updatedAt: updatedAt ?? this.updatedAt,
        base: base ?? this.base,
        changed: changed ?? this.changed,
        rev: rev ?? this.rev,
        editable: editable ?? this.editable,
        from: from ?? this.from,
      );

  /// The first line of the text, for the list.
  String get preview {
    final lines = text.split('\n').map((l) => l.trim()).where((l) => l.isNotEmpty);
    return lines.isEmpty ? '' : lines.first.replaceAll(RegExp(r'^(#{1,3} |- |> |\d+\. )'), '').replaceAll('**', '').replaceAll(RegExp(r'\*|\\'), '');
  }
}

/// A computer that was added. The key is not here: it is in secure storage, under [id].
class Computer {
  Computer({required this.id, required this.name, required this.hosts, required this.port, required this.fingerprint, this.project = '', this.lastSync});

  /// The fingerprint of its certificate. A computer is known by it.
  final String id;
  final String name;

  /// The addresses to try, newest working one first.
  final List<String> hosts;
  final int port;
  final String fingerprint;

  /// The project that was open on the computer the last time.
  final String project;
  final DateTime? lastSync;

  Computer copyWith({List<String>? hosts, int? port, String? name, String? project, DateTime? lastSync}) => Computer(
        id: id,
        name: name ?? this.name,
        hosts: hosts ?? this.hosts,
        port: port ?? this.port,
        fingerprint: fingerprint,
        project: project ?? this.project,
        lastSync: lastSync ?? this.lastSync,
      );
}
