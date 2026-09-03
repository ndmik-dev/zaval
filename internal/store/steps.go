package store

type Step struct {
	ID       int64
	TaskID   int64
	Title    string
	Done     bool
	Position int
}

func (s *Store) Steps(taskID int64) ([]Step, error) {
	rows, err := s.db.Query(`select id, task_id, title, done, position from task_steps where task_id = ? order by position, id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Step
	for rows.Next() {
		var st Step
		if err := rows.Scan(&st.ID, &st.TaskID, &st.Title, &st.Done, &st.Position); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func (s *Store) AddStep(taskID int64, title string) (Step, error) {
	res, err := s.db.Exec(`insert into task_steps (task_id, title, position)
		values (?, ?, coalesce((select max(position) from task_steps where task_id = ?), 0) + 1)`, taskID, title, taskID)
	if err != nil {
		return Step{}, err
	}
	id, _ := res.LastInsertId()
	return Step{ID: id, TaskID: taskID, Title: title}, nil
}

func (s *Store) ToggleStep(id int64) error {
	_, err := s.db.Exec(`update task_steps set done = not done where id = ?`, id)
	return err
}

func (s *Store) DeleteStep(id int64) error {
	_, err := s.db.Exec(`delete from task_steps where id = ?`, id)
	return err
}

// StepTask returns the task a step belongs to, so handlers can re-render it.
func (s *Store) StepTask(stepID int64) (int64, error) {
	var taskID int64
	err := s.db.QueryRow(`select task_id from task_steps where id = ?`, stepID).Scan(&taskID)
	return taskID, err
}

func (s *Store) LinkTask(linkID int64) (int64, error) {
	var taskID int64
	err := s.db.QueryRow(`select task_id from task_links where id = ?`, linkID).Scan(&taskID)
	return taskID, err
}
