package main

import "math"

// ExportOptions are the choices on the Export screen.
type ExportOptions struct {
	Author   string `json:"author"`   // printed under the title, and set in the e-book
	PageSize string `json:"pageSize"` // a key from pageSizes (Word and PDF)
	Print    bool   `json:"print"`    // PDF only: mirrored margins, chapters on right-hand pages
	Contents bool   `json:"contents"` // add a table of contents after the title page
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
