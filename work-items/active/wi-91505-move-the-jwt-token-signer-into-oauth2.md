---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: [wi-35767-publish-claim-issuance-as-claimmapping-public-operations]
change_kind: refactor
spec_impact: { kind: none, reason: "ID トークンとアクセストークンを組み立てて署名する Go のパッケージの置き場所だけを変える。発行するトークンの内容と署名、HTTP の応答、ドメインイベントは変えない。" }
---

# JWT の署名器を共有ライブラリから OAuth2 へ移す

## 動機

`tools/check/boundary-debt.json` の `shared-dependency` のうち 5 件は、`backend/shared/security/tokens_jose` から ClaimMapping、IdManagement、OAuth2、SigningKeys、Tenancy への依存である。
`tokens_jose` は OAuth2 の ID トークンとアクセストークンを組み立てて署名する。
OAuth2 の型とポート、ClaimMapping の発行、ユーザーの型を使うので、共有ライブラリの条件（D5）を満たさない。

## 対象範囲

- `tokens_jose` のうち OAuth2 のトークンを組み立てる部分を OAuth2 のパッケージへ移す。
- モジュールの型を使わない JOSE の署名の処理が残るなら、それだけを共有ライブラリに残す。
- 解消した違反 ID を台帳から消す。

## 対象外

- トークンの内容、署名の方式、鍵の選び方の変更。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D5：モジュールの型と業務上の意味を必要とする処理は共有ライブラリに置かない。
- D8：トークンの内容の変更（クレームの追加、`act` の扱い）は OAuth2 の要件から生じ、現在は共有ライブラリを変えている。OAuth2 へ移せば、その変更が OAuth2 で閉じる。

ClaimMapping への依存は、[ClaimMapping の発行規則の公開](../done/wi-35767-publish-claim-issuance-as-claimmapping-public-operations.md)で公開された操作へ置き換える。

## タスク

- [ ] T001 [Design] OAuth2 へ移す部分と共有ライブラリに残す部分を分ける。
- [ ] T002 [App] 移して利用側を書き換える。
- [ ] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 署名の鍵の選び方を移動で変えると、発行したトークンを検証できなくなる。
  移す前に、署名済みのトークンのヘッダーとクレームを特性化テストで固定する。
