// Package audit は audit モジュールの DI 組立を持つ。
// authentication / identity-management / oauth2 / tenancy / signing-keys /
// application / saml / wsfederation を横断する監査イベント read model の
// repository と検索用 salt store を Module へ集約する。
package audit

import (
	"github.com/ambi/idmagic/backend/audit/domain"
	"github.com/ambi/idmagic/backend/audit/internal/usecases"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

// Module は audit context が所有する repository の束。bootstrap は永続化 backend
// (memory / postgres) に応じてこれらを組み立て、Module へ渡すだけでよい。
type Module struct {
	AuditEventRepo  auditports.AuditEventRepository
	TenantSaltStore auditports.TenantSaltStore
	ExportArtifacts auditports.ExportArtifactStore
	ExportPolicy    domain.ExportPolicy
	ExportCodec     *support.CursorCodec
	metrics         *exportMetrics
	// eventPurger は保持期間の掃引が使う削除である。構築関数が AuditEventRepo と同じ
	// 保存先を入れる。AuditEventRepo を差し替えても、掃引の対象は変わらない。
	eventPurger usecases.EventPurger
}
