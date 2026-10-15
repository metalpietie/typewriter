package main

import (
	"strings"
	"testing"
)

func syncStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.shut() })
	return s
}

func addDraft(t *testing.T, s *Store, title, body string) Doc {
	t.Helper()
	d, err := s.newDraft(title, body)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func drafts(t *testing.T, s *Store) map[string]string { // title -> body
	t.Helper()
	list, _ := s.List("draft")
	out := map[string]string{}
	for _, d := range list {
		title, body, _ := draftFields(d)
		out[title] = body
	}
	return out
}

func doSync(t *testing.T, s *Store, req PhoneRequest) PhoneResponse {
	t.Helper()
	if req.Device == "" {
		req.Device = "phone-1"
	}
	if req.DeviceName == "" {
		req.DeviceName = "Pixel 7"
	}
	req.Project = s.projectInfo().ID
	r, err := s.syncNotes(req)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func changedBySID(r PhoneResponse) map[string]NoteOut {
	m := map[string]NoteOut{}
	for _, c := range r.Changed {
		m[c.SID] = c
	}
	return m
}

const sidA = "11111111-1111-4111-8111-111111111111"

func TestPhoneMakesANote(t *testing.T) {
	s := syncStore(t)
	r := doSync(t, s, PhoneRequest{Notes: []PhoneNote{{SID: sidA, Title: "Idea", Text: "A **big** idea.\n\n- one\n- two", UpdatedAt: "2026-10-14T10:00:00Z"}}})
	if len(r.Saved) != 1 || r.Saved[0].SID != sidA || r.Saved[0].Seq == 0 {
		t.Fatalf("saved: %+v", r.Saved)
	}
	got := drafts(t, s)
	if got["Idea"] != "<p>A <strong>big</strong> idea.</p><ul><li><p>one</p></li><li><p>two</p></li></ul>" {
		t.Fatalf("the page: %q", got["Idea"])
	}
	// the page says that it came from the phone
	notes, _ := s.readSyncNotes()
	if len(notes) != 1 || notes[0].From != "Pixel 7" || notes[0].SID != sidA {
		t.Fatalf("syncnote: %+v", notes)
	}
	// the phone does not get its own note back
	if len(r.Changed) != 0 || len(r.Deleted) != 0 {
		t.Fatalf("changed %v deleted %v", r.Changed, r.Deleted)
	}
	// a second sync with nothing new does nothing
	r2 := doSync(t, s, PhoneRequest{Since: r.Seq})
	if len(r2.Saved)+len(r2.Changed)+len(r2.Deleted) != 0 || len(drafts(t, s)) != 1 {
		t.Fatalf("second sync: %+v", r2)
	}
}

func TestNoteSentTwice(t *testing.T) {
	s := syncStore(t)
	n := PhoneNote{SID: sidA, Title: "Idea", Text: "text", UpdatedAt: "2026-10-14T10:00:00Z"}
	doSync(t, s, PhoneRequest{Notes: []PhoneNote{n}})
	r := doSync(t, s, PhoneRequest{Notes: []PhoneNote{n}}) // the answer was lost, so the phone sends it again
	if len(drafts(t, s)) != 1 || len(r.Saved) != 1 {
		t.Fatalf("a second copy was made: %v %+v", drafts(t, s), r.Saved)
	}
}

func TestComputerNotesGoToThePhone(t *testing.T) {
	s := syncStore(t)
	addDraft(t, s, "From the computer", "<p>Hello <strong>phone</strong></p>")
	r := doSync(t, s, PhoneRequest{})
	if len(r.Changed) != 1 || r.Changed[0].Title != "From the computer" || r.Changed[0].Text != "Hello **phone**" || !r.Changed[0].Editable || r.Changed[0].Seq == 0 {
		t.Fatalf("changed: %+v", r.Changed)
	}
	sid, seq := r.Changed[0].SID, r.Changed[0].Seq
	// nothing changed: the next sync is empty
	if r2 := doSync(t, s, PhoneRequest{Since: r.Seq}); len(r2.Changed) != 0 {
		t.Fatalf("again: %+v", r2.Changed)
	}
	// an edit on the computer comes with a new seq
	list, _ := s.List("draft")
	if _, err := s.updateDraft(list[0], "From the computer", "<p>Edited</p>"); err != nil {
		t.Fatal(err)
	}
	r3 := doSync(t, s, PhoneRequest{Since: r.Seq})
	c := changedBySID(r3)[sid]
	if len(r3.Changed) != 1 || c.Text != "Edited" || c.Seq <= seq {
		t.Fatalf("edit: %+v", r3.Changed)
	}
}

func TestPhoneEditsAComputerNote(t *testing.T) {
	s := syncStore(t)
	addDraft(t, s, "Plan", "<p>First</p>")
	r := doSync(t, s, PhoneRequest{})
	sid, seq := r.Changed[0].SID, r.Changed[0].Seq
	r2 := doSync(t, s, PhoneRequest{Since: r.Seq, Notes: []PhoneNote{{SID: sid, Title: "Plan", Text: "First, and more", Base: seq, UpdatedAt: "2026-10-14T10:00:00Z"}}})
	if len(r2.Saved) != 1 || r2.Saved[0].Seq <= seq {
		t.Fatalf("saved: %+v", r2.Saved)
	}
	if got := drafts(t, s); len(got) != 1 || got["Plan"] != "<p>First, and more</p>" {
		t.Fatalf("pages: %v", got)
	}
}

func TestClashPhoneNewer(t *testing.T) {
	s := syncStore(t)
	addDraft(t, s, "Plan", "<p>First</p>")
	r := doSync(t, s, PhoneRequest{})
	sid, seq := r.Changed[0].SID, r.Changed[0].Seq
	list, _ := s.List("draft")
	if _, err := s.updateDraft(list[0], "Plan", "<p>Computer edit</p>"); err != nil {
		t.Fatal(err)
	}
	r2 := doSync(t, s, PhoneRequest{Since: r.Seq, Notes: []PhoneNote{{SID: sid, Title: "Plan", Text: "Phone edit", Base: seq, UpdatedAt: "2099-01-01T00:00:00Z"}}})
	got := drafts(t, s)
	if got["Plan"] != "<p>Phone edit</p>" || got["Plan (earlier version)"] != "<p>Computer edit</p>" || len(got) != 2 {
		t.Fatalf("pages: %v", got)
	}
	if len(r2.Saved) != 1 || len(r2.Changed) != 1 || r2.Changed[0].Title != "Plan (earlier version)" {
		t.Fatalf("answer: %+v", r2)
	}
}

func TestClashComputerNewer(t *testing.T) {
	s := syncStore(t)
	addDraft(t, s, "Plan", "<p>First</p>")
	r := doSync(t, s, PhoneRequest{})
	sid, seq := r.Changed[0].SID, r.Changed[0].Seq
	list, _ := s.List("draft")
	if _, err := s.updateDraft(list[0], "Plan", "<p>Computer edit</p>"); err != nil {
		t.Fatal(err)
	}
	r2 := doSync(t, s, PhoneRequest{Since: r.Seq, Notes: []PhoneNote{{SID: sid, Title: "Plan", Text: "Phone edit", Base: seq, UpdatedAt: "2000-01-01T00:00:00Z"}}})
	got := drafts(t, s)
	if got["Plan"] != "<p>Computer edit</p>" || got["Plan (copy from Pixel 7)"] != "<p>Phone edit</p>" || len(got) != 2 {
		t.Fatalf("pages: %v", got)
	}
	if len(r2.Saved) != 0 {
		t.Fatalf("the phone's edit was not taken as the note: %+v", r2.Saved)
	}
	m := changedBySID(r2)
	if m[sid].Text != "Computer edit" || len(r2.Changed) != 2 { // the computer's version, and the copy
		t.Fatalf("the phone must get both: %+v", r2.Changed)
	}
}

func TestDeleteOnTheComputerGoesToThePhone(t *testing.T) {
	s := syncStore(t)
	d := addDraft(t, s, "Gone soon", "<p>x</p>")
	r := doSync(t, s, PhoneRequest{})
	sid := r.Changed[0].SID
	if _, err := s.Delete("draft", d.ID); err != nil {
		t.Fatal(err)
	}
	r2 := doSync(t, s, PhoneRequest{Since: r.Seq})
	if len(r2.Deleted) != 1 || r2.Deleted[0] != sid || len(r2.Changed) != 0 {
		t.Fatalf("answer: %+v", r2)
	}
	// a phone that is behind still hears about it, and one that is up to date does not hear it twice
	if r3 := doSync(t, s, PhoneRequest{Since: r2.Seq}); len(r3.Deleted) != 0 {
		t.Fatalf("again: %+v", r3.Deleted)
	}
}

func TestDeleteOnTheComputerAfterAPhoneEdit(t *testing.T) {
	s := syncStore(t)
	d := addDraft(t, s, "Plan", "<p>First</p>")
	r := doSync(t, s, PhoneRequest{})
	sid, seq := r.Changed[0].SID, r.Changed[0].Seq
	if _, err := s.Delete("draft", d.ID); err != nil {
		t.Fatal(err)
	}
	r2 := doSync(t, s, PhoneRequest{Since: r.Seq, Notes: []PhoneNote{{SID: sid, Title: "Plan", Text: "Written on the phone", Base: seq, UpdatedAt: "2026-10-14T10:00:00Z"}}})
	got := drafts(t, s)
	if len(got) != 1 || got["Plan (restored from Pixel 7)"] != "<p>Written on the phone</p>" {
		t.Fatalf("the phone's words are lost: %v", got)
	}
	if len(r2.Deleted) != 0 || len(r2.Saved) != 1 || changedBySID(r2)[sid].Title != "Plan (restored from Pixel 7)" {
		t.Fatalf("answer: %+v", r2)
	}
}

func TestRichNoteIsReadOnlyOnThePhone(t *testing.T) {
	s := syncStore(t)
	addDraft(t, s, "Linked", `<p>Talk to <span class="ref" data-ref="character:3">Mara</span></p>`)
	r := doSync(t, s, PhoneRequest{})
	c := r.Changed[0]
	if c.Editable || c.Text != "Talk to Mara" {
		t.Fatalf("changed: %+v", c)
	}
	// a phone that sends an edit anyway is turned down, and the link stays
	r2 := doSync(t, s, PhoneRequest{Since: r.Seq, Notes: []PhoneNote{{SID: c.SID, Title: "Linked", Text: "Talk to nobody", Base: c.Seq, UpdatedAt: "2026-10-14T10:00:00Z"}}})
	if len(r2.Rejected) != 1 || len(r2.Saved) != 0 {
		t.Fatalf("answer: %+v", r2)
	}
	if !strings.Contains(drafts(t, s)["Linked"], "data-ref") {
		t.Fatal("the link was removed")
	}
}

func TestOtherProjectAndBadNotes(t *testing.T) {
	s := syncStore(t)
	r, err := s.syncNotes(PhoneRequest{Project: "another-project", Notes: []PhoneNote{{SID: sidA, Title: "x", Text: "y"}}})
	if err != nil || !r.Mismatch || len(drafts(t, s)) != 0 {
		t.Fatalf("a note was saved in the wrong project: %+v %v", r, drafts(t, s))
	}
	r = doSync(t, s, PhoneRequest{Notes: []PhoneNote{
		{SID: "bad id!", Title: "x", Text: "y"},
		{SID: sidA, Title: strings.Repeat("t", 1000), Text: "ok\x00text"},
		{SID: "22222222-2222-4222-8222-222222222222", Title: "big", Text: strings.Repeat("x", maxText+1)},
	}})
	if len(r.Rejected) != 2 || len(r.Saved) != 1 {
		t.Fatalf("answer: %+v", r)
	}
	for title, body := range drafts(t, s) {
		if len([]rune(title)) > maxTitle || strings.Contains(body, "\x00") {
			t.Fatalf("an unchecked note was saved: %q", title)
		}
	}
}

func TestTwoPhones(t *testing.T) {
	s := syncStore(t)
	sidB := "33333333-3333-4333-8333-333333333333"
	r1 := doSync(t, s, PhoneRequest{Device: "p1", DeviceName: "Pixel", Notes: []PhoneNote{{SID: sidA, Title: "One", Text: "from the first"}}})
	r2 := doSync(t, s, PhoneRequest{Device: "p2", DeviceName: "Tablet", Notes: []PhoneNote{{SID: sidB, Title: "Two", Text: "from the second"}}})
	if changedBySID(r2)[sidA].Text != "from the first" {
		t.Fatalf("the second phone did not get the first phone's note: %+v", r2.Changed)
	}
	r3 := doSync(t, s, PhoneRequest{Device: "p1", DeviceName: "Pixel", Since: r1.Seq})
	if changedBySID(r3)[sidB].Text != "from the second" || changedBySID(r3)[sidB].From != "Tablet" {
		t.Fatalf("the first phone did not get the second phone's note: %+v", r3.Changed)
	}
}

func TestSeqIsPerProject(t *testing.T) {
	s := syncStore(t)
	a := doSync(t, s, PhoneRequest{Notes: []PhoneNote{{SID: sidA, Title: "x", Text: "y"}}})
	b := doSync(t, s, PhoneRequest{Notes: []PhoneNote{{SID: "44444444-4444-4444-8444-444444444444", Title: "z", Text: "w"}}})
	if b.Seq <= a.Seq {
		t.Fatalf("seq goes up: %d then %d", a.Seq, b.Seq)
	}
}
