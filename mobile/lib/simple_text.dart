// The buttons above the keyboard. A note is written in the simple form (see docs/phone-sync-protocol.md), so each button puts the marks in the text.
// These functions work on plain values (text, where the selection starts and ends), so they can be tested without a screen.

class Edit {
  const Edit(this.text, this.start, this.end);
  final String text;
  final int start, end;

  @override
  bool operator ==(Object other) => other is Edit && other.text == text && other.start == start && other.end == end;
  @override
  int get hashCode => Object.hash(text, start, end);
  @override
  String toString() => 'Edit("$text", $start, $end)';
}

int _starsBefore(String t, int at) {
  var n = 0;
  while (at - n - 1 >= 0 && t[at - n - 1] == '*') {
    n++;
  }
  return n;
}

int _starsAfter(String t, int at) {
  var n = 0;
  while (at + n < t.length && t[at + n] == '*') {
    n++;
  }
  return n;
}

/// Bold (`**`) or italic (`*`) around the selection. With nothing selected the marks go in as a pair, with the cursor between them.
/// If the selection already has the marks around it, they are taken off. (Three stars are bold and italic together.)
Edit toggleWrap(String text, int start, int end, String mark) {
  if (start > end) {
    final t = start;
    start = end;
    end = t;
  }
  final m = mark.length;
  final b = _starsBefore(text, start), a = _starsAfter(text, end);
  final wrapped = mark == '**' ? (b >= 2 && a >= 2) : (b % 2 == 1 && a % 2 == 1 && b >= 1 && a >= 1);
  if (wrapped) {
    return Edit(text.substring(0, start - m) + text.substring(start, end) + text.substring(end + m), start - m, end - m);
  }
  if (start == end) {
    return Edit(text.substring(0, start) + mark + mark + text.substring(end), start + m, start + m);
  }
  return Edit(text.substring(0, start) + mark + text.substring(start, end) + mark + text.substring(end), start + m, end + m);
}

enum LineMark { heading, bullet, numbered, quote }

final _headingRe = RegExp(r'^(#{1,3}) ');
final _bulletRe = RegExp(r'^- ');
final _numberRe = RegExp(r'^\d+\. ');
final _quoteRe = RegExp(r'^> ');

/// The lines that the selection touches, as a range in the text: from the start of the first line to the end of the last.
(int, int) _lineRange(String text, int start, int end) {
  if (start > end) {
    final t = start;
    start = end;
    end = t;
  }
  final from = start == 0 ? 0 : text.lastIndexOf('\n', start - 1) + 1;
  final last = end > start && end > 0 && text[end - 1] == '\n' ? end - 1 : end; // a selection that ends at the start of a line does not touch that line
  var to = last >= text.length ? text.length : text.indexOf('\n', last);
  if (to < 0) to = text.length;
  return (from, to);
}

/// A heading (the mark goes # then ## then ### then none), a bullet, a number or a quote at the start of each line that the selection touches.
/// The same button again takes the mark off. A numbered list is numbered 1, 2, 3 from the first line.
Edit toggleLines(String text, int start, int end, LineMark kind) {
  final (from, to) = _lineRange(text, start, end);
  final lines = text.substring(from, to).split('\n');
  RegExp own;
  switch (kind) {
    case LineMark.heading:
      own = _headingRe;
    case LineMark.bullet:
      own = _bulletRe;
    case LineMark.numbered:
      own = _numberRe;
    case LineMark.quote:
      own = _quoteRe;
  }
  final allHave = lines.where((l) => l.trim().isNotEmpty).isNotEmpty && lines.where((l) => l.trim().isNotEmpty).every((l) => own.hasMatch(l));
  String strip(String l) => l.replaceFirst(_headingRe, '').replaceFirst(_bulletRe, '').replaceFirst(_numberRe, '').replaceFirst(_quoteRe, '');
  final out = <String>[];
  var n = 0;
  for (final l in lines) {
    if (l.trim().isEmpty && lines.length > 1) {
      out.add(l);
      continue;
    }
    switch (kind) {
      case LineMark.heading:
        final m = _headingRe.firstMatch(l);
        final level = m == null ? 0 : m.group(1)!.length;
        final next = level >= 3 ? 0 : level + 1;
        out.add((next == 0 ? '' : '${'#' * next} ') + strip(l));
      case LineMark.bullet:
        out.add(allHave ? strip(l) : '- ${strip(l)}');
      case LineMark.numbered:
        n++;
        out.add(allHave ? strip(l) : '$n. ${strip(l)}');
      case LineMark.quote:
        out.add(allHave ? strip(l) : '> ${strip(l)}');
    }
  }
  final changed = out.join('\n');
  final newText = text.substring(0, from) + changed + text.substring(to);
  // the cursor moves with the text: it goes to the end of the changed lines
  return Edit(newText, from + changed.length, from + changed.length);
}
