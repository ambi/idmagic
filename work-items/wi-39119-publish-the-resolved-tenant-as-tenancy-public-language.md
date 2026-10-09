---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-09
priority: p2
depends_on: [wi-92970-adopt-information-hiding-modular-design]
change_kind: refactor
spec_impact: { kind: none, reason: "解決済みのテナントを運ぶ Go のパッケージの置き場所と、Tenancy の非公開パッケージへの直接の依存だけを変える。テナントの解決規則、HTTP の応答、発行するドメインイベント、永続状態は変えない。" }
---

# 解決済みのテナントを Tenancy の公開契約として公開する

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 54 件は、ほかのモジュールから Tenancy の非公開パッケージへの依存である。

| 参照先 | 件数 | 原因 |
| --- | --- | --- |
| `backend/tenancy`（ルートパッケージ） | 41 | 解決済みのテナントを `context.Context` に載せて読む `backend/tenancy/context.go` が、組み立て地点にだけ公開するルートパッケージにある |
| `backend/tenancy/usecases` | 11 | Tenancy のユースケースを直接呼ぶ |
| `backend/tenancy/db_postgres` | 2 | Tenancy のアダプターを直接使う |

テナントの境界は、ほぼすべてのモジュールが前提にする。
それにもかかわらず、その前提を受け取る経路が公開パッケージにないため、正しい前提の受け取りまでが非公開の実装への依存になっている。
共有ライブラリ（`shared/http/support_http`、`shared/storage/db_memory` など）から Tenancy への `shared-dependency` も、同じ受け渡しを原因に含む。

## 対象範囲

- `backend/tenancy/context.go` の `WithTenant`、`Tenant`、`TenantID`、`Issuer`、`URLPrefix` を、Tenancy の公開パッケージへ移し、呼び出し元を書き換える。
- ほかのモジュールが Tenancy の `usecases` と `db_postgres` を直接使う箇所を、公開操作を経由する形へ直す。
- 解消した違反 ID を `tools/check/boundary-debt.json` から消す。

## 対象外

- テナントの解決規則そのものの変更。
- Tenancy の `internal/` への移行。モジュールごとの `internal/` 移行の項目で扱う。

## 設計

境界を選ぶ判断手順の記録は、[モジュール設計の項目](done/wi-92970-adopt-information-hiding-modular-design.md)の「判断手順の適用」にある。
要点は次のとおりである。

- D1：解決済みのテナントの状態と、テナントのない文脈を default テナントへ落とさない規則の所有者は Tenancy なので、Tenancy を担当とする。
- D3：非公開の関数をそのまま公開するのではなく、取得と付与を Tenancy の公開契約として公開パッケージに置く。
- D5：値の型が `tenancy/domain.Tenant` であり、規則がテナント分離の保証そのものなので、共有ライブラリの条件を満たさない。

置き場所は `ports` を第一候補にする。
`context.Context` に値を載せる処理は計算ではなく実行時の受け渡しなので、`structure.md` が決定論的な計算だけを置くと定める `domain` には合わない。

採らない案は次のとおり。

| 案 | 採らない理由 |
| --- | --- |
| テナントの受け渡しを `backend/shared` へ移す | Tenancy の型と規則を運ぶので D5 を満たさず、`shared-dependency` へ形を変えるだけである |
| 境界の検査で Tenancy を例外にする | 検査にモジュール固有の例外を持ち込むと、責務表という宣言が検査の判断を説明しなくなる |

## 計画

1. `context.go` を公開パッケージへ移し、呼び出し元を一括で書き換える。
2. 残った `private-import` を一件ずつ公開操作の経由へ直す。

## タスク

- [ ] T001 [App] 解決済みのテナントの受け渡しを Tenancy の公開パッケージへ移す。
- [ ] T002 [App] Tenancy の `usecases` と `db_postgres` への直接の依存を公開操作の経由へ直す。
- [ ] T003 [Tooling] 解消した違反 ID を `tools/check/boundary-debt.json` から消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 公開パッケージへ移した操作が、テナントの境界ではない用途にも使われるおそれがある。
  移す操作は解決済みのテナントの受け渡しに限り、Tenancy の管理操作は公開しない。
