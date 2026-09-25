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
	if len(ps) != 2 {
		t.Fatalf("want 2 projects, got %d", len(ps))
	}
	if ps[0].Name != "Atlas" || ps[0].Slug != "atl" || ps[0].Kind != "work" {
		t.Errorf("unexpected first project: %+v", ps[0])
	}
	if _, err := s.ProjectBySlug("harbor"); err != nil {
		t.Fatal(err)
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
	nim, _ := s.ProjectBySlug("harbor")
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

func TestUpdateTask(t *testing.T) {
	s := testStore(t)
	s.Seed()
	atl, _ := s.ProjectBySlug("atl")
	task, _ := s.CreateTask(atl.ID, "x", "backlog")
	if err := s.UpdateTask(task.ID, "renamed", "some notes"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Task(task.ID)
	if got.Title != "renamed" || got.Notes != "some notes" {
		t.Fatalf("after update: %+v", got)
	}
}

func TestSearchTasks(t *testing.T) {
	s := testStore(t)
	s.Seed()
	atl, _ := s.ProjectBySlug("atl")
	s.CreateTask(atl.ID, "Ретраї webhook", "backlog")
	n, _ := s.CreateTask(atl.ID, "Webhook алерт", "now")
	d, _ := s.CreateTask(atl.ID, "webhook docs", "backlog")
	s.SetTaskState(d.ID, "done")
	s.CreateTask(atl.ID, "100% unrelated", "backlog")

	got, err := s.SearchTasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[0].ID != n.ID || got[3].State != "done" {
		t.Fatalf("search candidates: %+v", got)
	}
}

func TestReorder(t *testing.T) {
	s := testStore(t)
	s.Seed()
	atl, _ := s.ProjectBySlug("atl")
	a, _ := s.CreateTask(atl.ID, "a", "now")
	b, _ := s.CreateTask(atl.ID, "b", "now")
	c, _ := s.CreateTask(atl.ID, "c", "backlog")
	// c dragged to the top of now, a dragged down to backlog
	if err := s.Reorder([]int64{c.ID, b.ID}, []int64{a.ID}, nil); err != nil {
		t.Fatal(err)
	}
	now, _ := s.TasksByState("now")
	backlog, _ := s.TasksByState("backlog")
	if len(now) != 2 || now[0].ID != c.ID || now[1].ID != b.ID || !now[0].NowSince.Valid {
		t.Fatalf("now: %+v", now)
	}
	if len(backlog) != 1 || backlog[0].ID != a.ID || backlog[0].NowSince.Valid {
		t.Fatalf("backlog: %+v", backlog)
	}
	// b dragged into "Чекаю": keeps its state, gets a default waiting note; back out clears it.
	if err := s.Reorder([]int64{c.ID}, []int64{a.ID}, []int64{b.ID}); err != nil {
		t.Fatal(err)
	}
	w, _ := s.WaitingTasks()
	if len(w) != 1 || w[0].ID != b.ID || w[0].State != "now" || w[0].Waiting != "чекаю" {
		t.Fatalf("waiting via drag: %+v", w)
	}
	s.Reorder([]int64{c.ID, b.ID}, []int64{a.ID}, nil)
	if w, _ := s.WaitingTasks(); len(w) != 0 {
		t.Error("drag out of waiting must clear it")
	}
}

func TestDoneBetween(t *testing.T) {
	s := testStore(t)
	s.Seed()
	atl, _ := s.ProjectBySlug("atl")
	a, _ := s.CreateTask(atl.ID, "old", "backlog")
	b, _ := s.CreateTask(atl.ID, "new", "backlog")
	s.SetTaskState(a.ID, "done")
	s.SetTaskState(b.ID, "done")
	s.db.Exec(`update tasks set done_at = '2026-08-01 10:00:00' where id = ?`, a.ID)
	got, err := s.DoneBetween("2026-08-01 00:00:00", "2026-08-02 00:00:00")
	if err != nil || len(got) != 1 || got[0].ID != a.ID {
		t.Fatalf("got %v %+v", err, got)
	}
	if got, _ := s.DoneBetween("2000-01-01 00:00:00", "2100-01-01 00:00:00"); len(got) != 2 || got[0].ID != b.ID {
		t.Fatalf("order: %+v", got)
	}
}

func TestProjectsCRUD(t *testing.T) {
	s := testStore(t)
	s.Seed()
	p, err := s.CreateProject("newpet", "newpet", "pet", "#8A6A30")
	if err != nil {
		t.Fatal(err)
	}
	p.Name, p.Color, p.Kind = "New Pet", "#4E6B8C", "work"
	if err := s.UpdateProject(p); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Project(p.ID)
	if got.Name != "New Pet" || got.Color != "#4E6B8C" || got.Kind != "work" {
		t.Fatalf("after update: %+v", got)
	}
	s.CreateTask(p.ID, "x", "backlog")
	if err := s.DeleteProject(p.ID); err != ErrHasTasks {
		t.Errorf("delete with tasks: %v", err)
	}
	if _, err := s.CreateProject("Atlas", "dup", "work", "#000"); err == nil {
		t.Error("duplicate name accepted")
	}
	if err := s.UpdateProject(Project{ID: 999}); err != ErrNotFound {
		t.Errorf("missing: %v", err)
	}
}

func TestChecklist(t *testing.T) {
	s := testStore(t)
	s.Seed()
	atl, _ := s.ProjectBySlug("atl")
	a, _ := s.AddChecklistItem(atl.ID, "before", "Merge release branch")
	b, _ := s.AddChecklistItem(atl.ID, "before", "Run migrate check")
	s.AddChecklistItem(atl.ID, "after", "Check Sentry")
	if err := s.UpdateChecklistItem(b.ID, "Run migrations", "after"); err != nil {
		t.Fatal(err)
	}
	task, _ := s.CreateTask(atl.ID, "Rotate keys", "backlog")
	s.SetChecklistItemTask(b.ID, task.ID)
	s.ToggleChecklistItem(a.ID)
	items, _ := s.Checklist(atl.ID)
	if len(items) != 3 || !items[0].Done || items[1].Phase != "after" || items[1].TaskTitle.String != "Rotate keys" {
		t.Fatalf("items: %+v", items)
	}
	if it, err := s.ChecklistItem(b.ID); err != nil || it.Title != "Run migrations" || it.TaskState.String != "backlog" {
		t.Fatalf("item: %+v %v", it, err)
	}
	if pid, _ := s.ChecklistItemProject(a.ID); pid != atl.ID {
		t.Error("ChecklistItemProject wrong")
	}
	prog, _ := s.ChecklistProgress()
	if prog[atl.ID] != (Progress{Done: 1, Total: 3}) {
		t.Fatalf("progress: %+v", prog[atl.ID])
	}
	// Release: everything moves to history, the list is empty again.
	if err := s.MarkReleased(atl.ID); err != nil {
		t.Fatal(err)
	}
	if items, _ := s.Checklist(atl.ID); len(items) != 0 {
		t.Error("release must empty the checklist")
	}
	hist, _ := s.ReleaseHistory(atl.ID, 5)
	if len(hist) != 1 || len(hist[0].Items) != 3 || !hist[0].Items[0].Done || hist[0].Items[1].TaskTitle.String != "Rotate keys" {
		t.Fatalf("history: %+v", hist)
	}
	s.DeleteTask(task.ID)
	if hist, _ := s.ReleaseHistory(atl.ID, 5); hist[0].Items[1].TaskTitle.Valid {
		t.Error("deleted task must unlink from history")
	}
	c, _ := s.AddChecklistItem(atl.ID, "before", "x")
	s.DeleteChecklistItem(c.ID)
	if items, _ := s.Checklist(atl.ID); len(items) != 0 {
		t.Error("delete failed")
	}
}

func TestBackup(t *testing.T) {
	s := testStore(t)
	s.Seed()
	dir := filepath.Join(t.TempDir(), "backups")
	if err := s.Backup(dir, 30); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "dayboard-*.db"))
	if len(files) != 1 {
		t.Fatalf("backup files: %v", files)
	}
	copyDB, err := Open(files[0])
	if err != nil {
		t.Fatal(err)
	}
	defer copyDB.Close()
	if ps, _ := copyDB.Projects(); len(ps) != 2 {
		t.Errorf("backup has %d projects", len(ps))
	}
}
