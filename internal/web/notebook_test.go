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

func TestParseLine(t *testing.T) {
	in := parseLine("Полагодити ретраї #atl https://atlas.atlassian.net/browse/ATL-412", testProjects)
	if in.Title != "Полагодити ретраї" || in.Project == nil || in.Project.Slug != "atl" || len(in.Links) != 1 || in.Waiting != "" {
		t.Fatalf("got %+v", in)
	}
	in = parseLine("Оновити субтитри ? відповіді від Марти #ni", testProjects)
	if in.Title != "Оновити субтитри" || in.Waiting != "відповіді від Марти" || in.Project == nil || in.Project.Slug != "nim" {
		t.Fatalf("waiting: %+v", in)
	}
	in = parseLine("Що робити? #atl", testProjects)
	if in.Title != "Що робити?" || in.Waiting != "" {
		t.Fatalf("attached question mark is not a wait: %+v", in)
	}
	in = parseLine("Тег #nomatch лишається ?", testProjects)
	if in.Project != nil || in.Title != "Тег #nomatch лишається" || in.Waiting != "відповіді" {
		t.Fatalf("unknown tag, bare wait: %+v", in)
	}
	in = parseLine("Оцінка по білінгу → беклог", testProjects)
	if !in.ToBacklog || in.Title != "Оцінка по білінгу" {
		t.Fatalf("to backlog: %+v", in)
	}
	in = parseLine("Оцінка -> сьогодні", testProjects)
	if !in.ToToday || in.Title != "Оцінка" {
		t.Fatalf("to today: %+v", in)
	}
	in = parseLine("/release #atl", testProjects)
	if !in.Release || in.Project == nil || in.Project.Slug != "atl" || in.Title != "" {
		t.Fatalf("release: %+v", in)
	}
}

func TestRawTextRoundTrip(t *testing.T) {
	task := store.Task{Title: "Полагодити ретраї", Waiting: "ревʼю", Project: store.Project{Slug: "atl"},
		Links: []store.Link{{URL: "https://github.com/ndmik/zaval/pull/128"}}}
	raw := rawText(task)
	if raw != "Полагодити ретраї ? ревʼю #atl https://github.com/ndmik/zaval/pull/128" {
		t.Fatalf("raw: %q", raw)
	}
	in := parseLine(raw, testProjects)
	if in.Title != task.Title || in.Waiting != task.Waiting || in.Project.Slug != "atl" || len(in.Links) != 1 || in.Links[0].URL != task.Links[0].URL {
		t.Fatalf("round trip: %+v", in)
	}
}
