package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/signingkeys/internal/keys_vault"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5"
)

type VaultKeyStateStore struct{ Pool postgres.DB }

func loadVaultState(ctx context.Context, q *Queries) (keys_vault.KeySetState, error) {
	payload, err := q.GetVaultKeySet(ctx, GetVaultKeySetParams{TenantID: tenantports.TenantID(ctx), KeyUsage: string(signingports.KeyUsage(ctx)), ScopeID: signingports.KeyScope(ctx)})
	var state keys_vault.KeySetState
	if errors.Is(err, pgx.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	err = json.Unmarshal(payload, &state)
	return state, err
}

func (s VaultKeyStateStore) Load(ctx context.Context) (keys_vault.KeySetState, error) {
	return loadVaultState(ctx, New(s.Pool))
}
func (s VaultKeyStateStore) Healthy(ctx context.Context) bool { return s.Pool.Ping(ctx) == nil }
func (s VaultKeyStateStore) Update(ctx context.Context, update func(*keys_vault.KeySetState) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	if err := lockKeySet(ctx, q); err != nil {
		return err
	}
	state, err := loadVaultState(ctx, q)
	if err != nil {
		return err
	}
	if err := update(&state); err != nil {
		return err
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err := q.SaveVaultKeySet(ctx, SaveVaultKeySetParams{TenantID: tenantports.TenantID(ctx), KeyUsage: string(signingports.KeyUsage(ctx)), ScopeID: signingports.KeyScope(ctx), State: payload}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
