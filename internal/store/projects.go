package store

import (
	"database/sql"
	"errors"
)

// Project is a work or pet project. Slug is the #tag used in ⌘K.
// (The table still carries jira/repos/channels/on_board columns from earlier
// designs; they are not read.)
type Project struct {
	ID       int64
	Name     string
	Slug     string
	Color    string
	Kind     string // work | pet
	Position int
}

const projectCols = `id, name, slug, color, kind, position`

func scanProject(row interface{ Scan(...any) error }) (Project, error) {
	var p Project
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.Color, &p.Kind, &p.Position)
	return p, err
}

func (s *Store) Projects() ([]Project, error) {
	rows, err := s.db.Query(`select ` + projectCols + ` from projects order by position, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Project(id int64) (Project, error) {
	return scanProject(s.db.QueryRow(`select `+projectCols+` from projects where id = ?`, id))
}

func (s *Store) ProjectBySlug(slug string) (Project, error) {
	return scanProject(s.db.QueryRow(`select `+projectCols+` from projects where slug = ?`, slug))
}

// Seed inserts the initial projects when the database is empty.
func (s *Store) Seed() error {
	var n int
	if err := s.db.QueryRow(`select count(*) from projects`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	seed := []Project{
		{Name: "Atlas", Slug: "atl", Color: "#2F5D50", Kind: "work"},
		{Name: "Nimbus", Slug: "nim", Color: "#7C4A6B", Kind: "work"},
		{Name: "harbor", Slug: "harbor", Color: "#8A6A30", Kind: "pet"},
		{Name: "kite", Slug: "kite", Color: "#4E6B8C", Kind: "pet"},
		{Name: "moss", Slug: "moss", Color: "#7A5C99", Kind: "pet"},
		{Name: "zaval", Slug: "zaval", Color: "#3E7D6E", Kind: "pet"},
		{Name: "tide", Slug: "tide", Color: "#A0522D", Kind: "pet"},
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for i, p := range seed {
		_, err := tx.Exec(`insert into projects (name, slug, color, kind, position) values (?, ?, ?, ?, ?)`,
			p.Name, p.Slug, p.Color, p.Kind, i)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

var ErrNotFound = sql.ErrNoRows

func (s *Store) CreateProject(name, slug, kind, color string) (Project, error) {
	res, err := s.db.Exec(`insert into projects (name, slug, kind, color, position)
		values (?, ?, ?, ?, coalesce((select max(position) from projects), 0) + 1)`, name, slug, kind, color)
	if err != nil {
		return Project{}, err
	}
	id, _ := res.LastInsertId()
	return s.Project(id)
}

func (s *Store) UpdateProject(p Project) error {
	res, err := s.db.Exec(`update projects set name = ?, slug = ?, color = ?, kind = ? where id = ?`,
		p.Name, p.Slug, p.Color, p.Kind, p.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteProject removes a project that has no tasks; ErrHasTasks otherwise.
// DeleteProject removes a project together with its struck lines, release
// checklist and release history. Open lines block it: they must be moved or
// struck first, so nothing you still mean to do disappears with a project.
func (s *Store) DeleteProject(id int64) error {
	var n int
	if err := s.db.QueryRow(`select count(*) from tasks where project_id = ? and state != 'done'`, id).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrHasTasks
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		`delete from release_items where release_id in (select id from releases where project_id = ?)`,
		`delete from releases where project_id = ?`,
		`delete from release_templates where project_id = ?`,
		`delete from tasks where project_id = ?`,
		`delete from projects where id = ?`,
	} {
		if _, err := tx.Exec(q, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

var ErrHasTasks = errors.New("project has tasks")
