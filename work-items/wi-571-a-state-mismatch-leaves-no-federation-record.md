---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p3
depends_on: []
change_kind: bugfix
spec_impact: { kind: none, reason: "宣言済みの具体例が要求している発行を、実装が一部の分岐で行っていない。規範は動かさず実装を合わせる。" }
affected_spec:
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-001 }
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
- `tools/check/example-coverage-debt.json` から `EX-AUTHENTICATION-001-03` を外し、`backend/authentication/federation/handlers_http/refusal_effects_test.go` の `TestProtocolValidationRefusalCreatesNothingAndRecordsTheRejection` へ `state` 不一致の観測を足して、そのディレクティブが具体例を名指すようにする。

## Out of Scope

- シナリオと具体例の書き換え。実装を宣言へ合わせる。
- 失効した attempt の保持期間や掃除の規則。

## Verification

- `mise run check-spec`
- `mise run test-go-package -- ./backend/authentication/federation/usecases`
- `mise run test-go-package -- ./backend/authentication/federation/handlers_http`

## Risk Notes

- **応答だけを見て直したことにする。** `FederatedLoginRejected` はイベントであって応答ではない。発行を足したことは、拒否の応答ではなく記録の側からしか観測できない。
- **`state` の照合をプロトコル検証の後ろへ動かして揃える。** 照合が先にあるのは、発行していない `state` を持つ callback で上流の応答を検証させないためである。順序は変えず、発行だけを足す。
