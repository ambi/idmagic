---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# System が宣言する具体例 45 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は「no normative specification change against main」を返す。規範は動いていない。
動いたのは対応付けで、System が宣言する具体例 45 件のうち **43 件**を
`tools/check/example-coverage-debt.json` から外し、`//spec:covers` を持つテストへ結び付けた。
台帳の System 行は 45 件から 2 件へ減り、残る 2 件はどちらも `blocked_by` と `finding` を持つ。
消化の内訳は、既存テストへ観測を足したもの 22 件、新しく書いたもの 18 件、注記だけで済んだもの 3 件である。
**製品コードの振る舞いは変えていない。** 変えたのは 2 つの境界だけで、どちらも入力の受け取り方である。
`routes/index.tsx` の loader が式で持っていた起動時設定の判定を `demoLoginEnabled(env = import.meta.env)` へ
出し、`RenderConfigReference()` の内側に節・説明・registry を引数で受ける `renderConfigReference` を置いた。
新しく建てた観測点は 5 つのファイルである。運用資材と登録済み経路の突き合わせ
(`ops_assets_test.go`)、起動時設定の拒否が副作用の前で終わること
(`startup_refusal_e2e_test.go`)、`development` プロファイルだけがデモ資格情報を持つこと
(`demo_login_seed_test.go`)、API のエラー本文が英語であること
(`error_language_examples_test.go`)、生成物の乖離が再生成を促すこと
(`idmagic-route-reference/main_test.go`) である。
基盤へは `testing_stack` の `Browser.PostRawJSON` を足した。`json.Marshal` を通す入口では、
復号が失敗する要求そのものを作れなかった。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
