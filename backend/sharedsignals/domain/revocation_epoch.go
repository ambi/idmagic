package domain

import (
	"time"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// AgentRevocationEpoch は Agent ごとの revocation epoch を保持する 1 対 1 の付随
// レコード。Epoch 以前に issued_at を持つ access token / session は無効とみなす
// (fail-closed)。行が存在しない Agent は未失効を意味する。
type AgentRevocationEpoch struct {
	AgentID       string
	TenantID      string
	Epoch         time.Time
	Reason        RevocationReason
	AdvancedAt    time.Time
	SourceEventID *string
}

var agentRevocationEpochSchema = z.Struct(z.Shape{
	"AgentID":  z.String().Min(1).Required(),
	"TenantID": z.String().Min(1).Required(),
	"Epoch":    z.Time().Required(),
	"Reason": z.StringLike[RevocationReason]().TestFunc(
		func(value *RevocationReason, _ z.Ctx) bool { return value.Valid() },
		z.Message("revocation reason is not in enum"),
	).Required(),
	"AdvancedAt": z.Time().Required(),
})

// Validate は構造的妥当性 (spec/contexts/sharedsignals.yaml AgentRevocationEpoch) を検証する。
func (e AgentRevocationEpoch) Validate() error {
	return spec.Validate(agentRevocationEpochSchema, &e)
}

// Supersedes は、指定した issued_at (access token の iat) が本 epoch より前かどうかを
// 判定する。true なら当該 token は revoked とみなす (fail-closed、境界は issuedAt ==
// Epoch のとき「同時刻に発行された token は有効」とする — epoch 前進と発行が完全に
// 同時刻になることは実運用上ない)。
func (e AgentRevocationEpoch) Supersedes(issuedAt time.Time) bool {
	return issuedAt.Before(e.Epoch)
}
