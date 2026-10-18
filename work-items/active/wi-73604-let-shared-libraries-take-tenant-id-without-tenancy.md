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

### 今回の負債整理での見送りと再開条件

2026-10-18 時点で残るのは、support_http、rate limiter、二つの salt の保存先、メモリ保存先の計5件である。
署名器からの Tenancy 依存は wi-91505 で解消した。
middleware だけを移しても、support_http.Deps.TenantRepo と Cookie の名前や属性を決める処理に Tenancy の型への依存が残る。
メモリ保存先の DefaultTenant と TenantKey も、空の ID を default へ補う意味を持ち、型名を文字列へ写すだけでは共有の条件を満たさない。

今回は HTTP と背景処理の文脈の契約を混ぜる変更を見送る。
wi-73872 と合わせて、解決済みの設定を持つ文脈と ID だけの文脈、未解決の文脈の区別を先に決める。
再開条件は、テナント ID の読取りと未解決時の拒否を一か所に保ち、設定を持たない背景処理に空の Tenant を解決済みとして見せず、HTTP の origin と Cookie の分離も保つ設計をこの項目へ記録することである。
単に共有の getter を増やして、二つの文脈キーの優先順位と default の規則を別々に維持する実装は行わない。

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
