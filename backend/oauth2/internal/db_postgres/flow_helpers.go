package db_postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// nullable 列と domain の optional 値の相互変換。text codec 登録を前提に、
// UUID は Go 側で string として扱う。

func timestamptzOrNil(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func timestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	value := t.Time
	return &value
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	value := t.String
	return &value
}

func uuidOrNil(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{}
	}
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

func strDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
