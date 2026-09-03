package store

// DoneBetween returns tasks closed in [from, to), newest first.
// Bounds are UTC timestamps in TimeLayout.
func (s *Store) DoneBetween(from, to string) ([]Task, error) {
	return s.queryTasks(`where t.state = 'done' and t.done_at >= ? and t.done_at < ? order by t.done_at desc`, from, to)
}
