package ports

import (
	"context"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

// LifecycleWorkflowRepository は定義と immutable revision の tenant-scoped な
// 永続化境界。実行履歴は worker 用の別 port に切り出せるよう、ここでは定義だけを持つ。
type LifecycleWorkflowRepository interface {
	List(ctx context.Context, tenantID string) ([]*idmdomain.LifecycleWorkflow, error)
	Find(ctx context.Context, tenantID, workflowID string) (*idmdomain.LifecycleWorkflow, error)
	Save(ctx context.Context, workflow *idmdomain.LifecycleWorkflow) error
	FindRevision(ctx context.Context, tenantID, workflowID string, revision int64) (*idmdomain.LifecycleWorkflowRevision, error)
	SaveRevision(ctx context.Context, revision *idmdomain.LifecycleWorkflowRevision) error
}
