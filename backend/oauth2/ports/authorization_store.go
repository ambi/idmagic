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
	AttachAuthentication(ctx context.Context, id, sub string, authTime int64, amr []string, acr string) error
}

type AuthorizationCodeStore interface {
	Save(ctx context.Context, code *domain.AuthorizationCodeRecord) error
	Find(ctx context.Context, code string) (*domain.AuthorizationCodeRecord, error)
	// Redeem は code を atomic に redeemed にする。既に redeemed なら nil。
	Redeem(ctx context.Context, code string, now time.Time) (*domain.AuthorizationCodeRecord, error)
	// LinkFamily は成功交換時の refresh family を逆引きインデックスに紐付ける。
	LinkFamily(ctx context.Context, code, familyID string) error
}

type PARStore interface {
	Save(ctx context.Context, rec *domain.PARRecord) error
	Find(ctx context.Context, requestURI string) (*domain.PARRecord, error)
	Consume(ctx context.Context, requestURI string) (*domain.PARRecord, error)
}
