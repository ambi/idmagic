package cimd_http

import "net/http"

// NewTestFetcher は、外部テストからも既存の TLS 接続の差し替えを使うためのテスト専用の入口である。
// 製品の公開 API には含めない。
func NewTestFetcher(client *http.Client) *Fetcher { return newFetcherWithClient(client) }
