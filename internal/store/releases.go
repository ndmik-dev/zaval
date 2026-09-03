package store

import "database/sql"

type Release struct {
	ID         int64
	ProjectID  int64
	Name       string
	Date       sql.NullString // YYYY-MM-DD
	ReleasedAt sql.NullString
	CreatedAt  string
	Project    Project
	Done       int
	Total      int
	Items      []ReleaseItem // loaded by Release(id)
	Tasks      []Task        // loaded by Release(id)
}

type ReleaseItem struct {
	ID        int64
	ReleaseID int64
	Phase     string // before | after
	Title     string
	Detail    string
	URL       string
	Command   string
	Done      bool
	Position  int
}

type TemplateItem struct {
	ID        int64
	ProjectID int64
	Phase     string
	Title     string
	Detail    string
	URL       string
	Command   string
	Position  int
}

const releaseSelect = `
select r.id, r.project_id, r.name, r.date, r.released_at, r.created_at,
       ` + projectColsPrefixed + `,
       (select count(*) from release_items i where i.release_id = r.id and i.done = 1),
       (select count(*) from release_items i where i.release_id = r.id)
from releases r join projects p on p.id = r.project_id `

func (s *Store) queryReleases(where string, args ...any) ([]Release, error) {
	rows, err := s.db.Query(releaseSelect+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Release
	for rows.Next() {
		var r Release
		p := &r.Project
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Name, &r.Date, &r.ReleasedAt, &r.CreatedAt,
			&p.ID, &p.Name, &p.Slug, &p.Color, &p.Kind, &p.JiraKey, &p.JiraHost, &p.Repos, &p.Channels, &p.OnBoard, &p.Position,
			&r.Done, &r.Total); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Releases returns releases in preparation (dated ones first, soonest first)
// and the last released ones.
func (s *Store) Releases() (upcoming, past []Release, err error) {
	upcoming, err = s.queryReleases(`where r.released_at is null order by r.date is null, r.date, r.id`)
	if err != nil {
		return
	}
	past, err = s.queryReleases(`where r.released_at is not null order by r.released_at desc limit 12`)
	return
}

func (s *Store) Release(id int64) (Release, error) {
	rs, err := s.queryReleases(`where r.id = ?`, id)
	if err != nil {
		return Release{}, err
	}
	if len(rs) == 0 {
		return Release{}, ErrNotFound
	}
	r := rs[0]
	rows, err := s.db.Query(`select id, release_id, phase, title, detail, url, command, done, position from release_items where release_id = ? order by position, id`, id)
	if err != nil {
		return Release{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var it ReleaseItem
		if err := rows.Scan(&it.ID, &it.ReleaseID, &it.Phase, &it.Title, &it.Detail, &it.URL, &it.Command, &it.Done, &it.Position); err != nil {
			return Release{}, err
		}
		r.Items = append(r.Items, it)
	}
	if err := rows.Err(); err != nil {
		return Release{}, err
	}
	r.Tasks, err = s.queryTasks(`where t.release_id = ? order by case t.state when 'now' then 0 when 'backlog' then 1 else 2 end, t.position`, id)
	return r, err
}

// NextRelease is the upcoming release shown on the board: the project's own
// when projectID is set, otherwise the soonest of all.
func (s *Store) NextRelease(projectID int64) (*Release, error) {
	where := `where r.released_at is null order by r.date is null, r.date, r.id limit 1`
	var args []any
	if projectID != 0 {
		where = `where r.released_at is null and r.project_id = ? order by r.date is null, r.date, r.id limit 1`
		args = append(args, projectID)
	}
	rs, err := s.queryReleases(where, args...)
	if err != nil || len(rs) == 0 {
		return nil, err
	}
	r, err := s.Release(rs[0].ID)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateRelease copies the project's template into a fresh checklist.
func (s *Store) CreateRelease(projectID int64, name, date string) (Release, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Release{}, err
	}
	defer tx.Rollback()
	var d any
	if date != "" {
		d = date
	}
	res, err := tx.Exec(`insert into releases (project_id, name, date) values (?, ?, ?)`, projectID, name, d)
	if err != nil {
		return Release{}, err
	}
	id, _ := res.LastInsertId()
	if _, err := tx.Exec(`insert into release_items (release_id, phase, title, detail, url, command, position)
		select ?, phase, title, detail, url, command, position from release_templates where project_id = ? order by position, id`, id, projectID); err != nil {
		return Release{}, err
	}
	if err := tx.Commit(); err != nil {
		return Release{}, err
	}
	return s.Release(id)
}

func (s *Store) UpdateRelease(id int64, name, date string) error {
	var d any
	if date != "" {
		d = date
	}
	_, err := s.db.Exec(`update releases set name = ?, date = ? where id = ?`, name, d, id)
	return err
}

func (s *Store) SetReleased(id int64, released bool) error {
	if released {
		_, err := s.db.Exec(`update releases set released_at = ? where id = ?`, sqlNow, id)
		return err
	}
	_, err := s.db.Exec(`update releases set released_at = null where id = ?`, id)
	return err
}

func (s *Store) DeleteRelease(id int64) error {
	_, err := s.db.Exec(`delete from releases where id = ?`, id)
	return err
}

func (s *Store) AddReleaseItem(releaseID int64, phase, title, detail, url, command string) (ReleaseItem, error) {
	res, err := s.db.Exec(`insert into release_items (release_id, phase, title, detail, url, command, position)
		values (?, ?, ?, ?, ?, ?, coalesce((select max(position) from release_items where release_id = ?), 0) + 1)`,
		releaseID, phase, title, detail, url, command, releaseID)
	if err != nil {
		return ReleaseItem{}, err
	}
	id, _ := res.LastInsertId()
	return ReleaseItem{ID: id, ReleaseID: releaseID, Phase: phase, Title: title, Detail: detail, URL: url, Command: command}, nil
}

func (s *Store) ToggleReleaseItem(id int64) error {
	_, err := s.db.Exec(`update release_items set done = not done where id = ?`, id)
	return err
}

func (s *Store) DeleteReleaseItem(id int64) error {
	_, err := s.db.Exec(`delete from release_items where id = ?`, id)
	return err
}

func (s *Store) ReleaseItemRelease(itemID int64) (int64, error) {
	var id int64
	err := s.db.QueryRow(`select release_id from release_items where id = ?`, itemID).Scan(&id)
	return id, err
}

func (s *Store) Templates(projectID int64) ([]TemplateItem, error) {
	rows, err := s.db.Query(`select id, project_id, phase, title, detail, url, command, position from release_templates where project_id = ? order by position, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TemplateItem
	for rows.Next() {
		var it TemplateItem
		if err := rows.Scan(&it.ID, &it.ProjectID, &it.Phase, &it.Title, &it.Detail, &it.URL, &it.Command, &it.Position); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) AddTemplateItem(projectID int64, phase, title, detail, url, command string) (TemplateItem, error) {
	res, err := s.db.Exec(`insert into release_templates (project_id, phase, title, detail, url, command, position)
		values (?, ?, ?, ?, ?, ?, coalesce((select max(position) from release_templates where project_id = ?), 0) + 1)`,
		projectID, phase, title, detail, url, command, projectID)
	if err != nil {
		return TemplateItem{}, err
	}
	id, _ := res.LastInsertId()
	return TemplateItem{ID: id, ProjectID: projectID, Phase: phase, Title: title, Detail: detail, URL: url, Command: command}, nil
}

func (s *Store) UpdateTemplateItem(it TemplateItem) error {
	_, err := s.db.Exec(`update release_templates set phase = ?, title = ?, detail = ?, url = ?, command = ? where id = ?`,
		it.Phase, it.Title, it.Detail, it.URL, it.Command, it.ID)
	return err
}

func (s *Store) DeleteTemplateItem(id int64) error {
	_, err := s.db.Exec(`delete from release_templates where id = ?`, id)
	return err
}

func (s *Store) TemplateItemProject(itemID int64) (int64, error) {
	var id int64
	err := s.db.QueryRow(`select project_id from release_templates where id = ?`, itemID).Scan(&id)
	return id, err
}

// SetTaskRelease links a task to a release; 0 clears the link.
func (s *Store) SetTaskRelease(taskID, releaseID int64) error {
	var v any
	if releaseID != 0 {
		v = releaseID
	}
	_, err := s.db.Exec(`update tasks set release_id = ? where id = ?`, v, taskID)
	return err
}
