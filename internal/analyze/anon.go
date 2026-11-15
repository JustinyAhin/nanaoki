package analyze

import (
	"database/sql"

	"github.com/JustinyAhin/nanaoki/internal/anon"
)

// KnownProjects lists the project names in the store, so that --anon can also find them inside error text.
func KnownProjects(db *sql.DB) ([]string, error) {
	rs, err := db.Query(`select distinct project from events where project is not null`)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	var out []string
	for rs.Next() {
		var p string
		if err := rs.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rs.Err()
}

func (p *Pointer) anon(a *anon.Anon) {
	p.Project, p.Session, p.RawFile = a.Project(p.Project), a.Session(p.Session), a.Text(p.RawFile)
}

func anonPtrs(ps []Pointer, a *anon.Anon) {
	for i := range ps {
		ps[i].anon(a)
	}
}

// Anonymize replaces project names, session ids, home paths and private data in the report, in the order of
// the text report so that the aliases read project-a, project-b, ... from the top.
func (r *Report) Anonymize(a *anon.Anon) {
	r.Filter.Project = a.Project(r.Filter.Project)
	for i, w := range r.Coverage.Warnings {
		r.Coverage.Warnings[i] = a.Text(w)
	}
	for i := range r.Repeated {
		it := &r.Repeated[i]
		it.Key, it.Label = a.Text(it.Key), a.Text(it.Label)
		anonPtrs(it.Examples, a)
	}
	for i := range r.RAF.Top {
		it := &r.RAF.Top[i]
		it.Key, it.Label = a.Text(it.Key), a.Text(it.Label)
		it.EarliestFix.anon(a)
		if it.FixEvent != nil {
			it.FixEvent.anon(a)
		}
		anonPtrs(it.Examples, a)
	}
	for i := range r.Rules {
		it := &r.Rules[i]
		it.Key, it.Label = a.Text(it.Key), a.Text(it.Label)
		anonPtrs(it.Examples, a)
	}
	for i := range r.Suggestions {
		s := &r.Suggestions[i]
		s.For, s.Text, s.Target = a.Text(s.For), a.Text(s.Text), a.Text(s.Target)
	}
}
