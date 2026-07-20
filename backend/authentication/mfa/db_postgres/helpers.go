package db_postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func timestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	value := t.Time
	return &value
}
