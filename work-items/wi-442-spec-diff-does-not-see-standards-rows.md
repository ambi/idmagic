---
depends_on: []
status: pending
authors: [tn]
risk: low
created_at: 2026-08-29
priority: p3
change_kind: tooling
spec_impact: { kind: none, reason: "規範の差分を読む道具の視野を広げるだけで、規範シナリオ、規範 ID、TypeSpec シンボルを追加も変更もしない。" }
---

# `spec-diff` が `standards.md` の行の増減を見えるようにする

## Motivation

`mise run spec-diff` は「変更が規範仕様に何をしたか」を git から計算して読ませる道具で、`WORK_ITEM_FORMAT.md` は完了時の要約をここから読むことを求めている。見ているのは規範シナリオ、状態遷移の行、TypeSpec の宣言の 3 つである。

**`standards.md` の行は入っていない。**

[[wi-403-provisioning-declares-no-scim-conformance]] は `docs/contexts/provisioning/standards.md` を新設し、RFC 7643 / RFC 7644 の準拠範囲を 12 行宣言した。`spec-diff` の出力は `no normative specification change against main` だった。**成果物の全体が、規範の差分を読む道具から見えなかった。**

これは要約を書くときだけの不便ではない。`standards.md` の行は、`Adoption` が `excluded` から `partial` へ動けば製品の約束が変わり、`Strength` が `MUST` から `SHOULD` へ落ちれば守る強さが変わる、正真正銘の規範である。その変化がレビューの視野に入らないまま通る経路がいま開いている。行の削除も同じで、規範 ID は消えても誰も差分として気付かない。

## Scope

- `standards.md` の行を `spec-diff` の対象に加える。追加、削除、`Adoption` と `Strength` と `Statement` の変化を区別して出す。
- 規範 ID の削除を、シナリオの削除と同じ重さで出す。
- `docs/` 直下と Context 直下の両方の `standards.md` を対象にする。

## Out of Scope

- 規範 ID の被覆の検査。[[wi-418-normative-coverage-gates]] が持つ。
- `spec-diff` をゲートにすること。この道具は読解の補助であり、何も落とさないという位置づけを変えない。
- `glossary.md` と `internals.md` の差分。散文であり、行単位の意味を持たない。

## Design

未定。着手時に、行の同一性を規範 ID で取るか行全体で取るかを確定して本節に記録する。

## Plan

1. `standards.md` の行を読む関数を、既存の状態遷移の読み取りと同じ形で足す。
2. 差分の出力に、追加、削除、列ごとの変化を加える。
3. `spec-diff.test.ts` に、`Adoption` だけが変わった場合を含める。

## Tasks

- [ ] T001 [Design] 行の同一性の取り方を確定する。
- [ ] T002 [Test] `Adoption` の変化が差分に出ることのテストを RED で置く。
- [ ] T003 [Tooling] `spec-diff` に `standards.md` を足す。
- [ ] T004 [Verify] `mise run verify`。

## Verification

- `mise run test-tools`
- `mise run verify`
- 手動: `standards.md` の 1 行の `Adoption` を変えて `mise run spec-diff` を実行し、その行が出ることを確かめる。

## Risk Notes

リスクは low。読解の補助を広げるだけで、何も落とさない。出力が長くなりすぎると読まれなくなるので、変化した列だけを出すかどうかは実測してから決める。
