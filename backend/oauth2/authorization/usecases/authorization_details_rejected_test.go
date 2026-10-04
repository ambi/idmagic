package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-OAUTH2-053: 登録済みのクライアントの認可と PAR の要求の authorization_details を拒否したときだけ、理由 invalid_authorization_details の AuthorizationDetailsRejected が記録され、未登録のクライアントの要求と受理した要求では記録されないことを固定する。
func TestAuthorizationDetailsRefusalEmitsRejected(t *testing.T) {
	ctx := tenantContext()
	types := oauth2memory.NewAuthorizationDetailTypeRepository()
	seedPaymentType(types)
	var emitted []spec.DomainEvent
	record := func(event spec.DomainEvent) { emitted = append(emitted, event) }
	rejections := func() []*domain.AuthorizationDetailsRejected {
		var found []*domain.AuthorizationDetailsRejected
		for _, event := range emitted {
			if rejected, ok := event.(*domain.AuthorizationDetailsRejected); ok {
				found = append(found, rejected)
			}
		}
		return found
	}
	wantOneRejection := func(t *testing.T, err error, clientID string) {
		t.Helper()
		if errorCode(err) != "invalid_authorization_details" {
			t.Fatalf("error = %v, want invalid_authorization_details", err)
		}
		got := rejections()
		if len(got) != 1 || got[0].TenantID != tenancydomain.DefaultTenantID || got[0].ClientID != clientID ||
			got[0].Reason != "invalid_authorization_details" || got[0].At.IsZero() {
			t.Fatalf("rejections = %+v, want one for %s with reason invalid_authorization_details", got, clientID)
		}
	}

	authorizeDeps := newAuthorizeDeps(false)
	authorizeDeps.AuthzDetailTypeRepo = types
	authorizeDeps.Emit = record
	authorize := func(raw, clientID string) error {
		emitted = nil
		in := validAuthorizeInput()
		in.ClientID = clientID
		in.AuthorizationDetailsRaw = raw
		_, err := Authorize(ctx, authorizeDeps, in)
		return err
	}

	t.Run("authorize with an unregistered type", func(t *testing.T) {
		wantOneRejection(t, authorize(`[{"type":"data_access"}]`, "client"), "client")
	})
	t.Run("authorize with malformed JSON", func(t *testing.T) {
		wantOneRejection(t, authorize(`{not json`, "client"), "client")
	})
	t.Run("authorize from an unregistered client", func(t *testing.T) {
		err := authorize(`{not json`, "unknown-client")
		if errorCode(err) != "invalid_client" || len(rejections()) != 0 {
			t.Fatalf("error = %v, rejections = %+v; want invalid_client and none", err, rejections())
		}
	})
	t.Run("authorize with accepted details", func(t *testing.T) {
		err := authorize(rawDetails(t, []spec.AuthorizationDetail{paymentDetail(100, "initiate")}), "client")
		if err != nil || len(rejections()) != 0 {
			t.Fatalf("error = %v, rejections = %+v; want acceptance without a rejection", err, rejections())
		}
	})

	clientRepo := oauth2memory.NewClientRepository()
	clientRepo.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: "par-client",
		RedirectURIs: []string{"https://example.com/cb"}, GrantTypes: []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode},
	})
	parDeps := PARDeps{ClientRepo: clientRepo, Store: oauth2memory.NewPARStore(), AuthzDetailTypeRepo: types, Emit: record}
	push := func(raw string) error {
		emitted = nil
		_, err := PushAuthorizationRequest(ctx, parDeps, PARInput{
			ClientID:   "par-client",
			Parameters: map[string]string{"response_type": "code", "scope": "openid", "authorization_details": raw},
		}, time.Now().UTC())
		return err
	}

	t.Run("PAR with a schema violation", func(t *testing.T) {
		wantOneRejection(t, push(`[{"type":"payment_initiation","actions":["initiate"]}]`), "par-client")
	})
	t.Run("PAR with malformed JSON", func(t *testing.T) {
		wantOneRejection(t, push(`[`), "par-client")
	})
	t.Run("a registry failure is not a refusal", func(t *testing.T) {
		healthy := parDeps.AuthzDetailTypeRepo
		parDeps.AuthzDetailTypeRepo = failingDetailTypeRepository{}
		defer func() { parDeps.AuthzDetailTypeRepo = healthy }()
		err := push(`[{"type":"payment_initiation"}]`)
		if !errors.Is(err, errDetailTypeStore) || len(rejections()) != 0 {
			t.Fatalf("error = %v, rejections = %+v; want the store error and none", err, rejections())
		}
	})
}

var errDetailTypeStore = errors.New("detail type store unavailable")

// failingDetailTypeRepository は種類の保管先の障害を再現する。
type failingDetailTypeRepository struct {
	ports.AuthorizationDetailTypeRepository
}

func (failingDetailTypeRepository) FindByType(context.Context, string, string) (*domain.AuthorizationDetailType, error) {
	return nil, errDetailTypeStore
}
