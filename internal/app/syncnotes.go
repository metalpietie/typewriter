package app

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Syncing the Notes pages with phones. The computer is the hub: it counts every change with a number (seq),
// and each phone remembers the seq that it last got for each note. Time is used only to say which side of a clash is newer.
//
// A Notes page is a "draft" item. A "syncnote" item says that a page is synced: its sync id (sid), the seq of its last change,
// a fingerprint of its title and text, whether it was deleted, and which phone it came from.

// PhoneNote is a note that the phone sends: new, or changed since the last sync. The phone never sends a delete.
type PhoneNote struct {
	SID       string `json:"sid"`
	Title     string `json:"title"`
	Text      string `json:"text"` // the simple form (see notesimple.go)
	UpdatedAt string `json:"updatedAt"`
	Base      int64  `json:"base"` // the seq that the phone last got for this note. 0 for a note made on the phone.
}

// PhoneRequest is one sync.
type PhoneRequest struct {
	Device     string      `json:"device"`
	DeviceName string      `json:"deviceName"`
	Project    string      `json:"project"` // the project that the phone thinks is open
	Since      int64       `json:"since"`   // the seq that the phone last got
	Notes      []PhoneNote `json:"notes"`
}

// NoteOut is a note that the computer sends.
type NoteOut struct {
	SID       string `json:"sid"`
	Title     string `json:"title"`
	Text      string `json:"text"`
	Seq       int64  `json:"seq"`
	UpdatedAt string `json:"updatedAt"`
	Editable  bool   `json:"editable"` // false: shown on the phone, but changed only on the computer
	From      string `json:"from"`
}

// SavedNote tells the phone the seq of a note that the computer took.
type SavedNote struct {
	SID string `json:"sid"`
	Seq int64  `json:"seq"`
}

// ProjectInfo says which project the notes belong to.
type ProjectInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PhoneResponse is the answer to one sync.
type PhoneResponse struct {
	Project  ProjectInfo `json:"project"`
	Mismatch bool        `json:"mismatch"` // the phone asked for another project: nothing was changed
	Saved    []SavedNote `json:"saved"`
	Rejected []string    `json:"rejected"` // notes that the phone may not edit
	Changed  []NoteOut   `json:"changed"`  // new or edited on the computer after "since", and copies made by a clash
	Deleted  []string    `json:"deleted"`  // sync ids of notes deleted on the computer after "since"
	Seq      int64       `json:"seq"`
}

type syncNote struct {
	ID      int64  `json:"-"`
	SID     string `json:"sid"`
	Draft   int64  `json:"draft"`
	Seq     int64  `json:"seq"`
	Hash    string `json:"hash"`
	Deleted bool   `json:"deleted"`
	From    string `json:"from"`
}

var sidRe = regexp.MustCompile(`^[0-9a-zA-Z-]{8,64}$`)

const (
	maxTitle = 300
	maxText  = 200000
)

func noteHash(title, body string) string {
	sum := sha256.Sum256([]byte(title + "\x00" + body))
	return hex.EncodeToString(sum[:])
}

func newSID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// sqliteTime reads a time that SQLite wrote, or an RFC 3339 time from a phone.
func sqliteTime(s string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339, time.RFC3339Nano} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (s *Store) readSyncNotes() ([]*syncNote, error) {
	docs, err := s.List("syncnote")
	if err != nil {
		return nil, err
	}
	out := make([]*syncNote, 0, len(docs))
	for _, d := range docs {
		n := &syncNote{ID: d.ID}
		if json.Unmarshal([]byte(d.Data), n) == nil {
			out = append(out, n)
		}
	}
	return out, nil
}

func (s *Store) writeSyncNote(n *syncNote) error {
	b, _ := json.Marshal(n)
	saved, err := s.Save(Doc{ID: n.ID, Kind: "syncnote", Data: string(b)})
	n.ID = saved.ID
	return err
}

// nextSeq gives the next number. It is kept in a setting, so each project has its own count.
func (s *Store) nextSeq() (int64, error) {
	docs, err := s.List("setting")
	if err != nil {
		return 0, err
	}
	for _, d := range docs {
		var row struct {
			Key   string  `json:"key"`
			Value float64 `json:"value"`
		}
		if json.Unmarshal([]byte(d.Data), &row) == nil && row.Key == "syncSeq" {
			n := int64(row.Value) + 1
			_, err := s.Save(Doc{ID: d.ID, Kind: "setting", Data: fmt.Sprintf(`{"key":"syncSeq","value":%d}`, n)})
			return n, err
		}
	}
	_, err = s.Save(Doc{Kind: "setting", Data: `{"key":"syncSeq","value":1}`})
	return 1, err
}

func (s *Store) currentSeq() int64 {
	docs, _ := s.List("setting")
	for _, d := range docs {
		var row struct {
			Key   string  `json:"key"`
			Value float64 `json:"value"`
		}
		if json.Unmarshal([]byte(d.Data), &row) == nil && row.Key == "syncSeq" {
			return int64(row.Value)
		}
	}
	return 0
}

// draftFields reads the title and the text of a Notes page, and keeps every other field as it was.
func draftFields(d Doc) (title, body string, all map[string]any) {
	all = map[string]any{}
	_ = json.Unmarshal([]byte(d.Data), &all)
	title, _ = all["title"].(string)
	body, _ = all["body"].(string)
	return
}

func (s *Store) newDraft(title, body string) (Doc, error) {
	order := float64(0)
	docs, _ := s.List("draft")
	for _, d := range docs {
		var f struct {
			Order float64 `json:"order"`
		}
		if json.Unmarshal([]byte(d.Data), &f) == nil && f.Order > order {
			order = f.Order
		}
		if f.Order == 0 && float64(d.ID) > order {
			order = float64(d.ID)
		}
	}
	b, _ := json.Marshal(map[string]any{"title": title, "body": body, "order": order + 1})
	return s.Save(Doc{Kind: "draft", Data: string(b)})
}

func (s *Store) updateDraft(d Doc, title, body string) (Doc, error) {
	_, _, all := draftFields(d)
	all["title"], all["body"] = title, body
	b, _ := json.Marshal(all)
	d.Data = string(b)
	return s.Save(d)
}

// syncScan finds what changed on the computer since the last scan: a new page, an edited page, a deleted page. Each gets the next seq.
func (s *Store) syncScan() error {
	drafts, err := s.List("draft")
	if err != nil {
		return err
	}
	notes, err := s.readSyncNotes()
	if err != nil {
		return err
	}
	byDraft := map[int64]*syncNote{}
	for _, n := range notes {
		if !n.Deleted {
			byDraft[n.Draft] = n
		}
	}
	for _, d := range drafts {
		title, body, _ := draftFields(d)
		h := noteHash(title, body)
		n := byDraft[d.ID]
		switch {
		case n == nil:
			seq, err := s.nextSeq()
			if err != nil {
				return err
			}
			if err := s.writeSyncNote(&syncNote{SID: newSID(), Draft: d.ID, Seq: seq, Hash: h}); err != nil {
				return err
			}
		case n.Hash != h:
			seq, err := s.nextSeq()
			if err != nil {
				return err
			}
			n.Seq, n.Hash = seq, h
			if err := s.writeSyncNote(n); err != nil {
				return err
			}
		}
	}
	exists := map[int64]bool{}
	for _, d := range drafts {
		exists[d.ID] = true
	}
	for _, n := range notes {
		if !n.Deleted && !exists[n.Draft] {
			seq, err := s.nextSeq()
			if err != nil {
				return err
			}
			n.Deleted, n.Seq = true, seq
			if err := s.writeSyncNote(n); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) projectInfo() ProjectInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, _ := s.reg.find(s.reg.Current)
	return ProjectInfo{ID: s.reg.Current, Name: p.Name}
}

// cleanPhoneNote checks a note from a phone before anything is saved.
func cleanPhoneNote(p PhoneNote) (PhoneNote, bool) {
	if !sidRe.MatchString(p.SID) || len(p.Text) > maxText {
		return p, false
	}
	p.Title = strings.TrimSpace(strings.ReplaceAll(p.Title, "\n", " "))
	if len([]rune(p.Title)) > maxTitle {
		p.Title = string([]rune(p.Title)[:maxTitle])
	}
	p.Text = strings.ReplaceAll(p.Text, "\x00", "")
	return p, true
}

// syncNotes does one sync. The phone says what it changed, and the computer answers with what it took and what it changed itself.
func (s *Store) syncNotes(req PhoneRequest) (PhoneResponse, error) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	resp := PhoneResponse{Project: s.projectInfo(), Saved: []SavedNote{}, Rejected: []string{}, Changed: []NoteOut{}, Deleted: []string{}}
	if req.Project != "" && req.Project != resp.Project.ID {
		resp.Mismatch = true
		return resp, nil
	}
	device := strings.TrimSpace(req.DeviceName)
	if device == "" {
		device = "a phone"
	}
	if err := s.syncScan(); err != nil {
		return resp, err
	}
	notes, err := s.readSyncNotes()
	if err != nil {
		return resp, err
	}
	bySID := map[string]*syncNote{}
	for _, n := range notes {
		bySID[n.SID] = n
	}
	drafts, err := s.List("draft")
	if err != nil {
		return resp, err
	}
	draftByID := map[int64]Doc{}
	for _, d := range drafts {
		draftByID[d.ID] = d
	}

	mine := map[string]bool{}  // notes that this request saved: the phone has them already
	force := map[string]bool{} // notes that the phone must get again, whatever "since" says
	take := func(n *syncNote, d Doc, from string) error {
		seq, err := s.nextSeq()
		if err != nil {
			return err
		}
		title, body, _ := draftFields(d)
		n.Seq, n.Hash, n.Deleted = seq, noteHash(title, body), false
		if from != "" {
			n.From = from
		}
		return s.writeSyncNote(n)
	}
	// a new page with its own sync id
	makeCopy := func(title, body, from string) error {
		d, err := s.newDraft(title, body)
		if err != nil {
			return err
		}
		n := &syncNote{SID: newSID(), Draft: d.ID, From: from}
		bySID[n.SID] = n
		return take(n, d, from)
	}

	for _, raw := range req.Notes {
		p, ok := cleanPhoneNote(raw)
		if !ok {
			resp.Rejected = append(resp.Rejected, raw.SID)
			continue
		}
		body := simpleToHTML(p.Text)
		n := bySID[p.SID]
		switch {
		case n == nil: // a note made on the phone
			d, err := s.newDraft(p.Title, body)
			if err != nil {
				return resp, err
			}
			n = &syncNote{SID: p.SID, Draft: d.ID}
			bySID[n.SID] = n
			if err := take(n, d, device); err != nil {
				return resp, err
			}
			mine[n.SID] = true
			resp.Saved = append(resp.Saved, SavedNote{SID: n.SID, Seq: n.Seq})

		case n.Deleted: // deleted on the computer, but edited on the phone: the edit is not lost
			d, err := s.newDraft(p.Title+" (restored from "+device+")", body)
			if err != nil {
				return resp, err
			}
			n.Draft = d.ID
			if err := take(n, d, device); err != nil {
				return resp, err
			}
			mine[n.SID] = true
			force[n.SID] = true // the phone gets the new title
			resp.Saved = append(resp.Saved, SavedNote{SID: n.SID, Seq: n.Seq})

		default:
			d := draftByID[n.Draft]
			oldTitle, oldBody, _ := draftFields(d)
			if noteHash(p.Title, body) == n.Hash { // the same note again, for example after a failed answer: nothing to do
				mine[n.SID] = true
				resp.Saved = append(resp.Saved, SavedNote{SID: n.SID, Seq: n.Seq})
				continue
			}
			if _, editable := htmlToSimple(oldBody); !editable {
				resp.Rejected = append(resp.Rejected, p.SID) // the phone may not edit this one
				force[p.SID] = true
				continue
			}
			if n.Seq <= p.Base { // the computer did not change it since the phone last got it
				nd, err := s.updateDraft(d, p.Title, body)
				if err != nil {
					return resp, err
				}
				if err := take(n, nd, device); err != nil {
					return resp, err
				}
				draftByID[nd.ID] = nd
				mine[n.SID] = true
				resp.Saved = append(resp.Saved, SavedNote{SID: n.SID, Seq: n.Seq})
				continue
			}
			// both sides changed it: the newest stays as the note, the other is kept as a copy
			phoneAt, phoneOK := sqliteTime(p.UpdatedAt)
			deskAt, deskOK := sqliteTime(d.UpdatedAt)
			if deskOK && phoneOK && deskAt.After(phoneAt) {
				if err := makeCopy(p.Title+" (copy from "+device+")", body, device); err != nil {
					return resp, err
				}
				force[p.SID] = true // the phone gets the computer's version back
			} else {
				if err := makeCopy(oldTitle+" (earlier version)", oldBody, ""); err != nil {
					return resp, err
				}
				nd, err := s.updateDraft(d, p.Title, body)
				if err != nil {
					return resp, err
				}
				if err := take(n, nd, device); err != nil {
					return resp, err
				}
				draftByID[nd.ID] = nd
				mine[n.SID] = true
				resp.Saved = append(resp.Saved, SavedNote{SID: n.SID, Seq: n.Seq})
			}
		}
	}

	// what changed on the computer
	drafts, err = s.List("draft")
	if err != nil {
		return resp, err
	}
	for _, d := range drafts {
		draftByID[d.ID] = d
	}
	for _, n := range bySID {
		if mine[n.SID] && !force[n.SID] {
			continue
		}
		if n.Seq <= req.Since && !force[n.SID] {
			continue
		}
		if n.Deleted {
			resp.Deleted = append(resp.Deleted, n.SID)
			continue
		}
		d, ok := draftByID[n.Draft]
		if !ok {
			continue
		}
		title, body, _ := draftFields(d)
		text, editable := htmlToSimple(body)
		out := NoteOut{SID: n.SID, Title: title, Text: text, Seq: n.Seq, Editable: editable, From: n.From}
		if t, ok := sqliteTime(d.UpdatedAt); ok {
			out.UpdatedAt = t.UTC().Format(time.RFC3339)
		}
		resp.Changed = append(resp.Changed, out)
	}
	resp.Seq = s.currentSeq()
	return resp, nil
}
