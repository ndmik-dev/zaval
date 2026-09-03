package web

import (
	"testing"

	"github.com/ndmik-dev/zaval/internal/store"
)

var testProjects = []store.Project{
	{ID: 1, Name: "Atlas", Slug: "atl"},
	{ID: 2, Name: "Nimbus", Slug: "nim"},
	{ID: 3, Name: "harbor", Slug: "harbor"},
}

func TestParseQuick(t *testing.T) {
	q := parseQuick("Полагодити ретраї #atl ! https://atlas.atlassian.net/browse/ATL-412", testProjects)
	if q.Title != "Полагодити ретраї" || q.Project == nil || q.Project.Slug != "atl" || !q.Now || len(q.Links) != 1 {
		t.Fatalf("got %+v", q)
	}
	q = parseQuick("Субтитри #ni", testProjects)
	if q.Project == nil || q.Project.Slug != "nim" || q.Now || q.Title != "Субтитри" {
		t.Fatalf("name prefix: %+v", q)
	}
	q = parseQuick("Тег #nomatch лишається", testProjects)
	if q.Project != nil || q.Title != "Тег #nomatch лишається" {
		t.Fatalf("unknown tag: %+v", q)
	}
	q = parseQuick("#sky !зараз", testProjects)
	if q.Project == nil || q.Project.Slug != "harbor" || !q.Now || q.Title != "" {
		t.Fatalf("tags only: %+v", q)
	}
}
