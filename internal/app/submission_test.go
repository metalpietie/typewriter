package app

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSubmission(t *testing.T) {
	e, b1, _ := outlineStore(t)
	c := e.load("chapter")[0]
	e.store.Save(Doc{Kind: "scene", Data: fmt.Sprintf(`{"chapter":%v,"order":3,"body":"<p>Mara found the <em>letter</em>.</p><hr><p>She ran.</p>"}`, num(c, "id"))})

	if _, err := e.submission(0, ExportOptions{}); err == nil || !strings.Contains(err.Error(), "one book") {
		t.Fatalf("a series cannot be one submission: %v", err)
	}
	sub, err := e.submission(b1, ExportOptions{Author: "Ann Lee", Contact: "Ann Lee\n1 Clock Street\n\nann@example.com", Copyright: "© me", Dedication: "For you", Contents: true})
	if err != nil {
		t.Fatal(err)
	}
	if sub.surname != "Lee" || sub.short != "ONE" || len(sub.contact) != 3 || sub.words != 6 {
		t.Fatalf("submission: %+v", sub)
	}
	if got := sub.header(); got != "Lee / ONE / " {
		t.Fatalf("header %q", got)
	}
	for _, b := range sub.blocks {
		if b.kind == "title" || b.kind == "tochead" || strings.Contains(b.text, "For you") || strings.Contains(b.text, "© me") {
			t.Fatalf("no title, contents or front pages in a submission: %+v", b)
		}
		if b.kind == "rule" && b.text != "#" {
			t.Fatalf("a scene break is a #: %q", b.text)
		}
	}

	doc, err := sub.docx()
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(doc), int64(len(doc)))
	if err != nil {
		t.Fatal(err)
	}
	part := func(name string) string {
		for _, f := range z.File {
			if f.Name == name {
				r, _ := f.Open()
				b, _ := io.ReadAll(r)
				return string(b)
			}
		}
		t.Fatalf("no %s", name)
		return ""
	}
	body := part("word/document.xml")
	for _, want := range []string{"about 6 words", "1 Clock Street", ">ONE<", "by Ann Lee", "THE END", "<w:titlePg/>", "<w:i/>", ">#<"} {
		if !strings.Contains(body, want) {
			t.Errorf("the Word file has no %q", want)
		}
	}
	if !strings.Contains(part("word/header1.xml"), "Lee / ONE / ") || !strings.Contains(part("word/styles.xml"), `w:line="480"`) {
		t.Error("the header and the double spacing")
	}

	pdf, err := sub.pdf()
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("pdf: %v", err)
	}
}

func TestAboutWords(t *testing.T) {
	for n, want := range map[int]string{6: "about 6 words", 4249: "about 4,200 words", 4250: "about 4,300 words", 85432: "about 85,000 words", 120600: "about 121,000 words"} {
		if got := aboutWords(n); got != want {
			t.Errorf("aboutWords(%d) = %q, want %q", n, got, want)
		}
	}
	if got := shortTitle("The Brass Key and the Very Long Night"); got != "The Brass Key" {
		t.Errorf("short title %q", got)
	}
}

// TestSubmissionSample writes a sample submission to look at, when TYPEWRITER_SUBMISSION_OUT names a folder.
func TestSubmissionSample(t *testing.T) {
	dir := os.Getenv("TYPEWRITER_SUBMISSION_OUT")
	if dir == "" {
		t.Skip("set TYPEWRITER_SUBMISSION_OUT to a folder to write a sample")
	}
	e, b1, _ := outlineStore(t)
	para := strings.Repeat("<p>Mara found the letter under the old clock. The paper was yellow and soft, like cloth. She did not open it yet, because the bell of the bakery rang twice and <em>Tom</em> was early today.</p>", 12)
	for _, c := range e.load("chapter") {
		e.store.Save(Doc{Kind: "scene", Data: fmt.Sprintf(`{"chapter":%v,"order":9,"body":%q}`, num(c, "id"), para+"<hr>"+para)})
	}
	sub, err := e.submission(b1, ExportOptions{Author: "Ann Lee", Contact: "Ann Lee\n1 Clock Street\nBrassford, BR1 2AB\n+1 555 0100\nann@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	pdf, _ := sub.pdf()
	doc, _ := sub.docx()
	os.WriteFile(filepath.Join(dir, "submission.pdf"), pdf, 0o644)
	os.WriteFile(filepath.Join(dir, "submission.docx"), doc, 0o644)
}
