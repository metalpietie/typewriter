package app

import "math"

// ExportOptions are the choices on the Export screen.
type ExportOptions struct {
	Author   string `json:"author"`   // printed under the title, and set in the e-book
	PageSize string `json:"pageSize"` // a key from pageSizes (Word and PDF)
	Print    bool   `json:"print"`    // PDF only: mirrored margins, chapters on right-hand pages
	Contents bool   `json:"contents"` // add a table of contents after the title page

	// Pages and style of a manuscript. An empty text means no such page.
	Copyright    string `json:"copyright"`    // copyright page, after the title page
	Dedication   string `json:"dedication"`   // dedication page, after the copyright page
	Thanks       string `json:"thanks"`       // acknowledgments, after the last chapter
	About        string `json:"about"`        // about the author, at the very end
	SceneBreak   string `json:"sceneBreak"`   // "stars" (the default), "hash", "dashes" or "blank"
	ChapterStyle string `json:"chapterStyle"` // "full" (the default), "title", "number" or "plain"

	Cover string `json:"cover"` // a JPEG as a data URL, for the first page of a manuscript. Empty means no cover.

	// Non-fiction
	SectionHeadings bool   `json:"sectionHeadings"` // print the title of each section as a sub-heading, and no mark between sections
	Takeaways       bool   `json:"takeaways"`       // print "Key takeaways" at the end of each chapter that has some
	CitationStyle   string `json:"citationStyle"`   // "authordate" (the start), "numbered" or "notes"
	Bibliography    bool   `json:"bibliography"`    // print the list of the sources that were cited, after the last chapter
	IndexPage       bool   `json:"indexPage"`       // print an index of the terms that are marked in the text, at the end
	ExerciseList    bool   `json:"exerciseList"`    // print all the exercise boxes together, after the last chapter
	Nonfiction      bool   `json:"nonfiction"`      // the project is a non-fiction book: a scene is called a section

	// A submission (what is "submission", submission.go)
	Contact    string `json:"contact"`    // the lines at the top left of the title page: name, address, phone, email
	ShortTitle string `json:"shortTitle"` // the title in the page header; empty takes the first words of the title

	// The outline (what is "outline")
	OutlineDetails bool `json:"outlineDetails"` // add the point of view and the place of each scene
	OutlineEmpty   bool `json:"outlineEmpty"`   // list scenes that have no summary yet
}

// pageSize is a paper size in millimetres.
type pageSize struct {
	key  string
	w, h float64
}

var pageSizes = []pageSize{
	{"letter", 215.9, 279.4},
	{"a4", 210, 297},
	{"a5", 148, 210},
	{"6x9", 152.4, 228.6},     // US trade paperback
	{"5.5x8.5", 139.7, 215.9}, // digest
	{"5x8", 127, 203.2},       // pocket
}

func sizeOf(key string) pageSize {
	for _, s := range pageSizes {
		if s.key == key {
			return s
		}
	}
	return pageSizes[0]
}

// small pages get smaller margins and type, so a book page does not look empty.
func (s pageSize) small() bool { return s.w < 160 }

func (s pageSize) margin() float64 {
	if s.small() {
		return 18
	}
	return 25.4
}

func (s pageSize) fontPt() float64 {
	if s.small() {
		return 11
	}
	return 12
}

// twips converts millimetres to the unit Word uses (1/1440 inch).
func twips(mm float64) int { return int(math.Round(mm / 25.4 * 1440)) }

// sceneWord is what a scene is called in an export: "Scene" in a story, "Section" in a non-fiction book.
func (o ExportOptions) sceneWord() string {
	if o.Nonfiction {
		return "Section"
	}
	return "Scene"
}
