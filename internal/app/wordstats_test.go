package app

import (
	"database/sql"
	"testing"
)

func TestWordDaysReadsEveryProject(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	other, _ := s.AddProject("Practice")
	mustSave(t, s, "wordday", `{"date":"2026-10-20","words":300,"books":{"4":200,"5":100}}`)
	mustSave(t, s, "wordday", `{"date":"2026-10-21","words":-20}`) // cut more than written: the screen makes it 0
	s.withProject(other, false, func(d *sql.DB) error {
		_, err := d.Exec(`INSERT INTO docs(kind, data) VALUES('wordday', '{"date":"2026-10-20","words":50}'), ('wordday', 'not json')`)
		return err
	})

	all, err := s.WordDays()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("every project: %+v", all)
	}
	by := map[string]ProjectDays{}
	for _, p := range all {
		by[p.ID] = p
	}
	mine := by[defaultProject]
	if !mine.Current || len(mine.Days) != 2 || mine.Days[0].Words != 300 || mine.Days[0].Books["4"] != 200 || mine.Days[1].Words != -20 {
		t.Fatalf("the open project: %+v", mine)
	}
	if p := by[other]; p.Current || p.Name != "Practice" || len(p.Days) != 1 || p.Days[0].Words != 50 || p.Days[0].Books != nil {
		t.Fatalf("the other project (a damaged row counts as nothing): %+v", p)
	}
}
