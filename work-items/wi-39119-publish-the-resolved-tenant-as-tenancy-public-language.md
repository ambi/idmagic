---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-09
priority: p2
depends_on: [wi-92970-adopt-information-hiding-modular-design]
change_kind: refactor
spec_impact: { kind: none, reason: "解決済みのテナントを運ぶ Go のパッケージの置き場所と、Context Map に Tenancy からの関係を書き足すだけである。テナントの解決規則、HTTP の応答、発行するドメインイベント、永続状態は変えない。" }
---

# 解決済みのテナントを Tenancy の公開言語として公開する

## 動機

`tools/check/boundary-debt.json` の境界の負債のうち 95 件は、ほかの Context から Tenancy への依存である。
内訳は二種類ある。

| 種類 | 件数の目安 | 原因 |
| --- | --- | --- |
| 公開されていないパッケージの import（`private-import`） | `backend/tenancy` 41、`backend/tenancy/usecases` 11、`backend/tenancy/db_postgres` 2 | 解決済みのテナントを `context.Context` に載せて読む `backend/tenancy/context.go` が、公開言語ではない Context の直下のパッケージにある。ほかに、Tenancy のユースケースを直接呼ぶ箇所がある |
| Context Map にない向きの依存（`undeclared-context-edge`） | OAuth2 23、Authentication 20、SharedSignals 7、SigningKeys 6、Jobs 4、IdGovernance 4 ほか | Context Map は Tenancy から IdManagement と Application への関係しか宣言していない |

テナントの境界は、ほぼすべての Context が前提にする。
それにもかかわらず、その前提を受け取る経路が公開言語にも Context Map にもないため、正しい依存までが違反として負債に載っている。

## 対象範囲

- `backend/tenancy/context.go` の `WithTenant`、`Tenant`、`TenantID`、`Issuer`、`URLPrefix` を、Tenancy の公開言語（`backend/tenancy/ports` または `backend/tenancy/domain`）へ移し、呼び出し元を書き換える。
- `docs/design/architecture/logical.md` の Context Map に、Tenancy を Supplier とする関係を、実際に依存する Context の分だけ書き足す。
  書き足すのは、テナントの境界を受け取るだけの関係に限る。
- ほかの Context が Tenancy の `usecases` と `db_postgres` を直接使う箇所を、Tenancy の `ports` を経由する形へ直す。
- 解消した違反 ID を `tools/check/boundary-debt.json` から消す。

## 対象外

- テナントの解決規則そのものの変更。
- Tenancy 以外を Supplier とする未宣言の関係。残りの負債の棚卸し（wi-33994）で扱う。

## 設計

置き場所は `ports` を第一候補にする。
`context.Context` に値を載せる処理は計算ではなく実行時の受け渡しなので、`structure.md` が決定論的な計算だけを置くと定める `domain` には合わない。

Context Map への書き足しは、依存を正当化するための後付けではない。
テナントの境界は Open Host Service として全 Context へ公開する関係であり、いまの Context Map は IdManagement と Application の二つしか書いていないことが記述の欠落である。
書き足すときは、関係の名前を既存の `OHS/PL: tenant boundary` にそろえる。

採らない案は次のとおり。

| 案 | 採らない理由 |
| --- | --- |
| テナントの受け渡しを `backend/shared` へ移す | Tenancy の語彙（`domain.Tenant`）を運ぶので、`shared` を経由した Tenancy への到達として `shared-detour` に形を変えるだけである |
| 境界の検査で Tenancy を例外にする | 検査に Context 固有の例外を持ち込むと、Context Map という正本が検査の判断を説明しなくなる |

## 計画

1. Context Map の書き足しを先に行い、`undeclared-context-edge` のうちテナントの境界だけを受け取るものが消えることを確かめる。
2. `context.go` を移し、呼び出し元を一括で書き換える。
3. 残った `private-import` を一件ずつ `ports` 経由へ直す。

## タスク

- [ ] T001 [Design] `docs/design/architecture/logical.md` の Context Map に Tenancy からの関係を書き足す。
- [ ] T002 [App] 解決済みのテナントの受け渡しを Tenancy の公開言語へ移す。
- [ ] T003 [App] Tenancy の `usecases` と `db_postgres` への直接の依存を `ports` 経由へ直す。
- [ ] T004 [Tooling] 解消した違反 ID を `tools/check/boundary-debt.json` から消す。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run verify`

## リスク

- 書き足した関係が、テナントの境界ではない依存まで許してしまうおそれがある。
  関係を書き足した後も、Tenancy の `domain` と `ports` 以外への依存は `private-import` として検出されるので、許すのは公開言語への依存だけである。
