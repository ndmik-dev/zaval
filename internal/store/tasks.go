package store

import (
	"database/sql"
	"fmt"
	"strings"
)

type Task struct {
	ID           int64
	ProjectID    int64
	Title        string
	State        string
	Notes        string
	Position     int
	CreatedAt    string
	NowSince     sql.NullString
	DoneAt       sql.NullString
	Waiting      string // what the task waits for; empty = not waiting
	WaitingSince sql.NullString

	Project    Project
	Links      []Link
	Steps      []Step // loaded only by Task(id)
	StepsDone  int
	StepsTotal int
}

type Link struct {
	ID     int64
	TaskID int64
	URL    string
	Kind   string
	Label  string
	Meta   string
}

type Counts struct{ Now, Backlog int }

// Timestamps are stored the way SQLite's datetime('now') writes them, in UTC.
const TimeLayout = "2006-01-02 15:04:05"

const taskSelect = `
select t.id, t.project_id, t.title, t.state, t.notes, t.position, t.created_at, t.now_since, t.done_at, t.waiting, t.waiting_since,
       ` + projectColsPrefixed + `,
       (select count(*) from task_steps st where st.task_id = t.id and st.done = 1),
       (select count(*) from task_steps st where st.task_id = t.id)
from tasks t join projects p on p.id = t.project_id `

const projectColsPrefixed = `p.id, p.name, p.slug, p.color, p.kind, p.jira_key, p.jira_host, p.repos, p.channels, p.on_board, p.position`

func scanTask(rows *sql.Rows) (Task, error) {
	var t Task
	p := &t.Project
	err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.State, &t.Notes, &t.Position, &t.CreatedAt, &t.NowSince, &t.DoneAt, &t.Waiting, &t.WaitingSince,
		&p.ID, &p.Name, &p.Slug, &p.Color, &p.Kind, &p.JiraKey, &p.JiraHost, &p.Repos, &p.Channels, &p.OnBoard, &p.Position,
		&t.StepsDone, &t.StepsTotal)
	return t, err
}

func (s *Store) queryTasks(where string, args ...any) ([]Task, error) {
	rows, err := s.db.Query(taskSelect+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, s.attachLinks(out)
}

func (s *Store) attachLinks(tasks []Task) error {
	if len(tasks) == 0 {
		return nil
	}
	byID := make(map[int64]*Task, len(tasks))
	ids := make([]string, 0, len(tasks))
	args := make([]any, 0, len(tasks))
	for i := range tasks {
		byID[tasks[i].ID] = &tasks[i]
		ids = append(ids, "?")
		args = append(args, tasks[i].ID)
	}
	rows, err := s.db.Query(`select id, task_id, url, kind, label, meta from task_links where task_id in (`+strings.Join(ids, ",")+`) order by position, id`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.ID, &l.TaskID, &l.URL, &l.Kind, &l.Label, &l.Meta); err != nil {
			return err
		}
		t := byID[l.TaskID]
		t.Links = append(t.Links, l)
	}
	return rows.Err()
}

func (s *Store) TasksByState(state string) ([]Task, error) {
	return s.queryTasks(`where t.state = ? order by t.position, t.id`, state)
}

func (s *Store) DoneToday() ([]Task, error) {
	return s.queryTasks(`where t.state = 'done' and date(t.done_at, 'localtime') = date('now', 'localtime') order by t.done_at desc`)
}

func (s *Store) Task(id int64) (Task, error) {
	ts, err := s.queryTasks(`where t.id = ?`, id)
	if err != nil {
		return Task{}, err
	}
	if len(ts) == 0 {
		return Task{}, ErrNotFound
	}
	t := ts[0]
	if t.Steps, err = s.Steps(id); err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s *Store) UpdateTask(id int64, title, notes string) error {
	_, err := s.db.Exec(`update tasks set title = ?, notes = ? where id = ?`, title, notes, id)
	return err
}

func (s *Store) CreateTask(projectID int64, title, state string) (Task, error) {
	var nowSince any
	if state == "now" {
		nowSince = sqlNow
	}
	res, err := s.db.Exec(`insert into tasks (project_id, title, state, now_since, position)
		values (?, ?, ?, ?, coalesce((select max(position) from tasks where state = ?), 0) + 1)`,
		projectID, title, state, nowSince, state)
	if err != nil {
		return Task{}, err
	}
	id, _ := res.LastInsertId()
	return s.Task(id)
}

func (s *Store) ProjectCounts() (map[int64]Counts, error) {
	rows, err := s.db.Query(`select project_id, state, count(*) from tasks where state in ('now', 'backlog') group by project_id, state`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]Counts{}
	for rows.Next() {
		var pid int64
		var state string
		var n int
		if err := rows.Scan(&pid, &state, &n); err != nil {
			return nil, err
		}
		c := out[pid]
		if state == "now" {
			c.Now = n
		} else {
			c.Backlog = n
		}
		out[pid] = c
	}
	return out, rows.Err()
}

// sqlNow is passed as a bound value so callers never format timestamps themselves.
type sqlNowType struct{}

var sqlNow = sqlNowType{}

// SetTaskState moves a task between now, backlog and done, keeping the
// timestamps the board relies on (age in "now", done today) consistent.
func (s *Store) SetTaskState(id int64, state string) error {
	var q string
	switch state {
	case "now":
		q = `update tasks set state = 'now', now_since = coalesce(now_since, ?), done_at = null,
			position = coalesce((select max(position) from tasks where state = 'now'), 0) + 1 where id = ?`
	case "backlog":
		q = `update tasks set state = 'backlog', now_since = null, done_at = null,
			position = coalesce((select max(position) from tasks where state = 'backlog'), 0) + 1 where id = ?`
	case "done":
		q = `update tasks set state = 'done', done_at = ?, position = 0, waiting = '', waiting_since = null where id = ?`
	default:
		return fmt.Errorf("bad state %q", state)
	}
	var res sql.Result
	var err error
	if state == "backlog" {
		res, err = s.db.Exec(q, id)
	} else {
		res, err = s.db.Exec(q, sqlNow, id)
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteTask(id int64) error {
	_, err := s.db.Exec(`delete from tasks where id = ?`, id)
	return err
}

// SearchTasks finds tasks by title substring, open ones first.
func (s *Store) SearchTasks(q string, limit int) ([]Task, error) {
	return s.queryTasks(`where t.title like ? escape '\'
		order by case t.state when 'now' then 0 when 'backlog' then 1 else 2 end, t.done_at desc, t.position limit ?`,
		"%"+escapeLike(q)+"%", limit)
}

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// Reorder applies the drag result: each list is the full ordered set of ids
// for that state, so a task that was dragged across lists changes state too.
func (s *Store) Reorder(now, backlog []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, id := range now {
		if _, err := tx.Exec(`update tasks set state = 'now', position = ?, now_since = coalesce(now_since, ?), done_at = null where id = ?`, i+1, sqlNow, id); err != nil {
			return err
		}
	}
	for i, id := range backlog {
		if _, err := tx.Exec(`update tasks set state = 'backlog', position = ?, now_since = null, done_at = null where id = ?`, i+1, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetWaiting marks what the task waits for; an empty note clears it.
func (s *Store) SetWaiting(id int64, note string) error {
	if note == "" {
		_, err := s.db.Exec(`update tasks set waiting = '', waiting_since = null where id = ?`, id)
		return err
	}
	_, err := s.db.Exec(`update tasks set waiting = ?, waiting_since = coalesce(waiting_since, ?) where id = ?`, note, sqlNow, id)
	return err
}

// WaitingTasks lists open tasks that wait on something, oldest wait first.
func (s *Store) WaitingTasks() ([]Task, error) {
	return s.queryTasks(`where t.state in ('now', 'backlog') and t.waiting != '' order by t.waiting_since, t.id`)
}
