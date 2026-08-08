package spec

import (
	"fmt"

	"github.com/google/uuid"
)

func NewUUIDv4() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("uuid: %w", err)
	}
	return id.String(), nil
}
