---
status: completed
archived_from: 239e261929cfcc75bb27e16d42e0025359d5c7d6
---

# 仕様に書かれていない振る舞いを、仕様、実装、テストの各工程で機械的に検出する

`mise run spec-diff -- main` の結果は「no normative specification change against main」であり、規範仕様の差分はない。
`mise run check-spec` は、状態遷移表（マトリクス形式）の空のセル、語彙にない結果、状態の表にない状態、行のない状態、遷移の表と食い違う遷移を拒否するようになった。
新しい検査 `mise run check-unspecified-vocabulary`（`mise run check` にも含む）は、IdManagement の TypeSpec のエラーコードと Go のドメインイベント 72 件のうち、どの要件にも現れない 17 件（エラーコード 8 件、イベント 9 件）を `tools/check/unspecified-vocabulary-debt.json` に許容リストとして載せた。17 件は、廃止した操作ごとの要約表にだけ書かれていたものと、どこにも書かれていないものである。
`backend/shared/testing_statematrix` が機能仕様から状態機械を読み、`TestUserLifecycleFollowsTheStateMatrixInEveryCell` と `TestUserLifecycleFollowsTheStateMatrixAlongRandomWalks` が、UserLifecycle の全セルと乱数の操作列を管理 API で実行して表の予測と比べる。導入時点で表と実装は一致した。
仕様先行の開発ワークフローに、各検出の出力を (a)(b)(c) の分類と記録先へつなぐ表を加えた。
着手時に見つけた、どの遷移も到達しない `UserStatus` の三つの値を (c) とし、wi-98121 として起票した。

全文は `git show 239e261929cfcc75bb27e16d42e0025359d5c7d6:<このファイルのパス>` で読む。
