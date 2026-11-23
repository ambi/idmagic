// Package idempotency は業務の型に依存せず、要求の受付と暗号化した応答を保存する。
package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

const (
	Retention     = 24 * time.Hour
	recordContext = "http_idempotency"
	table         = "http_idempotency_records"
	field         = "response"
)

// Record はキーの平文と要求の本文を保持しない。応答がない記録は再実行の許可にならない。
type Record struct {
	ID, TenantID, PrincipalID, KeyHash, Fingerprint string
	CreatedAt, ExpiresAt                            time.Time
	KeyVersion                                      int32
	Ciphertext                                      []byte
}

type Response struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
}

type Cipher interface {
	Encrypt(ctx context.Context, tenantID, recordContext, table, recordID, field, plaintext string) (int, []byte, error)
	Decrypt(ctx context.Context, tenantID, recordContext, table, recordID, field string, keyVersion int, ciphertext []byte) (string, error)
}

// Store は受付を一意に確定し、受付ごとの ID に一致する応答だけを書き込む。
type Store interface {
	Claim(context.Context, Record, time.Time) (Record, bool, error)
	Complete(context.Context, Record) error
	PurgeExpired(context.Context, time.Time) error
	ListPending(context.Context, string, int32, int, []int32) ([]Record, error)
	ReplaceCipher(context.Context, Record, int32) (bool, error)
	PendingCount(context.Context, string, int32) (int, error)
}

type Ledger struct {
	store  Store
	cipher Cipher
}

func New(store Store, cipher Cipher) *Ledger { return &Ledger{store: store, cipher: cipher} }

func (l *Ledger) Claim(ctx context.Context, tenantID, principalID, key, fingerprint string, now time.Time) (Record, bool, error) {
	id, err := spec.NewUUIDv4()
	if err != nil {
		return Record{}, false, err
	}
	hash := sha256.Sum256([]byte(key))
	return l.store.Claim(ctx, Record{
		ID: id, TenantID: tenantID, PrincipalID: principalID, KeyHash: hex.EncodeToString(hash[:]),
		Fingerprint: fingerprint, CreatedAt: now, ExpiresAt: now.Add(Retention),
	}, now)
}

func (l *Ledger) Read(ctx context.Context, record Record) (Response, error) {
	plain, err := l.cipher.Decrypt(ctx, record.TenantID, recordContext, table, record.ID, field, int(record.KeyVersion), record.Ciphertext)
	if err != nil {
		return Response{}, err
	}
	var response Response
	err = json.Unmarshal([]byte(plain), &response)
	return response, err
}

func (l *Ledger) Complete(ctx context.Context, record Record, response Response) error {
	plain, err := json.Marshal(response)
	if err != nil {
		return err
	}
	version, ciphertext, err := l.cipher.Encrypt(ctx, record.TenantID, recordContext, table, record.ID, field, string(plain))
	if err != nil {
		return err
	}
	// DEK のバージョンは PostgreSQL の INTEGER と同じ範囲に制限する。
	if version <= 0 || int64(version) > 1<<31-1 {
		return errors.New("invalid response encryption key version")
	}
	record.KeyVersion, record.Ciphertext = int32(version), ciphertext
	return l.store.Complete(ctx, record)
}

func (l *Ledger) PurgeExpired(ctx context.Context, now time.Time) error {
	return l.store.PurgeExpired(ctx, now)
}

// ReencryptBatch と PendingCount は DataKeys の FieldMigrator に適合する。
// 期限切れの記録も削除されるまでは数え、参照が残る鍵の破棄を防ぐ。
func (l *Ledger) ReencryptBatch(ctx context.Context, tenantID string, activeVersion, batchSize int, excluded []int32) (int, error) {
	rows, err := l.store.ListPending(ctx, tenantID, int32(activeVersion), batchSize, excluded) //nolint:gosec // DEK のバージョンは INTEGER、件数は呼び出し元の上限以内。
	if err != nil {
		return 0, err
	}
	count := 0
	for _, row := range rows {
		plain, err := l.cipher.Decrypt(ctx, tenantID, recordContext, table, row.ID, field, int(row.KeyVersion), row.Ciphertext)
		if err != nil {
			return count, err
		}
		version, ciphertext, err := l.cipher.Encrypt(ctx, tenantID, recordContext, table, row.ID, field, plain)
		if err != nil {
			return count, err
		}
		old := row.KeyVersion
		row.KeyVersion, row.Ciphertext = int32(version), ciphertext //nolint:gosec // DEK のバージョンは INTEGER。
		updated, err := l.store.ReplaceCipher(ctx, row, old)
		if err != nil {
			return count, err
		}
		if updated {
			count++
		}
	}
	return count, nil
}

func (l *Ledger) PendingCount(ctx context.Context, tenantID string, activeVersion int) (int, error) {
	return l.store.PendingCount(ctx, tenantID, int32(activeVersion)) //nolint:gosec // DEK のバージョンは INTEGER。
}
