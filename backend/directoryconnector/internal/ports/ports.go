// Package ports は、Directory Connector が読む LDAP、呼ぶ Connector API、手元の状態の抽象である。
package ports

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
)

// Directory は、同期の回ごとに LDAP へ接続する。Connect は TLS を確立して bind した後に返る。
type Directory interface {
	Connect(ctx context.Context) (DirectorySession, error)
}

type DirectorySession interface {
	// RootInfo は RootDSE から差分の位置に使う値を読む。AD でない方言では空を返す。
	RootInfo(ctx context.Context) (domain.RootInfo, error)
	// Search はページングして検索し、ページごとに page を呼ぶ。page の失敗で検索を止める。
	Search(ctx context.Context, baseDN, filter string, attributes []string, page func([]domain.Entry) error) error
	Close() error
}

// IngestionAPI は Connector API である。各呼び出しは新しい JWT で署名する。
type IngestionAPI interface {
	Heartbeat(ctx context.Context) error
	StartRun(ctx context.Context, mode domain.Mode) (runID string, err error)
	SendBatch(ctx context.Context, runID, batchID string, items []domain.Item) error
	SendComputerBatch(ctx context.Context, runID, batchID string, items []domain.ComputerItem) error
	CompleteRun(ctx context.Context, runID string, itemCount int) error
}

// APIError は Connector API が返した 2xx 以外の応答である。接続の失敗はこの型にならない。
type APIError struct {
	Status int
	// Type は Problem Details の type から `urn:idmagic:error:` を除いた値である。
	Type string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("connector api: status %d %s", e.Status, e.Type)
}

type CursorStore interface {
	// Load は保存した位置を返す。位置がなければ nil を返す。
	Load(ctx context.Context) (*domain.Cursor, error)
	Save(ctx context.Context, cursor domain.Cursor) error
}

// Enrollment は登録の応答のうち、以後の JWT に使う値である。
type Enrollment struct {
	ConnectorID string `json:"connector_id"`
	KeyID       string `json:"key_id"`
	Audience    string `json:"audience"`
}

type EnrollmentAPI interface {
	Enroll(ctx context.Context, token string, jwk map[string]any) (Enrollment, error)
}

// CredentialStore は秘密鍵と登録の情報を、所有者だけが読める形で一緒に置き換える。
type CredentialStore interface {
	SaveCredentials(ctx context.Context, key *ecdsa.PrivateKey, enrollment Enrollment) error
}
