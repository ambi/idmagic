package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

const (
	defaultNameIDFormat = "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"
	defaultNameIDSource = "user_id"
)

// RelyingPartyAdministration は wsfedports.RelyingPartyAdministration を RP の保存先で実装する。
type RelyingPartyAdministration struct {
	RPRepo wsfedports.WsFedRelyingPartyRepository
}

var _ wsfedports.RelyingPartyAdministration = RelyingPartyAdministration{}

func (a RelyingPartyAdministration) Find(ctx context.Context, tenantID, wtrealm string) (*feddomain.WsFedRelyingParty, error) {
	return a.RPRepo.FindByWtrealm(ctx, tenantID, wtrealm)
}

func (a RelyingPartyAdministration) CreateForApplication(ctx context.Context, in wsfedports.CreateApplicationRelyingPartyInput) error {
	rp := &feddomain.WsFedRelyingParty{
		TenantID: in.TenantID, Wtrealm: in.Wtrealm, DisplayName: in.DisplayName, ReplyURLs: in.ReplyURLs,
		ClaimPolicy: claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{
			Format: nonEmpty(in.NameIDFormat, defaultNameIDFormat), SourceAttribute: nonEmpty(in.NameIDSource, defaultNameIDSource),
		}},
		CreatedAt: in.Now,
	}
	if err := rp.Validate(); err != nil {
		return err
	}
	existing, err := a.RPRepo.FindByWtrealm(ctx, in.TenantID, in.Wtrealm)
	if err != nil {
		return err
	}
	if existing != nil {
		return wsfedports.ErrRelyingPartyAlreadyRegistered
	}
	return a.RPRepo.Save(ctx, rp)
}

func (a RelyingPartyAdministration) Update(ctx context.Context, in wsfedports.UpdateRelyingPartyInput) error {
	rp, err := a.RPRepo.FindByWtrealm(ctx, in.TenantID, in.Wtrealm)
	if err != nil {
		return err
	}
	if rp == nil {
		return wsfedports.ErrRelyingPartyNotFound
	}
	if in.TokenType != nil && *in.TokenType != "" && !in.TokenType.Valid() {
		return wsfedports.ErrInvalidTokenType
	}
	if in.ReplyURLs != nil {
		rp.ReplyURLs = *in.ReplyURLs
	}
	if in.Audience != nil {
		rp.Audience = strings.TrimSpace(*in.Audience)
	}
	if in.TokenType != nil {
		rp.TokenType = *in.TokenType
	}
	if in.NameIDFormat != nil {
		rp.ClaimPolicy.NameID.Format = *in.NameIDFormat
	}
	if in.NameIDSource != nil {
		rp.ClaimPolicy.NameID.SourceAttribute = *in.NameIDSource
	}
	if in.Rules != nil {
		rp.ClaimPolicy.Rules = *in.Rules
	}
	rp.UpdatedAt = in.Now
	return a.RPRepo.Save(ctx, rp)
}

var (
	// ErrApplicationOwned は、Application に属する RP を管理 API から上書きまたは削除しようとしたことを表す。
	ErrApplicationOwned = errors.New("relying party is owned by an application")
	// ErrEntraRelyingParty は、Entra の定型設定を持つ RP を汎用の管理 API から上書きまたは削除しようとしたことを表す。
	ErrEntraRelyingParty = errors.New("relying party carries the Entra federation profile")
)

// RegisterRelyingPartyInput は、管理 API の登録が RP を置き換える内容である。
type RegisterRelyingPartyInput struct {
	TenantID    string
	Wtrealm     string
	DisplayName string
	ReplyURLs   []string
	Audience    string
	ClaimPolicy claimdomain.ClaimMappingPolicy
	Now         time.Time
}

// Register は管理 API の登録（REQ-WSFEDERATION-001）で、同じ wtrealm の RP を作成時刻を保って置き換えるか作る。
// Entra の定型設定を持つ RP は ErrEntraRelyingParty で、Application に属する RP は ErrApplicationOwned で、
// 長さの違反は Validate のエラーで拒否する。汎用の登録は Entra の定型設定と固定のクレームを捨てて
// 置き換えるので、Entra の RP には届かせない。
func (a RelyingPartyAdministration) Register(ctx context.Context, in RegisterRelyingPartyInput) (*feddomain.WsFedRelyingParty, bool, error) {
	existing, err := a.RPRepo.FindByWtrealm(ctx, in.TenantID, in.Wtrealm)
	if err != nil {
		return nil, false, err
	}
	if err := refuseOwned(existing); err != nil {
		return nil, false, err
	}
	rp := &feddomain.WsFedRelyingParty{
		TenantID: in.TenantID, Wtrealm: in.Wtrealm, DisplayName: in.DisplayName, ReplyURLs: in.ReplyURLs,
		Audience: in.Audience, ClaimPolicy: in.ClaimPolicy, CreatedAt: in.Now, UpdatedAt: in.Now,
	}
	if err := rp.Validate(); err != nil {
		return nil, false, err
	}
	if existing != nil {
		rp.CreatedAt = existing.CreatedAt
	}
	if err := a.RPRepo.Save(ctx, rp); err != nil {
		return nil, false, err
	}
	return rp, existing == nil, nil
}

// DeleteStandalone は管理 API の削除で、カタログの外の RP を冪等に消す。Entra の RP は
// ErrEntraRelyingParty で、Application に属する RP は ErrApplicationOwned で拒否して残す。
func (a RelyingPartyAdministration) DeleteStandalone(ctx context.Context, tenantID, wtrealm string) error {
	rp, err := a.RPRepo.FindByWtrealm(ctx, tenantID, wtrealm)
	if err != nil {
		return err
	}
	if err := refuseOwned(rp); err != nil {
		return err
	}
	return a.RPRepo.Delete(ctx, tenantID, wtrealm)
}

// SaveEntra は、Entra のドメインフェデレーションの構成が組み立てた RP を、Validate の後に保存する。
func (a RelyingPartyAdministration) SaveEntra(ctx context.Context, rp *feddomain.WsFedRelyingParty) error {
	if err := rp.Validate(); err != nil {
		return err
	}
	return a.RPRepo.Save(ctx, rp)
}

// refuseOwned は、汎用の管理 API が変えてはならない RP を拒否する。Entra の RP を先に判定する。
func refuseOwned(rp *feddomain.WsFedRelyingParty) error {
	switch {
	case rp == nil:
		return nil
	case rp.EntraProfile != nil:
		return ErrEntraRelyingParty
	case rp.ApplicationID != "":
		return ErrApplicationOwned
	}
	return nil
}

func (a RelyingPartyAdministration) BindApplication(ctx context.Context, tenantID, wtrealm, applicationID string) error {
	rp, err := a.RPRepo.FindByWtrealm(ctx, tenantID, wtrealm)
	if err != nil || rp == nil {
		return err
	}
	rp.ApplicationID = applicationID
	return a.RPRepo.Save(ctx, rp)
}

func (a RelyingPartyAdministration) DeleteForApplication(ctx context.Context, tenantID, wtrealm string) error {
	return a.RPRepo.Delete(ctx, tenantID, wtrealm)
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
