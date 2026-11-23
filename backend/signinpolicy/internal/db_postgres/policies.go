package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"
	"github.com/jackc/pgx/v5"
)

// SignInPolicyRepository は Application sign-in policy を PostgreSQL に永続化する。
type SignInPolicyRepository struct{ Pool sharedpg.DB }

func (r *SignInPolicyRepository) Save(ctx context.Context, policy *policydomain.AppSignInPolicy) error {
	_, err := r.Update(ctx, policy.TenantID, policy.ApplicationID, func(_ *policydomain.AppSignInPolicy) (*policydomain.AppSignInPolicy, error) { return policy, nil })
	return err
}

func (r *SignInPolicyRepository) Update(ctx context.Context, tenantID, applicationID string, change func(*policydomain.AppSignInPolicy) (*policydomain.AppSignInPolicy, error)) (*policydomain.AppSignInPolicy, error) {
	return sharedpg.InTransaction(ctx, r.Pool, func(tx pgx.Tx) (*policydomain.AppSignInPolicy, error) {
		if _, err := New(tx).LockApplicationForPolicy(ctx, LockApplicationForPolicyParams{TenantID: tenantID, ID: applicationID}); err != nil {
			return nil, err
		}
		bound := &SignInPolicyRepository{Pool: sharedpg.BindTransaction(tx)}
		current, err := bound.Get(ctx, tenantID, applicationID)
		if err != nil {
			return nil, err
		}
		updated, err := change(current)
		if err != nil {
			return nil, err
		}
		return updated, bound.save(ctx, updated)
	})
}

func signInPolicyFromFields(tenantID, applicationID string, rules []byte, createdAt, updatedAt time.Time) (*policydomain.AppSignInPolicy, error) {
	policy := &policydomain.AppSignInPolicy{
		TenantID: tenantID, ApplicationID: applicationID,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
		Rules: []policydomain.SignInRule{},
	}
	if len(rules) > 0 {
		if err := json.Unmarshal(rules, &policy.Rules); err != nil {
			return nil, err
		}
	}
	return policy, nil
}

func (r *SignInPolicyRepository) Get(ctx context.Context, tenantID, applicationID string) (*policydomain.AppSignInPolicy, error) {
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

func (r *SignInPolicyRepository) ListAll(ctx context.Context, tenantID string) ([]*policydomain.AppSignInPolicy, error) {
	rows, err := New(r.Pool).ListAppSignInPoliciesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*policydomain.AppSignInPolicy, 0, len(rows))
	for _, row := range rows {
		policy, err := signInPolicyFromFields(row.TenantID, row.ApplicationID, row.Rules, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, policy)
	}
	return out, nil
}

func (r *SignInPolicyRepository) save(ctx context.Context, policy *policydomain.AppSignInPolicy) error {
	rules := policy.Rules
	if rules == nil {
		rules = []policydomain.SignInRule{}
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

func (r *DefaultSignInPolicyRepository) Save(ctx context.Context, policy *policydomain.TenantDefaultSignInPolicy) error {
	_, err := r.Update(ctx, policy.TenantID, func(_ *policydomain.TenantDefaultSignInPolicy) (*policydomain.TenantDefaultSignInPolicy, error) {
		return policy, nil
	})
	return err
}

func (r *DefaultSignInPolicyRepository) Update(ctx context.Context, tenantID string, change func(*policydomain.TenantDefaultSignInPolicy) (*policydomain.TenantDefaultSignInPolicy, error)) (*policydomain.TenantDefaultSignInPolicy, error) {
	return sharedpg.InTransaction(ctx, r.Pool, func(tx pgx.Tx) (*policydomain.TenantDefaultSignInPolicy, error) {
		if _, err := New(tx).LockTenantForDefaultPolicy(ctx, tenantID); err != nil {
			return nil, err
		}
		bound := &DefaultSignInPolicyRepository{Pool: sharedpg.BindTransaction(tx)}
		current, err := bound.Get(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		updated, err := change(current)
		if err != nil {
			return nil, err
		}
		return updated, bound.save(ctx, updated)
	})
}

func (r *DefaultSignInPolicyRepository) Get(ctx context.Context, tenantID string) (*policydomain.TenantDefaultSignInPolicy, error) {
	row, err := New(r.Pool).GetTenantDefaultSignInPolicy(ctx, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	policy := &policydomain.TenantDefaultSignInPolicy{
		TenantID: row.TenantID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		Rules: []policydomain.SignInRule{},
	}
	if len(row.Rules) > 0 {
		if err := json.Unmarshal(row.Rules, &policy.Rules); err != nil {
			return nil, err
		}
	}
	return policy, nil
}

func (r *DefaultSignInPolicyRepository) save(ctx context.Context, policy *policydomain.TenantDefaultSignInPolicy) error {
	rules := policy.Rules
	if rules == nil {
		rules = []policydomain.SignInRule{}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return err
	}
	return New(r.Pool).UpsertTenantDefaultSignInPolicy(ctx, UpsertTenantDefaultSignInPolicyParams{
		TenantID: policy.TenantID, Rules: encoded, CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt,
	})
}
