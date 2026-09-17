---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-18
priority: p1
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: bugfix
affected_spec:
  - { path: spec/contexts/saml/main.tsp, symbol: IdMagic.Saml.Operations.RegisterSamlServiceProvider }
  - { path: spec/contexts/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.RegisterWsFedRelyingParty }
  - { path: spec/contexts/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.ConfigureEntraFederation }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantQuota }
---

# Echo の Bind でデコードする管理 API のリクエストボディを厳格にデコードする

## Motivation

[API ガイドライン](../docs/design/application/api-guidelines.md)の「リクエストボディのサイズ」と「未知のプロパティ」は、汎用 API の JSON のリクエストボディを 64 KiB に制限し、未知のプロパティを 400 で拒否すると定める。
次の 4 つの API 操作は、`support_http.DecodeJSON` ではなく Echo の `Bind` でデコードしているため、どちらのルールも適用されない。

- `RegisterSamlServiceProvider`
- `RegisterWsFedRelyingParty`
- `ConfigureEntraFederation`
- `UpdateTenantQuota`

綴りを誤ったプロパティは無視され、管理者が設定したはずの値が反映されないまま 200 または 201 が返る。
`Bind` はパスパラメーターとクエリパラメーターもリクエストの構造体へ書き込むため、宣言と異なる場所から値を読む可能性もある。

## Scope

- 上記 4 つの API 操作を `support_http.DecodeJSON` でデコードする。
- 未知のプロパティと 64 KiB の超過を 400 で拒否することを、HTTP の境界で固定する。
- API ガイドラインの適用状況を改める。

## Out of Scope

- Dynamic Client Registration。RFC 7591 に従い、未知のクライアントメタデータを無視する。

## Design

`Bind` を `DecodeJSON` に置き換え、デコードの失敗を既存と同じ 400 `invalid_request` で返す。
リクエストの構造体が `param` や `query` のタグを持つ場合は、その値をパスまたはクエリから明示的に読む。

## Plan

1. 4 つのリクエストの構造体のタグを確認する。
2. 1 つずつ Acceptance RED を確認してから置き換える。

## Tasks

- [ ] T001 [Acceptance] 未知のプロパティを送ると 400 になることを HTTP の境界で確認し、RED を記録する。
- [ ] T002 [App] `DecodeJSON` に置き換える。
- [ ] T003 [Docs] API ガイドラインの適用状況を改める。
- [ ] T004 [Verify] 変更を検証する。

## Verification

- `mise run check-contract-drift`
- `mise run verify`
- `mise run test-ui-e2e`

## Risk Notes

管理コンソールが TypeSpec に宣言していないプロパティを送っている場合、置き換え後に 400 になる。E2E テストで確かめる。
