---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "共有ライブラリがテナント ID を受け取る経路と、テナントを解決する middleware の置き場所だけを変える。テナントの解決規則、テナントの分離、HTTP の応答、永続状態は変えない。" }
---

# 共有ライブラリが Tenancy に依存せずにテナント ID を受け取るようにする

## 動機

`tools/check/boundary-debt.json` の `shared-dependency` のうち 5 件は、共有ライブラリから Tenancy への依存である。

| 共有ライブラリ | 使っているもの |
| --- | --- |
| `http/support_http` | テナントを解決して文脈へ載せる middleware |
| `ratelimit/db_postgres`、`security/salts_memory`、`security/salts_postgres` | 文脈からのテナント ID の取得（`tenantports.TenantID`） |
| `storage/db_memory` | テナントの型 |

保存先と塩の共有ライブラリが必要とするのはテナント ID の文字列だけである。
共有ライブラリが Tenancy に依存すると、共有ライブラリを経由してほかのモジュールが Tenancy へ到達し、共有ライブラリの変更が Tenancy の型の変更に引きずられる。

## 対象範囲

- 保存先と塩の共有ライブラリが、テナント ID を Tenancy の型なしで受け取るようにする。
- テナントを解決する middleware を Tenancy へ移し、組み立て地点で結ぶ。
- 解消した違反 ID を台帳から消す。

## 対象外

- テナントの解決規則と、テナントのない文脈を拒否する規則の変更。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D1：テナントの解決規則の所有者は Tenancy なので、middleware は Tenancy に置く。
- D5：テナント ID の文字列を受け取るだけなら、共有ライブラリの条件を満たす。

| 案 | 判断の材料 |
| --- | --- |
| 共有ライブラリの操作がテナント ID を引数で受け取る | 呼び出し側の変更が最も多いが、依存は最も単純になる |
| テナント ID だけを運ぶ文脈の受け渡しを共有ライブラリに置き、Tenancy が値を載せる | 呼び出し側は変わらない。[テナントの公開契約](../done/wi-39119-publish-the-resolved-tenant-as-tenancy-public-language.md)が退けたのは Tenancy の型を運ぶ共有化であり、文字列だけなら D5 を満たすかを確かめる。テナントのない文脈を default へ落とさない規則（`REQ-TENANCY-006`）を二か所で保たないことが条件になる |

## タスク

- [ ] T001 [Design] 案を比較して受け取り方を決める。
- [ ] T002 [App] 共有ライブラリと呼び出し側、middleware を書き換える。
- [ ] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- テナント ID の受け渡しを変えるときに、テナントのない文脈が default テナントへ落ちると、テナントの分離が破れる。
  `REQ-TENANCY-006` のテストが新しい経路を通ることを確かめる。
