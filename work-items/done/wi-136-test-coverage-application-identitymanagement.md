---
status: completed
authors: [tn]
risk: low
created_at: 2026-07-05
---

# application / identitymanagement の HTTP ハンドラおよびユースケースのユニットテスト追加

## Motivation

wi-129 のサブタスク C。Go バックエンドの `application` および `identitymanagement` コンテキストにおける HTTP ハンドラ層・ユースケース層のテストカバレッジが低い（application/http: 40.6%, identitymanagement/http: 37.8%, identitymanagement/usecases: 66.7%）。
これらはアプリケーション管理と ID 管理という中核機能を担うため、カバレッジ向上によりリグレッションリスクを低減する。

## Scope

- `internal/application/adapters/http` — 管理 API ハンドラ（handleListApplications, handleUpdateApplication 等）のテストを追加し、カバレッジを 75% 以上に引き上げる
- `internal/application/usecases` — カバレッジは 80.2% だが、エッジケース補強で 90% を目指す
- `internal/identitymanagement/adapters/http` — 管理 API ハンドラのテストを追加し、カバレッジを 75% 以上に引き上げる
- `internal/identitymanagement/usecases` — カバレッジを 66.7% → 80% 以上に引き上げる

## Out of Scope

- application/domain のテスト（0% → 別の作業で対応）
- postgres persistence adapter のテスト（wi-134 で対応）
- フロントエンドテスト（wi-130 で対応）

## Initial Context

- parent: wi-129-backend-test-coverage-improvement
- packages:
  - internal/application/adapters/http
  - internal/application/usecases
  - internal/identitymanagement/adapters/http
  - internal/identitymanagement/usecases

## Affected Guarantees

- application context の HTTP API 正常系・異常系が検証される
- identitymanagement context の HTTP API 正常系・異常系が検証される

## Verification

- `just verify-go`
- `just test-go-cover` で以下を確認:
  - `internal/application/adapters/http` ≥ 75%
  - `internal/application/usecases` ≥ 90%
  - `internal/identitymanagement/adapters/http` ≥ 75%
  - `internal/identitymanagement/usecases` ≥ 80%

## Risk Notes

既存テストとの整合を確認し、テスト用モック／スタブの設計を統一する。

## Plan

- application HTTP ハンドラと usecase の既存境界を変えず、正常系・異常系の追加テストでカバレッジを引き上げる
- identitymanagement HTTP ハンドラと usecase の admin / self-service 経路を追加テストで補強する
- `just test-go-cover` で対象 4 パッケージの閾値達成を確認し、`just verify-go` を最終ゲートにする

## Tasks

- [x] T001 application HTTP ハンドラの管理 API lifecycle / validation テストを追加
- [x] T002 application usecase の binding / category / ordering / sign-in policy edge case テストを追加
- [x] T003 identitymanagement HTTP ハンドラの user / group / agent / account profile / email change テストを追加
- [x] T004 identitymanagement usecase の admin user / group / agent / account profile edge case テストを追加
- [x] T005 `just test-go-cover` で対象 4 パッケージのカバレッジ閾値達成を確認

## Completion

- **Completed At**: 2026-07-07
- **Summary**:
  Added Go unit tests for application and identitymanagement HTTP handlers / usecases, covering lifecycle, validation, bindings, categories, ordering, sign-in policy, icon handling, admin users, groups, agents, account profile, email change, attribute definition filters, membership, credential binding, and repository error paths.
- **Verification Results**:
  - `just test-go` - 成功
  - `just test-go-cover` - 成功
    - `internal/application/adapters/http`: 76.3%
    - `internal/application/usecases`: 90.0%
    - `internal/identitymanagement/adapters/http`: 75.6%
    - `internal/identitymanagement/usecases`: 81.5%
