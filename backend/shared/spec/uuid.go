package spec

import "uuid"

// 担保するルールは docs/design/application/api-guidelines.md の「識別子」。
// uuid.New は将来ほかのバージョンを選び得るので、バージョン 4 を明示する。
func NewUUIDv4() (string, error) {
	return uuid.NewV4().String(), nil
}
