import 'models.dart';

/// Where the notes are kept. The sync engine and the screens only know this, so the engine can be tested without a database.
abstract class NoteStore {
  Future<List<Notebook>> notebooks();
  Future<Notebook?> notebook(String id);
  Future<void> saveNotebook(Notebook notebook);

  /// Moves every note of one notebook into another, then removes the first one (unless it is the local one).
  Future<void> mergeNotebook(String from, String into);

  /// Gives a notebook a new id (the phone met the computer: the local notebook becomes the project's).
  Future<void> adoptNotebook(String from, Notebook as);

  /// The notes of a notebook, newest edit first. [query] filters on the title and the text.
  Future<List<Note>> notes(String notebook, {String query = ''});
  Future<Note?> note(String sid);
  Future<void> saveNote(Note note);

  /// The notes that the next sync sends: changed on the phone, and allowed to be edited here.
  Future<List<Note>> changedNotes(String notebook);
  Future<void> deleteNotes(List<String> sids);

  /// Runs [body] as one step: all of it happens, or none of it.
  Future<T> transaction<T>(Future<T> Function() body);

  Future<String?> meta(String key);
  Future<void> setMeta(String key, String value);
}
