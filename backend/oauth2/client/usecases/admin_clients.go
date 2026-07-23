package usecases

// 管理者向け Client メタデータ操作 (Create / Update / Delete)。
// SCL OAuth2 bounded context の admin インターフェース群:
// CreateAdminOAuth2Client / UpdateAdminOAuth2Client / DeleteAdminOAuth2Client。

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

var ErrClientNotFound = errors.New("client not found")

type AdminOAuth2ClientDeps struct {
	ClientRepo oauthports.OAuth2ClientRepository
	Emit       func(spec.DomainEvent)
	// QuotaRepo enforces the tenant's Hard Quota on oauth2_clients (wi-160,
	// ADR-134), bridged into RegisterClient which owns the actual check.
	QuotaRepo tenantports.QuotaRepository
}

type CreateAdminOAuth2ClientInput struct {
	ActorUserID  string
	Registration RegisterClientInput
	Now          time.Time
}

func CreateAdminOAuth2Client(
	ctx context.Context,
	deps AdminOAuth2ClientDeps,
	in CreateAdminOAuth2ClientInput,
) (*RegisterClientResult, error) {
	// Emit is intentionally not bridged into RegisterClientDeps here: RegisterClient's
	// internal emit fires the dynamic-registration-specific ClientRegistered event,
	// which CreateAdminOAuth2Client must not duplicate (it emits its own
	// AdminOAuth2ClientCreated below). The quota check itself still runs (QuotaRepo is
	// bridged); on rejection we independently emit QuotaExceeded through our own Emit.
	result, err := RegisterClient(ctx, RegisterClientDeps{
		ClientRepo: deps.ClientRepo, QuotaRepo: deps.QuotaRepo,
	}, in.Registration, in.Now)
	if err != nil {
		if qErr, ok := errors.AsType[*tenancydomain.QuotaExceededError](err); ok {
			emit(deps.Emit, &tenancydomain.QuotaExceeded{At: adminNow(in.Now), TenantID: qErr.TenantID, Resource: qErr.Resource, HardLimit: true})
		}
		return nil, err
	}
	emit(deps.Emit, &domain.AdminOAuth2ClientCreated{
		At: adminNow(in.Now), TenantID: result.Client.TenantID, ActorUserID: in.ActorUserID, ClientID: result.Client.ClientID,
	})
	return result, nil
}

type UpdateAdminOAuth2ClientInput struct {
	ActorUserID     string
	ClientID        string
	ClientName      *string
	RedirectURIs    *[]string
	GrantTypes      *[]spec.GrantType
	ResponseTypes   *[]spec.ResponseType
	Scope           *string
	RequirePAR      *bool
	DpopBoundTokens *bool
	Now             time.Time
}

func UpdateAdminOAuth2Client(ctx context.Context, deps AdminOAuth2ClientDeps, in UpdateAdminOAuth2ClientInput) (*domain.OAuth2Client, error) {
	tenantID := tenancy.TenantID(ctx)
	client, err := deps.ClientRepo.FindByID(ctx, tenantID, in.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrClientNotFound
	}
	updated := *client
	changed := []string{}
	if in.ClientName != nil && !adminEqualOptionalString(client.ClientName, in.ClientName) {
		updated.ClientName = in.ClientName
		changed = append(changed, "client_name")
	}
	if in.RedirectURIs != nil && !slices.Equal(client.RedirectURIs, *in.RedirectURIs) {
		updated.RedirectURIs = slices.Clone(*in.RedirectURIs)
		changed = append(changed, "redirect_uris")
	}
	if in.GrantTypes != nil && !slices.Equal(client.GrantTypes, *in.GrantTypes) {
		updated.GrantTypes = slices.Clone(*in.GrantTypes)
		changed = append(changed, "grant_types")
	}
	if in.ResponseTypes != nil && !slices.Equal(client.ResponseTypes, *in.ResponseTypes) {
		updated.ResponseTypes = slices.Clone(*in.ResponseTypes)
		changed = append(changed, "response_types")
	}
	if in.Scope != nil && client.Scope != *in.Scope {
		updated.Scope = *in.Scope
		changed = append(changed, "scope")
	}
	if in.RequirePAR != nil && client.RequirePushedAuthorizationRequests != *in.RequirePAR {
		updated.RequirePushedAuthorizationRequests = *in.RequirePAR
		changed = append(changed, "require_pushed_authorization_requests")
	}
	if in.DpopBoundTokens != nil && client.DpopBoundAccessTokens != *in.DpopBoundTokens {
		updated.DpopBoundAccessTokens = *in.DpopBoundTokens
		changed = append(changed, "dpop_bound_access_tokens")
	}
	if len(changed) == 0 {
		return &updated, nil
	}
	updated.UpdatedAt = adminNow(in.Now)
	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if err := deps.ClientRepo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.AdminOAuth2ClientUpdated{
		At: adminNow(in.Now), TenantID: tenantID, ActorUserID: in.ActorUserID, ClientID: client.ClientID,
		ChangedFields: changed,
	})
	return &updated, nil
}

func DeleteAdminOAuth2Client(
	ctx context.Context,
	deps AdminOAuth2ClientDeps,
	actorUserID, clientID string,
	now time.Time,
) error {
	tenantID := tenancy.TenantID(ctx)
	client, err := deps.ClientRepo.FindByID(ctx, tenantID, clientID)
	if err != nil {
		return err
	}
	if client == nil {
		return ErrClientNotFound
	}
	if err := deps.ClientRepo.Delete(ctx, tenantID, clientID); err != nil {
		return err
	}
	if deps.QuotaRepo != nil {
		if err := tenancyusecases.DecrementQuota(ctx, deps.QuotaRepo, tenantID, tenancydomain.ResourceOAuth2Clients, 1); err != nil {
			return err
		}
	}
	emit(deps.Emit, &domain.AdminOAuth2ClientDeleted{
		At: adminNow(now), TenantID: tenantID, ActorUserID: actorUserID, ClientID: clientID,
	})
	return nil
}

func adminNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}

func adminEqualOptionalString(left, right *string) bool {
	return left == nil && right == nil ||
		left != nil && right != nil && *left == *right
}
