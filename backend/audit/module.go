// Package audit は audit bounded context の DI 組立を持つ (ADR-091)。
// authentication / identity-management / oauth2 / tenancy / signing-keys /
// application / saml / wsfederation を横断する監査イベント read model の
// repository と検索用 salt store を Module へ集約する。
package audit

import (
	auditports "github.com/ambi/idmagic/backend/audit/ports"
)

// Module は audit context が所有する repository の束。bootstrap は永続化 backend
// (memory / postgres_valkey) に応じてこれらを組み立て、Module へ渡すだけでよい。
type Module struct {
	AuditEventRepo  auditports.AuditEventRepository
	TenantSaltStore auditports.TenantSaltStore
}
