import 'package:flutter_test/flutter_test.dart';
import 'package:typewriter_notes/simple_text.dart';

void main() {
  group('bold and italic', () {
    test('wrap the selection', () {
      expect(toggleWrap('hello world', 6, 11, '**'), const Edit('hello **world**', 8, 13));
      expect(toggleWrap('hello world', 0, 5, '*'), const Edit('*hello* world', 1, 6));
    });
    test('nothing selected: a pair, with the cursor between', () {
      expect(toggleWrap('ab', 1, 1, '**'), const Edit('a****b', 3, 3));
    });
    test('the same button takes the marks off', () {
      expect(toggleWrap('hello **world**', 8, 13, '**'), const Edit('hello world', 6, 11));
      expect(toggleWrap('*hello* world', 1, 6, '*'), const Edit('hello world', 0, 5));
    });
    test('italic inside bold is not taken for italic marks', () {
      // **word**: the stars around "word" are bold. Italic adds one star each side.
      expect(toggleWrap('**word**', 2, 6, '*'), const Edit('***word***', 3, 7));
      // ***word***: bold and italic. Italic off leaves bold.
      expect(toggleWrap('***word***', 3, 7, '*'), const Edit('**word**', 2, 6));
      expect(toggleWrap('***word***', 3, 7, '**'), const Edit('*word*', 1, 5));
    });
    test('a selection that is backwards works', () {
      expect(toggleWrap('hello world', 11, 6, '**'), const Edit('hello **world**', 8, 13));
    });
  });

  group('lines', () {
    test('a heading goes # then ## then ### then none', () {
      var e = toggleLines('Title', 2, 2, LineMark.heading);
      expect(e.text, '# Title');
      e = toggleLines(e.text, 2, 2, LineMark.heading);
      expect(e.text, '## Title');
      e = toggleLines(e.text, 2, 2, LineMark.heading);
      expect(e.text, '### Title');
      e = toggleLines(e.text, 2, 2, LineMark.heading);
      expect(e.text, 'Title');
    });
    test('a bullet list on several lines, and off again', () {
      var e = toggleLines('one\ntwo\nthree', 0, 13, LineMark.bullet);
      expect(e.text, '- one\n- two\n- three');
      e = toggleLines(e.text, 0, e.text.length, LineMark.bullet);
      expect(e.text, 'one\ntwo\nthree');
    });
    test('a numbered list is numbered from 1', () {
      final e = toggleLines('a\nb\nc', 0, 5, LineMark.numbered);
      expect(e.text, '1. a\n2. b\n3. c');
    });
    test('only the lines that the selection touches', () {
      final e = toggleLines('one\ntwo\nthree', 5, 6, LineMark.quote);
      expect(e.text, 'one\n> two\nthree');
    });
    test('a new mark replaces the old one', () {
      expect(toggleLines('- item', 2, 2, LineMark.numbered).text, '1. item');
      expect(toggleLines('> said', 2, 2, LineMark.bullet).text, '- said');
    });
    test('empty text and the last line', () {
      expect(toggleLines('', 0, 0, LineMark.bullet).text, '- ');
      expect(toggleLines('a\nb', 3, 3, LineMark.bullet).text, 'a\n- b');
    });
    test('a blank line between lines stays blank', () {
      expect(toggleLines('a\n\nb', 0, 4, LineMark.bullet).text, '- a\n\n- b');
    });
  });
}
