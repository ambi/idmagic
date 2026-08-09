package support_http

// Opaque keyset pagination cursor codec (ADR-159): encode/verify a cursor as
// HMAC-SHA256(payload) over a base64url JSON payload. The cursor binds
// tenant_id, a caller-supplied query fingerprint (sort/filter), the boundary
// keyset value, a direction, and a format version, so a cursor cannot be
// replayed against a different tenant or a changed sort/filter. New cursors do
// not expire; version-zero cursors retain their legacy expiry during migration.
// Colocated with pagination.go /
// pagination_request.go rather than split into its own package: this codec
// has exactly one caller (ParsePageRequest/SetPageLinks below) and isn't a
// general-purpose crypto primitive reused across bounded contexts the way
// security/tokens_jose or security/passwords_argon2id are.

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalidCursor is returned (possibly wrapped) for any cursor that fails to
// decode: malformed token, bad signature, expired, or tenant/query mismatch.
// Callers map it to SCL InvalidRequestError.
var ErrInvalidCursor = errors.New("support_http: invalid cursor")

// Cursor is the decoded, unsigned content of a keyset pagination cursor.
type Cursor struct {
	Version   int           `json:"v,omitempty"`
	TenantID  string        `json:"tid"`
	QueryHash string        `json:"q"`
	After     string        `json:"a"`
	Direction PageDirection `json:"d,omitempty"`
	ExpiresAt time.Time     `json:"exp"`
}

// MarshalJSON omits a zero expiry. time.Time's own marshaler otherwise emits
// year-one even with omitempty, which would make new cursors look expiring.
func (cur Cursor) MarshalJSON() ([]byte, error) {
	type cursorWire struct {
		Version   int           `json:"v,omitempty"`
		TenantID  string        `json:"tid"`
		QueryHash string        `json:"q"`
		After     string        `json:"a"`
		Direction PageDirection `json:"d,omitempty"`
		ExpiresAt *time.Time    `json:"exp,omitempty"`
	}
	var expiresAt *time.Time
	if !cur.ExpiresAt.IsZero() {
		expiresAt = &cur.ExpiresAt
	}
	return json.Marshal(cursorWire{
		Version: cur.Version, TenantID: cur.TenantID, QueryHash: cur.QueryHash,
		After: cur.After, Direction: cur.Direction, ExpiresAt: expiresAt,
	})
}

const cursorVersion = 2

// PageDirection says which side of the signed keyset boundary to read.
type PageDirection string

const (
	PageForward  PageDirection = "forward"
	PageBackward PageDirection = "backward"
)

// CursorCodec signs and verifies Cursor values with a single symmetric secret.
type CursorCodec struct {
	secret []byte
}

func NewCursorCodec(secret []byte) *CursorCodec {
	return &CursorCodec{secret: secret}
}

// Encode signs cur and returns an opaque, URL-safe token.
func (c *CursorCodec) Encode(cur Cursor) (string, error) {
	payload, err := json.Marshal(cur)
	if err != nil {
		return "", fmt.Errorf("support_http: marshal cursor: %w", err)
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	sig := c.sign(encodedPayload)
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// Decode verifies the token's signature and expiry and returns the embedded Cursor.
func (c *CursorCodec) Decode(token string) (Cursor, error) {
	encodedPayload, encodedSig, ok := strings.Cut(token, ".")
	if !ok || encodedPayload == "" || encodedSig == "" {
		return Cursor{}, fmt.Errorf("%w: malformed token", ErrInvalidCursor)
	}
	sig, err := base64.RawURLEncoding.DecodeString(encodedSig)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: decode signature: %w", ErrInvalidCursor, err)
	}
	if subtle.ConstantTimeCompare(sig, c.sign(encodedPayload)) != 1 {
		return Cursor{}, fmt.Errorf("%w: signature mismatch", ErrInvalidCursor)
	}
	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: decode payload: %w", ErrInvalidCursor, err)
	}
	var cur Cursor
	if err := json.Unmarshal(payload, &cur); err != nil {
		return Cursor{}, fmt.Errorf("%w: unmarshal payload: %w", ErrInvalidCursor, err)
	}
	if !cur.ExpiresAt.IsZero() && time.Now().After(cur.ExpiresAt) {
		return Cursor{}, fmt.Errorf("%w: expired", ErrInvalidCursor)
	}
	if err := normalizeCursorVersion(&cur); err != nil {
		return Cursor{}, err
	}
	return cur, nil
}

func normalizeCursorVersion(cur *Cursor) error {
	switch cur.Version {
	case 0:
		if cur.Direction != "" {
			return fmt.Errorf("%w: legacy cursor has direction", ErrInvalidCursor)
		}
		cur.Direction = PageForward
	case cursorVersion:
		if !cur.ExpiresAt.IsZero() {
			return fmt.Errorf("%w: version %d cursor has expiry", ErrInvalidCursor, cursorVersion)
		}
		if cur.Direction != PageForward && cur.Direction != PageBackward {
			return fmt.Errorf("%w: invalid direction", ErrInvalidCursor)
		}
	default:
		return fmt.Errorf("%w: unsupported version", ErrInvalidCursor)
	}
	return nil
}

// DecodeForQuery decodes token and additionally requires it to match tenantID
// and queryHash (the caller's fingerprint of the current sort/filter), rejecting
// cursors issued for a different tenant or a since-changed query. On success it
// returns the embedded keyset ("after") value.
func (c *CursorCodec) DecodeForQuery(token, tenantID, queryHash string) (string, error) {
	cur, err := c.DecodeCursorForQuery(token, tenantID, queryHash)
	if err != nil {
		return "", err
	}
	return cur.After, nil
}

// DecodeCursorForQuery returns the full validated cursor, including its
// normalized direction. Legacy version-zero cursors are forward cursors.
func (c *CursorCodec) DecodeCursorForQuery(token, tenantID, queryHash string) (Cursor, error) {
	cur, err := c.Decode(token)
	if err != nil {
		return Cursor{}, err
	}
	if cur.TenantID != tenantID {
		return Cursor{}, fmt.Errorf("%w: tenant mismatch", ErrInvalidCursor)
	}
	if cur.QueryHash != queryHash {
		return Cursor{}, fmt.Errorf("%w: query mismatch", ErrInvalidCursor)
	}
	return cur, nil
}

func (c *CursorCodec) sign(encodedPayload string) []byte {
	mac := hmac.New(sha256.New, c.secret)
	mac.Write([]byte(encodedPayload))
	return mac.Sum(nil)
}
