package store

import (
	"database/sql"
	"errors"
)

type Project struct {
	ID       int64
	Name     string
	Slug     string
	Color    string
	Kind     string
	JiraKey  string
	JiraHost string
	Repos    string
	Channels string
	OnBoard  bool
	Position int
}

const projectCols = `id, name, slug, color, kind, jira_key, jira_host, repos, channels, on_board, position`

func scanProject(row interface{ Scan(...any) error }) (Project, error) {
	var p Project
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.Color, &p.Kind, &p.JiraKey, &p.JiraHost, &p.Repos, &p.Channels, &p.OnBoard, &p.Position)
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
		{Name: "Atlas", Slug: "atl", Color: "#2F5D50", Kind: "work", JiraKey: "ATL", OnBoard: true},
		{Name: "Nimbus", Slug: "nim", Color: "#7C4A6B", Kind: "work", JiraKey: "NIM", OnBoard: true},
		{Name: "harbor", Slug: "harbor", Color: "#8A6A30", Kind: "pet", OnBoard: true},
		{Name: "kite", Slug: "kite", Color: "#8A6A30", Kind: "pet", OnBoard: true},
		{Name: "moss", Slug: "moss", Color: "#8A6A30", Kind: "pet", OnBoard: true},
		{Name: "zaval", Slug: "zaval", Color: "#8A6A30", Kind: "pet", OnBoard: true},
		{Name: "tide", Slug: "tide", Color: "#8A6A30", Kind: "pet", OnBoard: false},
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for i, p := range seed {
		_, err := tx.Exec(`insert into projects (name, slug, color, kind, jira_key, on_board, position) values (?, ?, ?, ?, ?, ?, ?)`,
			p.Name, p.Slug, p.Color, p.Kind, p.JiraKey, p.OnBoard, i)
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
	res, err := s.db.Exec(`update projects set name = ?, slug = ?, color = ?, kind = ?, jira_key = ?, jira_host = ?, repos = ?, channels = ?, on_board = ? where id = ?`,
		p.Name, p.Slug, p.Color, p.Kind, p.JiraKey, p.JiraHost, p.Repos, p.Channels, p.OnBoard, p.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetProjectOnBoard(id int64, on bool) error {
	_, err := s.db.Exec(`update projects set on_board = ? where id = ?`, on, id)
	return err
}

// DeleteProject removes a project that has no tasks; ErrHasTasks otherwise.
func (s *Store) DeleteProject(id int64) error {
	var n int
	if err := s.db.QueryRow(`select count(*) from tasks where project_id = ?`, id).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrHasTasks
	}
	_, err := s.db.Exec(`delete from projects where id = ?`, id)
	return err
}

var ErrHasTasks = errors.New("project has tasks")
