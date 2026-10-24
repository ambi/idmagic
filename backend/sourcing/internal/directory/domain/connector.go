// Package domain は Directory Connector の登録と、その状態遷移を定める。
package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// ConnectorStatus は Directory Connector の状態。
type ConnectorStatus string

const (
	StatusAwaitingEnrollment ConnectorStatus = "awaiting_enrollment"
	StatusActive             ConnectorStatus = "active"
	StatusDisabled           ConnectorStatus = "disabled"
)

// EnrollmentLifetime は登録トークンの有効期間 (REQ-SOURCING-008)。
const EnrollmentLifetime = time.Hour

var (
	ErrDisplayNameRequired = errors.New("directory connector: display_name is required")
	// ErrEnrollmentUnavailable は、登録待ちでないか、登録トークンの期限が切れていることを表す。
	// 呼び出し元へは区別せずに invalid_token として返す。
	ErrEnrollmentUnavailable = errors.New("directory connector: enrollment is not available")
	ErrConnectorDisabled     = errors.New("directory connector: connector is disabled")
)

// Connector は、閉域のディレクトリから取り込む Directory Connector の IdMagic 側の登録。
// Revision は保存のたびに一つ進み、Repository が同時の更新を検出するのに使う。
type Connector struct {
	TenantID            string
	ID                  string
	DisplayName         string
	Status              ConnectorStatus
	EnrollmentTokenHash string
	EnrollmentExpiresAt time.Time
	PublicKey           map[string]any
	KeyID               string
	CreatedAt           time.Time
	EnrolledAt          *time.Time
	LastSeenAt          *time.Time
	DisabledAt          *time.Time
	Revision            int64
}

// NewConnector は、登録トークンのハッシュを持つ登録待ちの Connector を作る。
func NewConnector(tenantID, id, displayName, tokenHash string, now time.Time) (Connector, error) {
	name := strings.TrimSpace(displayName)
	if name == "" {
		return Connector{}, ErrDisplayNameRequired
	}
	if err := spec.CheckMaxChars("display_name", name, spec.LengthDisplayName); err != nil {
		return Connector{}, err
	}
	return Connector{
		TenantID: tenantID, ID: id, DisplayName: name, Status: StatusAwaitingEnrollment,
		EnrollmentTokenHash: tokenHash, EnrollmentExpiresAt: now.Add(EnrollmentLifetime),
		CreatedAt: now, Revision: 1,
	}, nil
}

// Enroll は、登録待ちで期限内の Connector に公開鍵を登録して Active にする。
// 登録トークンの照合は Repository の検索が済ませている。
func (c Connector) Enroll(publicKey map[string]any, keyID string, now time.Time) (Connector, error) {
	if c.Status != StatusAwaitingEnrollment || now.After(c.EnrollmentExpiresAt) {
		return Connector{}, ErrEnrollmentUnavailable
	}
	c.Status = StatusActive
	c.PublicKey = publicKey
	c.KeyID = keyID
	c.EnrollmentTokenHash = ""
	c.EnrollmentExpiresAt = time.Time{}
	c.EnrolledAt = &now
	c.Revision++
	return c, nil
}

// ReissueEnrollment は、登録済みの鍵と以前の登録トークンを破棄して登録待ちに戻す。
func (c Connector) ReissueEnrollment(tokenHash string, now time.Time) (Connector, error) {
	if c.Status == StatusDisabled {
		return Connector{}, ErrConnectorDisabled
	}
	c.Status = StatusAwaitingEnrollment
	c.PublicKey = nil
	c.KeyID = ""
	c.EnrolledAt = nil
	c.EnrollmentTokenHash = tokenHash
	c.EnrollmentExpiresAt = now.Add(EnrollmentLifetime)
	c.Revision++
	return c, nil
}

// Disable は Connector を終端の Disabled にする。二つ目の値は状態を変えたかどうかで、
// 無効化済みの Connector には false を返し、何も変えない。
func (c Connector) Disable(now time.Time) (Connector, bool) {
	if c.Status == StatusDisabled {
		return c, false
	}
	c.Status = StatusDisabled
	c.EnrollmentTokenHash = ""
	c.EnrollmentExpiresAt = time.Time{}
	c.DisabledAt = &now
	c.Revision++
	return c, true
}
