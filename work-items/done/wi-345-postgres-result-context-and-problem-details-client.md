---
status: completed
authors: [tn]
risk: high
created_at: 2026-08-09
depends_on: []
change_kind: bugfix
initial_context:
  scl:
    System:
      - interfaces.BackendErrorResponse
      - scenarios.既知のバックエンドエラーコードはUIで翻訳される
  source:
    - backend/shared/storage/db_postgres/base.go
    - frontend/src/api
  tests:
    - backend/shared/storage/db_postgres
    - frontend/src/api
  stop_before_reading:
    - backend/oauth2
    - backend/saml
affected_spec:
  - { context: System, kind: interface, element: BackendErrorResponse }
  - { context: System, kind: scenario, element: 既知のバックエンドエラーコードはUIで翻訳される }
  - { context: System, kind: scenario, element: PostgreSQLクエリの期限は結果読取完了まで維持される }
---

# PostgreSQLの結果読取を早期キャンセルせずProblem DetailsをUIへ正しく伝える

## Motivation

PostgreSQL 共通ラッパーが `Query` / `QueryRow` の返却直後に内部 timeout context を
cancel しているため、呼び出し側が `Rows.Next` / `Row.Scan` で結果を読む前に
`context canceled` となる競合がある。CSV user import は job 自体が成功しても状態取得が
500 になり、worker の job claim も同じ理由で不定期に失敗する。

さらに generic HTTP 500 は RFC 9457 Problem Details の `detail` / `type` を返す一方、
frontend の API client は旧 envelope の `message` / `error_description` / `error` しか
解釈しない。その結果、バックエンドへ到達して明確なエラー応答を受け取っている場合にも
「認証サービスに接続できませんでした」という通信障害用文言が表示される。

## Scope

- `spec/contexts/system.yaml` の `BackendErrorResponse` と UI error scenario。
- `backend/shared/storage/db_postgres` の `Query` / `QueryRow` timeout context lifetime と
  circuit breaker の結果判定。
- `frontend/src/api` の共通および個別 fetch 経路における RFC 9457 Problem Details 解釈。
- 単一行・複数行・成功・期限切れと、旧 envelope / Problem Details の回帰テスト。
- SCL 派生 artifact の同期。

## Out of Scope

- OAuth2、SCIM、SharedSignals SET delivery など、各標準が固有形式を定める error envelope の変更。
- PostgreSQL pool sizing、query timeout 値、circuit breaker の閾値変更。
- HTTP error status や既存 stable error code の再分類。

## Design

- `Query` の timeout cancel は返却時ではなく、結果集合の `Close` または終端到達時に行う。
  wrapper は pgx の `Rows` 契約を透過的に保ち、呼び出し側の既存 `defer rows.Close()` を
  completion boundary として利用する。
- `QueryRow` は `Scan` が実際の成功・失敗確定点であるため、timeout cancel と circuit breaker
  計上を `Scan` 完了後に行う。`QueryRow` 呼び出しだけを成功として数えない。
- frontend は旧 JSON error envelope と RFC 9457 を同じ共通解釈関数へ通す。
  stable code は従来どおり優先し、人間可読文は `message`、`error_description`、`detail`、
  `title` の順で解決する。Problem `type` の `urn:idmagic:error:` suffix を stable code として扱う。
- 文法は固定 field の選択のみで、再帰・認証判定を含まないため fuzz/property test は追加しない。

## Plan

1. SCL に PostgreSQL result consumption と Problem Details 表示の受け入れ条件を追加し検証する。
2. PostgreSQL adapter の単一行・複数行 context lifetime テストを先に失敗させ、wrapper を修正する。
3. frontend API error parser の Problem Details テストを先に失敗させ、全 fetch 経路を共通化する。
4. backend / frontend の局所テスト、SCL render、全体 verify を実行する。
5. completion を記録し `work-items/done/` へ移動してコミットする。

## Tasks

- [x] T001 [SCL] `System.BackendErrorResponse` と関連 scenarios に結果読取期限・Problem Details 表示契約を追加する。`just check-scl` passed。
- [x] T002 [Adapter] PostgreSQL `Query` / `QueryRow` の結果読取完了まで timeout context を保持する。RED: `TestResilientDBQueryRowDoesNotCancelBeforeScan` / `TestResilientDBQueryDoesNotCancelBeforeRowsClose` / `TestResilientDBQueryRowReportsScanFailureToCircuitBreaker` が早期 `context canceled` で fail することを先に確認。実起動で判明した正常な `pgx.ErrNoRows` の誤失敗計上は `TestResilientDBQueryRowDoesNotTripCircuitBreakerOnNoRows` が2回目に `circuit breaker is open` となる RED を確認（scenario `PostgreSQLクエリの期限は結果読取完了まで維持される`）→ GREEN (`just test-go-package ./backend/shared/storage/db_postgres`)。
- [x] T003 [UI Adapter] 旧 envelope と RFC 9457 を全 frontend fetch 経路で共通解釈する。RED: `core.test.ts` / `account.test.ts` / `authFlow.test.ts` の Problem Details 回帰テストが通信障害 fallback と code 欠落で fail することを先に確認（scenario `既知のバックエンドエラーコードはUIで翻訳される`）→ GREEN。generic request、account、auth flow、application icon、branding asset の fetch 経路を共通 parser へ統一。
- [x] T004 [Verify] SCL 派生物を同期し、Go / UI / repository 全体の検証と `just dev` 起動確認を通す。API `server listening`、worker `worker listening` 後に警告なしで複数 poll 継続を確認。

## Verification

- `just check-work-items`
- `just check-ids`
- `just check-scl`
- `just test-go-package ./backend/shared/storage/db_postgres`
- `just test-ui-unit-file src/api/core.test.ts`
- `just scl-render`
- `just verify`

## Risk Notes

DB wrapper は全 PostgreSQL repository が共有し、frontend error parser は全 UI API 呼び出しが
共有するため blast radius が大きい。pgx の connection release 契約を wrapper で壊さないこと、
cancel の多重呼び出しを安全にすること、プロトコル固有 envelope を generic Problem Details と
誤認しないことを局所テストと全体 race test で確認する。

## Completion

- **Completed At**: 2026-08-09
- **Summary**:
  PostgreSQL の query timeout context を `Row.Scan` / `Rows.Close` または iteration 終端まで
  保持し、実際の結果読取成否を circuit breaker へ一度だけ報告するよう修正した。正常な
  `pgx.ErrNoRows` は breaker failure から除外する。frontend は旧 error envelope と RFC 9457
  Problem Details を共通解釈し、generic request と独自 fetch 経路の双方で `detail` / `title` と
  `type` suffix を利用する。
- **Verification Results**:
  - `just check-scl` - passed (27 SCL files)
  - `just test-go-package ./backend/shared/storage/db_postgres` - passed
  - `just test-ui-unit` - passed (553 tests)
  - `just typecheck-ui` - passed
  - `just scl-render` - passed
  - `just verify` - passed (11 parallel gates, final run after no-rows correction)
  - `just verify-go` - passed (lint and race-enabled all-package tests, final run after no-rows correction)
  - `just dev` - API `server listening` and worker `worker listening`; startup seed completed and no claim warning observed before intentional Ctrl-C

### Affected Guarantees State

- PostgreSQL query timeout: passed — deadline remains active through result consumption without premature cancellation.
- PostgreSQL availability classification: passed — no-row lookups do not open the circuit breaker.
- Backend error presentation: passed — valid Problem Details no longer falls back to the network-unavailable message.
- Protocol-specific errors: unchanged — OAuth2 / SCIM / SET delivery envelopes retain their existing parsers and contracts.
