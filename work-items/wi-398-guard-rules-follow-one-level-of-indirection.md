---
depends_on: [wi-390-security-control-test-standard-and-gate]
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-23
priority: p1
change_kind: tooling
spec_impact: { kind: none, reason: "検査の強化であり、製品の振る舞いを変えない。検査が見つけた欠陥は個別に直す。" }
---

# 拒否の応答を書くヘルパーを 1 段挟むと R1 が見えなくなる

## Motivation

[[wi-390-security-control-test-standard-and-gate]] の R1 は「ガードが `WriteProblem` の戻り値を返している」形を落とす。`WriteProblem` は応答を書いた結果 (成功時は `nil`) を返すので、この形のガードは拒否を呼び出し元へ伝えられない。

**この判定は 1 段の間接で抜ける。** [[wi-391-refusal-declaration-floor-and-reinventory]] の作業中に、`requireAuthorizationAdmin` (`backend/authorization/handlers_http/routes.go`) と `requireWorkflowAdmin` (`backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go`) が `WriteAdminAccessError` の戻り値を返しており、当時の `WriteAdminAccessError` は `WriteProblem` の結果をそのまま返していた。つまり両ヘルパーは 403 を書いたうえで `nil` を返し、`if err := d.requireXxx(c); err != nil { return err }` は素通りしていた。

実害が出ていた。非管理者による認可モデルの登録は 403 を返しながらモデルの版を作っており、ライフサイクルワークフローの一覧は 403 の本文に続けてワークフロー一覧をそのまま書き出していた。wi-391 が `WriteAdminAccessError` に番人を返させて塞いだが、**塞いだのは 1 つの writer であって、形そのものではない**。同じ形の writer を新しく書けば、また R1 の外側に出る。

## Scope

- 「応答を書いてその結果を返す」関数を 1 つの writer 名に固定せず、推移的に求める。すべての `return` が応答書き込み (またはその性質を持つ別の関数) である関数を不動点で集め、その集合をガード位置で返している箇所を落とす。
- 既存の違反を洗い出し、番人を返す形へ直す。
- `security-controls.test.ts` に、間接が 1 段・2 段の場合をそれぞれ固定する。

## Out of Scope

- ユースケース層のセンチネル。[[wi-393-guard-rules-reach-the-usecase-layer]] が持つ。
- 拒否の宣言の最低要件。[[wi-391-refusal-declaration-floor-and-reinventory]] が持つ。

## Design

未定。着手時に、不動点の計算をどこまで行うか (パッケージ内に閉じるか、`backend` 全体か) と、偽陽性の見込みを測ってから決める。

## Plan

- 先に現状を測る。推移的な判定で何件が落ちるかを見てから、検査に入れるかどうかを決める。件数が多ければ、wi-390 と同じく縮むだけの許可リストを置く。

## Tasks

- [ ] T001 [Survey] 推移的な判定で落ちる箇所を数える。
- [ ] T002 [Design] 不動点の範囲と許可リストの要否を決める。
- [ ] T003 [Tooling] R1 を推移的な判定へ広げる。
- [ ] T004 [Guardrail] 間接 1 段・2 段の違反を固定するテストを置く。
- [ ] T005 [Fix] 見つかった違反を直す。

## Verification

- `mise run check`
- `mise run verify`
- 手動: `WriteAdminAccessError` を「書いて `nil` を返す」形へ戻すと、それを包むヘルパーが落ちることを確認する。

## Risk Notes

リスクは medium。推移的な判定は偽陽性を出しやすい。応答を書いて返すのが正しい関数 (ハンドラー本体、エラー写像) を落とすと、開発者は除外を書いて先へ進む。判定の決め手は wi-390 と同じく「呼び出し元が続行の判断にその値を使っているか」であり、そこは動かさない。
