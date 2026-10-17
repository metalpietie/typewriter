package app

import (
	"strings"
	"testing"
)

func TestHTMLToSimple(t *testing.T) {
	cases := []struct {
		name, in, want string
		editable       bool
	}{
		{"empty", "", "", true},
		{"plain text page", "one\ntwo", "one\n\ntwo", true},
		{"paragraphs", "<p>One.</p><p>Two.</p>", "One.\n\nTwo.", true},
		{"headings", "<h1>A</h1><h2>B</h2><h3>C</h3>", "# A\n\n## B\n\n### C", true},
		{"deep heading is level 3", "<h5>Deep</h5>", "### Deep", true},
		{"bold and italic", "<p>a <strong>b</strong> <em>c</em> <strong><em>d</em></strong></p>", "a **b** *c* ***d***", true},
		{"bullets", "<ul><li><p>x</p></li><li><p>y</p></li></ul>", "- x\n- y", true},
		{"numbers", "<ol><li><p>x</p></li><li><p>y</p></li></ol>", "1. x\n2. y", true},
		{"quote", "<blockquote><p>q1</p><p>q2</p></blockquote>", "> q1\n> q2", true},
		{"a line break", "<p>a<br>b</p>", "a\nb", true},
		{"stars are escaped", "<p>2 * 3 \\ 4</p>", `2 \* 3 \\ 4`, true},
		{"a line that looks like a mark", "<p>- not a list</p>", `\- not a list`, true},
		{"a link to a character", `<p>Hello <span class="ref" data-ref="character:3">Mara</span></p>`, "Hello Mara", false},
		{"a comment", `<p><span data-comment="4" class="cmt">words</span></p>`, "words", false},
		{"a citation", `<p>fact<span data-cite="1"></span></p>`, "fact", false},
		{"a footnote", `<p>fact<span data-footnote="x"></span></p>`, "fact", false},
		{"underline", "<p><u>u</u></p>", "u", false},
		{"a box", `<div data-callout="idea"><p>in a box</p></div>`, "in a box", false},
		{"a rule", "<p>a</p><hr><p>b</p>", "a\n\nb", false},
		{"a list in a list", "<ul><li><p>a</p><ul><li><p>b</p></li></ul></li></ul>", "- a b", false},
	}
	for _, c := range cases {
		got, ed := htmlToSimple(c.in)
		if got != c.want || ed != c.editable {
			t.Errorf("%s: got %q editable=%v, want %q editable=%v", c.name, got, ed, c.want, c.editable)
		}
	}
}

func TestSimpleToHTML(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"One.\n\nTwo.", "<p>One.</p><p>Two.</p>"},
		{"# A\n\n## B\n\n### C", "<h1>A</h1><h2>B</h2><h3>C</h3>"},
		{"a **b** *c*", "<p>a <strong>b</strong> <em>c</em></p>"},
		{"- x\n- y", "<ul><li><p>x</p></li><li><p>y</p></li></ul>"},
		{"1. x\n2. y", "<ol><li><p>x</p></li><li><p>y</p></li></ol>"},
		{"> q1\n> q2", "<blockquote><p>q1</p><p>q2</p></blockquote>"},
		{"a\nb", "<p>a<br>b</p>"},
		{`2 \* 3 \\ 4`, `<p>2 * 3 \ 4</p>`},
		{`\- not a list`, "<p>- not a list</p>"},
		{`\*stars\*`, "<p>*stars*</p>"},
		{"a < b & c", "<p>a &lt; b &amp; c</p>"},
		{"half **open", "<p>half **open</p>"},
		{"one *lonely star", "<p>one *lonely star</p>"},
		{"- x\ntext", "<p>- x<br>text</p>"},
		{"#not a heading", "<p>#not a heading</p>"},
	}
	for _, c := range cases {
		if got := simpleToHTML(c.in); got != c.want {
			t.Errorf("%q: got %q want %q", c.in, got, c.want)
		}
	}
}

// What the computer gives to the phone, and the phone gives back unchanged, must not change the note.
func TestSimpleRoundTrip(t *testing.T) {
	inputs := []string{
		"<p>One.</p><p>Two <strong>bold</strong> and <em>italic</em>.</p>",
		"<h1>Title</h1><ul><li><p>a</p></li><li><p>b</p></li></ul><ol><li><p>c</p></li></ol><blockquote><p>q</p></blockquote>",
		"<p>2 * 3 and a \\ slash</p><p>- looks like a list</p>",
		"<p>line<br>break</p>",
	}
	for _, in := range inputs {
		text, ed := htmlToSimple(in)
		if !ed {
			t.Fatalf("should be editable: %s", in)
		}
		back := simpleToHTML(text)
		again, _ := htmlToSimple(back)
		if again != text {
			t.Errorf("not stable:\n html  %s\n text  %q\n back  %s\n again %q", in, text, back, again)
		}
		if strings.Contains(back, "\\") && !strings.Contains(in, "\\") {
			t.Errorf("a backslash leaked: %s", back)
		}
	}
}
