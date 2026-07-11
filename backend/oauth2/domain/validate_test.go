package domain

import (
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// wi-129 の Validate() happy/failure カバレッジのうち client/consent/認可詳細タイプ系を
// 移設 (wi-173)。

func TestValidateHappyAndFailure(t *testing.T) {
	now := time.Now().UTC()

	validClient := OAuth2Client{
		ClientID: "demo", ClientType: spec.ClientConfidential,
		RedirectURIs: []string{"https://app.example.com/cb"},
		GrantTypes:   []spec.GrantType{spec.GrantAuthorizationCode}, ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: AuthMethodClientSecretBasic, IDTokenSignedResponseAlg: spec.SigAlgPS256,
		FapiProfile: FapiNone, CreatedAt: now, UpdatedAt: now,
	}
	// authorization_code グラントだが redirect_uris が無いので失敗する。
	badClient := validClient
	badClient.RedirectURIs = nil

	validConsent := Consent{UserID: "user_1", ClientID: "demo", Scopes: []string{"openid"}, State: ConsentGranted, GrantedAt: now, ExpiresAt: now}
	badConsent := validConsent
	badConsent.Scopes = nil

	validDetailType := AuthorizationDetailType{
		TenantID: tenancydomain.DefaultTenantID, Type: "payment", DisplayTemplate: "{{.amount}}", State: DetailTypeEnabled,
		Schema:    AuthorizationDetailsSchema{Rules: []AuthorizationDetailFieldRule{{Name: "amount", Semantics: DetailFieldExact}}},
		CreatedAt: now, UpdatedAt: now,
	}
	badDetailType := validDetailType
	badDetailType.Schema = AuthorizationDetailsSchema{Rules: nil}

	cases := []struct {
		name    string
		v       interface{ Validate() error }
		wantErr bool
	}{
		{"client ok", validClient, false},
		{"client bad", badClient, true},
		{"consent ok", validConsent, false},
		{"consent bad", badConsent, true},
		{"detail type ok", validDetailType, false},
		{"detail type bad", badDetailType, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.v.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("%s: expected error, got nil", c.name)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("%s: expected valid, got %v", c.name, err)
			}
		})
	}
}
