package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
)

// CrossAppAccessPolicyRepository は ID-JAG の許可を PostgreSQL に保つ。一意の組の重複は
// ON CONFLICT で判定し、読んでから書く競合を残さない。
type CrossAppAccessPolicyRepository struct{ Pool sharedpg.DB }

func issuePolicyFromRow(row *CrossAppAccessIssuePolicy) (*domain.CrossAppAccessIssuePolicy, error) {
	policy := &domain.CrossAppAccessIssuePolicy{
		TenantID: row.TenantID, ID: row.ID, ClientID: row.ClientID, Audience: row.Audience,
		RelyingPartyClientID: row.RelyingPartyClientID, TargetClientID: row.TargetClientID,
		ReleaseEmail: row.ReleaseEmail, CreatedAt: row.CreatedAt,
	}
	for _, field := range []struct {
		raw []byte
		out *[]string
	}{{row.Resources, &policy.Resources}, {row.Scopes, &policy.Scopes}, {row.AuthorizationDetailTypes, &policy.AuthorizationDetailTypes}} {
		if err := json.Unmarshal(field.raw, field.out); err != nil {
			return nil, err
		}
	}
	return policy, nil
}

func redeemPolicyFromRow(row *CrossAppAccessRedeemPolicy) (*domain.CrossAppAccessRedeemPolicy, error) {
	policy := &domain.CrossAppAccessRedeemPolicy{
		TenantID: row.TenantID, ID: row.ID, ProviderID: row.ProviderID, ClientID: row.ClientID, CreatedAt: row.CreatedAt,
	}
	if err := json.Unmarshal(row.Scopes, &policy.Scopes); err != nil {
		return nil, err
	}
	return policy, nil
}

func (r *CrossAppAccessPolicyRepository) ListIssuePolicies(ctx context.Context, tenantID string) ([]*domain.CrossAppAccessIssuePolicy, error) {
	rows, err := New(r.Pool).ListCrossAppAccessIssuePolicies(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.CrossAppAccessIssuePolicy, 0, len(rows))
	for _, row := range rows {
		policy, err := issuePolicyFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, policy)
	}
	return out, nil
}

func (r *CrossAppAccessPolicyRepository) FindIssuePolicy(ctx context.Context, tenantID, id string) (*domain.CrossAppAccessIssuePolicy, error) {
	row, err := New(r.Pool).GetCrossAppAccessIssuePolicy(ctx, GetCrossAppAccessIssuePolicyParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return issuePolicyFromRow(row)
}

func (r *CrossAppAccessPolicyRepository) CreateIssuePolicy(ctx context.Context, policy *domain.CrossAppAccessIssuePolicy) error {
	resources, err := json.Marshal(policy.Resources)
	if err != nil {
		return err
	}
	scopes, err := json.Marshal(policy.Scopes)
	if err != nil {
		return err
	}
	types := policy.AuthorizationDetailTypes
	if types == nil {
		types = []string{}
	}
	detailTypes, err := json.Marshal(types)
	if err != nil {
		return err
	}
	inserted, err := New(r.Pool).InsertCrossAppAccessIssuePolicy(ctx, InsertCrossAppAccessIssuePolicyParams{
		TenantID: policy.TenantID, ID: policy.ID, ClientID: policy.ClientID, Audience: policy.Audience,
		RelyingPartyClientID: policy.RelyingPartyClientID, TargetClientID: policy.TargetClientID,
		Resources: resources, Scopes: scopes, AuthorizationDetailTypes: detailTypes,
		ReleaseEmail: policy.ReleaseEmail, CreatedAt: policy.CreatedAt,
	})
	if err != nil {
		return err
	}
	if inserted == 0 {
		return ports.ErrCrossAppAccessPolicyExists
	}
	return nil
}

func (r *CrossAppAccessPolicyRepository) DeleteIssuePolicy(ctx context.Context, tenantID, id string) error {
	return New(r.Pool).DeleteCrossAppAccessIssuePolicy(ctx, DeleteCrossAppAccessIssuePolicyParams{TenantID: tenantID, ID: id})
}

func (r *CrossAppAccessPolicyRepository) ListRedeemPolicies(ctx context.Context, tenantID string) ([]*domain.CrossAppAccessRedeemPolicy, error) {
	rows, err := New(r.Pool).ListCrossAppAccessRedeemPolicies(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.CrossAppAccessRedeemPolicy, 0, len(rows))
	for _, row := range rows {
		policy, err := redeemPolicyFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, policy)
	}
	return out, nil
}

func (r *CrossAppAccessPolicyRepository) FindRedeemPolicy(ctx context.Context, tenantID, id string) (*domain.CrossAppAccessRedeemPolicy, error) {
	row, err := New(r.Pool).GetCrossAppAccessRedeemPolicy(ctx, GetCrossAppAccessRedeemPolicyParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return redeemPolicyFromRow(row)
}

func (r *CrossAppAccessPolicyRepository) FindRedeemPolicyFor(ctx context.Context, tenantID, providerID, clientID string) (*domain.CrossAppAccessRedeemPolicy, error) {
	row, err := New(r.Pool).GetCrossAppAccessRedeemPolicyFor(ctx, GetCrossAppAccessRedeemPolicyForParams{TenantID: tenantID, ProviderID: providerID, ClientID: clientID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return redeemPolicyFromRow(row)
}

func (r *CrossAppAccessPolicyRepository) CreateRedeemPolicy(ctx context.Context, policy *domain.CrossAppAccessRedeemPolicy) error {
	scopes, err := json.Marshal(policy.Scopes)
	if err != nil {
		return err
	}
	inserted, err := New(r.Pool).InsertCrossAppAccessRedeemPolicy(ctx, InsertCrossAppAccessRedeemPolicyParams{
		TenantID: policy.TenantID, ID: policy.ID, ProviderID: policy.ProviderID, ClientID: policy.ClientID,
		Scopes: scopes, CreatedAt: policy.CreatedAt,
	})
	if err != nil {
		return err
	}
	if inserted == 0 {
		return ports.ErrCrossAppAccessPolicyExists
	}
	return nil
}

func (r *CrossAppAccessPolicyRepository) DeleteRedeemPolicy(ctx context.Context, tenantID, id string) error {
	return New(r.Pool).DeleteCrossAppAccessRedeemPolicy(ctx, DeleteCrossAppAccessRedeemPolicyParams{TenantID: tenantID, ID: id})
}
