---
status: pending
authors: [tn]
risk: low
created_at: 2026-07-05
---

# crypto / spec / authentication HTTP / http/support のユニットテスト追加

## Motivation

wi-129 のサブタスク D。共有基盤パッケージ（crypto: 48.3%, spec: 56.6%, http/support: 41.0%）および authentication HTTP ハンドラ（34.6%）のテストカバレッジが不十分で、認証・暗号処理・HTTP 基盤という重要レイヤーの品質保証が弱い。
これらのカバレッジを向上させ、セキュリティ関連ロジックの信頼性を高める。

## Scope

- `internal/shared/adapters/crypto` — カバレッジを 48.3% → 75% 以上に引き上げる
- `internal/shared/spec` — カバレッジを 56.6% → 75% 以上に引き上げる
- `internal/authentication/adapters/http` — カバレッジを 34.6% → 70% 以上に引き上げる
- `internal/shared/adapters/http/support` — auth, tenant_middleware, CSRF, consent 等の 0% 関数群のテストを追加し、カバレッジを 41.0% → 70% 以上に引き上げる

## Out of Scope

- authentication/usecases（68.6% — 本 wi 対象外）
- authentication/domain（100% — 対応不要）
- http/server パッケージ（60.3% — 本 wi 対象外）

## Initial Context

- parent: wi-129-backend-test-coverage-improvement
- packages:
  - internal/shared/adapters/crypto
  - internal/shared/spec
  - internal/authentication/adapters/http
  - internal/shared/adapters/http/support

## Affected Guarantees

- 暗号処理（鍵生成・署名検証等）の正当性が検証される
- OpenAPI spec バリデーションの網羅度が向上する
- 認証フローの HTTP ハンドラが正常系・異常系ともに検証される
- HTTP 共通基盤（CSRF, テナント解決, 認可, レスポンスヘルパー）が検証される

## Verification

- `just verify-go`
- `just test-go-cover` で以下を確認:
  - `internal/shared/adapters/crypto` ≥ 75%
  - `internal/shared/spec` ≥ 75%
  - `internal/authentication/adapters/http` ≥ 70%
  - `internal/shared/adapters/http/support` ≥ 70%

## Risk Notes

crypto パッケージのテストでは鍵ペアの生成等で外部依存が生じる可能性があるが、既存テストの構成に揃える。
http/support の認証・テナント解決テストではモック設計に注意が必要。
