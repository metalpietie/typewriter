package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"strconv"
	"strings"
	"testing"
)

func testCover(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 255 / w), uint8(y * 255 / h), 120, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestDecodeCover(t *testing.T) {
	url := testCover(t, 400, 600)
	c := decodeCover(url)
	if c == nil || c.w != 400 || c.h != 600 {
		t.Fatalf("cover %+v", c)
	}
	for name, bad := range map[string]string{
		"empty": "", "no comma": "data:image/jpeg;base64", "png type": strings.Replace(url, "image/jpeg", "image/png", 1),
		"not base64": "data:image/jpeg;base64,@@@", "not a picture": "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte("hello")),
	} {
		if decodeCover(bad) != nil {
			t.Errorf("%s must give no cover", name)
		}
	}
}

func TestFitIn(t *testing.T) {
	if w, h := fitIn(400, 600, 100, 150); w != 100 || h != 150 {
		t.Errorf("same shape fills the box: %v %v", w, h)
	}
	if w, h := fitIn(400, 600, 215.9, 279.4); h != 279.4 || w >= 215.9 { // a tall picture on a letter page: fit the height
		t.Errorf("tall picture: %v %v", w, h)
	}
	if w, h := fitIn(600, 400, 100, 150); w != 100 || h >= 150 {
		t.Errorf("wide picture: %v %v", w, h)
	}
	if w, h := fitIn(0, 0, 10, 20); w != 10 || h != 20 {
		t.Errorf("no size: %v %v", w, h)
	}
}

func wellFormedXML(t *testing.T, name string, data string) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(data))
	for {
		if _, err := d.Token(); err == io.EOF {
			return
		} else if err != nil {
			t.Errorf("%s is not valid XML: %v", name, err)
			return
		}
	}
}

func TestCoverInWordAndEPUB(t *testing.T) {
	url := testCover(t, 600, 900)
	blocks := sampleBlocks()

	docx, err := buildDocx(blocks, ExportOptions{Cover: url})
	if err != nil {
		t.Fatal(err)
	}
	_, files := unzipAll(t, docx)
	if _, ok := files["word/media/cover.jpg"]; !ok {
		t.Fatal("word: the picture is missing")
	}
	for _, n := range []string{"[Content_Types].xml", "word/_rels/document.xml.rels", "word/document.xml"} {
		wellFormedXML(t, "docx "+n, files[n])
	}
	if !strings.Contains(files["[Content_Types].xml"], `Extension="jpg"`) || !strings.Contains(files["word/_rels/document.xml.rels"], `Id="rId2"`) {
		t.Error("word: the picture is not declared")
	}
	if !strings.Contains(files["word/document.xml"], `r:embed="rId2"`) || strings.Index(files["word/document.xml"], "<w:drawing>") > strings.Index(files["word/document.xml"], "My Book") {
		t.Error("word: the cover must come first")
	}
	plain, _ := buildDocx(blocks, ExportOptions{})
	if _, files := unzipAll(t, plain); files["word/media/cover.jpg"] != "" || strings.Contains(files["word/document.xml"], "<w:drawing>") {
		t.Error("word: no cover was asked for")
	}

	epub, err := buildEPUBWithCover(blocks, "My Book", "A. Writer", decodeCover(url))
	if err != nil {
		t.Fatal(err)
	}
	zr, ef := unzipAll(t, epub)
	if zr.File[0].Name != "mimetype" {
		t.Error("epub: mimetype must stay first")
	}
	if ef["OEBPS/images/cover.jpg"] == "" || !strings.Contains(ef["OEBPS/content.opf"], `properties="cover-image"`) || !strings.Contains(ef["OEBPS/content.opf"], `<meta name="cover" content="cover-image"/>`) {
		t.Error("epub: the cover is not declared")
	}
	if !strings.Contains(ef["OEBPS/cover.xhtml"], "images/cover.jpg") {
		t.Error("epub: no cover page")
	}
	spine := ef["OEBPS/content.opf"][strings.Index(ef["OEBPS/content.opf"], "<spine"):]
	if !strings.HasPrefix(strings.TrimSpace(strings.SplitN(spine, "\n", 2)[1]), `<itemref idref="p1"/>`) || !strings.Contains(ef["OEBPS/content.opf"], `<item id="p1" href="cover.xhtml"`) {
		t.Error("epub: the cover must be first in the reading order")
	}
	for name, body := range ef {
		if strings.HasSuffix(name, "xhtml") || strings.HasSuffix(name, ".opf") || strings.HasSuffix(name, ".ncx") {
			wellFormedXML(t, "epub "+name, body)
		}
	}
	if plainEpub, _ := buildEPUB(blocks, "My Book", "A. Writer"); bytes.Contains(plainEpub, []byte("cover.xhtml")) {
		t.Error("epub: no cover was asked for")
	}
}

func pdfPages(data []byte) int { return bytes.Count(data, []byte("/Type /Page\n")) }

func TestCoverInPDF(t *testing.T) {
	url := testCover(t, 600, 900)
	blocks := sampleBlocks()
	without, err := buildPDF(blocks, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	with, err := buildPDF(blocks, ExportOptions{Cover: url})
	if err != nil {
		t.Fatal(err)
	}
	if pdfPages(with) != pdfPages(without)+1 {
		t.Errorf("a cover adds one page: %d and %d", pdfPages(without), pdfPages(with))
	}
	// a print-ready PDF has no cover: the printer wants it as its own file
	printed, err := buildPDF(blocks, ExportOptions{Cover: url, Print: true})
	if err != nil {
		t.Fatal(err)
	}
	plainPrint, _ := buildPDF(blocks, ExportOptions{Print: true})
	if pdfPages(printed) != pdfPages(plainPrint) {
		t.Errorf("print layout keeps no cover: %d and %d", pdfPages(printed), pdfPages(plainPrint))
	}
	// with a table of contents, both runs agree on the page numbers
	if _, err := buildPDF(withContents(blocks), ExportOptions{Cover: url}); err != nil {
		t.Error(err)
	}
	// a bad cover is ignored
	if bad, err := buildPDF(blocks, ExportOptions{Cover: "data:image/jpeg;base64,xx"}); err != nil || pdfPages(bad) != pdfPages(without) {
		t.Errorf("bad cover: %v", err)
	}
}

// A cover goes with its book.
func TestCoverGoesWithItsBook(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	book, _ := s.Save(Doc{Kind: "book", Data: `{"title":"One"}`})
	other, _ := s.Save(Doc{Kind: "book", Data: `{"title":"Two"}`})
	s.Save(Doc{Kind: "cover", Data: `{"book":` + itoa(book.ID) + `,"image":"data:image/jpeg;base64,AAAA"}`})
	s.Save(Doc{Kind: "cover", Data: `{"book":` + itoa(other.ID) + `,"image":"data:image/jpeg;base64,AAAA"}`})
	removed, err := s.Delete("book", book.ID)
	if err != nil || len(removed) != 2 {
		t.Fatalf("removed %d, %v", len(removed), err)
	}
	if got, _ := s.ListBy("cover", "book", book.ID); len(got) != 0 {
		t.Error("the cover of a deleted book stayed")
	}
	if got, _ := s.ListBy("cover", "book", other.ID); len(got) != 1 {
		t.Error("the cover of another book was removed")
	}
	if err := s.Restore(removed); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ListBy("cover", "book", book.ID); len(got) != 1 {
		t.Error("Undo must bring the cover back")
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
