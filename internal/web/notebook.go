package web

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ndmik-dev/zaval/internal/links"
	"github.com/ndmik-dev/zaval/internal/store"
)

// The notebook: a page per day, a line per task. A line is edited as one
// string — "Полагодити ретраї #atl ? ревʼю https://…" — and parsed on save.
// The stored model is unchanged: an open line is a task in "now", a struck
// line is "done" on the day it was closed, the backlog page is "backlog".

// releasePrefix marks a line that unfolds a project's release checklist.
const releasePrefix = "/release"

// lineInput is what one raw line parses into.
type lineInput struct {
	Title     string
	Project   *store.Project
	Links     []links.Link
	Waiting   string // what the line waits for; "" = not waiting
	ToBacklog bool   // "→ беклог" at the end
	ToToday   bool   // "→ сьогодні" at the end
	Release   bool   // "/release #tag": the line is a release checklist
}

var trailingCommand = regexp.MustCompile(`(?i)\s*(?:→|->)\s*(беклог|backlog|сьогодні|today)\s*$`)

// parseLine reads the notebook syntax: #tag picks the project, a standalone
// "?" starts the waiting note, links become chips, "→ беклог" moves the line.
func parseLine(raw string, projects []store.Project) lineInput {
	var in lineInput
	raw = strings.TrimSpace(raw)
	if m := trailingCommand.FindStringSubmatch(raw); m != nil {
		switch strings.ToLower(m[1]) {
		case "беклог", "backlog":
			in.ToBacklog = true
		default:
			in.ToToday = true
		}
		raw = strings.TrimSpace(raw[:len(raw)-len(m[0])])
	}
	fields := strings.Fields(raw)
	if len(fields) > 0 && (fields[0] == releasePrefix || fields[0] == "/реліз") {
		in.Release = true
		fields = fields[1:]
	}
	var title, waiting, urls []string
	inWait := false
	for _, tok := range fields {
		switch {
		case tok == "?":
			inWait = true
		case strings.HasPrefix(tok, "http://") || strings.HasPrefix(tok, "https://"):
			urls = append(urls, tok) // a link is a link wherever it sits in the line
		case strings.HasPrefix(tok, "#") && len(tok) > 1 && matchProject(tok[1:], projects) != nil:
			in.Project = matchProject(tok[1:], projects)
		case inWait:
			waiting = append(waiting, tok)
		default:
			title = append(title, tok)
		}
	}
	in.Title, in.Links = links.Parse(strings.Join(append(title, urls...), " "))
	if inWait {
		in.Waiting = strings.Join(waiting, " ")
		if in.Waiting == "" {
			in.Waiting = "відповіді"
		}
	}
	return in
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

// rawText is the line as the editor shows it: what parseLine would read back
// into the same task.
func rawText(t store.Task) string {
	if isRelease(t) {
		return releasePrefix + " #" + t.Project.Slug
	}
	parts := []string{t.Title}
	if t.Waiting != "" {
		parts = append(parts, "?", t.Waiting)
	}
	parts = append(parts, "#"+t.Project.Slug)
	for _, l := range t.Links {
		parts = append(parts, l.URL)
	}
	return strings.Join(parts, " ")
}

func isRelease(t store.Task) bool {
	return t.State != "done" && strings.HasPrefix(t.Title, releasePrefix)
}

// line is one row of the notebook.
type line struct {
	store.Task
	Raw     string
	Age     int // days waiting, shown from the second day
	Release *releaseBlock
}

// releaseBlock is a project's checklist unfolded under a "/release" line.
type releaseBlock struct {
	Items       []store.ChecklistItem
	Done, Total int
}

type dayBlock struct {
	Key   string
	Label string
	Lines []line
}

type notebookData struct {
	shell
	Weekday  string
	DayLabel string
	Today    []line     // open lines, in priority order
	Struck   []line     // closed today
	Days     []dayBlock // earlier days, newest first
	NewTag   string     // the project a new line takes without a #tag
}

type backlogBand struct {
	store.Project
	Lines []line
}

type backlogData struct {
	shell
	Bands  []backlogBand
	Total  int
	NewTag string
}

const feedSpan = 45 // days of struck lines under today

func (s *Server) notebook(w http.ResponseWriter, r *http.Request) {
	s.respondNotebook(w, r)
}

func (s *Server) backlog(w http.ResponseWriter, r *http.Request) {
	s.respondBacklog(w, r)
}

// respondNotebook re-renders the page a mutation came from: the notebook by
// default, the backlog or the releases page when the context says so.
func (s *Server) respondNotebook(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	switch r.FormValue("page") {
	case "backlog":
		s.respondBacklog(w, r)
		return
	case "releases":
		s.respondReleases(w, r)
		return
	}
	if r.Header.Get("HX-Request") == "" && r.Method != http.MethodGet {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	d, err := s.notebookData(time.Now(), preferredProject(r))
	if err != nil {
		s.fail(w, "notebook", err)
		return
	}
	d.Undo = undoFrom(r)
	if r.Header.Get("HX-Request") != "" {
		s.renderPart(w, "notebook", "app", d)
		return
	}
	s.render(w, "notebook", d)
}

func (s *Server) respondBacklog(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "" && r.Method != http.MethodGet {
		http.Redirect(w, r, "/backlog", http.StatusSeeOther)
		return
	}
	sh, err := s.shell("Беклог", "backlog", "")
	if err != nil {
		s.fail(w, "backlog", err)
		return
	}
	d := backlogData{shell: sh, NewTag: s.newTag(sh, "backlog", preferredProject(r))}
	d.Ctx = map[string]string{"page": "backlog"}
	d.Undo = undoFrom(r)
	tasks, err := s.store.TasksByState("backlog")
	if err != nil {
		s.fail(w, "backlog", err)
		return
	}
	now := time.Now()
	byProject := map[int64][]line{}
	for _, t := range tasks {
		l, err := s.line(t, now)
		if err != nil {
			s.fail(w, "backlog", err)
			return
		}
		byProject[t.ProjectID] = append(byProject[t.ProjectID], l)
		d.Total++
	}
	for _, p := range sh.Projects {
		if ls := byProject[p.ID]; len(ls) > 0 {
			d.Bands = append(d.Bands, backlogBand{p.Project, ls})
		}
	}
	if r.Header.Get("HX-Request") != "" {
		s.renderPart(w, "backlog", "app", d)
		return
	}
	s.render(w, "backlog", d)
}

func (s *Server) notebookData(now time.Time, remembered string) (notebookData, error) {
	sh, err := s.shell("Сьогодні", "today", "")
	if err != nil {
		return notebookData{}, err
	}
	d := notebookData{shell: sh, NewTag: s.newTag(sh, "now", remembered)}
	d.Ctx = map[string]string{"page": "today"}
	d.Weekday = ukWeekdays[now.Weekday()] + ","
	d.DayLabel = fmt.Sprintf("%d %s", now.Day(), ukMonths[now.Month()-1])

	open, err := s.store.TasksByState("now")
	if err != nil {
		return d, err
	}
	for _, t := range open {
		l, err := s.line(t, now)
		if err != nil {
			return d, err
		}
		d.Today = append(d.Today, l)
	}
	today := dayStart(now)
	done, err := s.store.DoneBetween(utc(today.AddDate(0, 0, -feedSpan)), utc(today.AddDate(0, 0, 1)))
	if err != nil {
		return d, err
	}
	for _, t := range done { // newest first
		day := localDay(t.DoneAt.String, now.Location())
		l := line{Task: t, Raw: rawText(t)}
		if sameDay(day, today) {
			d.Struck = append(d.Struck, l)
			continue
		}
		key := day.Format("2006-01-02")
		if len(d.Days) == 0 || d.Days[len(d.Days)-1].Key != key {
			d.Days = append(d.Days, dayBlock{Key: key, Label: ukDate(day)})
		}
		d.Days[len(d.Days)-1].Lines = append(d.Days[len(d.Days)-1].Lines, l)
	}
	return d, nil
}

// line dresses a task for the page; a "/release" line loads its checklist.
func (s *Server) line(t store.Task, now time.Time) (line, error) {
	l := line{Task: t, Raw: rawText(t)}
	if t.Waiting != "" && t.WaitingSince.Valid {
		l.Age = ageDays(t.WaitingSince.String, now)
	}
	if isRelease(t) {
		items, err := s.store.Checklist(t.ProjectID)
		if err != nil {
			return l, err
		}
		b := &releaseBlock{Items: items, Total: len(items)}
		for _, it := range items {
			if it.Done {
				b.Done++
			}
		}
		l.Release = b
	}
	return l, nil
}

// createLine turns the new-line input into a task. Without a #tag the line
// takes the project of the line above it, then the last one used.
func (s *Server) createLine(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	projects, err := s.store.Projects()
	if err != nil {
		s.fail(w, "projects", err)
		return
	}
	in := parseLine(r.FormValue("raw"), projects)
	if in.Title == "" && !in.Release {
		s.respondNotebook(w, r)
		return
	}
	state := "now"
	if r.FormValue("page") == "backlog" || in.ToBacklog {
		state = "backlog"
	}
	if in.Project == nil {
		in.Project = s.projectAbove(projects, state, preferredProject(r))
	}
	if in.Project == nil {
		s.respondNotebook(w, r)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "lastp", Value: in.Project.Slug, Path: "/", MaxAge: 86400 * 365, SameSite: http.SameSiteLaxMode})
	title := in.Title
	if in.Release {
		title = releasePrefix
	}
	task, err := s.store.CreateTask(in.Project.ID, title, state)
	if err != nil {
		s.fail(w, "create line", err)
		return
	}
	if err := s.applyLine(task, in); err != nil {
		s.fail(w, "create line", err)
		return
	}
	s.respondNotebook(w, r)
}

// updateLine re-parses an edited line. An emptied line is deleted.
func (s *Server) updateLine(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	task, err := s.store.Task(pathID(r))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	projects, err := s.store.Projects()
	if err != nil {
		s.fail(w, "projects", err)
		return
	}
	in := parseLine(r.FormValue("raw"), projects)
	if in.Title == "" && !in.Release {
		if err := s.store.DeleteTask(task.ID); err != nil {
			s.fail(w, "delete line", err)
			return
		}
		s.remember(task)
		offerUndo(r, "Рядок видалено", "restore", task.ID, "")
		s.respondNotebook(w, r)
		return
	}
	title := in.Title
	if in.Release {
		title = releasePrefix
	}
	if title != task.Title {
		if err := s.store.UpdateTask(task.ID, title, task.Notes); err != nil {
			s.fail(w, "update line", err)
			return
		}
	}
	if in.Project != nil && in.Project.ID != task.ProjectID {
		if err := s.store.SetProject(task.ID, in.Project.ID); err != nil {
			s.fail(w, "update line", err)
			return
		}
	}
	if err := s.applyLine(task, in); err != nil {
		s.fail(w, "update line", err)
		return
	}
	s.respondNotebook(w, r)
}

// applyLine writes the parts of a parsed line that live outside the title:
// links (replaced as a set), the waiting note, and a move between pages.
func (s *Server) applyLine(task store.Task, in lineInput) error {
	have := map[string]bool{}
	for _, l := range task.Links {
		have[l.URL] = true
	}
	want := map[string]bool{}
	for _, l := range in.Links {
		want[l.URL] = true
		if !have[l.URL] {
			if _, err := s.store.AddLink(task.ID, l.URL, l.Kind, l.Label, l.Meta); err != nil {
				return err
			}
		}
	}
	for _, l := range task.Links {
		if !want[l.URL] {
			if err := s.store.DeleteLink(l.ID); err != nil {
				return err
			}
		}
	}
	if in.Waiting != task.Waiting {
		if err := s.store.SetWaiting(task.ID, in.Waiting); err != nil {
			return err
		}
	}
	switch {
	case in.ToBacklog && task.State != "backlog":
		return s.store.SetTaskState(task.ID, "backlog")
	case in.ToToday && task.State != "now":
		return s.store.SetTaskState(task.ID, "now")
	}
	return nil
}

// newTag is the slug a new line would take, shown in the empty line's placeholder.
func (s *Server) newTag(sh shell, state, remembered string) string {
	projects := make([]store.Project, len(sh.Projects))
	for i, p := range sh.Projects {
		projects[i] = p.Project
	}
	if p := s.projectAbove(projects, state, remembered); p != nil {
		return p.Slug
	}
	return ""
}

// projectAbove is the project of the last open line on the page, else the
// remembered one, else the first project.
func (s *Server) projectAbove(projects []store.Project, state, remembered string) *store.Project {
	if ts, err := s.store.TasksByState(state); err == nil && len(ts) > 0 {
		last := ts[len(ts)-1]
		for i := range projects {
			if projects[i].ID == last.ProjectID {
				return &projects[i]
			}
		}
	}
	for i := range projects {
		if projects[i].Slug == remembered {
			return &projects[i]
		}
	}
	if len(projects) > 0 {
		return &projects[0]
	}
	return nil
}

// preferredProject is the project the request names, else the last one used.
func preferredProject(r *http.Request) string {
	if p := r.FormValue("p"); p != "" {
		return p
	}
	if c, err := r.Cookie("lastp"); err == nil {
		return c.Value
	}
	return ""
}

// strikeLine crosses a line out, or brings a struck one back to today.
func (s *Server) strikeLine(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.Task(pathID(r))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	state, label := "done", "Закреслено"
	if task.State == "done" {
		state, label = "now", "Повернуто"
	}
	if err := s.store.SetTaskState(task.ID, state); err != nil {
		s.fail(w, "strike", err)
		return
	}
	offerUndo(r, label, "state", task.ID, undoState(task))
	s.respondNotebook(w, r)
}

// moveLineHandler shifts a line one step up or down within its page (⌥↑ / ⌥↓).
func (s *Server) moveLineHandler(w http.ResponseWriter, r *http.Request) {
	s.moveLine(w, r, pathID(r))
}

func (s *Server) moveLine(w http.ResponseWriter, r *http.Request, id int64) {
	task, err := s.store.Task(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	list, err := s.store.TasksByState(task.State)
	if err != nil {
		s.fail(w, "move", err)
		return
	}
	order := make([]int64, len(list))
	at := -1
	for i, t := range list {
		order[i] = t.ID
		if t.ID == task.ID {
			at = i
		}
	}
	to, back := at, "down"
	if r.FormValue("dir") == "up" {
		to--
	} else {
		to++
		back = "up"
	}
	if at >= 0 && to >= 0 && to < len(order) {
		order[at], order[to] = order[to], order[at]
		if err := s.store.SetPositions(task.State, order); err != nil {
			s.fail(w, "move", err)
			return
		}
		offerUndo(r, "Пересунуто", "move", task.ID, back)
	}
	s.respondNotebook(w, r)
}

// orderLines applies a drag: the full order of one page's lines.
func (s *Server) orderLines(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	state := r.FormValue("state")
	if state != "now" && state != "backlog" {
		http.Error(w, "bad state", http.StatusBadRequest)
		return
	}
	if err := s.store.SetPositions(state, ids(r.Form["id"])); err != nil {
		s.fail(w, "order", err)
		return
	}
	s.respondNotebook(w, r)
}

// sendLine moves a line to the other page: → беклог or → сьогодні.
func (s *Server) sendLine(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != "now" && state != "backlog" {
		http.Error(w, "bad state", http.StatusBadRequest)
		return
	}
	task, err := s.store.Task(pathID(r))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.store.SetTaskState(task.ID, state); err != nil {
		s.fail(w, "send", err)
		return
	}
	label := "→ беклог"
	if state == "now" {
		label = "→ сьогодні"
	}
	offerUndo(r, label, "state", task.ID, undoState(task))
	s.respondNotebook(w, r)
}

// releaseLine archives the checklist under a "/release" line and strikes the
// line with the score, so the release stays in the day as a journal entry.
func (s *Server) releaseLine(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.Task(pathID(r))
	if err != nil || !isRelease(task) {
		http.NotFound(w, r)
		return
	}
	items, err := s.store.Checklist(task.ProjectID)
	if err != nil {
		s.fail(w, "release", err)
		return
	}
	done := 0
	for _, it := range items {
		if it.Done {
			done++
		}
	}
	if err := s.store.MarkReleased(task.ProjectID); err != nil {
		s.fail(w, "release", err)
		return
	}
	title := fmt.Sprintf("Реліз %s · %d / %d", task.Project.Name, done, len(items))
	if err := s.store.UpdateTask(task.ID, title, task.Notes); err != nil {
		s.fail(w, "release", err)
		return
	}
	if err := s.store.SetTaskState(task.ID, "done"); err != nil {
		s.fail(w, "release", err)
		return
	}
	s.respondNotebook(w, r)
}

// deleteLine removes a line outright; the toast can bring it back.
func (s *Server) deleteLine(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.Task(pathID(r))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.store.DeleteTask(task.ID); err != nil {
		s.fail(w, "delete line", err)
		return
	}
	s.remember(task)
	offerUndo(r, "Рядок видалено", "restore", task.ID, "")
	s.respondNotebook(w, r)
}

// searchTasks matches open and recently closed tasks against q, open ones first.
func (s *Server) searchTasks(q string, limit int) ([]store.Task, error) {
	all, err := s.store.SearchTasks()
	if err != nil {
		return nil, err
	}
	var out []store.Task
	for _, t := range all {
		if matchQuery(t.Title, q) {
			out = append(out, t)
			if len(out) == limit {
				break
			}
		}
	}
	return out, nil
}

func pathID(r *http.Request) int64 {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id
}

func ids(raw []string) []int64 {
	out := make([]int64, 0, len(raw))
	for _, v := range raw {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			out = append(out, id)
		}
	}
	return out
}

func (s *Server) fail(w http.ResponseWriter, what string, err error) {
	log.Printf("%s: %v", what, err)
	http.Error(w, "db error", http.StatusInternalServerError)
}
