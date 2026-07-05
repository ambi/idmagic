---
id: wi-129-backend-test-coverage-improvement
title: Go バックエンドのテスト網羅度（カバレッジ）の向上
created_at: 2026-07-05
authors: [tn]
status: pending
risk: medium
---

# Motivation
Go バックエンドのテスト網羅度（カバレッジ）が現在全体の 42.4% と低く、機能が十分に検証されていないリスクがある。
特に `postgres` persistence adapter、`scim/usecases`、`application/domain` などの重要なロジック部分のカバレッジが 0% であるため、これらのカバレッジを高めることで、バックエンドの信頼性と堅牢性を向上させる。

# Scope
- **Domain & Use Case Testing**:
  - `internal/application/domain/application.go` の `ValidateApplication` や `ValidateBinding` などの不変条件バリデーションの単体テストを実装する。
  - `internal/scim/usecases` の SCIM Inbound Provisioning ユースケースに対するテストを実装する。
- **Infrastructure & Adapter Testing**:
  - `internal/shared/adapters/persistence/postgres` パッケージのテスト環境を整備し、データの永続化処理に対するテストを実装する（現在カバレッジ 0%）。
  - `internal/wsfederation/adapters/wstrust` の WS-Trust 関連のテストを実装する（現在カバレッジ 0%）。
- **Overall Coverage Improvement**:
  - カバレッジが 50% 未満の主要コンテキスト (`internal/application/usecases`, `internal/authentication/adapters/http`, `internal/oauth2/domain` など) のテストを追加する。
  - Go バックエンド全体のテストカバレッジを 75% 以上に引き上げる。

# Out of Scope
- フロントエンド（ui）のテスト環境構築およびテスト追加（`wi-130` で対応）。
- CIにおけるテストカバレッジ強制ルールの適用（`wi-131` で対応）。

# Verification
- `just verify-go`
- `just test-go-cover` (全体のカバレッジが 75% 以上に向上し、該当パッケージのカバレッジが改善していること)

# Risk Notes
テスト用の PostgreSQL コンテナやデータベースフィクスチャのセットアップが複雑になる可能性がある。
既存 of Valkey/Postgres モック/本物の接続テストの構成を確認し、安定したテスト実行を確保する必要がある。
カバレッジの数値のみを追い求め、価値の低いテストを量産しないように注意する。
