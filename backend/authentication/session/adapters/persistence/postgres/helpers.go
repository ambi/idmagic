package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

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

// uuidOrNil / uuidPtr は空文字を「値なし」として扱う nullable UUID の変換 (ADR-084 の
// text codec 登録を前提に、Go 側は常に string で UUID を表現する)。
func uuidOrNil(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{}
	}
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

func uuidPtr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	v, err := u.Value()
	if err != nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
