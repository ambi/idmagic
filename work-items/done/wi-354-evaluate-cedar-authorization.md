---
status: completed
authors: [tn]
risk: high
created_at: 2026-08-11
depends_on: [wi-355-replace-scl-architecture-ledgers-and-adrs]
change_kind: tooling
initial_context:
  specification:
    - spec/contexts/oauth2/SPECIFICATION.md#REQ-OAUTH2-004
    - spec/contexts/oauth2/SPECIFICATION.md#RFC8707-MCP-RESOURCE-BINDING
  typespec: []
  source:
    - backend/oauth2/ports/authorizer.go
    - backend/oauth2/token/usecases/exchange_token.go
    - backend/oauth2/token/usecases/role_policies.go
    - backend/shared/policy/authorization_local/authorizer.go
    - backend/shared/policy/authorization_http/authorizer.go
    - backend/shared/spec/policy.go
    - backend/cmd/internal/bootstrap/authorizer.go
  tests:
    - backend/oauth2/token/usecases/exchange_token_test.go
    - backend/shared/spec/admin_policy_test.go
    - backend/shared/policy/authorization_http/authorizer_test.go
  stop_before_reading:
    - frontend
spec_impact:
  kind: none
  reason: Cedar の採否を評価する将来作業であり、この work item 自体は現行の製品仕様や認可挙動を変更しない。
---

# Cedar を IdMagic の実行時認可正本として採用するか再評価する

## Motivation

specification の認可記述を廃止した後も、認可要件と Go の実行時 evaluator の間に重複が残り得る。一方、
Cedar の導入は policy schema、entity projection、validator、運用、移行を伴うため、仕様体系の簡素化と
同時に採用を確定すると変更リスクと管理対象を増やす。

## Scope

- Cedar / cedar-go の実行時・検証時の成熟度を実装時点で再調査する。
- 代表的な一つの認可経路で、既存 Go evaluator を実際に置換する pilot を設計する。
- policy、schema、entity projection、テスト、運用負荷を比較し、採用または不採用を決める。

## Out of Scope

- specification置換作業中のCedar導入。
- 文書としてだけCedar policyを追加し、Go evaluatorと二重管理すること。

## Design

- Cedarを採る場合の必須条件は、対象経路の実行時判断をCedarへ移し、同じ意味のGo rule mapを削除できることとする。
- pilot は既に `Authorizer` port を通る token-exchange を対象にし、grant 宣言、scope の単調縮小、
  resource/audience 一致を Cedar policy と schema で表現する。既存 evaluator との permit/deny parity、
  undefined action・不正 projection・validator/evaluator error の fail-closed、投影込みの性能を受け入れ基準とする。
- 採否は validator の互換性保証、性能、障害時の fail-closed、ローカル/remote AuthZEN seam との互換性、
  role-policy inspection を含む単一正本化の可否で判断する。
- **Decision (2026-08-14): Cedar は採用しない。** cedar-go v1.8.0 の authorizer は対象経路を表現でき、
  parity と fail-closed も成立したが、schema validator は互換性保証外の `x/exp` である。また
  `RulesForAction` が role-policy inspection の表示データを兼ねるため、runtime evaluation だけを Cedar へ移しても
  token-exchange の3 requirementを持つGo tableを削除できず、二重管理になる。
- 再評価条件は、validator が安定 API になること、policy から inspection metadata を同じ正本で導出できること、
  wi-53 の relationship facts を bounded entity projection として供給できることの3点とする。
- wi-53 は、既存 AuthZEN-style `Authorizer` seam に Go evaluator / relationship facts provider を組み込む方針で進める。

## Plan

- OAuth2 Design の削除済み JSON policy 参照を、現行 Go rule table と AuthZEN adapter seam へ同期する。
- token-exchange の runtime test を先に追加し、cedar-go v1.8.0 の一時 adapter で比較する。
- schema・policy・entity projection・fail-closed と benchmark を評価し、pilot と Cedar 依存は採否後に撤去する。
- 採否、移行費用、撤去可能な既存コード量、再評価条件を Completion に記録する。

## Tasks

- [x] T001 [Research] cedar-go v1.8.0 (2026-06-01) を確認した。core authorizer は stable だが、
  schema parser/validator は `x/exp/schema` 配下で semantic-version compatibility の対象外である。
- [x] T002 [Design] token-exchange と3条件を pilot 対象にし、parity、fail-closed、schema validation、
  projection 込み benchmark、Go rule map 撤去可否を受け入れ基準にした。
- [x] T003 [Pilot] `TestTokenExchangePilotParityAndFailClosed` を `NewTokenExchangePilot` 未定義で RED 確認
  (RFC8707-MCP-RESOURCE-BINDING) → Cedar policy/schema、entity projection、policy/entity/request validation、
  undefined action・不正 projection の fail-closed を実装して GREEN。評価後に pilot を撤去した。
- [x] T004 [Decision] Cedar は不採用とした。280行の一時 pilot (adapter 148行、test 132行) と dependency が必要で、
  benchmark は Cedar 16.5–17.1 us/op・23,797–23,798 B/op・248 allocs/op、既存 Go evaluator
  66.16–66.85 ns/op・0 B/op・0 allocs/opだった。既存 table の対象 action と3 requirementは
  role-policy inspection に必要なため撤去できない。
- [x] T005 [Verify] 対象 package、全 Go test、repository check、spec diff、標準 verification を通した。

## Verification

- `just check-spec`
- `just check-api-compat`
- `just check-boundaries`
- `just test-go-package ./backend/shared/policy/authorization_cedar` (一時 pilot、GREEN 後に撤去)
- `just benchmark-go-package ./backend/shared/policy/authorization_cedar BenchmarkTokenExchange` (一時 pilot)
- `just test-go`
- `just check`
- `just verify`

## Risk Notes

認可エンジンの置換はsecurity-criticalである。文書追加だけで採用済みにせず、runtime pilotとfail-closed
テストが成立した場合だけ本移行を提案する。

## Completion

- **Completed At**: 2026-08-14
- **Summary**:
  cedar-go v1.8.0 の token-exchange pilot で grant 宣言、scope 縮小、resource/audience 一致を
  Cedar policy/schemaへ移し、既存 evaluatorとの decision parity、schema validation、undefined action・
  不正projection・evaluation errorのfail-closedを確認した。runtime判断は表現可能だったが、validatorが
  互換性保証外の実験APIで、role-policy inspection用Go tableも撤去できないためCedarは不採用とした。
  pilot、Cedar dependency、schema、policyは撤去済みで、wi-53は既存Go evaluator方針で進める。
- **Semantic Difference**:
  `just spec-diff` は `no normative specification change against main`。製品API・認証・認可結果に差分はない。
  OAuth2 Designの存在しないJSON policy参照を、現行Go rule table、local/remote AuthZEN adapter seam、
  Cedar不採用と再評価条件へ同期した。Go package benchmark用の汎用just recipeを追加した。
- **Pilot Results**:
  一時実装はadapter 148行・test 132行。Apple M3で5回測定し、Cedarは16.5–17.1 us/op、
  23,797–23,798 B/op、248 allocs/op、既存Go evaluatorは66.16–66.85 ns/op、0 B/op、0 allocs/opだった。
  token-exchangeの3 requirementは`RulesForAction`経由でinspectionにも必要なため、runtimeを移しても
  action-to-requirement tableは撤去できない。
- **Verification Results**:
  - `just check-spec` - passed (44 OAuth2 scenarios、761 TypeSpec symbols)。
  - `just check-api-compat` / `just check-boundaries` - passed。
  - 一時pilotの`just test-go-package` - RED (`NewTokenExchangePilot`未定義) → GREEN (全parity/fail-closed cases)。
  - 一時pilotの`just benchmark-go-package` - passed (5 runs、上記結果)。
  - `just test-go` / `just check` - passed (368 work items、dependency recordsともにvalid)。
  - `just verify` - passed (Go/UI/tooling/spec/API compatibility、11秒)。
- **Left Undone**: なし。Out of Scopeの全面移行、二重policy管理、remote Cedar service、wi-53 ReBAC実装は行っていない。
