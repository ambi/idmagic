// Package evaluation_http は、公開契約 POST /resource-access/evaluate を呼ぶ。ワイヤの型は TypeSpec に合わせて
// ここで持ち、サーバーの内部の型を共有しない。
package evaluation_http

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ambi/idmagic/backend/mcpgateway/internal/domain"
)

const maxResponseBytes = 1 << 20

// Request は一回の評価の入力である。
type Request struct {
	CorrelationID       string
	Token               string
	Operation           domain.Operation
	OperationKey        string
	OperationParameters string
	// Attributes は規則の条件が読む JSON オブジェクトで、空なら送らない。
	Attributes string
	// DPoPProof があれば htm と htu を添える。
	DPoPProof string
	HTU       string
	// TokenExchangeResource があれば、許可とともに交換の許可を求める。
	TokenExchangeResource string
}

// Result は評価の結果である。Available が偽なら判断はなく、許可として扱わない。
type Result struct {
	Available         bool
	Permitted         bool
	Reasons           []string
	RequestID         string
	RetryAfter        string
	ApprovedOperation *ApprovedOperationUse
	// TokenExchangeGrant は下流宛の交換の根拠で、ログへ出さない。
	TokenExchangeGrant string
	// Remediation は承認に求める明細の JSON のオブジェクトで、MCP クライアントへ返し、ログへ出さない。
	Remediation json.RawMessage
}

// ApprovedOperationUse は転送前に取得した承認の使用権で、業務の成功を表さない。
type ApprovedOperationUse struct {
	ApprovalRequestID string
	OperationDigest   string
	Replayed          bool
}

// Client は機密クライアントの資格情報で評価を呼ぶ。
type Client struct {
	EvaluateURL  string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

// Evaluate は判断を求める。通信の失敗と 200 以外と解釈できない応答は、すべて判断なしの結果にする。
// 第二の戻り値は運用者のための原因であり、MCP クライアントへ返さない。
func (c Client) Evaluate(ctx context.Context, in Request) (Result, error) {
	form := url.Values{
		"token":         {in.Token},
		"resource_type": {in.Operation.ResourceType},
		"resource_id":   {in.Operation.ResourceID},
		"relation":      {in.Operation.Relation},
	}
	if in.DPoPProof != "" {
		form.Set("dpop_proof", in.DPoPProof)
		form.Set("htm", http.MethodPost)
		form.Set("htu", in.HTU)
	}
	if in.TokenExchangeResource != "" {
		form.Set("token_exchange_resource", in.TokenExchangeResource)
	}
	if in.OperationParameters != "" {
		form.Set("operation_parameters", in.OperationParameters)
	}
	if in.OperationKey != "" {
		form.Set("operation_key", in.OperationKey)
	}
	if in.Attributes != "" {
		form.Set("attributes", in.Attributes)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.EvaluateURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Result{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-Request-ID", in.CorrelationID)
	request.SetBasicAuth(c.ClientID, c.ClientSecret)
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = response.Body.Close() }()
	result := Result{RequestID: response.Header.Get("X-Request-ID")}
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusTooManyRequests {
			result.RetryAfter = response.Header.Get("Retry-After")
		}
		return result, fmt.Errorf("evaluation returned status %d", response.StatusCode)
	}
	var body struct {
		Decision           *bool    `json:"decision"`
		Reasons            []string `json:"reasons"`
		TokenExchangeGrant string   `json:"token_exchange_grant"`
		ApprovedOperation  *struct {
			ApprovalRequestID string `json:"approval_request_id"`
			OperationDigest   string `json:"operation_digest"`
			Replayed          *bool  `json:"replayed"`
		} `json:"approved_operation"`
		AuthorizationRemediation json.RawMessage `json:"authorization_remediation"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(&body); err != nil {
		return result, fmt.Errorf("evaluation response: %w", err)
	}
	if body.Decision == nil {
		return result, fmt.Errorf("evaluation response has no decision")
	}
	if approval := body.ApprovedOperation; approval != nil {
		digest, err := hex.DecodeString(approval.OperationDigest)
		if approval.ApprovalRequestID == "" || err != nil || len(digest) != 32 || approval.Replayed == nil {
			return result, fmt.Errorf("evaluation response has an invalid approved_operation")
		}
		result.ApprovedOperation = &ApprovedOperationUse{
			ApprovalRequestID: approval.ApprovalRequestID, OperationDigest: approval.OperationDigest,
			Replayed: *approval.Replayed,
		}
	}
	if len(body.AuthorizationRemediation) > 0 {
		remediation, err := compactRemediation(body.AuthorizationRemediation)
		if err != nil {
			return result, err
		}
		result.Remediation = remediation
	}
	result.Available, result.Permitted, result.Reasons = true, *body.Decision, body.Reasons
	result.TokenExchangeGrant = body.TokenExchangeGrant
	return result, nil
}

// compactRemediation は authorization_details に配列を持つ JSON のオブジェクトだけを受け、
// MCP クライアントへ渡す前に空白を除く。形の誤った値は解釈できない応答として扱う。
func compactRemediation(raw json.RawMessage) (json.RawMessage, error) {
	var shape struct {
		AuthorizationDetails []json.RawMessage `json:"authorization_details"`
	}
	if err := json.Unmarshal(raw, &shape); err != nil || shape.AuthorizationDetails == nil {
		return nil, fmt.Errorf("evaluation response has an invalid authorization_remediation")
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, raw); err != nil {
		return nil, fmt.Errorf("evaluation response has an invalid authorization_remediation")
	}
	return compacted.Bytes(), nil
}
