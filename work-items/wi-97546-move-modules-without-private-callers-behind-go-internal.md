---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on: [wi-50428-move-one-module-behind-go-internal]
change_kind: refactor
spec_impact: { kind: none, reason: "外から非公開パッケージへの import がないモジュールのパッケージを Go の internal/ へ移し、責務表の公開方式と公開パッケージを書き換えるだけである。HTTP の応答、認証方式、永続状態、ドメインイベント、外向きの通知は変えない。" }
---

# 外から非公開パッケージへの import がないモジュールを Go の internal/ へ移す

## 動機

モジュール設計の項目は、非公開の実装の最終的な配置を Go の `internal/` と定め、全モジュールを公開方式 `legacy` のまま残した。
最初の一つのモジュールの移行は別の項目で手順を確かめるが、二つ目以降の移行を受け持つ項目がない。
このままでは、文書が定める最終形と実装の乖離が解消されない。

ほかのモジュールが非公開パッケージを import しているモジュールは、その依存を解消するまで `internal/` へ移せない。
移すと Go のビルドが失敗するからである。
起票時の負債台帳では、外から非公開パッケージへ import されるモジュールは次の 8 個だった。

| モジュール | 外から非公開パッケージへの import |
| --- | --- |
| Tenancy | 54 |
| Authentication | 19 |
| ClaimMapping | 11 |
| Jobs | 9 |
| OAuth2 | 6 |
| IdManagement | 6 |
| WsFederation | 4 |
| Application | 2 |

残りの 12 個（IdGovernance、Authorization、Audit、Provisioning、Sourcing、ApiTokens、Seeding、SigningKeys、DataKeys、Saml、WorkloadIdentity、SharedSignals のうち、最初の移行の項目が選んだものを除く）は、ほかのモジュールの依存を変えずに移せる。
この項目はそれらを移し、残る 8 個は負債の解消後に[残りのモジュールの移行の項目](wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md)で移す。

## 対象範囲

- 着手時に `mise run check-boundaries` と台帳から、外から非公開パッケージへの import がないモジュールを再計測して対象を確定する。
- 対象の各モジュールの本番パッケージを、ルートパッケージ、公開パッケージ、`internal/` のどれかに分類して移す。
- 責務表の公開方式を `internal` にし、公開パッケージを列挙する。
- 組み立て地点からの非公開パッケージへの直接の import を、ルートパッケージの操作へ移す。
- 最初の移行の項目で見つかった手順の不足が構造の文書へ反映されていなければ、ここで反映する。

## 対象外

- 外から非公開パッケージへ import されるモジュールの移行と、その依存の解消。
- `legacy` の公開方式の撤去。
- モジュールの分割と統合。
- 機能スライス間の規則の追加。

## 設計

公開パッケージを決めることは公開範囲の宣言であり、[境界を選ぶ判断手順](../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)を適用する。
現在 `legacy` の判定で公開扱いの `domain` と `ports` を、そのまま公開パッケージへ列挙することを既定にしない。
ほかのモジュールと組み立て地点が実際に使う型と操作を調べ、使われていない `domain` は `internal/` へ移す（D3）。

モジュールごとに、外側に残るパッケージの分類、公開パッケージとその利用元、組み立て地点の結線の変更を設計の節へ記録する。
移行はモジュールごとに独立しているので、モジュール単位でコミットし、途中で止めても `legacy` と `internal` の併用で検査が通る状態を保つ。

## タスク

- [ ] T001 [Inventory] 対象のモジュールを再計測して確定し、モジュールごとに外部の利用元と組み立て地点からの import を記録する。
- [ ] T002 [Design] モジュールごとにパッケージの分類と公開パッケージを判断手順で決める。
- [ ] T003 [App] モジュールごとにパッケージを移し、利用元と組み立て地点を書き換え、責務表を `internal` にする。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 外部の利用元に合わせて公開パッケージを広げると、`legacy` の判定より公開範囲が広がる。
  公開パッケージの追加ごとに D3 で、公開操作で表せるかを比べる。
- 移すパッケージ数が多く、差分が大きくなる。
  モジュール単位のコミットに分け、パッケージの移動と結線の変更を読み分けられるようにする。
