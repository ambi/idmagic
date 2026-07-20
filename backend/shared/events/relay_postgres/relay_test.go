package relay_postgres

import (
	"context"
	"sync"
	"testing"

	eventports "github.com/ambi/idmagic/backend/shared/events/ports"
	publishersLog "github.com/ambi/idmagic/backend/shared/events/publishers_log"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestMain(m *testing.M) {
	pgtest.Main(m)
}

// fakePublisher は Relay の drain ループを transport 非依存に検証するためのテスト用 Publisher。
type fakePublisher struct {
	name string
	err  error
	mu   sync.Mutex
	got  []eventports.OutboxMessage
}

func (f *fakePublisher) Publish(_ context.Context, m eventports.OutboxMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.got = append(f.got, m)
	return nil
}

func (f *fakePublisher) Name() string { return f.name }
func (f *fakePublisher) Close()       {}

func resetOutbox(t *testing.T, db sharedpg.DB) {
	t.Helper()
	if _, err := db.Exec(context.Background(), "DELETE FROM outbox"); err != nil {
		t.Fatalf("reset outbox: %v", err)
	}
}

func insertOutbox(t *testing.T, db sharedpg.DB, eventType, topic, payload string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(context.Background(),
		`INSERT INTO outbox (event_type, topic, payload) VALUES ($1,$2,$3) RETURNING id`,
		eventType, topic, []byte(payload),
	).Scan(&id); err != nil {
		t.Fatalf("insert outbox: %v", err)
	}
	return id
}

// 成功時: 行が published としてマークされ published_to=Publisher.Name()、
// Publisher へ topic/key/payload/event_type/outbox_id が渡る。
func TestRelayTickPublishesAndMarks(t *testing.T) {
	db := pgtest.Require(t)
	resetOutbox(t, db)
	ctx := context.Background()

	id := insertOutbox(t, db, "AccessTokenIssued", "oauth2.token.v1", `{"userId":"u-123"}`)

	pub := &fakePublisher{name: "fake"}
	relay := NewRelay(db, pub)
	if err := relay.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	if len(pub.got) != 1 {
		t.Fatalf("published %d messages, want 1", len(pub.got))
	}
	msg := pub.got[0]
	if msg.Topic != "oauth2.token.v1" || msg.EventType != "AccessTokenIssued" || msg.OutboxID != id {
		t.Errorf("unexpected message: %+v", msg)
	}
	if msg.Key != "u-123" { // partitionKey extracts userId for AccessTokenIssued
		t.Errorf("ordering key = %q, want u-123", msg.Key)
	}

	var publishedTo string
	if err := db.QueryRow(ctx,
		"SELECT published_to FROM outbox WHERE id=$1 AND published_at IS NOT NULL", id,
	).Scan(&publishedTo); err != nil {
		t.Fatalf("row not marked published: %v", err)
	}
	if publishedTo != "fake" {
		t.Errorf("published_to = %q, want fake", publishedTo)
	}
}

// 失敗時: 行は未 published のまま、attempts 増加・last_error 記録。
func TestRelayTickFailureRecordsAttempt(t *testing.T) {
	db := pgtest.Require(t)
	resetOutbox(t, db)
	ctx := context.Background()

	id := insertOutbox(t, db, "AccessTokenIssued", "oauth2.token.v1", `{"userId":"u-1"}`)

	pub := &fakePublisher{name: "fake", err: context.DeadlineExceeded}
	relay := NewRelay(db, pub)
	if err := relay.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	var attempts int
	var publishedAt *string
	var lastError *string
	if err := db.QueryRow(ctx,
		"SELECT attempts, published_at::text, last_error FROM outbox WHERE id=$1", id,
	).Scan(&attempts, &publishedAt, &lastError); err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	if publishedAt != nil {
		t.Errorf("published_at = %v, want NULL", *publishedAt)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
	if lastError == nil || *lastError == "" {
		t.Error("last_error not recorded")
	}
}

// BatchSize は 1 tick で drain する行数を制限する。
func TestRelayTickRespectsBatchSize(t *testing.T) {
	db := pgtest.Require(t)
	resetOutbox(t, db)
	ctx := context.Background()

	for range 5 {
		insertOutbox(t, db, "AccessTokenIssued", "oauth2.token.v1", `{"userId":"u"}`)
	}

	pub := &fakePublisher{name: "fake"}
	relay := NewRelay(db, pub)
	relay.BatchSize = 2
	if err := relay.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if len(pub.got) != 2 {
		t.Fatalf("published %d messages, want 2 (batch size)", len(pub.got))
	}

	var remaining int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE published_at IS NULL").Scan(&remaining); err != nil {
		t.Fatalf("count remaining: %v", err)
	}
	if remaining != 3 {
		t.Errorf("remaining unpublished = %d, want 3", remaining)
	}
}

// 空の outbox でもエラーにならない。
func TestRelayTickEmpty(t *testing.T) {
	db := pgtest.Require(t)
	resetOutbox(t, db)
	pub := &fakePublisher{name: "fake"}
	if err := NewRelay(db, pub).Tick(context.Background()); err != nil {
		t.Fatalf("Tick on empty outbox: %v", err)
	}
	if len(pub.got) != 0 {
		t.Errorf("published %d messages on empty outbox, want 0", len(pub.got))
	}
}

// 各 Publisher アダプタは Name() を持ち published_to へ記録できる。
func TestPublisherNames(t *testing.T) {
	if got := publishersLog.NewLogPublisher().Name(); got != "log" {
		t.Errorf("LogPublisher.Name() = %q, want log", got)
	}
}
