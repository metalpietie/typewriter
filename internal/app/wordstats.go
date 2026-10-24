package app

import (
	"database/sql"
	"encoding/json"
)

// Words written per day in every project, for the Statistics screen ("All projects"). Each project counts its own days
// (the wordday rows, see progress.ts); this reads them all, so the screen can add them up.

// WordDay is the count of one day in one project. Books splits it by book (book id → words); days from before the
// split was kept have no Books.
type WordDay struct {
	Date  string         `json:"date"`
	Words int            `json:"words"`
	Books map[string]int `json:"books,omitempty"`
}

// ProjectDays is the writing days of one project.
type ProjectDays struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Group   string    `json:"group"`
	Current bool      `json:"current"`
	Problem string    `json:"problem"` // set when the data file of this project could not be read
	Days    []WordDay `json:"days"`
}

// WordDays returns the writing days of every project.
func (s *Store) WordDays() ([]ProjectDays, error) {
	s.mu.RLock()
	reg := s.reg
	s.mu.RUnlock()
	out := []ProjectDays{}
	for _, p := range reg.Projects {
		pd := ProjectDays{ID: p.ID, Name: p.Name, Group: p.Group, Current: p.ID == reg.Current, Days: []WordDay{}}
		err := s.withProject(p.ID, false, func(d *sql.DB) error {
			days, err := readWordDays(d)
			pd.Days = days
			return err
		})
		if err != nil {
			pd.Problem = err.Error()
		}
		out = append(out, pd)
	}
	return out, nil
}

func readWordDays(d *sql.DB) ([]WordDay, error) {
	r, err := d.Query(`SELECT data FROM docs WHERE kind = 'wordday' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	out := []WordDay{}
	for r.Next() {
		var data string
		if err := r.Scan(&data); err != nil {
			return nil, err
		}
		var w WordDay
		if json.Unmarshal([]byte(data), &w) != nil || w.Date == "" {
			continue // a damaged row counts as nothing
		}
		out = append(out, w)
	}
	return out, r.Err()
}
