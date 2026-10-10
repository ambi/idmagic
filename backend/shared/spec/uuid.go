package spec

import (
	"fmt"

	"github.com/google/uuid"
)

// 担保するルールは docs/design/application/api-guidelines.md の「識別子」。
func NewUUIDv4() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("uuid: %w", err)
	}
	return id.String(), nil
}
