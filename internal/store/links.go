package store

func (s *Store) AddLink(taskID int64, url, kind, label, meta string) (Link, error) {
	res, err := s.db.Exec(`insert into task_links (task_id, url, kind, label, meta, position)
		values (?, ?, ?, ?, ?, coalesce((select max(position) from task_links where task_id = ?), 0) + 1)`,
		taskID, url, kind, label, meta, taskID)
	if err != nil {
		return Link{}, err
	}
	id, _ := res.LastInsertId()
	return Link{ID: id, TaskID: taskID, URL: url, Kind: kind, Label: label, Meta: meta}, nil
}

func (s *Store) DeleteLink(id int64) error {
	_, err := s.db.Exec(`delete from task_links where id = ?`, id)
	return err
}
