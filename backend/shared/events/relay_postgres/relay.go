package relay_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	eventports "github.com/ambi/idmagic/backend/shared/events/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

type outboxRecord struct {
	ID        int64
	EventType string
	Topic     string
	Payload   []byte
}

// Relay は outbox テーブルを skip-locked で drain し、Publisher へ転送する
// transport 中立のリレー。
type Relay struct {
	DB           sharedpg.DB
	Pub          eventports.Publisher
	PollInterval time.Duration
	BatchSize    int
}

// NewRelay は既定のポーリング間隔とバッチサイズで Relay を組み立てる。
func NewRelay(db sharedpg.DB, pub eventports.Publisher) *Relay {
	return &Relay{DB: db, Pub: pub, PollInterval: 200 * time.Millisecond, BatchSize: 100}
}

func (r *Relay) Close() { r.Pub.Close() }

// Run は ctx がキャンセルされるまで PollInterval 間隔で Tick を回す。
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.PollInterval)
	defer ticker.Stop()
	for {
		if err := r.Tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Tick は未 published の outbox 行を 1 バッチ drain し、Publisher へ転送する。
// FOR UPDATE SKIP LOCKED により複数 relay が競合せず分担できる。
func (r *Relay) Tick(ctx context.Context) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT id,event_type,topic,payload FROM outbox
WHERE published_at IS NULL ORDER BY id FOR UPDATE SKIP LOCKED LIMIT $1`, r.BatchSize)
	if err != nil {
		return err
	}
	var batch []outboxRecord
	for rows.Next() {
		var rec outboxRecord
		if err := rows.Scan(&rec.ID, &rec.EventType, &rec.Topic, &rec.Payload); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, rec)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(batch) == 0 {
		return tx.Commit(ctx)
	}

	var published []int64
	for _, rec := range batch {
		msg := eventports.OutboxMessage{
			Topic:     rec.Topic,
			Key:       partitionKey(rec.EventType, rec.Payload),
			Payload:   rec.Payload,
			EventType: rec.EventType,
			OutboxID:  rec.ID,
		}
		if err := r.Pub.Publish(ctx, msg); err != nil {
			if _, updateErr := tx.Exec(ctx, `UPDATE outbox SET attempts=attempts+1,last_error=$1,updated_at=now() WHERE id=$2`,
				truncate(err.Error(), 500), rec.ID); updateErr != nil {
				return updateErr
			}
			continue
		}
		published = append(published, rec.ID)
	}
	if len(published) > 0 {
		if _, err := tx.Exec(ctx, `UPDATE outbox SET published_at=now(),published_to=$2,
attempts=attempts+1,last_error=NULL,updated_at=now() WHERE id=ANY($1)`, published, r.Pub.Name()); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// partitionKey はイベント種別ごとに ordering/partition key となる集約識別子を
// payload から取り出す。transport によらず同じキーを用いて per-aggregate ordering を保つ。
func partitionKey(eventType string, payload []byte) string {
	var value map[string]any
	if json.Unmarshal(payload, &value) != nil {
		return ""
	}
	fields := map[string][]string{
		"ClientRegistered":             {"ClientID", "clientId"},
		"AdminOAuth2ClientCreated":     {"ClientID", "clientId"},
		"AdminOAuth2ClientUpdated":     {"ClientID", "clientId"},
		"AdminOAuth2ClientDeleted":     {"ClientID", "clientId"},
		"UserAuthenticated":            {"UserID", "userId"},
		"AuthenticationFailed":         {"Username", "username"},
		"LoginThrottled":               {"KeyHash", "keyHash"},
		"PasswordChanged":              {"UserID", "userId"},
		"ConsentGranted":               {"UserID", "userId"},
		"ConsentRevoked":               {"UserID", "userId"},
		"AuthorizationCodeIssued":      {"UserID", "userId"},
		"AuthorizationCodeRedeemed":    {"UserID", "userId"},
		"AccessTokenIssued":            {"UserID", "userId"},
		"RefreshTokenIssued":           {"FamilyID", "familyId"},
		"RefreshTokenRotated":          {"FamilyID", "familyId"},
		"TokenRevoked":                 {"TokenID", "tokenId"},
		"TokenIntrospected":            {"RSClientID", "rsClientId"},
		"TokenExchanged":               {"SubjectUserID", "subjectUserID"},
		"TokenExchangeRejected":        {"ActorUserID", "actorUserID"},
		"RefreshTokenReuseDetected":    {"FamilyID", "familyId"},
		"SigningKeyRotated":            {"NewKID", "newKid"},
		"PARStored":                    {"ClientID", "clientId"},
		"DeviceAuthorizationRequested": {"ClientID", "clientId"},
		"DeviceAuthorizationApproved":  {"ClientID", "clientId"},
		"DeviceAuthorizationDenied":    {"ClientID", "clientId"},
		"AgentRegistered":              {"AgentID", "agentId"},
		"AgentUpdated":                 {"AgentID", "agentId"},
		"AgentDisabled":                {"AgentID", "agentId"},
		"AgentEnabled":                 {"AgentID", "agentId"},
		"AgentKilled":                  {"AgentID", "agentId"},
		"AgentDeleted":                 {"AgentID", "agentId"},
		"AgentOwnerChanged":            {"AgentID", "agentId"},
		"AgentCredentialBound":         {"AgentID", "agentId"},
		"AgentCredentialUnbound":       {"AgentID", "agentId"},
	}
	for _, field := range fields[eventType] {
		if v, ok := value[field]; ok {
			return fmt.Sprint(v)
		}
	}
	return ""
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
