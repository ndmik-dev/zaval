package web

// shell is what the layout and rail need on every page.
type shell struct {
	Title       string
	Nav         string
	Filter      string
	Projects    []projectItem
	HiddenCount int
	Open        *taskRow
}

func (s *Server) shell(title, nav, filter string) (shell, error) {
	sh := shell{Title: title, Nav: nav, Filter: filter}
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
