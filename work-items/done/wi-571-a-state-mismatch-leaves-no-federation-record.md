---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p3
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v3
spec_impact: { kind: none, reason: "宣言済みの具体例が要求している発行を、実装が一部の分岐で行っていない。規範は動かさず実装を合わせる。" }
documentation_impact:
  level: none
  reason: 宣言済みの EX-AUTHENTICATION-001-03 へ実装を合わせる。FederatedLoginRejected は宣言済みのイベント種別であり、公開契約も運用手順も変わらない。
  references: []
initial_context:
  specification: [docs/domain/authentication/scenarios.feature.md#REQ-AUTHENTICATION-001]
  typespec: [spec/contexts/authentication/models.tsp#FederatedLoginRejected]
  source:
    - backend/authentication/federation/usecases/flow.go
    - backend/authentication/federation/domain/events.go
    - backend/authentication/federation/domain/models.go
    - backend/authentication/federation/db_memory/repositories.go
    - backend/authentication/federation/db_postgres/repositories.go
    - backend/authentication/federation/db_postgres/federation.sql
    - backend/authentication/federation/handlers_http/routes.go
  tests:
    - backend/authentication/federation/handlers_http/refusal_effects_test.go
    - backend/authentication/federation/usecases/flow_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - backend/authentication/federation/protocol_oidc
    - backend/authentication/federation/protocol_saml
    - frontend
affected_spec:
  - { path: docs/domain/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-001 }
primary_use_cases:
  - id: state-mismatch-records-rejection
    requirement: REQ-AUTHENTICATION-001
    observable_result: 発行していない `state`、または消費済みの `state` を持つ callback は 401 で拒否され、LoginSession も関連付けも作られず、上流の応答は検証に到達せず、`Reason` が `state_mismatch` の FederatedLoginRejected が 1 件発行される。
    unit_test: { path: backend/authentication/federation/usecases/flow_test.go, name: TestCompleteLoginRejectsAStateThatIsNotTheOneItIssued, task: test-go-race }
    e2e_test: { path: backend/authentication/federation/handlers_http/refusal_effects_test.go, name: TestMismatchedCallbackCreatesNothingAndRecordsTheRejection, task: test-go-race }
    unit_fault_model: CompleteLogin が `Attempts.Consume` の拒否で発行せずに return する。
    e2e_fault_model: 発行はするが、`Reason` がプロトコル検証の失敗と同じ値になり、監査から 2 つの拒否を区別できない。
---

# `state` が一致しないフェデレーションの callback が記録を残さない

## Motivation

`EX-AUTHENTICATION-001-03` は「`state`、`nonce`、issuer、audience、署名、時刻のいずれかが一致しない」callback について、拒否に加えて `FederatedLoginRejected` の発行を要求する。

実装が発行するのは 6 つのうち後半の 5 つだけである。`CompleteLogin` は `deps.Attempts.Consume` が失敗した時点で `return` し、上流の応答の検証まで到達しない。`FederatedLoginRejected` はその検証が落ちたときにしか発行されない (`backend/authentication/federation/usecases/flow.go`)。

監査から見ると、この欠落は「上流の応答が壊れている攻撃は記録に残り、発行していない `state` を送りつける攻撃は残らない」という差になる。後者のほうが総当たりしやすい。

[[wi-539-back-authentication-examples-with-tests]] がこれを測り、`EX-AUTHENTICATION-001-03` を台帳へ残した。

## Scope

- `state` の照合で拒否した callback も `FederatedLoginRejected` を発行する。
- 発行の理由 (`Reason`) は、プロトコル検証の失敗と区別できる値にする。
- `tools/check/example-coverage-debt.json` から `EX-AUTHENTICATION-001-03` を外し、`backend/authentication/federation/handlers_http/refusal_effects_test.go` の `TestProtocolValidationRefusalCreatesNothingAndRecordsTheRejection` へ `state` 不一致の観測を足して、そのディレクティブが具体例を名指すようにする。テストは両方の拒否を扱うので `TestMismatchedCallbackCreatesNothingAndRecordsTheRejection` へ改名する。

## Out of Scope

- シナリオと具体例の書き換え。実装を宣言へ合わせる。
- 失効した attempt の保持期間や掃除の規則。

## Design

`CompleteLogin` の順序は変えない。`deps.Attempts.Consume` が attempt を返さなかったとき、その拒否を `FederatedLoginRejected` として発行してから error を返す。

| 項目 | 内容 |
| --- | --- |
| 発行の条件 | `Consume` の error が `ports.ErrAttemptNotFound` または `ports.ErrAttemptConsumed` のとき。それ以外の error (保存層の障害) は `state` の照合結果ではないので発行しない |
| `Reason` | `state_mismatch`。未発行、消費済み、期限切れの `state` はいずれも「生きている attempt と一致しない」ため区別しない。プロトコル検証の失敗 (`protocol_validation_failed`) とは別の値にする |
| `ProviderID` | 空文字列。attempt が見つからない以上、callback からは接続を特定できない。callback の URL にも接続の識別子は無い |
| 期限切れ | Postgres の `AttemptStore.Consume` は期限切れを `ErrAttemptConsumed` で返すが、メモリ実装は domain の ad hoc な error を返す。メモリ実装を Postgres に揃え、期限切れを `ErrAttemptConsumed` で返す |

発行の値は定数 `federationdomain.RejectionStateMismatch` と `RejectionProtocolValidationFailed` に名前を付け、テストもその定数で読む。

### 障害モデル

| 障害 | 検出するテスト |
| --- | --- |
| `Consume` の拒否で発行せずに return する | `TestCompleteLoginRejectsAStateThatIsNotTheOneItIssued`、`TestMismatchedCallbackCreatesNothingAndRecordsTheRejection` |
| 発行の `Reason` をプロトコル検証の失敗と同じ値にする | 同上 (`Reason` を読む) |
| 保存層の障害も `state_mismatch` として記録する | `TestCompleteLoginDoesNotRecordAStoreFailureAsAStateMismatch` |
| 照合をプロトコル検証の後ろへ動かす | 既存の `driver.completed` の観測 |

## Tasks

- [x] T001 [Acceptance] 台帳から `EX-AUTHENTICATION-001-03` を外し、HTTP テストへ `state` 不一致の観測を足して RED を確かめる。
- [x] T002 [Domain] `Reason` の定数を足す。
- [x] T003 [Use Case] 単体 RED を確かめ、`CompleteLogin` が `state` の拒否を発行する。
- [x] T004 [Adapters] メモリ実装の期限切れを `ErrAttemptConsumed` に揃える。
- [x] T005 [Verify] 故障注入、変異テスト、`mise run verify` を通す。

実行する検査の手順は次のとおりである。

- RED、GREEN、故障注入: `mise run test-go-test -- <package> <test>`
- 振る舞いが GREEN になった時点: `mise run lint-go`、`mise run test-go-changed`

## Verification

- `mise run check-spec`
- `mise run test-go-package -- ./backend/authentication/federation/usecases`
- `mise run test-go-package -- ./backend/authentication/federation/handlers_http`

## Risk Notes

- **応答だけを見て直したことにする。** `FederatedLoginRejected` はイベントであって応答ではない。発行を足したことは、拒否の応答ではなく記録の側からしか観測できない。
- **`state` の照合をプロトコル検証の後ろへ動かして揃える。** 照合が先にあるのは、発行していない `state` を持つ callback で上流の応答を検証させないためである。順序は変えず、発行だけを足す。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` は main に対する規範の差分を報告しない。宣言済みの `EX-AUTHENTICATION-001-03` へ実装を合わせた。
  `CompleteLogin` は、`Attempts.Consume` が `ErrAttemptNotFound` または `ErrAttemptConsumed` で `state` を拒否したとき、
  `Reason` が `state_mismatch`、`ProviderID` が空の `FederatedLoginRejected` を発行してから error を返す。
  照合を上流の応答の検証より先に置く順序は変えていない。保存層の障害は照合の結果ではないので記録しない。
  `Reason` の値は `federationdomain.RejectionStateMismatch` と、既存の値に名前を付けた
  `RejectionProtocolValidationFailed` の 2 つの定数にした。
  メモリ実装の `AttemptStore.Consume` は期限切れを domain の ad hoc な error で返していたので、Postgres 実装と同じ
  `ErrAttemptConsumed` にそろえた。
  HTTP のテストは `state` の不一致と上流の応答の検証失敗を両方扱うので
  `TestMismatchedCallbackCreatesNothingAndRecordsTheRejection` へ改名し、ディレクティブで `EX-AUTHENTICATION-001-03` を
  名指した。`tools/check/example-coverage-debt.json` の被覆台帳から当該行を外した。
- **Acceptance RED Evidence**:
  - **Test**: `TestMismatchedCallbackCreatesNothingAndRecordsTheRejection`
    (`backend/authentication/federation/handlers_http/refusal_effects_test.go`)
  - **Requirement**: REQ-AUTHENTICATION-001
  - **Observed Failure**: 実装前に、サブテスト「発行していない state」が
    `FederatedLoginRejected が 0 件、期待は 1 件: events=[]` で失敗した。
    サブテスト「上流の応答が検証に落ちる」と対照は実装前から通っていた。
  - **Detection Reason**: production と同じ start と callback の往復で、検証を通る claims を返す driver に
    発行していない `state` を送り、応答、関連付け、セッション、driver の呼び出し回数、発行されたイベントの `Reason` を読む。
- **E2E RED Evidence**:
  - **Test**: `TestMismatchedCallbackCreatesNothingAndRecordsTheRejection`（Acceptance と同じ HTTP 境界のテスト）
  - **Requirement**: REQ-AUTHENTICATION-001
  - **Observed Failure**: `FederatedLoginRejected が 0 件、期待は 1 件: events=[]`
  - **Detection Reason**: 拒否の応答ではなく、Emit へ届いたイベントを値ごと読む。
- **Unit RED Evidence**:
  - **Test**: `TestCompleteLoginRejectsAStateThatIsNotTheOneItIssued`
    (`backend/authentication/federation/usecases/flow_test.go`)、
    `TestAttemptStoreRefusesAnExpiredAttemptAsConsumed`
    (`backend/authentication/federation/db_memory/repositories_test.go`)
  - **Requirement**: REQ-AUTHENTICATION-001
  - **Observed Failure**: 前者は `rejections=[] after an unknown state; want one "state_mismatch" without a provider`。
    後者は `expired consume err=federated login attempt expired, want ErrAttemptConsumed`。
  - **Detection Reason**: 前者は未発行の `state` と消費済みの `state` の 2 つの拒否それぞれで、記録が 1 件ずつ増えることと、
    正当な callback では増えないことを読む。後者は期限切れの attempt が照合の拒否を表す sentinel で返ることを読む。
- **Primary Use Case Evidence**:
  - id: state-mismatch-records-rejection
    unit_red: '実装前の `TestCompleteLoginRejectsAStateThatIsNotTheOneItIssued` は `rejections=[] after an unknown state` で失敗した。'
    e2e_red: '実装前の `TestMismatchedCallbackCreatesNothingAndRecordsTheRejection` は、発行していない state について `FederatedLoginRejected が 0 件、期待は 1 件` で失敗した。'
    unit_fault_injection: '発行の条件を常に偽にすると、単体テストが `rejections=[] after an unknown state` で失敗した。'
    e2e_fault_injection: '発行の `Reason` を `protocol_validation_failed` に差し替えると、HTTP テストが `Reason:protocol_validation_failed、期待は Reason="state_mismatch"` で失敗した。'
- **Change-Resistance Results**:
  1. 発行の条件を常に偽にすると、単体テストと HTTP テストの両方が記録 0 件を観測して失敗した。
  2. `Reason` を `protocol_validation_failed` に差し替えると、単体テストと HTTP テストの両方が失敗した。
  3. 条件を「`ErrLinkConflict` 以外のすべての error」に広げると、`TestCompleteLoginDoesNotRecordAStoreFailureAsAStateMismatch` が
     `events=[FederatedLoginRejected] after a store failure; want none` で失敗した。
  4. 条件から `ErrAttemptConsumed` を外すと、単体テストが再送した `state` の 2 件目の記録がないことで失敗した。
  5. メモリ実装が期限切れの error をそのまま返すと、`TestAttemptStoreRefusesAnExpiredAttemptAsConsumed` が失敗した。
  いずれの差し替えも復元済みである。
  `mise run test-go-mutation` は `backend/authentication/federation/usecases`（Killed 62、Lived 2）と
  `backend/authentication/federation/db_memory`（Killed 5、Lived 1）で実行した。`CompleteLogin` の変異はすべて殺された。
  生き残った `broker.go` 146 行目（`UnlinkIdentity`）、`flow.go` 65 行目（`StartLogin` の PKCE 生成の error 判定）、
  `db_memory/repositories.go` 35 行目（`ConnectionRepository.Save`）はいずれも今回変更していない既存行である。
  `errors.Is` の論理和は変異器の演算子に含まれないため、上記 1、3、4 の手動注入で確かめた。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run test-go-package -- ./backend/authentication/federation/...` - passed
  - `mise run lint-go` - 0 issues
  - `mise run verify` - passed (exit 0)
  - `mise run test-ui-e2e` - 未実行。変更はイベントの発行だけで HTTP の応答を変えず、ブラウザへ届かない。
