---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
---

# Provisioning が宣言する標準の最後の 1 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/provisioning/standards.md` の 1 行を引き取る。

**この文書は 13 行のうち 12 行が既に名指しを持つ。** [[wi-238-scim-inbound-list-query-conformance]] の適合作業が id を名指すテストを書いたからであり、消化が可能であることの実例になっている。残る 1 行は `RFC7643-OUT-GROUP-RESOURCES` だけで、採用は `partial` である。**12 行を書いた作業がこの 1 行だけを残したという事実そのものが、この行の観測が他の 12 行と違う形を要求している徴候である。** 単に忘れられただけなのか、`partial` の境界が書きにくかったのかを、まずそこから読む。

## Scope

- `RFC7643-OUT-GROUP-RESOURCES`（`partial`）を消化する。
- その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// RFC7643-OUT-GROUP-RESOURCES: <この行の何を固定しているか>` の注記を足す。
- `partial` の観測は、採用した範囲の振る舞いと、採用していない範囲がどう扱われるか（拒否するのか、単に提供しないのか）の両方を持つ。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。
- 12 行が既に消化されているので、この行が消えた時点で `docs/contexts/provisioning/standards.md` は台帳から完全に外れる。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。Sourcing が持つ同じ RFC の 5 行は [[wi-505-back-sourcing-standards-rows-with-tests]] が持つ。
- 既に名指しを持つ 12 行の観測の見直し。名指しが実在の検証に付いているかを疑う作業は本項目の対象ではない。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。`partial` の境界が現状と食い違うと判明した場合は規範の変更であり、別の work item が扱う。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

`partial` の観測は 2 つ要る。片方だけでは、全部を採用している実装とも、何も採用していない実装とも区別できない。

`RFC7643-OUT-GROUP-RESOURCES` は Group リソースの送出を扱う行である。したがって観測すべき境界は、送出する範囲と送出しない範囲の間にある。行の `Statement` がその境界をどこに引いているかを読み、境界の内側で実際に送出されること、および外側が拒否されるのか単に含まれないのかを、その書きぶりに合わせて観測する。

**先に、なぜこの 1 行だけが残ったのかを読む。** 既に名指しを持つ 12 行のテストが `RFC7643-OUT-GROUP-RESOURCES` の範囲に触れていながら名指していないだけなら、必要なのは境界の観測を足すことである。触れてすらいないなら、Group の送出そのものにテストが無いということなので、そこから書く。この 2 つは作業量が違うので、着手前に区別する。

## Plan

1. 既に名指しを持つ 12 行のテストを読み、`RFC7643-OUT-GROUP-RESOURCES` の範囲に触れているものがあるかを確かめる。
2. 行の `Statement` が引いている採用の境界を読み、境界の内側と外側それぞれの観測を決める。
3. 境界の内側を消化する。
4. 境界の外側を、拒否なのか非提供なのかに合わせて消化する。
5. `standards-coverage-debt.json` から `RFC7643-OUT-GROUP-RESOURCES` を外し、`docs/contexts/provisioning/standards.md` が台帳から完全に外れたことを確かめる。

## Tasks

- [ ] T001 [Inventory] 既存の 12 行のテストが `RFC7643-OUT-GROUP-RESOURCES` の範囲に触れているかを確かめ、必要な作業量を区別する。
- [ ] T002 [Acceptance] 採用の境界の内側を消化する。
- [ ] T003 [Acceptance] 採用の境界の外側を、拒否か非提供かに合わせて消化する。
- [ ] T004 [Ledger] `RFC7643-OUT-GROUP-RESOURCES` を `standards-coverage-debt.json` から外す。
- [ ] T005 [Verify] `mise run verify`。

## Verification

- `RFC7643-OUT-GROUP-RESOURCES` が `tools/check/standards-coverage-debt.json` から消えている。
- `docs/contexts/provisioning/standards.md` の行が 1 つも台帳に残っていない。
- 注記を足したテストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **1 行だから注記だけで済ませる。** 件数が少ないことは、既存テストが行を区別している根拠にはならない。むしろ 12 行を書いた作業がこの 1 行を残したという事実が、区別が付いていないことの徴候である。T001 を飛ばさない。
- **`partial` を片側だけで消化する。** 境界の両側を観測する。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
