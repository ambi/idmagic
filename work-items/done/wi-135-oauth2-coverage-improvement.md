---
id: wi-135-oauth2-coverage-improvement
title: oauth2 パッケージ (usecases / adapters/http) のテストカバレッジ向上
created_at: 2026-07-05
authors: [tn]
status: completed
risk: low
---

# Motivation
`wi-129` (Go バックエンドのテスト網羅度向上) の一環として、コアプロトコルである OAuth2 / OIDC 関連のユースケース（`usecases`）および HTTP ハンドラー（`adapters/http`）のテストカバレッジを向上させ、堅牢性を高める。
現在 `internal/oauth2/usecases` (55.0%) および `internal/oauth2/adapters/http` (48.5%) は未カバーのエッジケースが多く、バグの温床となるリスクがあるため、テストを拡充して正常系・異常系・境界値などのカバレッジを高める。

# Scope
- `internal/oauth2/usecases` パッケージ of テスト拡充
  - 特に 0% の関数がある `admin_clients.go`, `admin_consents.go`, `complete_login.go`, `device_flow.go` (DenyUserCode), `introspect_token.go`, `push_authorization_request.go`, `tenant_key_health.go` などに対するテスト。
- `internal/oauth2/adapters/http` パッケージ of テスト拡充
  - 特に `device_handler.go`, `register_handler.go`, `token_handler.go` (Revoke) などの HTTP レベルでのテスト。

# Out of Scope
- `internal/oauth2/domain` パッケージのテスト拡充 (すでに 83.6% と十分高いため)。
- テスト環境やフレームワーク自体の改修。

# Verification
- `just verify-go`
- `go test -cover ./internal/oauth2/usecases/...`
- `go test -cover ./internal/oauth2/adapters/http/...`

# Risk Notes
- 既存のロジックを変更するわけではなくテストコードを追加するだけであるため、デプロイや動作への影響リスクは非常に低い。

# Completion
- **Completed At**: 2026-07-05
- **Summary**:
  oauth2/usecases の全未カバー関数（`admin_clients.go`, `admin_consents.go`, `complete_login.go`,
  `device_flow.go` の DenyUserCode, `introspect_token.go`, `push_authorization_request.go`,
  `tenant_key_health.go`）および oauth2/adapters/http の未テスト HTTP ハンドラー
  (`device_handler.go`, `register_handler.go`, `token_handler.go`)
  に対して正常系・異常系・境界値のテストコードを追加し、全体 Go テストカバレッジを
  54.4% から 56.1% に向上させた。
- **Verification Results**:
  - `just verify-go` - 成功
  - `go test -cover ./internal/oauth2/usecases/...` - 成功
  - `go test -cover ./internal/oauth2/adapters/http/...` - 成功
