package store

// DoneBetween returns tasks closed in [from, to), newest first.
// Bounds are UTC timestamps in TimeLayout.
func (s *Store) DoneBetween(from, to string) ([]Task, error) {
	return s.queryTasks(`where t.state = 'done' and t.done_at >= ? and t.done_at < ? order by t.done_at desc`, from, to)
}

// DoneByDay counts closed tasks per local day in [from, to), keyed YYYY-MM-DD.
func (s *Store) DoneByDay(from, to string) (map[string]int, error) {
	rows, err := s.db.Query(`select date(done_at, 'localtime'), count(*) from tasks
		where state = 'done' and done_at >= ? and done_at < ? group by 1`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var day string
		var n int
		if err := rows.Scan(&day, &n); err != nil {
			return nil, err
		}
		out[day] = n
	}
	return out, rows.Err()
}
