// Package evaluation_http は、公開契約 POST /resource-access/evaluate を呼ぶ。ワイヤの型は TypeSpec に合わせて
// ここで持ち、サーバーの内部の型を共有しない。
package evaluation_http

import (
	"context"
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
	CorrelationID string
	Token         string
	Operation     domain.Operation
	// DPoPProof があれば htm と htu を添える。
	DPoPProof string
	HTU       string
}

// Result は評価の結果である。Available が偽なら判断はなく、許可として扱わない。
type Result struct {
	Available  bool
	Permitted  bool
	Reasons    []string
	RequestID  string
	RetryAfter string
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
		Decision *bool    `json:"decision"`
		Reasons  []string `json:"reasons"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(&body); err != nil {
		return result, fmt.Errorf("evaluation response: %w", err)
	}
	if body.Decision == nil {
		return result, fmt.Errorf("evaluation response has no decision")
	}
	result.Available, result.Permitted, result.Reasons = true, *body.Decision, body.Reasons
	return result, nil
}
