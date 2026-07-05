---
id: wi-134-memory-persistence-adapter-roundtrip-tests
title: memory 永続化アダプタの全ストアに対する往復（Roundtrip）テストの追加
created_at: 2026-07-05
authors: [tn]
status: completed
risk: low
---

# Motivation
`wi-129` (Go バックエンドのテスト網羅度向上) の一環として、インメモリで動作する永続化アダプタ（`memory` パッケージ）内の全ストア（リポジトリおよびストア）について、作成・取得・更新・削除などの往復（Roundtrip）操作が正しく機能することを検証するテストコードを追加し、カバレッジを向上させるとともに、アダプタ実装の不具合を早期に検知できるようにする。

# Scope
- `internal/shared/adapters/persistence/memory` パッケージ内のすべてのリポジトリ / ストアを対象としたテストの追加・拡充。
- 特に以下の、現在テストが不足しているか存在しないストアに対する往復テストの実装：
  - `access_token_denylist.go`
  - `agents.go`
  - `authorization_codes.go`
  - `authorization_detail_types.go`
  - `authorization_requests.go`
  - `clients.go`
  - `consents.go`
  - `device_codes.go`
  - `email_change_token.go`
  - `groups.go`
  - `mfa.go`
  - `par.go`
  - `password_history.go`
  - `password_reset_token.go`
  - `refresh_tokens.go`
  - `replay.go`
  - `saml_service_providers.go`
  - `sessions.go`
  - `tenants.go`
  - `users.go`

# Out of Scope
- PostgreSQL アダプタ（`postgres`）のテスト環境整備とテスト追加（タスク A の範囲外、後のタスクで実施）。
- `memory` パッケージ以外のカバレッジ改善。

# Verification
- `just verify-go`
- `go test -v ./internal/shared/adapters/persistence/memory/...` がすべて通ること。

# Risk Notes
- 特になし。シンプルなメモリ内データ構造（マップやリスト）に対するテストであるため、外部依存（DBやネットワークなど）がなく、テスト実行は高速かつ安定しているはず。

# Completion
- **Completed At**: 2026-07-05
- **Summary**:
  memory 永続化アダプタの全ストア（リポジトリおよびストア）に対して、作成、取得、更新、削除等の往復（Roundtrip）操作が正常に行えることを検証するテストを網羅的に追加・拡充した。
  これに伴い、`internal/shared/adapters/persistence/memory` パッケージのテストカバレッジは 25.1% から 95.0% に向上し、全体のカバレッジも 43.1% から 54.4% へと改善した。
  また、Go backend の linter 指摘事項（gofumpt, contextcheck, revive, unparam 等）を修正し、`just verify` が正常に通ることを確認した。
- **Verification Results**:
  - `just verify` - 成功
  - `go test -cover ./internal/shared/adapters/persistence/memory/...` - 成功
