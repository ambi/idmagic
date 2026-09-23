package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type AuthorizationRequestStore interface {
	Save(ctx context.Context, req *domain.AuthorizationRequest) error
	Find(ctx context.Context, id string) (*domain.AuthorizationRequest, error)
	UpdateState(ctx context.Context, id string, state spec.AuthorizationCodeFlowState) error
	AttachAuthentication(ctx context.Context, id, sub string, authTime int64, amr []string, acr, sid string) error
}

type AuthorizationCodeStore interface {
	Save(ctx context.Context, code *domain.AuthorizationCodeRecord) error
	Find(ctx context.Context, code string) (*domain.AuthorizationCodeRecord, error)
	// Redeem は code を atomic に redeemed にする。既に redeemed なら nil。
	Redeem(ctx context.Context, code string, now time.Time) (*domain.AuthorizationCodeRecord, error)
	// MarkExpired は state='issued' の code を atomic に expired にする。
	// 既に redeemed または expired なら何もせず nil を返す (再提示の replay 検出は
	// 呼び出し側が別に行うので、ここでは issued からの遷移だけを扱う)。
	MarkExpired(ctx context.Context, code string) (*domain.AuthorizationCodeRecord, error)
	// LinkFamily は成功交換時の refresh family を逆引きインデックスに紐付ける。
	LinkFamily(ctx context.Context, code, familyID string) error
}

type PARStore interface {
	Save(ctx context.Context, rec *domain.PARRecord) error
	Find(ctx context.Context, requestURI string) (*domain.PARRecord, error)
	Consume(ctx context.Context, requestURI string) (*domain.PARRecord, error)
}
