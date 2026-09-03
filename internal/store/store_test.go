package store

import (
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestMigrateAndSeed(t *testing.T) {
	s := testStore(t)
	if err := s.Seed(); err != nil {
		t.Fatal(err)
	}
	if err := s.Seed(); err != nil {
		t.Fatal("second seed:", err)
	}
	ps, err := s.Projects()
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 7 {
		t.Fatalf("want 7 projects, got %d", len(ps))
	}
	if ps[0].Name != "Atlas" || ps[0].Slug != "atl" || !ps[0].OnBoard {
		t.Errorf("unexpected first project: %+v", ps[0])
	}
	p, err := s.ProjectBySlug("tide")
	if err != nil {
		t.Fatal(err)
	}
	if p.OnBoard {
		t.Error("tide should be hidden from the board")
	}
	// Re-opening must not re-run migrations.
	if _, err := Open(filepath.Join(t.TempDir(), "other.db")); err != nil {
		t.Fatal(err)
	}
}

func TestTasks(t *testing.T) {
	s := testStore(t)
	if err := s.Seed(); err != nil {
		t.Fatal(err)
	}
	atl, _ := s.ProjectBySlug("atl")
	nim, _ := s.ProjectBySlug("nim")
	a, err := s.CreateTask(atl.ID, "first", "now")
	if err != nil {
		t.Fatal(err)
	}
	if !a.NowSince.Valid {
		t.Error("task created in now must have now_since")
	}
	if _, err := s.CreateTask(nim.ID, "second", "backlog"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTask(atl.ID, "third", "backlog"); err != nil {
		t.Fatal(err)
	}
	now, _ := s.TasksByState("now")
	backlog, _ := s.TasksByState("backlog")
	if len(now) != 1 || len(backlog) != 2 {
		t.Fatalf("now=%d backlog=%d", len(now), len(backlog))
	}
	if backlog[0].Title != "second" || backlog[1].Title != "third" {
		t.Errorf("backlog order wrong: %s, %s", backlog[0].Title, backlog[1].Title)
	}
	if now[0].Project.Name != "Atlas" {
		t.Errorf("project not joined: %+v", now[0].Project)
	}
	counts, _ := s.ProjectCounts()
	if counts[atl.ID] != (Counts{Now: 1, Backlog: 1}) || counts[nim.ID] != (Counts{Backlog: 1}) {
		t.Errorf("counts: %+v", counts)
	}
}

func TestSetTaskState(t *testing.T) {
	s := testStore(t)
	s.Seed()
	atl, _ := s.ProjectBySlug("atl")
	task, _ := s.CreateTask(atl.ID, "x", "backlog")

	if err := s.SetTaskState(task.ID, "now"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Task(task.ID)
	if got.State != "now" || !got.NowSince.Valid {
		t.Fatalf("after now: %+v", got)
	}
	since := got.NowSince.String

	// Done keeps now_since (history), sets done_at.
	s.SetTaskState(task.ID, "done")
	got, _ = s.Task(task.ID)
	if got.State != "done" || !got.DoneAt.Valid || got.NowSince.String != since {
		t.Fatalf("after done: %+v", got)
	}
	today, _ := s.DoneToday()
	if len(today) != 1 {
		t.Fatalf("done today: %d", len(today))
	}

	// Back to backlog clears both.
	s.SetTaskState(task.ID, "backlog")
	got, _ = s.Task(task.ID)
	if got.State != "backlog" || got.NowSince.Valid || got.DoneAt.Valid {
		t.Fatalf("after backlog: %+v", got)
	}
	if err := s.SetTaskState(999, "now"); err != ErrNotFound {
		t.Errorf("missing task: %v", err)
	}
	if err := s.SetTaskState(task.ID, "weird"); err == nil {
		t.Error("bad state accepted")
	}
	if err := s.DeleteTask(task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Task(task.ID); err != ErrNotFound {
		t.Errorf("deleted task still found: %v", err)
	}
}

func TestLinks(t *testing.T) {
	s := testStore(t)
	s.Seed()
	atl, _ := s.ProjectBySlug("atl")
	task, _ := s.CreateTask(atl.ID, "x", "backlog")
	if _, err := s.AddLink(task.ID, "https://a", "jira", "ATL-1", ""); err != nil {
		t.Fatal(err)
	}
	l2, _ := s.AddLink(task.ID, "https://b", "other", "b", "")
	got, _ := s.Task(task.ID)
	if len(got.Links) != 2 || got.Links[0].Label != "ATL-1" {
		t.Fatalf("links: %+v", got.Links)
	}
	s.DeleteLink(l2.ID)
	got, _ = s.Task(task.ID)
	if len(got.Links) != 1 {
		t.Fatalf("after delete: %+v", got.Links)
	}
	s.DeleteTask(task.ID)
	var n int
	s.db.QueryRow(`select count(*) from task_links`).Scan(&n)
	if n != 0 {
		t.Error("links must cascade on task delete")
	}
}

func TestStepsAndUpdate(t *testing.T) {
	s := testStore(t)
	s.Seed()
	atl, _ := s.ProjectBySlug("atl")
	task, _ := s.CreateTask(atl.ID, "x", "backlog")
	a, _ := s.AddStep(task.ID, "one")
	s.AddStep(task.ID, "two")
	s.ToggleStep(a.ID)
	got, _ := s.Task(task.ID)
	if len(got.Steps) != 2 || !got.Steps[0].Done || got.StepsDone != 1 || got.StepsTotal != 2 {
		t.Fatalf("steps: %+v done=%d total=%d", got.Steps, got.StepsDone, got.StepsTotal)
	}
	if tid, _ := s.StepTask(a.ID); tid != task.ID {
		t.Error("StepTask wrong")
	}
	s.DeleteStep(a.ID)
	if err := s.UpdateTask(task.ID, "renamed", "some notes"); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Task(task.ID)
	if got.Title != "renamed" || got.Notes != "some notes" || len(got.Steps) != 1 {
		t.Fatalf("after update: %+v", got)
	}
}
