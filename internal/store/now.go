package store

import (
	"database/sql/driver"
	"time"
)

// Value formats the current UTC time the same way datetime('now') does.
func (sqlNowType) Value() (driver.Value, error) {
	return time.Now().UTC().Format(TimeLayout), nil
}
