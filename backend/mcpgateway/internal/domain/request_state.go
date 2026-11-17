package domain

// 承認を待つ再送で MCP クライアントが送り返す requestState (REQ-MCPGATEWAY-014)。ゲートウェイは状態を
// 持たないので、承認リクエストの参照を認証付き暗号で封じてクライアントに預ける。ここは時刻、鍵、乱数を
// 入力に取る計算であり、評価と転送の作用は持たない。

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

// RequestStatePrefix は、このゲートウェイが封緘した状態の印である。印のない状態は上流が発行したものとして読まない。
const RequestStatePrefix = "igw1."

const requestStateKeyInfo = "idmagic-mcp-gateway request-state v1"

var (
	// ErrRequestStateInvalid は、開けないか、改ざんされたか、中身の欠けた状態である。
	ErrRequestStateInvalid = errors.New("request state is invalid")
	// ErrRequestStateExpired は、承認リクエストの期限を過ぎた状態である。
	ErrRequestStateExpired = errors.New("request state has expired")
)

// RequestState は再送が運ぶ参照である。主体を持たない。ゲートウェイはトークンを検証せず、主体の一致は
// IdMagic が参照する評価で確かめる。
type RequestState struct {
	ApprovalRequestID string
	// OperationDigest は OperationDigest の値で、再送のツール名と引数が起票したときと同じかを確かめる。
	OperationDigest string
	ExpiresAt       time.Time
}

type sealedRequestState struct {
	Version           int    `json:"v"`
	ApprovalRequestID string `json:"a"`
	OperationDigest   string `json:"d"`
	ExpiresAt         int64  `json:"e"`
}

// RequestStateKey はクライアントシークレットから状態の鍵を導く。同じ登録の複数台のゲートウェイが、
// 設定を増やさずに同じ状態を開ける。
func RequestStateKey(clientSecret string) []byte {
	key, err := hkdf.Key(sha256.New, []byte(clientSecret), nil, requestStateKeyInfo, 32)
	if err != nil {
		// 長さ 32 の SHA-256 の導出は失敗しない。
		panic(err)
	}
	return key
}

// SealRequestState は状態を AES-256-GCM で封じる。関連データに resource を使い、別のゲートウェイの状態を開かせない。
func SealRequestState(key []byte, resource string, state RequestState, random io.Reader) (string, error) {
	plaintext, err := json.Marshal(sealedRequestState{
		Version: 1, ApprovalRequestID: state.ApprovalRequestID, OperationDigest: state.OperationDigest,
		ExpiresAt: state.ExpiresAt.UnixNano(),
	})
	if err != nil {
		return "", err
	}
	aead, err := requestStateAEAD(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(random, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, plaintext, []byte(resource))
	return RequestStatePrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

// OpenRequestState は状態を開く。開けない状態は ErrRequestStateInvalid、期限を過ぎた状態は ErrRequestStateExpired である。
func OpenRequestState(key []byte, resource, sealed string, now time.Time) (RequestState, error) {
	encoded, found := strings.CutPrefix(sealed, RequestStatePrefix)
	if !found {
		return RequestState{}, ErrRequestStateInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return RequestState{}, ErrRequestStateInvalid
	}
	aead, err := requestStateAEAD(key)
	if err != nil {
		return RequestState{}, err
	}
	if len(raw) < aead.NonceSize() {
		return RequestState{}, ErrRequestStateInvalid
	}
	plaintext, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte(resource))
	if err != nil {
		return RequestState{}, ErrRequestStateInvalid
	}
	var opened sealedRequestState
	if err := json.Unmarshal(plaintext, &opened); err != nil || opened.Version != 1 ||
		opened.ApprovalRequestID == "" || opened.OperationDigest == "" {
		return RequestState{}, ErrRequestStateInvalid
	}
	state := RequestState{
		ApprovalRequestID: opened.ApprovalRequestID, OperationDigest: opened.OperationDigest,
		ExpiresAt: time.Unix(0, opened.ExpiresAt).UTC(),
	}
	if !now.Before(state.ExpiresAt) {
		return RequestState{}, ErrRequestStateExpired
	}
	return state, nil
}

// OperationDigest はツール名と引数の digest である。引数は値を変えずにキーの順序と空白だけを正規化する。
// 数の表記は読み替えないので、同じ値でも表記が違えば別の操作になる。
func OperationDigest(tool string, arguments map[string]json.RawMessage) (string, error) {
	normalized, err := json.Marshal(struct {
		Tool      string                     `json:"tool"`
		Arguments map[string]json.RawMessage `json:"arguments"`
	}{Tool: tool, Arguments: normalizeArguments(arguments)})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(normalized)
	return hex.EncodeToString(sum[:]), nil
}

// normalizeArguments は入れ子のオブジェクトのキーも並べ替える。json.Marshal は map のキーを整列し、
// RawMessage は空白を除いて書くが、RawMessage の中のオブジェクトのキーは並べ替えない。
func normalizeArguments(arguments map[string]json.RawMessage) map[string]json.RawMessage {
	normalized := make(map[string]json.RawMessage, len(arguments))
	for key, value := range arguments {
		normalized[key] = normalizeJSON(value)
	}
	return normalized
}

func normalizeJSON(value json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 {
		return value
	}
	switch trimmed[0] {
	case '{':
		var object map[string]json.RawMessage
		if json.Unmarshal(trimmed, &object) != nil {
			return value
		}
		encoded, err := json.Marshal(normalizeArguments(object))
		if err != nil {
			return value
		}
		return encoded
	case '[':
		var array []json.RawMessage
		if json.Unmarshal(trimmed, &array) != nil {
			return value
		}
		for i := range array {
			array[i] = normalizeJSON(array[i])
		}
		encoded, err := json.Marshal(array)
		if err != nil {
			return value
		}
		return encoded
	}
	return value
}

func requestStateAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
