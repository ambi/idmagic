// Package proxy_http は、MCP の入口で解釈、評価、転送、記録を編成する。
package proxy_http

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ambi/idmagic/backend/mcpgateway/internal/domain"
	"github.com/ambi/idmagic/backend/mcpgateway/internal/evaluation_http"
)

// 実装が定義できる範囲 (-32000 から -32019) から割り当てるコードである。
const (
	codeForbidden       = -32000
	codeUnauthenticated = -32001
	codeUnavailable     = -32002
	codeUpstreamFailed  = -32003
)

const (
	outcomeForwarded       = "forwarded"
	outcomeDenied          = "denied"
	outcomeUnauthenticated = "unauthenticated"
	outcomeInvalid         = "invalid"
	outcomeUnavailable     = "unavailable"
	outcomeUpstreamFailed  = "upstream_failed"
)

const (
	metadataPath   = "/.well-known/oauth-protected-resource"
	maxLoggedValue = 128
)

// Evaluator は評価の呼び出しである。
type Evaluator interface {
	Evaluate(ctx context.Context, in evaluation_http.Request) (evaluation_http.Result, error)
}

// Recorder はツール別の指標を記録する。
type Recorder interface {
	Request(method, tool, outcome string)
	Evaluation(elapsed time.Duration)
}

// Options はハンドラーの依存である。
type Options struct {
	Resource       *url.URL
	Upstream       *url.URL
	RealmURL       string
	AllowedOrigins []string
	MaxBodyBytes   int64
	Mapping        domain.Mapping
	Evaluator      Evaluator
	Audit          *slog.Logger
	Metrics        Recorder
	Transport      http.RoundTripper
}

// Handler は MCP の入口である。
type Handler struct {
	options     Options
	path        string
	metadataURL string
	metadata    []byte
	proxy       *httputil.ReverseProxy
}

// NewHandler はハンドラーを組み立てる。
func NewHandler(options Options) (*Handler, error) {
	path := options.Resource.EscapedPath()
	if path == "" {
		path = "/"
	}
	metadata, err := json.Marshal(map[string]any{
		"resource": options.Resource.String(), "authorization_servers": []string{options.RealmURL},
	})
	if err != nil {
		return nil, err
	}
	base := url.URL{Scheme: options.Resource.Scheme, Host: options.Resource.Host, Path: metadataPath}
	if path != "/" {
		base.Path += options.Resource.Path
	}
	h := &Handler{options: options, path: path, metadataURL: base.String(), metadata: metadata}
	h.proxy = &httputil.ReverseProxy{
		Rewrite:        h.rewrite,
		Transport:      options.Transport,
		FlushInterval:  -1,
		ModifyResponse: restoreCorrelationID,
		ErrorHandler:   h.upstreamFailed,
		ErrorLog:       slog.NewLogLogger(options.Audit.Handler(), slog.LevelWarn),
	}
	return h, nil
}

type requestRecord struct {
	correlationID, method, methodLabel, tool, toolLabel string
	operation                                           domain.Operation
	outcome, evaluationRequestID                        string
	reasons                                             []string
}

type recordKey struct{}

// ServeHTTP は REQ-MCPGATEWAY-002 から 006 を編成する。転送は評価の許可の後の一か所だけにある。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == metadataPath || r.URL.Path == metadataPath+h.path {
		h.serveMetadata(w, r)
		return
	}
	record := &requestRecord{correlationID: newCorrelationID(), outcome: outcomeInvalid}
	w.Header().Set("X-Request-ID", record.correlationID)
	// 転送の途中で接続が切れると ReverseProxy は panic で中断するので、記録は defer に置く。
	defer h.finish(context.WithoutCancel(r.Context()), record)
	h.enforce(w, r.WithContext(context.WithValue(r.Context(), recordKey{}, record)), record)
}

func (h *Handler) enforce(w http.ResponseWriter, r *http.Request, record *requestRecord) {
	if r.URL.EscapedPath() != h.path {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" && !slices.Contains(h.options.AllowedOrigins, origin) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !acceptableContent(r.Header) {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}
	token, ok := presentedToken(r.Header)
	if !ok {
		record.outcome = outcomeUnauthenticated
		h.challenge(w, nil, "")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, h.options.MaxBodyBytes))
	if err != nil {
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "request body could not be read", http.StatusBadRequest)
		return
	}
	message, violation := domain.Inspect(r.Header.Values, body)
	if violation != nil {
		writeViolation(w, violation)
		return
	}
	record.method = message.Method
	record.tool = truncateRunes(message.ToolName, maxLoggedValue)
	evaluation := domain.Decide(message, h.options.Mapping)
	record.toolLabel = evaluation.ToolLabel
	record.operation = evaluation.Operation
	switch evaluation.Kind {
	case domain.MethodNotFound:
		record.reasons = []string{evaluation.Reason}
		writeError(w, http.StatusNotFound, message.ID, domain.CodeMethodNotFound, "method not found", nil)
		return
	case domain.DenyWithoutEvaluation:
		record.methodLabel, record.outcome, record.reasons = message.Method, outcomeDenied, []string{evaluation.Reason}
		writeError(w, http.StatusForbidden, message.ID, codeForbidden, "forbidden", nil)
		return
	}
	record.methodLabel = message.Method
	proof := r.Header.Values("DPoP")
	if len(proof) > 1 {
		record.outcome = outcomeUnauthenticated
		h.challenge(w, message.ID, `error="invalid_token"`)
		return
	}
	started := time.Now()
	in := evaluation_http.Request{
		CorrelationID: record.correlationID, Token: token, Operation: evaluation.Operation, HTU: h.options.Resource.String(),
	}
	if len(proof) == 1 {
		in.DPoPProof = proof[0]
	}
	result, err := h.options.Evaluator.Evaluate(r.Context(), in)
	h.options.Metrics.Evaluation(time.Since(started))
	record.evaluationRequestID = result.RequestID
	if err != nil || !result.Available {
		record.outcome = outcomeUnavailable
		if err != nil {
			record.reasons = []string{"evaluation_failed"}
			h.options.Audit.LogAttrs(r.Context(), slog.LevelWarn, "evaluation failed",
				slog.String("correlation_id", record.correlationID), slog.String("error", err.Error()))
		}
		if result.RetryAfter != "" {
			w.Header().Set("Retry-After", result.RetryAfter)
		}
		writeError(w, http.StatusServiceUnavailable, message.ID, codeUnavailable, "authorization is unavailable", nil)
		return
	}
	record.reasons = result.Reasons
	if !result.Permitted {
		if slices.Contains(result.Reasons, "token_rejected") || slices.Contains(result.Reasons, "sender_proof_invalid") {
			record.outcome = outcomeUnauthenticated
			h.challenge(w, message.ID, `error="invalid_token"`)
			return
		}
		record.outcome = outcomeDenied
		writeError(w, http.StatusForbidden, message.ID, codeForbidden, "forbidden", nil)
		return
	}
	if evaluation.Kind != domain.EvaluateAndForward {
		record.outcome = outcomeDenied
		record.reasons = []string{"tool_not_mapped"}
		writeError(w, http.StatusForbidden, message.ID, codeForbidden, "forbidden", nil)
		return
	}
	record.outcome = outcomeForwarded
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))
	h.proxy.ServeHTTP(w, r)
}

func (h *Handler) serveMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(h.metadata)
}

func (h *Handler) rewrite(request *httputil.ProxyRequest) {
	target := *h.options.Upstream
	request.Out.URL = &target
	request.Out.Host = target.Host
	request.Out.Header.Del("Authorization")
	request.Out.Header.Del("DPoP")
	request.Out.Header.Del("Cookie")
	if record, ok := request.In.Context().Value(recordKey{}).(*requestRecord); ok {
		request.Out.Header.Set("X-Request-ID", record.correlationID)
	}
}

// restoreCorrelationID は上流の X-Request-ID で相関 ID を上書きさせない。
func restoreCorrelationID(response *http.Response) error {
	if record, ok := response.Request.Context().Value(recordKey{}).(*requestRecord); ok {
		response.Header.Set("X-Request-ID", record.correlationID)
	}
	return nil
}

func (h *Handler) upstreamFailed(w http.ResponseWriter, r *http.Request, err error) {
	if record, ok := r.Context().Value(recordKey{}).(*requestRecord); ok {
		record.outcome = outcomeUpstreamFailed
		h.options.Audit.LogAttrs(r.Context(), slog.LevelWarn, "upstream failed",
			slog.String("correlation_id", record.correlationID), slog.String("error", err.Error()))
	}
	writeError(w, http.StatusBadGateway, nil, codeUpstreamFailed, "the MCP server is unavailable", nil)
}

func (h *Handler) challenge(w http.ResponseWriter, id json.RawMessage, parameter string) {
	value := fmt.Sprintf("Bearer resource_metadata=%q", h.metadataURL)
	if parameter != "" {
		value = "Bearer " + parameter + fmt.Sprintf(", resource_metadata=%q", h.metadataURL)
	}
	w.Header().Set("WWW-Authenticate", value)
	writeError(w, http.StatusUnauthorized, id, codeUnauthenticated, "unauthenticated", nil)
}

func (h *Handler) finish(ctx context.Context, record *requestRecord) {
	h.options.Metrics.Request(record.methodLabel, record.toolLabel, record.outcome)
	h.options.Audit.LogAttrs(ctx, slog.LevelInfo, "mcp gateway request",
		slog.String("kind", "mcp_gateway_request"),
		slog.String("correlation_id", record.correlationID),
		slog.String("method", truncateRunes(record.method, maxLoggedValue)),
		slog.String("tool", record.tool),
		slog.String("resource_type", record.operation.ResourceType),
		slog.String("relation", record.operation.Relation),
		slog.String("resource_id", truncateRunes(record.operation.ResourceID, maxLoggedValue)),
		slog.String("outcome", record.outcome),
		slog.Any("reasons", record.reasons),
		slog.String("evaluation_request_id", record.evaluationRequestID),
	)
}

// acceptableContent は JSON の平文だけを受ける。圧縮された本文は解釈と転送の間で読み替えの余地を作る。
func acceptableContent(headers http.Header) bool {
	mediaType, _, err := mime.ParseMediaType(headers.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return false
	}
	encodings := headers.Values("Content-Encoding")
	return len(encodings) == 0 || (len(encodings) == 1 && strings.EqualFold(encodings[0], "identity"))
}

// presentedToken はトークンを解析せずに取り出す。
func presentedToken(headers http.Header) (string, bool) {
	values := headers.Values("Authorization")
	if len(values) != 1 {
		return "", false
	}
	scheme, token, found := strings.Cut(values[0], " ")
	if !found || token == "" || (!strings.EqualFold(scheme, "Bearer") && !strings.EqualFold(scheme, "DPoP")) {
		return "", false
	}
	return token, true
}

type errorData struct {
	Supported []string `json:"supported"`
}

type errorBody struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    *errorData `json:"data,omitempty"`
}

type errorEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Error   errorBody       `json:"error"`
}

func writeViolation(w http.ResponseWriter, violation *domain.Violation) {
	var data *errorData
	if violation.UnsupportedVersionError {
		data = &errorData{Supported: []string{domain.ProtocolVersion}}
	}
	writeError(w, http.StatusBadRequest, violation.ID, violation.Code, violation.Message, data)
}

func writeError(w http.ResponseWriter, status int, id json.RawMessage, code int, message string, data *errorData) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	envelope := errorEnvelope{JSONRPC: "2.0", ID: id, Error: errorBody{Code: code, Message: message, Data: data}}
	if err := json.NewEncoder(w).Encode(envelope); err != nil {
		return
	}
}

func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit])
}

func newCorrelationID() string {
	var raw [16]byte
	_, _ = rand.Read(raw[:])
	return hex.EncodeToString(raw[:])
}
