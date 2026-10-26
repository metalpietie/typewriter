package app

import "testing"

func TestBibleShowsOwnFields(t *testing.T) {
	s, err := OpenStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shut)
	mustSave(t, s, "character", `{"name":"Mara","fields":[{"k":"Magic","v":"brass"},{"k":"Rank","v":""},{"k":"","v":"no name"},{"k":"Odd"}]}`)
	mustSave(t, s, "location", `{"name":"Brassford","fields":[{"k":"Ruler","v":"the guild"}]}`)
	mustSave(t, s, "faction", `{"name":"Clockmakers","fields":[{"k":"Colours","v":"gold"}]}`)
	mustSave(t, s, "item", `{"name":"Key","fields":[{"k":"Metal","v":"brass"}]}`)
	got := map[string]string{}
	for _, b := range (&Exporter{store: s}).bible() {
		if b.kind == "kv" {
			got[b.label] = b.text
		}
	}
	for k, v := range map[string]string{"Magic": "brass", "Field": "no name", "Ruler": "the guild", "Colours": "gold", "Metal": "brass"} {
		if got[k] != v {
			t.Errorf("%s: got %q, want %q", k, got[k], v)
		}
	}
	if _, ok := got["Rank"]; ok {
		t.Error("an empty field is left out")
	}
	if _, ok := got["Odd"]; ok {
		t.Error("a field with no value is left out, not printed as <nil>")
	}
}
