---
depends_on: [wi-390-security-control-test-standard-and-gate]
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-23
priority: p1
change_kind: tooling
spec_impact: { kind: none, reason: "検査の強化であり、製品の振る舞いを変えない。検査が見つけた欠陥は個別に直す。" }
initial_context:
  source:
    - tools/check/src/security-controls.ts
    - tools/check/src/check-security-controls.ts
    - backend/shared/http/support_http/auth.go
    - backend/shared/http/support_http/csrf.go
    - backend/authorization/handlers_http/routes.go
    - backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go
  tests: [tools/check/src/security-controls.test.ts]
  stop_before_reading: [spec, frontend]
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

**「応答を書いた結果を返す関数」の一覧を、増えなくなるまで広げながら作る。** 出発点は echo 自身の応答書き込みメソッドである。まず `return c.JSON(...)` のように、引数で受け取った `*echo.Context` に対する書き込みを呼んでその結果を返している関数を一覧に入れる。次に、一覧に入っている関数を呼んでその結果を返している関数を一覧に足す。これを新しく足せるものが無くなるまで繰り返すと、`c.JSON` から何段離れていても「応答を書いた結果を返す」関数がすべて集まる。

wi-390 の R1 は `WriteProblem` という 1 つの名前を直接見ていた。名前を 1 つ選ぶこと自体が穴の原因なので、出発点はこのリポジトリの規約ではなくフレームワークの境界に取る。`WriteProblem` も `NoStoreJSON` も名指しせず、`c.JSON` からたどって一覧に入る。

**1 つでも該当する `return` があれば一覧に入れる。** Scope は「すべての `return` が応答書き込みである関数」と書いていたが、測ったところこれでは目的の形が落ちない。事故当時の `WriteAdminAccessError` は末尾に `return err` (写像できなかったエラーの素通し) を持つので、「すべての `return`」を条件にすると一覧から外れてしまう。危険なのは「応答を書いた結果をそのまま返す経路が 1 つでもあるか」なので、すべてではなく 1 つでも、を条件にする。

**誤検知は位置で切る。** 一覧は 394 件まで広がる — 応答を書いて返すのはハンドラーとエラー写像の正しい仕事なので、これは想定どおりである。落とす条件は wi-390 から動かさない: 要求スコープであり、かつ呼び出し元が `if err := f(c); err != nil` で続行を判断している関数だけを見る。番人を返す今の `WriteAdminAccessError` は `return refused(...)` なので一覧に入らず、`refused` 自身も変数と番人しか返さないので入らない。

**探す範囲は `backend` 全体。** 検査は既に名前ベースで `backend` 全体を読んでおり、パッケージ内に閉じると `support_http` の writer を包む各文脈のヘルパーがまさに見えなくなる。

**許可リストは置かない。** 現在の作業ツリーで新規の指摘は 0 件だった。縮むだけのリストであっても、空のリストは次に足す口実になる。

**R2 の同じ穴も塞ぐ。** R2 が「要求ガード」を認識する条件も `WriteProblem` という名前に固定されていた。R1 と同じ一覧で判定する。

## Plan

- 先に現状を測る。推移的な判定で何件が落ちるかを見てから、検査に入れるかどうかを決める。件数が多ければ、wi-390 と同じく縮むだけの許可リストを置く。→ 測定済み。下記のとおり許可リストは不要。

## Survey

wi-391 の修正前 (`637991ba^`) と現在の作業ツリーの両方に、推移版の判定を当てた。

| ツリー | 現行 R1 | 推移版 R1 |
|---|---|---|
| `637991ba^` (wi-391 修正前) | 0 件 | **4 件** |
| 現在 | 0 件 | 0 件 |

修正前に落ちた 4 件は、いずれも `WriteAdminAccessError` の戻り値をガード位置で返していた:

- `backend/authorization/handlers_http/routes.go` の `requireAuthorizationAdmin`
- `backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go` の `requireWorkflowAdmin`
- `backend/apitoken/handlers_http/routes.go` の `requireAdmin`
- `backend/authentication/federation/handlers_http/routes.go` の `requireAdmin`

**wi-398 が想定していたのは前の 2 つだが、実際には同じ形が 4 箇所にあった。** wi-391 は writer を直したので 4 つとも塞がっているが、形を落とす判定はどれも持っていなかった。

R2 の要求ガード集合は 3 件から 4 件へ増える (`validateEntraSourceAnchors` が加わる) が、破棄されている呼び出しは無いので指摘は 0 件のままである。

## Tasks

- [x] T001 [Survey] 推移的な判定で落ちる箇所を数える。修正前 4 件 / 現在 0 件。
- [x] T002 [Design] 不動点の範囲と許可リストの要否を決める。`backend` 全体、許可リストなし。
- [x] T003 [Tooling] R1 を推移的な判定へ広げる。R2 の writer 判定も同じ集合に載せ替えた。
- [x] T004 [Guardrail] 間接 1 段・2 段の違反を固定するテストを置く。
- [x] T005 [Fix] 見つかった違反を直す。現在のツリーに違反は無く、修正は不要だった。

## Verification

- `mise run check`
- `mise run verify`
- 手動: `WriteAdminAccessError` を「書いて `nil` を返す」形へ戻すと、それを包むヘルパーが落ちることを確認する。

## Risk Notes

リスクは medium。推移的な判定は偽陽性を出しやすい。応答を書いて返すのが正しい関数 (ハンドラー本体、エラー写像) を落とすと、開発者は除外を書いて先へ進む。判定の決め手は wi-390 と同じく「呼び出し元が続行の判断にその値を使っているか」であり、そこは動かさない。

実測では、この懸念は現れなかった。「応答を書いた結果を返す」関数は 394 件に広がるが、そのうちガード位置で呼ばれている要求スコープの関数は 1 件も無い。R2 が認識する要求ガードは 3 件から 4 件へ増えたが、結果を捨てている呼び出しは無いので指摘は増えていない。

## Completion

- **Completed At**: 2026-08-23
- **Summary**:
  R1 と R2 が「応答を書いてその結果を返す」関数を認識する方法を、`WriteProblem` という名前の直接照合から、echo の応答書き込みメソッドを出発点にした一覧の構築へ替えた。ガードが writer をどれだけ深く包んでも同じ形として落ちる。製品の振る舞いは変えていない (`mise run spec-diff`: `no normative specification change against main`)。指摘の本文は `requireAuthorizationAdmin -> WriteAdminAccessError -> WriteProblem -> NoStoreJSON -> c.JSON` のように連鎖を示すので、どこを直せばよいかが指摘だけで分かる。R2 の要求ガード判定も同じ一覧に載せ替えた。
  検査の強化のみで、`backend/` への変更は無い。現在のツリーに違反は無かったため許可リストは置いていない。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run test-tools` - passed (167 tests, 0 fail)。RED を先に確認した: 新規 3 件 (間接 1 段・2 段・`c.NoContent` 直返し) が実装前に失敗し、実装後に通った。
  - `mise run check-security-controls` - passed (180 declared refusal(s), 18 promised by a 403 on a state change, 131 awaiting a test)
  - `mise run typecheck-tools` / `mise run lint-tools` - passed。`lint-tools` の警告 2 件は `check/src/specification-doc.ts` の既存のもので、本変更とは無関係。
  - 手動: `WriteAdminAccessError` を「書いて `nil` を返す」形へ戻すと、それを包む 4 つのヘルパー (`requireAuthorizationAdmin`, `requireWorkflowAdmin`, `apitoken` と `authentication/federation` の `requireAdmin`) がすべて R1 で落ちた。元へ戻して再度 passed を確認した。
  - 遡及確認: wi-391 の修正前 (`637991ba^`) のツリーへ当てると 4 件を指摘する。現行 R1 は同じツリーで 0 件だった。
