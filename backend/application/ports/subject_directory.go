package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/application/domain"
)

// SubjectDirectory は Application に割り当てられる主体がテナント内に実在するかを照合する。
// User と Group の保存表現は Application Context の外に留める。
type SubjectDirectory interface {
	SubjectExists(
		ctx context.Context,
		tenantID string,
		subjectType domain.AssignmentSubjectType,
		subjectID string,
	) (bool, error)
}
