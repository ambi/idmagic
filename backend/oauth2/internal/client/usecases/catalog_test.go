package usecases

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/internal/client/db_memory"
)

//spec:covers REQ-APPLICATION-006, REQ-APPLICATION-007: カタログの関連付け、クレームの部分更新と削除は対象クライアントだけを変更する。
func TestCatalogOperationsPreserveUnchangedClientState(t *testing.T) {
	blank, custom := "  ", " employee_id "
	empty := []claimdomain.ClaimMappingRule{}
	rules := []claimdomain.ClaimMappingRule{{ClaimType: "department", Source: claimdomain.ClaimSourceFixed, FixedValue: "eng", Required: true}}
	for _, tc := range []struct {
		name                string
		policy              *claimdomain.ClaimMappingPolicy
		sub                 *string
		rules               *[]claimdomain.ClaimMappingRule
		wantSub, wantFormat string
		wantRules           []claimdomain.ClaimMappingRule
	}{
		{"new policy", nil, nil, nil, "user_id", "user_id", nil},
		{"blank sub", nil, &blank, &empty, "user_id", "user_id", empty},
		{"custom sub", nil, &custom, &rules, "employee_id", "user_id", rules},
		{"preserve existing", &claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{Format: "format", SourceAttribute: "old"}, Rules: rules}, nil, nil, "old", "format", rules},
		{"replace rules", &claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{Format: "format"}, Rules: rules}, nil, &empty, "user_id", "format", empty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repo := db_memory.NewClientRepository()
			client := &domain.OAuth2Client{TenantID: "tenant", ClientID: "client", Scope: "openid", ClaimPolicy: tc.policy}
			other := &domain.OAuth2Client{TenantID: "other", ClientID: "client", ApplicationID: "other-app"}
			if err := repo.Save(ctx, client); err != nil {
				t.Fatal(err)
			}
			if err := repo.Save(ctx, other); err != nil {
				t.Fatal(err)
			}
			a := ClientAdministration{Deps: AdminOAuth2ClientDeps{ClientRepo: repo}}
			if err := a.BindApplication(ctx, "tenant", "client", "app"); err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
			if err := a.UpdateClaims(ctx, "tenant", "client", clientports.ClientClaimUpdate{SubSourceAttribute: tc.sub, Rules: tc.rules, Now: now}); err != nil {
				t.Fatal(err)
			}
			got, _ := repo.FindByID(ctx, "tenant", "client")
			if got.ApplicationID != "app" || got.Scope != "openid" || got.UpdatedAt != now || got.ClaimPolicy.NameID.SourceAttribute != tc.wantSub || got.ClaimPolicy.NameID.Format != tc.wantFormat || !reflect.DeepEqual(got.ClaimPolicy.Rules, tc.wantRules) {
				t.Fatalf("client=%+v policy=%+v", got, got.ClaimPolicy)
			}
			if err := a.DeleteForApplication(ctx, "tenant", "client"); err != nil {
				t.Fatal(err)
			}
			if got, _ := repo.FindByID(ctx, "tenant", "client"); got != nil {
				t.Fatal("client was not deleted")
			}
			if got, _ := repo.FindByID(ctx, "other", "client"); !reflect.DeepEqual(got, other) {
				t.Fatalf("other tenant changed: %+v", got)
			}
		})
	}
}

type catalogFailureRepository struct {
	clientports.OAuth2ClientRepository
	findErr, saveErr, deleteErr error
}

func (r catalogFailureRepository) FindByID(ctx context.Context, tenantID, clientID string) (*domain.OAuth2Client, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.OAuth2ClientRepository.FindByID(ctx, tenantID, clientID)
}

func (r catalogFailureRepository) Save(ctx context.Context, client *domain.OAuth2Client) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	return r.OAuth2ClientRepository.Save(ctx, client)
}

func (r catalogFailureRepository) Delete(ctx context.Context, tenantID, clientID string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	return r.OAuth2ClientRepository.Delete(ctx, tenantID, clientID)
}

func TestCatalogOperationsPreserveMissingAndStoreFailures(t *testing.T) {
	ctx := context.Background()
	repo := db_memory.NewClientRepository()
	a := ClientAdministration{Deps: AdminOAuth2ClientDeps{ClientRepo: repo}}
	if err := a.BindApplication(ctx, "tenant", "missing", "app"); err != nil {
		t.Fatalf("missing binding: %v", err)
	}
	if err := a.UpdateClaims(ctx, "tenant", "missing", clientports.ClientClaimUpdate{}); !errors.Is(err, clientports.ErrCatalogClientNotFound) {
		t.Fatalf("missing claims: %v", err)
	}
	failure := errors.New("store failed")
	if err := repo.Save(ctx, &domain.OAuth2Client{TenantID: "tenant", ClientID: "client"}); err != nil {
		t.Fatal(err)
	}
	for _, failing := range []catalogFailureRepository{
		{OAuth2ClientRepository: repo, findErr: failure},
		{OAuth2ClientRepository: repo, saveErr: failure},
	} {
		a := ClientAdministration{Deps: AdminOAuth2ClientDeps{ClientRepo: failing}}
		if err := a.BindApplication(ctx, "tenant", "client", "app"); !errors.Is(err, failure) {
			t.Fatalf("bind: %v", err)
		}
		if err := a.UpdateClaims(ctx, "tenant", "client", clientports.ClientClaimUpdate{}); !errors.Is(err, failure) {
			t.Fatalf("claims: %v", err)
		}
	}
	a.Deps.ClientRepo = catalogFailureRepository{OAuth2ClientRepository: repo, deleteErr: failure}
	if err := a.DeleteForApplication(ctx, "tenant", "client"); !errors.Is(err, failure) {
		t.Fatalf("delete: %v", err)
	}
}
