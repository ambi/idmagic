// Package db_postgres は TokenVault の PostgreSQL の保存先である。秘密は FieldCipher で
// レコードに束縛して暗号化する (REQ-TOKENVAULT-003)。
package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

const (
	recordContext   = "tokenvault"
	connectorsTable = "token_vault_connectors"
	connectionTable = "token_vault_connections"
	attemptsTable   = "token_vault_attempts"
	secretField     = "client_secret"
	accessField     = "access_token"
	refreshField    = "refresh_token"
	verifierField   = "code_verifier"
)

// SecretCipher は項目の暗号化である。実装は backend/datakeys.FieldCipher である。
type SecretCipher interface {
	Encrypt(ctx context.Context, tenantID, recordContext, table, recordID, field, plaintext string) (keyVersion int, ciphertext []byte, err error)
	Decrypt(ctx context.Context, tenantID, recordContext, table, recordID, field string, keyVersion int, ciphertext []byte) (plaintext string, err error)
}

// 列の値は UUID なので、それ以外を問い合わせると PostgreSQL が 22P02 で失敗する。
// パスやトークンの主体はそのままここへ届くので、存在しない行として扱う。
func validIDs(ids ...string) bool {
	for _, id := range ids {
		if uuid.Validate(id) != nil {
			return false
		}
	}
	return true
}

// ConnectorRepository は ports.ConnectorRepository の実装である。
type ConnectorRepository struct {
	Pool   sharedpg.DB
	Cipher SecretCipher
}

func (r *ConnectorRepository) encryptSecret(ctx context.Context, connector *domain.Connector) (int32, []byte, error) {
	version, ciphertext, err := r.Cipher.Encrypt(ctx, connector.TenantID, recordContext, connectorsTable, connector.ID, secretField, connector.ClientSecret)
	return int32(version), ciphertext, err //nolint:gosec // G115: DEK version is a small monotonic counter
}

// Create はコネクタを保存する。名前が重複すれば ErrConnectorConflict を返す。
func (r *ConnectorRepository) Create(ctx context.Context, c *domain.Connector) error {
	version, ciphertext, err := r.encryptSecret(ctx, c)
	if err != nil {
		return err
	}
	scopes, rules, err := connectorJSON(c)
	if err != nil {
		return err
	}
	inserted, err := New(r.Pool).InsertConnector(ctx, InsertConnectorParams{
		ID: c.ID, TenantID: c.TenantID, Name: c.Name, Status: string(c.Status),
		AuthorizationEndpoint: c.AuthorizationEndpoint, TokenEndpoint: c.TokenEndpoint, RevocationEndpoint: c.RevocationEndpoint,
		ClientID: c.ClientID, TokenEndpointAuthMethod: string(c.TokenEndpointAuthMethod),
		SecretKeyVersion: version, SecretCiphertext: ciphertext, AllowedScopes: scopes, Rules: rules,
		Version: c.Version, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	})
	if err != nil {
		return err
	}
	if inserted == 0 {
		return ports.ErrConnectorConflict
	}
	return nil
}

// Update は読んだ版が保存の値と一致するときだけ更新する。
func (r *ConnectorRepository) Update(ctx context.Context, c *domain.Connector) error {
	version, ciphertext, err := r.encryptSecret(ctx, c)
	if err != nil {
		return err
	}
	scopes, rules, err := connectorJSON(c)
	if err != nil {
		return err
	}
	updated, err := New(r.Pool).UpdateConnector(ctx, UpdateConnectorParams{
		TenantID: c.TenantID, ID: c.ID, Name: c.Name, Status: string(c.Status),
		AuthorizationEndpoint: c.AuthorizationEndpoint, TokenEndpoint: c.TokenEndpoint, RevocationEndpoint: c.RevocationEndpoint,
		ClientID: c.ClientID, TokenEndpointAuthMethod: string(c.TokenEndpointAuthMethod),
		SecretKeyVersion: version, SecretCiphertext: ciphertext, AllowedScopes: scopes, Rules: rules,
		UpdatedAt: c.UpdatedAt, Version: c.Version,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ports.ErrConnectorConflict
		}
		return err
	}
	if updated == 0 {
		return ports.ErrConnectorConflict
	}
	c.Version++
	return nil
}

// Find はコネクタを返す。なければ nil を返す。
func (r *ConnectorRepository) Find(ctx context.Context, tenantID, id string) (*domain.Connector, error) {
	if !validIDs(id) {
		return nil, nil
	}
	row, err := New(r.Pool).FindConnector(ctx, FindConnectorParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toDomain(ctx, row)
}

// List はテナントのコネクタを名前の順に返す。
func (r *ConnectorRepository) List(ctx context.Context, tenantID string) ([]*domain.Connector, error) {
	rows, err := New(r.Pool).ListConnectors(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Connector, 0, len(rows))
	for _, row := range rows {
		connector, err := r.toDomain(ctx, row)
		if err != nil {
			return nil, err
		}
		result = append(result, connector)
	}
	return result, nil
}

// Delete はコネクタを消す。接続と連携は外部キーで消える。
func (r *ConnectorRepository) Delete(ctx context.Context, tenantID, id string) (bool, error) {
	if !validIDs(id) {
		return false, nil
	}
	deleted, err := New(r.Pool).DeleteConnector(ctx, DeleteConnectorParams{TenantID: tenantID, ID: id})
	return deleted > 0, err
}

func (r *ConnectorRepository) toDomain(ctx context.Context, row *TokenVaultConnector) (*domain.Connector, error) {
	secret, err := r.Cipher.Decrypt(ctx, row.TenantID, recordContext, connectorsTable, row.ID, secretField, int(row.SecretKeyVersion), row.SecretCiphertext)
	if err != nil {
		return nil, err
	}
	connector := &domain.Connector{
		ID: row.ID, TenantID: row.TenantID, Name: row.Name, Status: domain.ConnectorStatus(row.Status),
		AuthorizationEndpoint: row.AuthorizationEndpoint, TokenEndpoint: row.TokenEndpoint, RevocationEndpoint: row.RevocationEndpoint,
		ClientID: row.ClientID, TokenEndpointAuthMethod: domain.TokenEndpointAuthMethod(row.TokenEndpointAuthMethod),
		ClientSecret: secret, Version: row.Version, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if err := json.Unmarshal(row.AllowedScopes, &connector.AllowedScopes); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.Rules, &connector.Rules); err != nil {
		return nil, err
	}
	return connector, nil
}

func connectorJSON(c *domain.Connector) (scopes, rules []byte, err error) {
	if scopes, err = marshalList(c.AllowedScopes); err != nil {
		return nil, nil, err
	}
	if c.Rules == nil {
		return scopes, []byte("[]"), nil
	}
	rules, err = json.Marshal(c.Rules)
	return scopes, rules, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func marshalList(values []string) ([]byte, error) {
	if values == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(values)
}

// ConnectionRepository は ports.ConnectionRepository の実装である。
type ConnectionRepository struct {
	Pool   sharedpg.DB
	Cipher SecretCipher
}

type sealedTokens struct {
	version pgtype.Int4
	access  []byte
	refresh []byte
}

// connectionRecordID は暗号文を束縛するレコードの識別である。再連携しても変わらない組を使う。
func connectionRecordID(connectorID, userID string) string {
	return connectorID + "/" + userID
}

// seal はアクセストークンと更新トークンを接続に束縛して暗号化する。2 項目は同じ版で揃える
// 必要があるので、ローテーションと重なって版が分かれたら暗号化し直す。
func seal(ctx context.Context, cipher SecretCipher, tenantID, recordID, access, refresh string) (sealedTokens, error) {
	version, accessCiphertext, err := cipher.Encrypt(ctx, tenantID, recordContext, connectionTable, recordID, accessField, access)
	if err != nil {
		return sealedTokens{}, err
	}
	sealed := sealedTokens{version: pgtype.Int4{Int32: int32(version), Valid: true}, access: accessCiphertext} //nolint:gosec // G115: DEK version is a small monotonic counter
	if refresh == "" {
		return sealed, nil
	}
	refreshVersion, refreshCiphertext, err := cipher.Encrypt(ctx, tenantID, recordContext, connectionTable, recordID, refreshField, refresh)
	if err != nil {
		return sealedTokens{}, err
	}
	if refreshVersion != version {
		return seal(ctx, cipher, tenantID, recordID, access, refresh)
	}
	sealed.refresh = refreshCiphertext
	return sealed, nil
}

func pgTime(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromPgTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func (r *ConnectionRepository) toDomain(ctx context.Context, row *TokenVaultConnection) (*domain.Connection, error) {
	connection := &domain.Connection{
		ID: row.ID, TenantID: row.TenantID, ConnectorID: row.ConnectorID, UserID: row.UserID,
		Status: domain.ConnectionStatus(row.Status), ExpiresAt: fromPgTime(row.ExpiresAt), TokenVersion: row.TokenVersion,
		LastUsedAt: fromPgTime(row.LastUsedAt), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if err := json.Unmarshal(row.GrantedScopes, &connection.GrantedScopes); err != nil {
		return nil, err
	}
	if !row.TokenKeyVersion.Valid {
		return connection, nil
	}
	var err error
	version := int(row.TokenKeyVersion.Int32)
	recordID := connectionRecordID(row.ConnectorID, row.UserID)
	if connection.AccessToken, err = r.Cipher.Decrypt(ctx, row.TenantID, recordContext, connectionTable, recordID, accessField, version, row.AccessTokenCiphertext); err != nil {
		return nil, err
	}
	if len(row.RefreshTokenCiphertext) > 0 {
		if connection.RefreshToken, err = r.Cipher.Decrypt(ctx, row.TenantID, recordContext, connectionTable, recordID, refreshField, version, row.RefreshTokenCiphertext); err != nil {
			return nil, err
		}
	}
	return connection, nil
}

// Find は接続を返す。なければ nil を返す。
func (r *ConnectionRepository) Find(ctx context.Context, tenantID, connectorID, userID string) (*domain.Connection, error) {
	if !validIDs(connectorID, userID) {
		return nil, nil
	}
	row, err := New(r.Pool).FindConnection(ctx, FindConnectionParams{TenantID: tenantID, ConnectorID: connectorID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toDomain(ctx, row)
}

// ListByUser は本人の接続を返す。一覧はトークンを使わないが、復号の失敗を隠さない。
func (r *ConnectionRepository) ListByUser(ctx context.Context, tenantID, userID string) ([]*domain.Connection, error) {
	if !validIDs(userID) {
		return []*domain.Connection{}, nil
	}
	rows, err := New(r.Pool).ListConnectionsByUser(ctx, ListConnectionsByUserParams{TenantID: tenantID, UserID: userID})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Connection, 0, len(rows))
	for _, row := range rows {
		connection, err := r.toDomain(ctx, row)
		if err != nil {
			return nil, err
		}
		result = append(result, connection)
	}
	return result, nil
}

// SaveLinked は組の行を作るか置き換える。
func (r *ConnectionRepository) SaveLinked(ctx context.Context, c *domain.Connection) error {
	sealed, err := seal(ctx, r.Cipher, c.TenantID, connectionRecordID(c.ConnectorID, c.UserID), c.AccessToken, c.RefreshToken)
	if err != nil {
		return err
	}
	scopes, err := marshalList(c.GrantedScopes)
	if err != nil {
		return err
	}
	return New(r.Pool).SaveLinkedConnection(ctx, SaveLinkedConnectionParams{
		ID: c.ID, TenantID: c.TenantID, ConnectorID: c.ConnectorID, UserID: c.UserID, GrantedScopes: scopes,
		TokenKeyVersion: sealed.version, AccessTokenCiphertext: sealed.access, RefreshTokenCiphertext: sealed.refresh,
		ExpiresAt: pgTime(c.ExpiresAt), CreatedAt: c.UpdatedAt,
	})
}

// AcquireRefreshLease は更新のリースを取る。
func (r *ConnectionRepository) AcquireRefreshLease(ctx context.Context, tenantID, id string, tokenVersion int64, now, until time.Time) (bool, error) {
	acquired, err := New(r.Pool).AcquireRefreshLease(ctx, AcquireRefreshLeaseParams{
		TenantID: tenantID, ID: id, TokenVersion: tokenVersion,
		RefreshLeaseUntil:   pgtype.Timestamptz{Time: now, Valid: true},
		RefreshLeaseUntil_2: pgtype.Timestamptz{Time: until, Valid: true},
	})
	return acquired > 0, err
}

// ReleaseRefreshLease はリースを手放す。
func (r *ConnectionRepository) ReleaseRefreshLease(ctx context.Context, tenantID, id string, tokenVersion int64) error {
	return New(r.Pool).ReleaseRefreshLease(ctx, ReleaseRefreshLeaseParams{TenantID: tenantID, ID: id, TokenVersion: tokenVersion})
}

// StoreRefreshed は版が一致するときだけ更新の結果を保存する。
func (r *ConnectionRepository) StoreRefreshed(ctx context.Context, c *domain.Connection, tokenVersion int64) (bool, error) {
	sealed, err := seal(ctx, r.Cipher, c.TenantID, connectionRecordID(c.ConnectorID, c.UserID), c.AccessToken, c.RefreshToken)
	if err != nil {
		return false, err
	}
	scopes, err := marshalList(c.GrantedScopes)
	if err != nil {
		return false, err
	}
	stored, err := New(r.Pool).StoreRefreshedConnection(ctx, StoreRefreshedConnectionParams{
		TenantID: c.TenantID, ID: c.ID, TokenVersion: tokenVersion, GrantedScopes: scopes,
		TokenKeyVersion: sealed.version, AccessTokenCiphertext: sealed.access, RefreshTokenCiphertext: sealed.refresh,
		ExpiresAt: pgTime(c.ExpiresAt), UpdatedAt: c.UpdatedAt,
	})
	return stored > 0, err
}

// MarkReauthorizationRequired は版が一致するときだけトークンを消す。
func (r *ConnectionRepository) MarkReauthorizationRequired(ctx context.Context, tenantID, id string, tokenVersion int64, now time.Time) (bool, error) {
	changed, err := New(r.Pool).MarkConnectionReauthorizationRequired(ctx, MarkConnectionReauthorizationRequiredParams{
		TenantID: tenantID, ID: id, TokenVersion: tokenVersion, UpdatedAt: now,
	})
	return changed > 0, err
}

// TouchLastUsed は最後の利用の時刻を記録する。
func (r *ConnectionRepository) TouchLastUsed(ctx context.Context, tenantID, id string, now time.Time) error {
	return New(r.Pool).TouchConnectionLastUsed(ctx, TouchConnectionLastUsedParams{
		TenantID: tenantID, ID: id, LastUsedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
}

// Delete は接続を消して返す。行があれば消すことを先に確定し、復号の失敗で解除を妨げない。
func (r *ConnectionRepository) Delete(ctx context.Context, tenantID, connectorID, userID string) (*domain.Connection, error) {
	if !validIDs(connectorID, userID) {
		return nil, nil
	}
	row, err := New(r.Pool).DeleteConnection(ctx, DeleteConnectionParams{TenantID: tenantID, ConnectorID: connectorID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	connection, err := r.toDomain(ctx, row)
	if err != nil {
		return &domain.Connection{ID: row.ID, TenantID: row.TenantID, ConnectorID: row.ConnectorID, UserID: row.UserID}, nil //nolint:nilerr // 上流の失効を諦めるだけで、解除は成功である。
	}
	return connection, nil
}

// AttemptStore は ports.AttemptStore の実装である。
type AttemptStore struct {
	Pool   sharedpg.DB
	Cipher SecretCipher
}

// Save は連携を保存し、テナントの期限切れの連携を片付ける。
func (s *AttemptStore) Save(ctx context.Context, a *domain.ConnectionAttempt) error {
	version, ciphertext, err := s.Cipher.Encrypt(ctx, a.TenantID, recordContext, attemptsTable, a.ID, verifierField, a.CodeVerifier)
	if err != nil {
		return err
	}
	scopes, err := marshalList(a.RequestedScopes)
	if err != nil {
		return err
	}
	queries := New(s.Pool)
	if err := queries.DeleteExpiredAttempts(ctx, DeleteExpiredAttemptsParams{TenantID: a.TenantID, ExpiresAt: time.Now().UTC()}); err != nil {
		return err
	}
	return queries.SaveAttempt(ctx, SaveAttemptParams{
		ID: a.ID, TenantID: a.TenantID, ConnectorID: a.ConnectorID, UserID: a.UserID, StateHash: a.StateHash,
		VerifierKeyVersion: int32(version), VerifierCiphertext: ciphertext, RequestedScopes: scopes, ExpiresAt: a.ExpiresAt, //nolint:gosec // G115: DEK version is a small monotonic counter
	})
}

// Consume は一致する期限内の連携を原子的に消して返す。
func (s *AttemptStore) Consume(ctx context.Context, tenantID, connectorID, userID string, stateHash []byte, now time.Time) (*domain.ConnectionAttempt, error) {
	if !validIDs(connectorID, userID) {
		return nil, nil
	}
	row, err := New(s.Pool).ConsumeAttempt(ctx, ConsumeAttemptParams{
		TenantID: tenantID, ConnectorID: connectorID, UserID: userID, StateHash: stateHash, ExpiresAt: now,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	verifier, err := s.Cipher.Decrypt(ctx, row.TenantID, recordContext, attemptsTable, row.ID, verifierField, int(row.VerifierKeyVersion), row.VerifierCiphertext)
	if err != nil {
		return nil, err
	}
	attempt := &domain.ConnectionAttempt{
		ID: row.ID, TenantID: row.TenantID, ConnectorID: row.ConnectorID, UserID: row.UserID, StateHash: row.StateHash,
		CodeVerifier: verifier, ExpiresAt: row.ExpiresAt,
	}
	return attempt, json.Unmarshal(row.RequestedScopes, &attempt.RequestedScopes)
}

var (
	_ ports.ConnectorRepository  = (*ConnectorRepository)(nil)
	_ ports.ConnectionRepository = (*ConnectionRepository)(nil)
	_ ports.AttemptStore         = (*AttemptStore)(nil)
)
