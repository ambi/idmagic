package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const maxFilteredToolsListBytes = 4 << 20

// ValidAllowedTools は評価サーバーから受けた一覧を検証する。nil は未対応の応答であり、空の一覧とは区別する。
func ValidAllowedTools(names []string) bool {
	if names == nil || len(names) > 100 {
		return false
	}
	seen := map[string]bool{}
	for _, name := range names {
		if name == "" || utf8.RuneCountInString(name) > 128 || seen[name] {
			return false
		}
		seen[name] = true
	}
	return true
}

// FilterToolsListResponse は上流の一ページを検査し、指定された名前の定義だけを返す。
// 上限まで読み切ってから本文を差し替え、未検査の SSE の断片を呼び出し元へ渡さない。
func FilterToolsListResponse(response *http.Response, id json.RawMessage, allowed []string) error {
	if response == nil || response.Body == nil {
		return errors.New("tools/list has no response body")
	}
	originalBody := response.Body
	defer func() { _ = originalBody.Close() }()
	content, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || response.StatusCode != http.StatusOK || !ValidAllowedTools(allowed) || (response.Header.Get("Content-Encoding") != "" && response.Header.Get("Content-Encoding") != "identity") {
		return errors.New("tools/list response is not supported")
	}
	raw, err := readToolsListBody(response.Body, 30*time.Second)
	if err != nil {
		return err
	}
	message, err := toolsListMessage(raw, content)
	if err != nil {
		return err
	}
	filtered, err := filterToolsListMessage(message, id, allowed)
	if err != nil {
		return err
	}
	response.Body = io.NopCloser(bytes.NewReader(filtered))
	response.ContentLength = int64(len(filtered))
	response.TransferEncoding = nil
	response.Header.Set("Content-Type", "application/json")
	response.Header.Set("Content-Length", strconv.Itoa(len(filtered)))
	response.Header.Set("Cache-Control", "no-store")
	for _, header := range []string{"Content-Encoding", "ETag", "Last-Modified", "Digest", "Content-Digest", "Repr-Digest"} {
		response.Header.Del(header)
	}
	return nil
}

func readToolsListBody(body io.ReadCloser, timeout time.Duration) ([]byte, error) {
	// 一覧では応答を読み切るので、終わらない SSE に接続を占有させない。
	timer := time.AfterFunc(timeout, func() { _ = body.Close() })
	raw, err := io.ReadAll(io.LimitReader(body, maxFilteredToolsListBytes+1))
	withinTime := timer.Stop()
	if !withinTime || err != nil || len(raw) > maxFilteredToolsListBytes {
		return nil, errors.New("tools/list response exceeded its budget")
	}
	return raw, nil
}

func toolsListMessage(raw []byte, content string) ([]byte, error) {
	if content == "application/json" {
		return raw, nil
	}
	if content != "text/event-stream" {
		return nil, errors.New("tools/list response is not JSON or SSE")
	}
	text := strings.ReplaceAll(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\r", "\n")
	var found []byte
	for event := range strings.SplitSeq(text, "\n\n") {
		var data []string
		for line := range strings.SplitSeq(event, "\n") {
			if value, ok := strings.CutPrefix(line, "data:"); ok {
				data = append(data, strings.TrimPrefix(value, " "))
			}
		}
		if len(data) == 0 {
			continue
		}
		body := []byte(strings.Join(data, "\n"))
		if err := rejectAmbiguousKeys(body); err != nil {
			return nil, err
		}
		var envelope map[string]json.RawMessage
		if json.Unmarshal(body, &envelope) != nil || envelope == nil {
			return nil, errors.New("tools/list SSE event is not an object")
		}
		if _, notification := envelope["method"]; notification && envelope["id"] == nil {
			continue
		}
		if found != nil {
			return nil, errors.New("tools/list SSE has multiple responses")
		}
		found = body
	}
	if found == nil {
		return nil, errors.New("tools/list SSE has no response")
	}
	return found, nil
}

func filterToolsListMessage(raw, id json.RawMessage, allowed []string) ([]byte, error) {
	if !utf8.Valid(raw) {
		return nil, errors.New("tools/list response is not UTF-8")
	}
	if err := rejectAmbiguousKeys(raw); err != nil {
		return nil, err
	}
	var envelope struct {
		JSONRPC string                     `json:"jsonrpc"`
		ID      json.RawMessage            `json:"id"`
		Result  map[string]json.RawMessage `json:"result"`
		Error   json.RawMessage            `json:"error,omitempty"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.JSONRPC != "2.0" || !sameToolsListID(envelope.ID, id) || envelope.Result == nil || len(envelope.Error) != 0 {
		return nil, errors.New("tools/list response has no matching result")
	}
	var tools []json.RawMessage
	if json.Unmarshal(envelope.Result["tools"], &tools) != nil || tools == nil {
		return nil, errors.New("tools/list has no tools array")
	}
	kept := []json.RawMessage{}
	seen := map[string]bool{}
	for _, tool := range tools {
		var definition map[string]json.RawMessage
		var name string
		if json.Unmarshal(tool, &definition) != nil || json.Unmarshal(definition["name"], &name) != nil || name == "" || seen[name] {
			return nil, errors.New("tools/list has invalid or duplicate tool names")
		}
		seen[name] = true
		if slices.Contains(allowed, name) {
			kept = append(kept, tool)
		}
	}
	if cursor, present := envelope.Result["nextCursor"]; present && string(bytes.TrimSpace(cursor)) != "null" {
		var value string
		if json.Unmarshal(cursor, &value) != nil {
			return nil, errors.New("tools/list has an invalid cursor")
		}
	}
	encodedTools, err := json.Marshal(kept)
	if err != nil {
		return nil, err
	}
	envelope.Result["tools"] = encodedTools
	envelope.Result["ttlMs"] = json.RawMessage(`0`)
	envelope.Result["cacheScope"] = json.RawMessage(`"private"`)
	return json.Marshal(envelope)
}

func sameToolsListID(left, right json.RawMessage) bool {
	if validID(left) == nil || validID(right) == nil {
		return false
	}
	var leftString, rightString string
	if json.Unmarshal(left, &leftString) == nil && json.Unmarshal(right, &rightString) == nil {
		return leftString == rightString
	}
	return bytes.Equal(bytes.TrimSpace(left), bytes.TrimSpace(right))
}
