package web

import "time"

// shell is what the layout and rail need on every page.
type shell struct {
	Title       string
	Nav         string
	Filter      string
	Projects    []projectItem
	HiddenCount int
	Open        *taskRow          // task shown in the drawer, if any
	Ctx         map[string]string // hidden fields every drawer request carries back (page, filters)
	AuthOn      bool
}

func (s *Server) shell(title, nav, filter string) (shell, error) {
	sh := shell{Title: title, Nav: nav, Filter: filter, AuthOn: s.auth.enabled()}
	projects, err := s.store.Projects()
	if err != nil {
		return sh, err
	}
	counts, err := s.store.ProjectCounts()
	if err != nil {
		return sh, err
	}
	for _, p := range projects {
		if !p.OnBoard {
			sh.HiddenCount++
			continue
		}
		sh.Projects = append(sh.Projects, projectItem{p, counts[p.ID]})
	}
	return sh, nil
}

func (sh shell) work() []projectItem {
	var out []projectItem
	for _, p := range sh.Projects {
		if p.Kind == "work" {
			out = append(out, p)
		}
	}
	return out
}

// openTask loads the task for the drawer; a missing id just leaves it closed.
func (s *Server) openTask(id int64, now time.Time) *taskRow {
	if id == 0 {
		return nil
	}
	t, err := s.store.Task(id)
	if err != nil {
		return nil
	}
	row := taskRow{Task: t}
	if t.State == "now" && t.NowSince.Valid {
		row.Age = ageDays(t.NowSince.String, now)
	}
	return &row
}
