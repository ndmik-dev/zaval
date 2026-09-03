package web

import (
	"strings"

	"github.com/ndmik-dev/zaval/internal/links"
	"github.com/ndmik-dev/zaval/internal/store"
)

// quickInput is what the ⌘K line parses into:
//
//	"Полагодити ретраї #atl ! https://…/ATL-412"
//
// #slug picks the project by slug or name prefix, a lone "!" (or "!зараз")
// sends the task straight to "Зараз", links become chips.
type quickInput struct {
	Title   string
	Project *store.Project
	Now     bool
	Links   []links.Link
}

func parseQuick(raw string, projects []store.Project) quickInput {
	var q quickInput
	var rest []string
	for _, tok := range strings.Fields(raw) {
		switch {
		case tok == "!" || strings.EqualFold(tok, "!зараз") || strings.EqualFold(tok, "!now"):
			q.Now = true
		case strings.HasPrefix(tok, "#") && len(tok) > 1:
			if p := matchProject(tok[1:], projects); p != nil {
				q.Project = p
				continue
			}
			rest = append(rest, tok)
		default:
			rest = append(rest, tok)
		}
	}
	q.Title, q.Links = links.Parse(strings.Join(rest, " "))
	return q
}

func matchProject(prefix string, projects []store.Project) *store.Project {
	prefix = strings.ToLower(prefix)
	for i := range projects {
		p := &projects[i]
		if strings.HasPrefix(strings.ToLower(p.Slug), prefix) || strings.HasPrefix(strings.ToLower(p.Name), prefix) {
			return p
		}
	}
	return nil
}
