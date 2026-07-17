package ports

import (
	"context"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
)

// LifecycleWorkflowRepository は定義と immutable revision の tenant-scoped な
// 永続化境界。実行履歴は worker 用の別 port に切り出せるよう、ここでは定義だけを持つ。
type LifecycleWorkflowRepository interface {
	List(ctx context.Context, tenantID string) ([]*igdomain.LifecycleWorkflow, error)
	Find(ctx context.Context, tenantID, workflowID string) (*igdomain.LifecycleWorkflow, error)
	Save(ctx context.Context, workflow *igdomain.LifecycleWorkflow) error
	FindRevision(ctx context.Context, tenantID, workflowID string, revision int64) (*igdomain.LifecycleWorkflowRevision, error)
	SaveRevision(ctx context.Context, revision *igdomain.LifecycleWorkflowRevision) error
}
