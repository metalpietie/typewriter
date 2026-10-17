package app

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestSaveRefusesBadData(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)

	if _, err := s.Save(Doc{Kind: "version", Data: "not json"}); err == nil {
		t.Fatal("text that is not JSON must not be saved")
	}
	sc, err := s.Save(Doc{Kind: "scene", Data: `{"x":1}`})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(Doc{ID: sc.ID, Kind: "book", Data: `{}`}); err == nil {
		t.Fatal("saving an id with the wrong kind must fail, not pretend")
	}
	if _, err := s.Save(Doc{ID: 99999, Kind: "scene", Data: `{}`}); err == nil {
		t.Fatal("saving an item that is gone must fail")
	}
	// the data is still readable after the refused saves
	if _, err := s.ListBy("version", "scene", sc.ID); err != nil {
		t.Fatalf("a search must still work: %v", err)
	}
	if got, err := s.Save(Doc{ID: sc.ID, Kind: "scene", Data: `{"x":2}`}); err != nil || got.ID != sc.ID {
		t.Fatalf("a normal update must work: %v", err)
	}
}

func TestFileNamesKeepLetters(t *testing.T) {
	cases := map[string]string{
		"Les Misérables": "Les Misérables",
		"Война и мир":    "Война и мир",
		"龍の物語":           "龍の物語",
		"A/B:C?":         "ABC",
		"  ":             "export",
	}
	for in, want := range cases {
		if got := safeName(in); got != want {
			t.Errorf("safeName(%q) = %q, want %q", in, got, want)
		}
	}
}

// Characters that XML forbids must not reach the Word or e-book files.
func TestExportsAreValidXMLWithOddCharacters(t *testing.T) {
	odd := "<p>a ￾ b ￿ c \x00 d \x0b e \xff f</p>"
	blocks := []block{{kind: "title", text: "T￾"}, {kind: "h1", text: "Ch￿", anchor: "c1"}}
	blocks = append(blocks, flushFirst(chapterBlocks(odd))...)

	check := func(name string, data []byte, suffixes ...string) {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, f := range zr.File {
			ok := false
			for _, s := range suffixes {
				ok = ok || strings.HasSuffix(f.Name, s)
			}
			if !ok {
				continue
			}
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			d := xml.NewDecoder(bytes.NewReader(b))
			for {
				if _, err := d.Token(); err == io.EOF {
					break
				} else if err != nil {
					t.Errorf("%s %s is not valid XML: %v", name, f.Name, err)
					break
				}
			}
		}
	}
	docx, err := buildDocx(blocks, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	check("docx", docx, ".xml")
	epub, err := buildEPUB(blocks, "Title￾", "Au￿thor")
	if err != nil {
		t.Fatal(err)
	}
	check("epub", epub, "xhtml", ".opf", ".ncx")
}
