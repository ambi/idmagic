package domain

// 人が承認した操作と、実行しようとする操作の照合 (REQ-OAUTH2-065)。ここは純粋な計算であり、
// 使用権の比較交換はストアが、実行と結果の記録はリソースサーバーが受け持つ。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

// maxSafeJSONInteger is the largest magnitude a float64 holds without merging neighbours.
const maxSafeJSONInteger = 1 << 53

// maxSafeSignificantDigits is the precision every float64 round-trips in decimal.
const maxSafeSignificantDigits = 15

// ErrInvalidOperationParameters means the actual arguments have no single meaning to compare.
var ErrInvalidOperationParameters = errors.New("operation parameters are not a comparable JSON object")

// OperationBinding is the operation a resource server is about to execute.
type OperationBinding struct {
	DetailType string
	Location   string
	Action     string
	Identifier string
}

// ApprovalUse records which resource server and operation key hold the right to execute.
type ApprovalUse struct {
	KeyHash   string    `json:"key_hash"`
	ClaimedAt time.Time `json:"claimed_at"`
}

// ApprovedOperationMatch is the outcome of comparing an approval with an actual operation.
type ApprovedOperationMatch int

const (
	ApprovedOperationMatched ApprovedOperationMatch = iota
	ApprovedOperationMismatched
	ApprovedOperationExpired
)

// HashOperationKey scopes the key to the resource server, so that two servers using the
// same key never share a right, and keeps the plaintext out of storage.
func HashOperationKey(resourceServerClientID, operationKey string) string {
	sum := sha256.Sum256([]byte(strconv.Itoa(len(resourceServerClientID)) + ":" + resourceServerClientID + operationKey))
	return hex.EncodeToString(sum[:])
}

// CanonicalizeOperationParameters digests actual arguments. An empty input equals an empty
// object. Duplicate keys and numbers a float64 would round are refused: approved values are
// decoded as float64, so a rounded number could match an amount the human never saw.
func CanonicalizeOperationParameters(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return DigestOperationFields(nil)
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	value, err := decodeStrictJSON(decoder)
	if err != nil {
		return "", ErrInvalidOperationParameters
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return "", ErrInvalidOperationParameters
	}
	fields, ok := value.(map[string]any)
	if !ok {
		return "", ErrInvalidOperationParameters
	}
	return DigestOperationFields(fields)
}

func decodeStrictJSON(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch typed := token.(type) {
	case json.Delim:
		if typed == '[' {
			return decodeStrictArray(decoder)
		}
		return decodeStrictObject(decoder)
	case json.Number:
		number, err := typed.Float64()
		if err != nil || math.Abs(number) >= maxSafeJSONInteger || significantDigits(typed.String()) > maxSafeSignificantDigits {
			return nil, ErrInvalidOperationParameters
		}
		return number, nil
	default:
		return typed, nil
	}
}

// significantDigits counts the mantissa digits of a JSON number, ignoring leading zeros.
func significantDigits(number string) int {
	mantissa, _, _ := strings.Cut(strings.ToLower(number), "e")
	mantissa = strings.TrimLeft(strings.NewReplacer("-", "", ".", "").Replace(mantissa), "0")
	return len(mantissa)
}

func decodeStrictArray(decoder *json.Decoder) (any, error) {
	out := []any{}
	for decoder.More() {
		value, err := decodeStrictJSON(decoder)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	_, err := decoder.Token()
	return out, err
}

func decodeStrictObject(decoder *json.Decoder) (any, error) {
	out := map[string]any{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, _ := token.(string)
		if _, duplicate := out[key]; duplicate {
			return nil, ErrInvalidOperationParameters
		}
		value, err := decodeStrictJSON(decoder)
		if err != nil {
			return nil, err
		}
		out[key] = value
	}
	_, err := decoder.Token()
	return out, err
}

// DigestOperationFields digests decoded fields. encoding/json orders map keys, which makes
// the encoding independent of the key order and whitespace of the input.
func DigestOperationFields(fields map[string]any) (string, error) {
	if fields == nil {
		fields = map[string]any{}
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

// MatchApprovedOperation compares the approved details with the actual operation. An empty
// identifier or location never acts as a wildcard: the human approved one named target.
func MatchApprovedOperation(rec *ApprovalRequest, actual OperationBinding, digest string, now time.Time) ApprovedOperationMatch {
	if !now.Before(rec.ExpiresAt) {
		return ApprovedOperationExpired
	}
	for _, detail := range rec.AuthorizationDetails {
		if detail.Type != actual.DetailType || detail.Identifier == "" || detail.Identifier != actual.Identifier ||
			!slices.Contains(detail.Locations, actual.Location) || !slices.Contains(detail.Actions, actual.Action) {
			continue
		}
		approved, err := DigestOperationFields(detail.Fields)
		if err == nil && approved == digest {
			return ApprovedOperationMatched
		}
	}
	return ApprovedOperationMismatched
}
