---
status: pending
authors: [tn]
risk: low
reversibility: irreversible
created_at: 2026-08-30
change_kind: bugfix
priority: p3
depends_on: []
affected_spec:
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

## Verification

- `mise run check-spec`
- `mise run check-api-compat`
- `mise run check-status-drift`

## Risk Notes

`operationId` は生成クライアントのメソッド名になる。付け替えは、その名前を呼んでいるコードを壊す。`reversibility: irreversible` としたのはこのためで、どちらの id を残すかは 1 つずつ決める。
