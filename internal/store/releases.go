package store

import "database/sql"

// ChecklistItem is one line of a project's release checklist. The same rows
// serve as the template and as the current run: Done is reset on release.
type ChecklistItem struct {
	ID        int64
	ProjectID int64
	Phase     string // before | after
	Title     string
	Detail    string
	URL       string
	Command   string
	Done      bool
	Position  int
	TaskID    sql.NullInt64
	TaskTitle sql.NullString
	TaskState sql.NullString
}

type ReleaseRecord struct {
	ID         int64
	ProjectID  int64
	ReleasedAt string
}

type Progress struct{ Done, Total int }

func (s *Store) Checklist(projectID int64) ([]ChecklistItem, error) {
	rows, err := s.db.Query(`select c.id, c.project_id, c.phase, c.title, c.detail, c.url, c.command, c.done, c.position, c.task_id, t.title, t.state
		from release_templates c left join tasks t on t.id = c.task_id where c.project_id = ? order by c.position, c.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChecklistItem
	for rows.Next() {
		var it ChecklistItem
		if err := rows.Scan(&it.ID, &it.ProjectID, &it.Phase, &it.Title, &it.Detail, &it.URL, &it.Command, &it.Done, &it.Position, &it.TaskID, &it.TaskTitle, &it.TaskState); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ChecklistProgress returns done/total per project for every project with items.
func (s *Store) ChecklistProgress() (map[int64]Progress, error) {
	rows, err := s.db.Query(`select project_id, sum(done), count(*) from release_templates group by project_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]Progress{}
	for rows.Next() {
		var pid int64
		var p Progress
		if err := rows.Scan(&pid, &p.Done, &p.Total); err != nil {
			return nil, err
		}
		out[pid] = p
	}
	return out, rows.Err()
}

func (s *Store) AddChecklistItem(projectID int64, phase, title string) (ChecklistItem, error) {
	res, err := s.db.Exec(`insert into release_templates (project_id, phase, title, position)
		values (?, ?, ?, coalesce((select max(position) from release_templates where project_id = ?), 0) + 1)`,
		projectID, phase, title, projectID)
	if err != nil {
		return ChecklistItem{}, err
	}
	id, _ := res.LastInsertId()
	return ChecklistItem{ID: id, ProjectID: projectID, Phase: phase, Title: title}, nil
}

func (s *Store) UpdateChecklistItem(id int64, title, command string) error {
	_, err := s.db.Exec(`update release_templates set title = ?, command = ? where id = ?`, title, command, id)
	return err
}

// SetChecklistItemTask points a line at a task; 0 clears it.
func (s *Store) SetChecklistItemTask(id, taskID int64) error {
	var v any
	if taskID != 0 {
		v = taskID
	}
	_, err := s.db.Exec(`update release_templates set task_id = ? where id = ?`, v, id)
	return err
}

func (s *Store) ToggleChecklistItem(id int64) error {
	_, err := s.db.Exec(`update release_templates set done = not done where id = ?`, id)
	return err
}

func (s *Store) DeleteChecklistItem(id int64) error {
	_, err := s.db.Exec(`delete from release_templates where id = ?`, id)
	return err
}

func (s *Store) ChecklistItemProject(id int64) (int64, error) {
	var pid int64
	err := s.db.QueryRow(`select project_id from release_templates where id = ?`, id).Scan(&pid)
	return pid, err
}

func (s *Store) ResetChecklist(projectID int64) error {
	_, err := s.db.Exec(`update release_templates set done = 0 where project_id = ?`, projectID)
	return err
}

// MarkReleased records a release for the project and clears the checklist.
func (s *Store) MarkReleased(projectID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`insert into releases (project_id, name, released_at) values (?, '', ?)`, projectID, sqlNow); err != nil {
		return err
	}
	if _, err := tx.Exec(`update release_templates set done = 0 where project_id = ?`, projectID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ReleaseHistory(projectID int64, limit int) ([]ReleaseRecord, error) {
	rows, err := s.db.Query(`select id, project_id, released_at from releases where project_id = ? and released_at is not null order by released_at desc limit ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReleaseRecord
	for rows.Next() {
		var r ReleaseRecord
		var at sql.NullString
		if err := rows.Scan(&r.ID, &r.ProjectID, &at); err != nil {
			return nil, err
		}
		r.ReleasedAt = at.String
		out = append(out, r)
	}
	return out, rows.Err()
}
