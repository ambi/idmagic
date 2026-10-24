package ports

import (
	"context"
	"errors"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	"github.com/ambi/idmagic/backend/wsfederation/domain"
)

var (
	ErrRelyingPartyNotFound = errors.New("relying party does not exist")
	// ErrRelyingPartyAlreadyRegistered は、作成しようとした wtrealm の RP がテナントにすでにあることを表す。
	ErrRelyingPartyAlreadyRegistered = errors.New("relying party is already registered")
	ErrInvalidTokenType              = errors.New("token type is invalid")
)

// RelyingPartyAdministration は、Application のカタログが自身に属する RP を作成、更新、
// 関連付け、削除する操作である。既定値と設定値の検査は WsFederation が保つ。
type RelyingPartyAdministration interface {
	// Find は wtrealm に一致する RP を返す。存在しなければ (nil, nil)。
	Find(ctx context.Context, tenantID, wtrealm string) (*domain.WsFedRelyingParty, error)
	// CreateForApplication は新しい RP を保存する。wtrealm が長さの上限を超えれば Validate のエラーを、
	// 同じ wtrealm の RP があれば、Entra の RP か、どの Application に属するかを問わず
	// ErrRelyingPartyAlreadyRegistered を返す。
	CreateForApplication(ctx context.Context, in CreateApplicationRelyingPartyInput) error
	// Update は指定した項目だけを変える。RP がなければ ErrRelyingPartyNotFound を返す。
	Update(ctx context.Context, in UpdateRelyingPartyInput) error
	// BindApplication は、カタログの作成後に保存側の関連付けを同期する。RP がなければ何もしない。
	BindApplication(ctx context.Context, tenantID, wtrealm, applicationID string) error
	// DeleteForApplication は、カタログが所有する RP を Application の削除の後に消す。
	DeleteForApplication(ctx context.Context, tenantID, wtrealm string) error
}

// CreateApplicationRelyingPartyInput の NameID の項目は、空なら WsFederation の既定値を使う。
type CreateApplicationRelyingPartyInput struct {
	TenantID     string
	Wtrealm      string
	DisplayName  string
	ReplyURLs    []string
	NameIDFormat string
	NameIDSource string
	Now          time.Time
}

type UpdateRelyingPartyInput struct {
	TenantID     string
	Wtrealm      string
	ReplyURLs    *[]string
	Audience     *string
	TokenType    *domain.WsFedTokenType
	NameIDFormat *string
	NameIDSource *string
	Rules        *[]claimdomain.ClaimMappingRule
	Now          time.Time
}
