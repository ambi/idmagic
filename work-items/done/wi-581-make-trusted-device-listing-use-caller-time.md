---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-15
priority: p1
depends_on: []
change_kind: refactor
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 公開仕様と利用者向けの振る舞いは変わらず、内部の時刻依存だけを明示するためである。
  references: []
initial_context:
  specification: [docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-029]
  typespec: []
  source:
    - backend/authentication/trusteddevice/usecases/trusted_devices.go
    - backend/authentication/trusteddevice/ports/trusted_device_repository.go
    - backend/authentication/trusteddevice/db_memory/trusted_devices.go
    - backend/authentication/trusteddevice/db_postgres/trusted_devices.go
  tests: [backend/authentication/trusteddevice/usecases/trusted_devices_test.go]
  stop_before_reading: [backend/shared/http, frontend]
spec_impact:
  kind: none
  reason: 信頼済みデバイスの有効期限判定が既存仕様どおり一つの評価時刻を使うよう、内部ポートの時刻依存を明示するだけである。
---

# 信頼済みデバイス一覧の評価時刻を呼び出し側から渡す

## Motivation

信頼済みデバイス一覧のユースケースは評価時刻を引数で受け取るが、リポジトリは絶対期限の判定に実時刻を使っている。
二つの時計がずれると、ユースケースが有効と判断すべきデバイスをリポジトリが先に除外する。

## Scope

- 信頼済みデバイス一覧のリポジトリポートへ評価時刻を渡す。
- メモリ実装と PostgreSQL 実装で同じ評価時刻を使う。
- 固定時刻を使うユースケーステストで回帰を検出する。

## Out of Scope

- 信頼済みデバイスの有効期限規則を変更しない。
- HTTP API の契約を変更しない。

## Design

`ListActiveByUser` は `now time.Time` を受け取り、絶対期限の検索条件に使う。
時刻は `ListActive` の入力境界からリポジトリへ渡し、各アダプターが独自に時計を読まない構成とする。

実時刻に合わせてテスト fixture を作る案は採用しない。
その案では二つの時計が残り、テストの再現性も下がるためである。

## Plan

ポート、メモリ実装、PostgreSQL 実装、呼び出し側を順に更新し、固定時刻の回帰テストとパッケージ検証を行う。

## Tasks

- [x] T001 [Readiness] 対象仕様、ポート、実装、テストを確認する。
- [x] T002 [RED] 固定時刻を使う既存テストが実時刻とのずれを検出することを記録する。
- [x] T003 [Refactor] 一覧取得の評価時刻をポート引数へ移す。
- [x] T004 [Verify] 変更範囲と全体を検証する。

## Verification

- `mise run test-go-test -- ./backend/authentication/trusteddevice/usecases TestRevokeOneScopesToTheOwnerAndIsIdempotent`
- `mise run test-go-changed`
- `mise run verify`

## Risk Notes

ポートの全実装と呼び出し側を同時に更新しないと、コンパイルエラーまたはアダプター間の判定差が生じる。
コンパイラーと変更パッケージテストで全経路を検査する。

## Completion

- **Completed At**: 2026-09-16
- **Summary**:
  `mise run spec-diff` は main に対する規範仕様差分がないと報告した。
  `ListActiveByUser` は呼び出し側の評価時刻を受け取り、メモリ実装と PostgreSQL 実装が独自に実時刻を読まない構成になった。
- **Acceptance RED Evidence**:
  - **Test**: `TestRevokeOneScopesToTheOwnerAndIsIdempotent`。
  - **Requirement**: REQ-AUTHENTICATION-029
  - **Observed Failure**: 2026-08-15 を評価時刻として発行したデバイスが、リポジトリ内の実日付 2026-09-15 では期限切れと判定され、一覧が空になった。
  - **Detection Reason**: ユースケースが受け取った時刻とリポジトリが読む時刻のずれによる誤除外を、失効操作へ進む前の一覧件数で検出する。
- **Unit RED Evidence**:
  - **Test**: `TestListActiveByUserUsesTheProvidedEvaluationTime`。
  - **Requirement**: N/A: 公開仕様を変えない内部ポートの時刻境界を検査するためである。
  - **Observed Failure**: 旧ポートには評価時刻の引数がなく、テストは引数過多のコンパイルエラーになった。
  - **Detection Reason**: メモリ実装が呼び出し側の評価時刻を受け取れない設計を、型検査で拒否する。
- **Change-Resistance Results**:
  - アダプターが `time.Now()` を読む旧実装を代表故障とし、受け入れテストが空の一覧を返して検出することを確認した。
  - 変更は時刻引数の配線であり、分岐または演算子のロジックを追加していないため mutation test は実行していない。
- **Verification Results**:
  - `mise run test-go-test -- ./backend/authentication/trusteddevice/db_memory TestListActiveByUserUsesTheProvidedEvaluationTime` - passed
  - `mise run test-go-test -- ./backend/authentication/trusteddevice/usecases TestRevokeOneScopesToTheOwnerAndIsIdempotent` - passed
  - `mise run test-go-package -- ./backend/authentication/trusteddevice/db_postgres` - passed
  - `mise run lint-go` - 0 issues
  - `mise run test-go-changed` - passed
  - `mise run verify` - passed
