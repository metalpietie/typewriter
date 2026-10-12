package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/jpeg"
	"strings"
)

// The cover of a manuscript. The screen makes a JPEG (from a picture you chose, or drawn from a template)
// and sends it as a data URL. These helpers read it so the Word, PDF and e-book writers can use it.

// coverData is a cover picture and its size in pixels.
type coverData struct {
	jpeg []byte
	w, h int
}

const maxCoverBytes = 20 << 20 // 20 MB

// decodeCover reads a "data:image/jpeg;base64,..." string. It returns nil when there is no cover, or when it is not a usable JPEG.
func decodeCover(dataURL string) *coverData {
	_, b64, ok := strings.Cut(dataURL, ",")
	if !ok || !strings.HasPrefix(dataURL, "data:image/jpeg") {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(raw) == 0 || len(raw) > maxCoverBytes {
		return nil
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 {
		return nil
	}
	return &coverData{jpeg: raw, w: cfg.Width, h: cfg.Height}
}

// fitIn gives the size of a picture of ratio w:h that fits in a box, with the same ratio.
// A picture that is almost the same shape as the box fills it.
func fitIn(w, h, boxW, boxH float64) (float64, float64) {
	if w <= 0 || h <= 0 {
		return boxW, boxH
	}
	pa, ba := w/h, boxW/boxH
	if d := (pa - ba) / ba; d > -0.04 && d < 0.04 {
		return boxW, boxH
	}
	if pa > ba {
		return boxW, boxW / pa
	}
	return boxH * pa, boxH
}

const emuPerMM = 36000

// docxCover is the paragraph that holds the cover picture in a Word file, then a page break.
// r:embed "rId2" points to word/media/cover.jpg. The picture fits inside the page margins.
func docxCover(c *coverData, availWmm, availHmm float64) string {
	w, h := fitIn(float64(c.w), float64(c.h), availWmm, availHmm)
	cx, cy := int64(w*emuPerMM), int64(h*emuPerMM)
	return fmt.Sprintf(`<w:p><w:pPr><w:pStyle w:val="Plain"/><w:jc w:val="center"/></w:pPr><w:r><w:drawing>`+
		`<wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:docPr id="1" name="Cover" descr="Cover"/>`+
		`<a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">`+
		`<pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:nvPicPr><pic:cNvPr id="0" name="cover.jpg"/><pic:cNvPicPr/></pic:nvPicPr>`+
		`<pic:blipFill><a:blip r:embed="rId2"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>`+
		`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic>`+
		`</a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p><w:p><w:r><w:br w:type="page"/></w:r></w:p>`, cx, cy, cx, cy)
}
