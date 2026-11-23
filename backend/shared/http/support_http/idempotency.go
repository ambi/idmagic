package support_http

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/idempotency"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/labstack/echo/v5"
)

// WithIdempotency は各操作の認証、認可と CSRF の検査を済ませた後に呼ぶ。
func WithIdempotency(c *echo.Context, deps Deps, principalID string, next echo.HandlerFunc) error {
	values := c.Request().Header.Values("Idempotency-Key")
	if len(values) == 0 {
		return next(c)
	}
	key, valid := parseIdempotencyKey(values)
	if !valid {
		return WriteProblem(c, http.StatusBadRequest, "invalid_request", "Idempotency-Key must be one non-empty structured string of at most 255 bytes.")
	}
	if deps.Idempotency == nil {
		return errors.New("idempotency storage is unavailable")
	}
	fingerprint, err := requestFingerprint(c.Request())
	if err != nil {
		return WriteProblem(c, http.StatusBadRequest, "invalid_request", "The request body is invalid or exceeds its size limit.")
	}
	ctx := c.Request().Context()
	record, acquired, err := deps.Idempotency.Claim(ctx, RequestTenantID(c), principalID, key, fingerprint, time.Now().UTC())
	if err != nil {
		return err
	}
	if record.Fingerprint != fingerprint {
		return WriteProblem(c, http.StatusUnprocessableEntity, "idempotency_key_reused", "The idempotency key was used with a different request.")
	}
	if !acquired {
		if len(record.Ciphertext) == 0 {
			return WriteProblem(c, http.StatusConflict, "idempotency_key_in_progress", "The earlier request is still running or its outcome is unknown. Do not retry with a new key without checking its effects.")
		}
		response, err := deps.Idempotency.Read(ctx, record)
		if err != nil {
			return err
		}
		return writeIdempotentResponse(c, response)
	}

	// 応答は保存が済むまでネットワークへ出さない。途中停止、panic、保存失敗のいずれも
	// 受付を残し、処理が済んだか不明な操作を再実行させない。
	original := c.Response()
	buffer := &idempotentResponseBuffer{header: original.Header().Clone()}
	c.SetResponse(buffer)
	defer c.SetResponse(original)
	if err := next(c); err != nil {
		c.Echo().HTTPErrorHandler(c, err)
	}
	if buffer.status == 0 {
		buffer.WriteHeader(http.StatusOK)
	}
	response := idempotency.Response{Status: buffer.status, Body: buffer.body.Bytes(), Headers: make(map[string][]string)}
	for _, header := range idempotentResponseHeaders {
		if value := buffer.header.Values(header); len(value) > 0 {
			response.Headers[header] = value
		}
	}
	// クライアントの切断だけで結果の保存を失わない。新しい期限はこの短い保存にだけ使う。
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := deps.Idempotency.Complete(saveCtx, record, response); err != nil {
		return err
	}
	c.SetResponse(original)
	maps.Copy(original.Header(), buffer.header)
	original.WriteHeader(buffer.status)
	_, err = original.Write(buffer.body.Bytes())
	return err
}

var idempotentResponseHeaders = []string{"Content-Type", "Cache-Control", "Content-Disposition", "Location", "Retry-After"}

func writeIdempotentResponse(c *echo.Context, response idempotency.Response) error {
	maps.Copy(c.Response().Header(), response.Headers)
	if strings.HasPrefix(c.Response().Header().Get("Content-Type"), ProblemContentType) {
		var problem map[string]json.RawMessage
		if err := json.Unmarshal(response.Body, &problem); err != nil {
			return err
		}
		if id := logging.RequestIDFromContext(c.Request().Context()); id != "" {
			instance, err := json.Marshal(id)
			if err != nil {
				return err
			}
			problem["instance"] = instance
		} else {
			delete(problem, "instance")
		}
		var err error
		response.Body, err = json.Marshal(problem)
		if err != nil {
			return err
		}
	}
	c.Response().WriteHeader(response.Status)
	_, err := c.Response().Write(response.Body)
	return err
}

type idempotentResponseBuffer struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (w *idempotentResponseBuffer) Header() http.Header { return w.header }
func (w *idempotentResponseBuffer) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *idempotentResponseBuffer) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(body)
}

// RFC 9651 の sf-string。Go の文字列リテラルで許す \n、\x、Unicode エスケープは受けない。
func parseIdempotencyKey(values []string) (string, bool) {
	if len(values) != 1 {
		return "", false
	}
	raw := strings.Trim(values[0], " \t")
	if len(raw) < 3 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", false
	}
	var key strings.Builder
	for i := 1; i < len(raw)-1; i++ {
		ch := raw[i]
		switch ch {
		case '\\':
			i++
			if i >= len(raw)-1 || (raw[i] != '\\' && raw[i] != '"') {
				return "", false
			}
			ch = raw[i]
		case '"':
			return "", false
		}
		if ch < 0x20 || ch > 0x7e {
			return "", false
		}
		key.WriteByte(ch)
		if key.Len() > 255 {
			return "", false
		}
	}
	return key.String(), key.Len() > 0
}

func requestFingerprint(request *http.Request) (string, error) {
	normalizedContentType := request.Header.Get("Content-Type")
	contentType, params, err := mime.ParseMediaType(normalizedContentType)
	if err != nil {
		contentType = normalizedContentType
	} else {
		normalizedContentType = mime.FormatMediaType(contentType, params)
	}
	limit := int64(maxJSONBodyBytes)
	if contentType == "text/csv" {
		limit = 64 << 20
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, limit+1))
	if err != nil {
		return "", err
	}
	if int64(len(body)) > limit {
		return "", errors.New("idempotent request body exceeds its limit")
	}
	request.Body = io.NopCloser(bytes.NewReader(body))
	if contentType == "" {
		contentType = "application/json"
		normalizedContentType = contentType
	}
	if contentType == "application/json" {
		var compact bytes.Buffer
		if json.Compact(&compact, body) == nil {
			body = compact.Bytes()
		}
	}
	bodyHash := sha256.Sum256(body)
	input := struct{ Method, Path, Query, ContentType, BodyHash string }{
		request.Method, AdminContractPath(request.URL.EscapedPath()), request.URL.RawQuery,
		normalizedContentType, hex.EncodeToString(bodyHash[:]),
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:]), nil
}
