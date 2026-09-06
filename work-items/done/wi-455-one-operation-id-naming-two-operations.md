---
status: completed
authors: [tn]
risk: low
reversibility: irreversible
created_at: 2026-08-30
change_kind: bugfix
priority: p3
depends_on: []
evidence_policy: risk-based-v3
documentation_impact:
  level: removal_notice
  reason: 重複していた operationId の一部を経路固有名へ置き換えるため、生成クライアントでは旧名として上書きされていた operation が除去され、新しいメソッド名として現れる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-455-unique-operation-ids.md }
    - { kind: upgrade_note, path: docs/releases/upgrades/wi-455-unique-operation-ids.md }
initial_context:
  specification:
    - docs/standards.md#openapi-specification-311
    - docs/contexts/saml/decisions.md
    - docs/contexts/oauth2/decisions.md
  typespec:
    - IdMagic.Saml.Operations.SamlSingleSignOn1
    - IdMagic.Authentication.Operations.CompleteFederatedLogin1
    - IdMagic.OAuth2.Operations.EndSession1
  source:
    - tools/generate-contract/src/main.ts
    - tools/check/src/status-drift.ts
    - backend/provisioning/handlers_http/routes.go
    - backend/provisioning/handlers_http/handlers.go
    - backend/sharedsignals/handlers_http/routes.go
  tests:
    - tools/generate-contract/src/contract.test.ts
    - tools/check/src/status-drift.test.ts
  stop_before_reading:
    - frontend
    - backend/sourcing
primary_use_cases:
  - id: unique-operation-ids
    requirement: OPENAPI31-OPERATION-ID
    observable_result: 生成 OpenAPI の全 operationId が一意で、実行時契約生成が重複を黙って捨てない。
    unit_test: { path: tools/generate-contract/src/contract.test.ts, name: rejects duplicate operation IDs and reports both routes, task: test-tools }
    e2e_test: { path: tools/generate-contract/src/contract.test.ts, name: generated contract entry point rejects duplicate operation IDs, task: test-tools }
    unit_fault_model: operationId を Set で重複排除して後続経路を黙って捨てる。
    e2e_fault_model: check-spec から重複検査を外して重複 TypeSpec を合格させる。
affected_spec:
  - { path: docs/standards.md, requirement: OPENAPI31-OPERATION-ID }
  - { path: spec/contexts/saml/main.tsp, symbol: IdMagic.Saml.Operations.SamlSingleSignOn1 }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.CompleteFederatedLogin1 }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.EndSession1 }
---

# 1 つの `operationId` が 2 つ以上の operation を名乗っている状態を解く

## Motivation

`wi-386` が 333 operation を総なめする過程で、`operationId` が一意でないことが分かった。TypeSpec の側は `SamlSingleSignOn1` から `SamlSingleSignOn4` のように別の記号を持つが、`@TypeSpec.OpenAPI.operationId` はすべて同じ文字列を書いている。

| operationId | 名乗る operation の数 |
| --- | --- |
| `SamlSingleSignOn` | 4 |
| `SamlSingleLogout` | 4 |
| `CompleteFederatedLogin` | 2 |
| `PublishSamlMetadata` | 2 |
| `DownloadSamlSigningCertificate` | 2 |
| `EndSession` | 2 |
| `WsTrustIssue` ほか | 要調査 |

OpenAPI 3.1 は `operationId` を文書内で一意と定める。重複していると、生成クライアントのメソッド名が衝突するか、後から読んだ方に上書きされる。`wi-386` の監査も、同じ id が 2 行に出るせいで表が読みにくくなった。

同じ形の問題が Go 側にもある。2 つのパッケージが `handleListDeliveries` という同じ handler 名を持っており、`wi-386` の検査は「どちらの本体を読めばよいか決められない」として `ListProvisioningDeliveries` と `ListSecurityEventDeliveries` を未解決に落としている。

## Scope

- 重複する `operationId` を数え上げ、1 つずつ「同じ操作を 2 つの経路で提供しているのか」「別の操作なのか」を決める。
- 別の操作なら別の id を与える。同じ操作なら、なぜ 2 つの経路があるのかを owning context の `decisions.md` に残す。
- `operationId` の一意性を `mise run check-spec` で検査する。
- Go 側の handler 名の衝突を解き、`wi-386` の未解決 2 件を閉じる。

## Out of Scope

- 経路そのものの統廃合。SAML のテナント既定プロファイルと名前付きプロファイルは、どちらも標準が定めた探索経路から到達する。

## Design

OpenAPI の operation は `{ name, method, path, deprecated, apiTokenScopes }` として収集し、同じ `name` が別の `(method, path)` から現れた時点で両経路を含む検査エラーにする。SAML は既定/名前付きプロファイルと GET/POST binding を名前に反映し、現在の実行時契約が保持していた名前付き GET の base ID は維持する。OIDC federation callback と GET EndSession も既存 ID を維持する。Go の handler 名はパッケージをまたぐ検査器でも一意になるよう、業務操作名を含める。入力は OpenAPI 文書、出力は生成契約または検査エラーであり、時刻、乱数、永続化、通知は関与しない。

## Plan

1. OpenAPI 3.1.1 の採用行と経路併存の判断を仕様へ記録し、重複 operationId を分離する。
2. 重複を黙って捨てる生成器の Unit RED と CLI 境界の E2E RED を確認する。
3. 収集処理を深い純粋モジュールへ分離し、生成入口から重複を拒否する。
4. 2 つの ListDeliveries handler を業務操作名へ改名し、status drift の未解決を閉じる。

## Tasks

- [x] T001 [Spec] OPENAPI31-OPERATION-ID、SAML/OAuth2 の決定、16 operation の一意な ID を追加する。
- [x] T002 [Unit] 重複 ID fixture が黙って 1 件へ縮む RED を確認する (OPENAPI31-OPERATION-ID)。
- [x] T003 [Tooling] operation 収集を純粋モジュールへ分離し、重複時に両経路を報告する。
- [x] T004 [Adapter] provisioning/sharedsignals の ListDeliveries handler 名を分離する。
- [x] T005 [Acceptance] CLI と `mise run check-spec` が重複を拒否し、status drift の未解決 2 件が消えることを確認する。
- [x] T006 [Verify] 重複検査の障害注入と `mise run verify` を通す。

## Verification

- `mise run check-spec`
- `mise run check-api-compat`
- `mise run check-status-drift`

## Risk Notes

`operationId` は生成クライアントのメソッド名になる。付け替えは、その名前を呼んでいるコードを壊す。`reversibility: irreversible` としたのはこのためで、どちらの id を残すかは 1 つずつ決める。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff` は `OPENAPI31-OPERATION-ID` の追加を報告した。SAML の既定/名前付きプロファイルと GET/POST binding、SAML/OIDC callback、GET/POST EndSession に経路固有の `operationId` を与え、336 operation を一意にした。契約生成器は重複を黙って縮約せず両経路を示して拒否する。provisioning と sharedsignals の配送一覧 handler 名も分離し、status drift の曖昧さを解消した。
- **Primary Use Case Evidence**:
  - id: unique-operation-ids
    unit_red: >-
      `rejects duplicate operation IDs and reports both routes` は収集モジュールが存在しない状態で module-not-found となり、その後の初期実装では重複を黙って縮約したため失敗した。
    e2e_red: >-
      `generated contract entry point rejects duplicate operation IDs` は重複 fixture を CLI に渡しても拒否する入口がなく失敗した。
    unit_fault_injection: >-
      重複時の throw を継続処理へ変えると、単体テストが例外未発生を検出した。
    e2e_fault_injection: >-
      同じ故障により CLI が終了コード 0 を返し、入口の受け入れテストが失敗した。さらに `PostEndSession` を `EndSession` に戻すと `mise run check-spec` が GET/POST の両経路を示して拒否した。
- **Change-Resistance Results**:
  リスクは low だが、重複拒否を無効化する代表故障を注入した。純粋な収集関数と実 CLI の両方が故障を検出し、TypeSpec 側の重複再導入も `check-spec` が検出した。すべて復元後に再実行して成功した。
- **Verification Results**:
  - `mise run check-spec` - passed (336 unique operation IDs)
  - `mise run check-api-compat` - passed
  - `mise run check-status-drift` - passed (0 findings; handler ambiguity resolved)
  - `mise run verify` - passed
