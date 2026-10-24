package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/signinpolicy/domain"
	"github.com/jackc/pgx/v5"
)

// SignInPolicyRepository は Application sign-in policy を PostgreSQL に永続化する。
type SignInPolicyRepository struct{ Pool sharedpg.DB }

func signInPolicyFromFields(tenantID, applicationID string, rules []byte, createdAt, updatedAt time.Time) (*domain.AppSignInPolicy, error) {
	policy := &domain.AppSignInPolicy{
		TenantID: tenantID, ApplicationID: applicationID,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
		Rules: []domain.SignInRule{},
	}
	if len(rules) > 0 {
		if err := json.Unmarshal(rules, &policy.Rules); err != nil {
			return nil, err
		}
	}
	return policy, nil
}

func (r *SignInPolicyRepository) Get(ctx context.Context, tenantID, applicationID string) (*domain.AppSignInPolicy, error) {
	row, err := New(r.Pool).GetAppSignInPolicy(ctx, GetAppSignInPolicyParams{
		TenantID: tenantID, ApplicationID: applicationID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return signInPolicyFromFields(row.TenantID, row.ApplicationID, row.Rules, row.CreatedAt, row.UpdatedAt)
}

func (r *SignInPolicyRepository) ListAll(ctx context.Context, tenantID string) ([]*domain.AppSignInPolicy, error) {
	rows, err := New(r.Pool).ListAppSignInPoliciesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.AppSignInPolicy, 0, len(rows))
	for _, row := range rows {
		policy, err := signInPolicyFromFields(row.TenantID, row.ApplicationID, row.Rules, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, policy)
	}
	return out, nil
}

func (r *SignInPolicyRepository) Save(ctx context.Context, policy *domain.AppSignInPolicy) error {
	rules := policy.Rules
	if rules == nil {
		rules = []domain.SignInRule{}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return err
	}
	return New(r.Pool).UpsertAppSignInPolicy(ctx, UpsertAppSignInPolicyParams{
		ApplicationID: policy.ApplicationID, Rules: encoded,
		CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt,
	})
}

func (r *SignInPolicyRepository) Delete(ctx context.Context, tenantID, applicationID string) error {
	return New(r.Pool).DeleteAppSignInPolicy(ctx, DeleteAppSignInPolicyParams{
		TenantID: tenantID, ApplicationID: applicationID,
	})
}

// DefaultSignInPolicyRepository はテナント既定 sign-in policy を PostgreSQL に永続化する。
type DefaultSignInPolicyRepository struct{ Pool sharedpg.DB }

func (r *DefaultSignInPolicyRepository) Get(ctx context.Context, tenantID string) (*domain.TenantDefaultSignInPolicy, error) {
	row, err := New(r.Pool).GetTenantDefaultSignInPolicy(ctx, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	policy := &domain.TenantDefaultSignInPolicy{
		TenantID: row.TenantID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		Rules: []domain.SignInRule{},
	}
	if len(row.Rules) > 0 {
		if err := json.Unmarshal(row.Rules, &policy.Rules); err != nil {
			return nil, err
		}
	}
	return policy, nil
}

func (r *DefaultSignInPolicyRepository) Save(ctx context.Context, policy *domain.TenantDefaultSignInPolicy) error {
	rules := policy.Rules
	if rules == nil {
		rules = []domain.SignInRule{}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return err
	}
	return New(r.Pool).UpsertTenantDefaultSignInPolicy(ctx, UpsertTenantDefaultSignInPolicyParams{
		TenantID: policy.TenantID, Rules: encoded, CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt,
	})
}
