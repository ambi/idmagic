---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "クレームの発行と開示規則の検証を呼ぶ経路を、ClaimMapping の公開パッケージへ変えるだけである。発行するクレーム、開示の下限、HTTP の応答、ドメインイベントは変えない。" }
---

# クレームの発行と開示規則の検証を ClaimMapping の公開操作にする

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 11 件は、Application、OAuth2、Saml、WsFederation から `backend/claimmapping/usecases` への依存である。
四つのプロトコルは、同じ開示規則でクレームを発行するために `IssueClaimsWithFloor`、`ResolveUserAttributes`、`ResolveTenantAttributeDefs`、`ValidateClaimReleaseRules`、`ClaimIssuanceResult`、`TenantAttributeSchemaRepo` を直接使っている。
ClaimMapping の非公開の実装に依存しているので、ClaimMapping の内部を変えると四つのプロトコルへ変更が波及する。

## 対象範囲

- クレームの発行、利用者の属性の解決、属性の定義の解決、開示規則の検証を、ClaimMapping の公開パッケージの操作として公開する。
- 四つのプロトコルの呼び出しを公開操作の経由へ直す。
- 解消した違反 ID を台帳から消す。

## 対象外

- 開示規則と発行するクレームの変更。
- ClaimMapping の残りのパッケージの `internal/` への移動。[残るモジュールの移行](wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md)で行う。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D1：クレームの開示規則と開示の下限（floor）の所有者は ClaimMapping なので、ClaimMapping を担当とする。
- D3：四つのプロトコルが使う操作は、発行、属性の解決、定義の解決、規則の検証に限られる。この範囲だけを公開し、属性スキーマの保存先は公開しない。

| 案 | 判断 |
| --- | --- |
| 発行と検証を公開操作にする | 採る。開示の下限を通らずにクレームを組み立てる経路をなくせる |
| `TenantAttributeSchemaRepo` などの保存先を公開し、各プロトコルが組み立てる | 採らない。開示の下限を迂回できる公開範囲が残る |
| ClaimMapping を各プロトコルへ分けて複製する | 採らない。同じ規則を四か所で保つことになる |

公開パッケージの名前と、`TenantAttributeSchemaRepo` の型を公開操作の引数から外せるかは、着手時に `ports` と専用の公開パッケージを比べて決める。

## タスク

- [ ] T001 [Design] 公開する操作の一覧と引数の型を決める。
- [ ] T002 [App] 公開操作を置き、四つのプロトコルの呼び出しを書き換える。
- [ ] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 発行の経路を書き換えるときに、開示の下限を適用しない経路が生じるおそれがある。
  各プロトコルの発行のテストが、下限を固定しているかを着手時に確かめ、固定していなければ特性化テストを先に置く。
