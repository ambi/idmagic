package support_http

// Opaque keyset pagination cursor codec. Version 3 keeps tenant and
// query fingerprints out of the wire payload and binds them as HMAC associated
// data. Legacy version-zero and version-two JSON cursors remain decodable.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("support_http: invalid cursor")

const (
	cursorVersion       = 3
	legacyCursorVersion = 2
	v3TagBytes          = 16
	v3MaxFieldBytes     = 4096
)

type PageDirection string

const (
	PageForward  PageDirection = "forward"
	PageBackward PageDirection = "backward"
)

type PageAnchor string

const (
	PageAnchorKeyset PageAnchor = ""
	PageAnchorEnd    PageAnchor = "end"
)

// Cursor is the decoded cursor content. TenantID and QueryHash are serialized
// only by legacy formats; v3 uses them solely as HMAC associated data.
type Cursor struct {
	Version   int           `json:"v,omitempty"`
	TenantID  string        `json:"tid"`
	QueryHash string        `json:"q"`
	After     string        `json:"a"`
	Direction PageDirection `json:"d,omitempty"`
	Anchor    PageAnchor    `json:"-"`
	Page      int           `json:"-"`
	ExpiresAt time.Time     `json:"exp"`
}

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

type CursorCodec struct {
	secret []byte
}

func NewCursorCodec(secret []byte) *CursorCodec {
	return &CursorCodec{secret: secret}
}

func (c *CursorCodec) Encode(cur Cursor) (string, error) {
	if cur.Version == cursorVersion {
		return c.encodeV3(cur)
	}
	payload, err := json.Marshal(cur)
	if err != nil {
		return "", fmt.Errorf("support_http: marshal cursor: %w", err)
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	sig := c.signLegacy(encodedPayload)
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// Decode supports legacy cursors whose tenant/query are in the signed payload.
// A v3 cursor must be decoded with DecodeCursorForQuery so its associated data
// is available for signature verification.
func (c *CursorCodec) Decode(token string) (Cursor, error) {
	if strings.HasPrefix(token, "v3.") {
		return Cursor{}, fmt.Errorf("%w: v3 cursor requires tenant and query context", ErrInvalidCursor)
	}
	if strings.HasPrefix(token, "v") {
		return Cursor{}, fmt.Errorf("%w: unsupported version", ErrInvalidCursor)
	}
	encodedPayload, encodedSig, ok := strings.Cut(token, ".")
	if !ok || encodedPayload == "" || encodedSig == "" || strings.Contains(encodedSig, ".") {
		return Cursor{}, fmt.Errorf("%w: malformed token", ErrInvalidCursor)
	}
	sig, err := base64.RawURLEncoding.DecodeString(encodedSig)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: decode signature: %w", ErrInvalidCursor, err)
	}
	if subtle.ConstantTimeCompare(sig, c.signLegacy(encodedPayload)) != 1 {
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
	if err := normalizeLegacyCursor(&cur); err != nil {
		return Cursor{}, err
	}
	return cur, nil
}

func normalizeLegacyCursor(cur *Cursor) error {
	switch cur.Version {
	case 0:
		if cur.Direction != "" {
			return fmt.Errorf("%w: legacy cursor has direction", ErrInvalidCursor)
		}
		cur.Direction = PageForward
	case legacyCursorVersion:
		if !cur.ExpiresAt.IsZero() {
			return fmt.Errorf("%w: version %d cursor has expiry", ErrInvalidCursor, legacyCursorVersion)
		}
		if cur.Direction != PageForward && cur.Direction != PageBackward {
			return fmt.Errorf("%w: invalid direction", ErrInvalidCursor)
		}
	default:
		return fmt.Errorf("%w: unsupported version", ErrInvalidCursor)
	}
	return nil
}

func (c *CursorCodec) DecodeForQuery(token, tenantID, queryHash string) (string, error) {
	cur, err := c.DecodeCursorForQuery(token, tenantID, queryHash)
	if err != nil {
		return "", err
	}
	return cur.After, nil
}

func (c *CursorCodec) DecodeCursorForQuery(token, tenantID, queryHash string) (Cursor, error) {
	if strings.HasPrefix(token, "v3.") {
		return c.decodeV3ForQuery(token, tenantID, queryHash)
	}
	if strings.HasPrefix(token, "v") {
		return Cursor{}, fmt.Errorf("%w: unsupported version", ErrInvalidCursor)
	}
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

func (c *CursorCodec) encodeV3(cur Cursor) (string, error) {
	if cur.Direction != PageForward && cur.Direction != PageBackward {
		return "", fmt.Errorf("%w: invalid direction", ErrInvalidCursor)
	}
	if cur.Page <= 0 {
		return "", fmt.Errorf("%w: invalid page", ErrInvalidCursor)
	}
	var payload bytes.Buffer
	if cur.Direction == PageForward {
		payload.WriteByte(0)
	} else {
		payload.WriteByte(1)
	}
	switch cur.Anchor {
	case PageAnchorKeyset:
		payload.WriteByte(0)
	case PageAnchorEnd:
		if cur.After != "" {
			return "", fmt.Errorf("%w: end anchor has keyset", ErrInvalidCursor)
		}
		payload.WriteByte(1)
	default:
		return "", fmt.Errorf("%w: invalid anchor", ErrInvalidCursor)
	}
	writeUvarint(&payload, uint64(cur.Page))
	if cur.Anchor == PageAnchorKeyset {
		primary, id, err := splitKeyset(cur.After)
		if err != nil || primary == "" || id == "" {
			return "", fmt.Errorf("%w: malformed keyset", ErrInvalidCursor)
		}
		if err := writeLengthPrefixed(&payload, []byte(primary)); err != nil {
			return "", err
		}
		if parsed, err := uuid.Parse(id); err == nil {
			payload.WriteByte(1)
			payload.Write(parsed[:])
		} else {
			payload.WriteByte(0)
			if err := writeLengthPrefixed(&payload, []byte(id)); err != nil {
				return "", err
			}
		}
	}
	payloadBytes := payload.Bytes()
	tag := c.signV3(payloadBytes, cur.TenantID, cur.QueryHash)
	return "v3." + base64.RawURLEncoding.EncodeToString(payloadBytes) + "." + base64.RawURLEncoding.EncodeToString(tag), nil
}

func (c *CursorCodec) decodeV3ForQuery(token, tenantID, queryHash string) (Cursor, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "v3" || parts[1] == "" || parts[2] == "" {
		return Cursor{}, fmt.Errorf("%w: malformed v3 token", ErrInvalidCursor)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(payload) == 0 || len(payload) > v3MaxFieldBytes*2+64 {
		return Cursor{}, fmt.Errorf("%w: invalid v3 payload", ErrInvalidCursor)
	}
	tag, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(tag) != v3TagBytes {
		return Cursor{}, fmt.Errorf("%w: invalid v3 tag", ErrInvalidCursor)
	}
	if subtle.ConstantTimeCompare(tag, c.signV3(payload, tenantID, queryHash)) != 1 {
		return Cursor{}, fmt.Errorf("%w: signature mismatch", ErrInvalidCursor)
	}
	r := bytes.NewReader(payload)
	directionByte, err := r.ReadByte()
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: missing direction", ErrInvalidCursor)
	}
	var direction PageDirection
	switch directionByte {
	case 0:
		direction = PageForward
	case 1:
		direction = PageBackward
	default:
		return Cursor{}, fmt.Errorf("%w: invalid direction", ErrInvalidCursor)
	}
	anchorByte, err := r.ReadByte()
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: missing anchor", ErrInvalidCursor)
	}
	page, err := binary.ReadUvarint(r)
	if err != nil || page == 0 || page > uint64(math.MaxInt) {
		return Cursor{}, fmt.Errorf("%w: invalid page", ErrInvalidCursor)
	}
	cur := Cursor{Version: cursorVersion, TenantID: tenantID, QueryHash: queryHash, Direction: direction, Page: int(page)}
	switch anchorByte {
	case 0:
		cur.Anchor = PageAnchorKeyset
		primary, err := readLengthPrefixed(r)
		if err != nil || len(primary) == 0 {
			return Cursor{}, fmt.Errorf("%w: invalid primary key", ErrInvalidCursor)
		}
		idKind, err := r.ReadByte()
		if err != nil {
			return Cursor{}, fmt.Errorf("%w: missing id kind", ErrInvalidCursor)
		}
		var id string
		switch idKind {
		case 0:
			rawID, err := readLengthPrefixed(r)
			if err != nil || len(rawID) == 0 {
				return Cursor{}, fmt.Errorf("%w: invalid text id", ErrInvalidCursor)
			}
			id = string(rawID)
		case 1:
			var rawID [16]byte
			if _, err := io.ReadFull(r, rawID[:]); err != nil {
				return Cursor{}, fmt.Errorf("%w: invalid UUID id", ErrInvalidCursor)
			}
			id = uuid.UUID(rawID).String()
		default:
			return Cursor{}, fmt.Errorf("%w: invalid id kind", ErrInvalidCursor)
		}
		cur.After = joinKeyset(string(primary), id)
	case 1:
		cur.Anchor = PageAnchorEnd
		if cur.Direction != PageBackward {
			return Cursor{}, fmt.Errorf("%w: end anchor must be backward", ErrInvalidCursor)
		}
	default:
		return Cursor{}, fmt.Errorf("%w: invalid anchor", ErrInvalidCursor)
	}
	if r.Len() != 0 {
		return Cursor{}, fmt.Errorf("%w: trailing payload", ErrInvalidCursor)
	}
	return cur, nil
}

func writeUvarint(w *bytes.Buffer, value uint64) {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], value)
	w.Write(buf[:n])
}

func writeLengthPrefixed(w *bytes.Buffer, value []byte) error {
	if len(value) > v3MaxFieldBytes {
		return fmt.Errorf("%w: cursor field too long", ErrInvalidCursor)
	}
	writeUvarint(w, uint64(len(value)))
	w.Write(value)
	return nil
}

func readLengthPrefixed(r *bytes.Reader) ([]byte, error) {
	length, err := binary.ReadUvarint(r)
	if err != nil || length > v3MaxFieldBytes {
		return nil, fmt.Errorf("%w: invalid length", ErrInvalidCursor)
	}
	lengthInt := int(length)
	if lengthInt > r.Len() {
		return nil, fmt.Errorf("%w: invalid length", ErrInvalidCursor)
	}
	value := make([]byte, lengthInt)
	if _, err := io.ReadFull(r, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (c *CursorCodec) signLegacy(encodedPayload string) []byte {
	mac := hmac.New(sha256.New, c.secret)
	mac.Write([]byte(encodedPayload))
	return mac.Sum(nil)
}

func (c *CursorCodec) signV3(payload []byte, tenantID, queryHash string) []byte {
	var input bytes.Buffer
	writeAssociatedData(&input, []byte("v3"))
	writeAssociatedData(&input, []byte(tenantID))
	writeAssociatedData(&input, []byte(queryHash))
	writeAssociatedData(&input, payload)
	mac := hmac.New(sha256.New, c.secret)
	mac.Write(input.Bytes())
	return mac.Sum(nil)[:v3TagBytes]
}

func writeAssociatedData(w *bytes.Buffer, value []byte) {
	writeUvarint(w, uint64(len(value)))
	w.Write(value)
}
