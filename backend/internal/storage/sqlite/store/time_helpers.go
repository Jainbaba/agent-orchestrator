package store

import (
	"database/sql"
	"time"
)

func timePtrToNull(value *time.Time) sql.NullTime {
	if value == nil || value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func nullTimeToPtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	copy := value.Time
	return &copy
}
