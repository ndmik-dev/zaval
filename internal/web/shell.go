package web

import "github.com/ndmik-dev/zaval/internal/store"

type projectItem struct {
	store.Project
	store.Counts
}

// shell is what the layout needs on every page.
type shell struct {
	Title    string
	Nav      string
	Filter   string
	Projects []projectItem
	Ctx      map[string]string // page parameters every mutation carries back
	AuthOn   bool
	Undo     *undo // toast after a reversible action
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
