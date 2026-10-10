---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on: [wi-35767-publish-claim-issuance-as-claimmapping-public-operations]
change_kind: refactor
spec_impact: { kind: none, reason: "SAML のアサーションを組み立てて署名する Go のパッケージの置き場所と、それを使う側の依存だけを変える。発行するアサーション、署名、HTTP の応答、ドメインイベントは変えない。" }
---

# SAML のアサーションの生成を WsFederation の非公開パッケージから出す

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 4 件は、Saml と Tenancy から `backend/wsfederation/tokens_saml` への依存である。
`tokens_saml` は SAML 1.1 と SAML 2.0 のアサーションを組み立てて署名し、WS-Federation と SAML の両方のプロトコルが使う。
Tenancy の handler は、依存の束に署名器の型（`SignerProvider`）を持っているだけである。

`tokens_saml` は WsFederation の型、ClaimMapping の発行結果、SigningKeys の型を使うので、そのままでは共有ライブラリの条件（D5）を満たさない。

## 対象範囲

- アサーションの生成の置き場所を決めて移し、Saml と WsFederation の呼び出しを書き換える。
- Tenancy の handler の依存の束から署名器の型を外すか、公開された型に置き換える。
- 解消した違反 ID を台帳から消す。

## 対象外

- アサーションの内容と署名の変更。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

| 案 | 判断の材料 |
| --- | --- |
| WsFederation の公開パッケージにする | 移動が最小。Saml から WsFederation への辺は残る |
| プロトコルに依存しない組み立てと署名を共有ライブラリへ分け、WS-Federation と SAML の固有の部分を各モジュールへ残す | D5 を満たす形に分けられれば、Saml と Tenancy から WsFederation への辺が消える |
| SAML のアサーションを担う新しいモジュールを作る | D8。独立して変えたい変更シナリオがあるかを確かめてから比べる |

着手時に `tokens_saml` が WsFederation の型を使う箇所を数え、共有ライブラリへ分けられるかを判断する。
ClaimMapping の発行結果の型は、[ClaimMapping の発行規則の公開](../done/wi-35767-publish-claim-issuance-as-claimmapping-public-operations.md)の後の公開された型を使う。

## タスク

- [ ] T001 [Design] 案を比較して置き場所を決める。
- [ ] T002 [App] 生成を移し、利用側を書き換える。
- [ ] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 署名の対象の正規化を移動で変えると、既存の SP がアサーションを拒否する。
  移す前に、署名済みのアサーションの出力を特性化テストで固定する。
