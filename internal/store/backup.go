package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Backup writes a consistent copy of the database into dir as
// dayboard-YYYY-MM-DD.db (VACUUM INTO, safe while serving) and keeps the newest `keep`.
func (s *Store) Backup(dir string, keep int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := filepath.Join(dir, "dayboard-"+time.Now().Format("2006-01-02")+".db")
	os.Remove(name) // VACUUM INTO refuses to overwrite
	if _, err := s.db.Exec(fmt.Sprintf(`vacuum into '%s'`, name)); err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(dir, "dayboard-*.db"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for len(files) > keep {
		os.Remove(files[0])
		files = files[1:]
	}
	return nil
}
