---
status: completed
archived_from: 83c79e84f6f8a032d37151231d16c977400f10c0
---

# IdGovernance が宣言する具体例 21 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `main` に対して `no normative specification change` を返した。
IdGovernance が台帳に残していた 21 件の具体例をすべて実装と照合し、`//spec:covers` を付けたテストへ結び付けて台帳から外した。
既存テストへディレクティブを足すだけで済んだ件は無かった。
`EX-IDGOVERNANCE-005-01`、`-013-02`、`-013-03` は既存テストに観測（重ならないメンバーシップ、下書きのトリガー、全アクションの blocked）を足した。
残りの 18 件は新しいテストを書いた。画面の 4 件は作成画面と編集画面の操作、実行系は IdManagement のユースケースと jobs の Runner を通した観測、管理 API はテナント "acme" を含む `Register` の組み立て、同一トランザクションは埋め込み PostgreSQL である。
照合で 5 件の食い違いが分かった。具体例以外の規範（TypeSpec の操作説明、`decisions.md`、`states.md`）とも照合し、すべて実装の欠陥と判定して、利用者の指示により本項目で直した。
管理 API は入力の検証失敗を `invalid_workflow`、存在しないワークフローを 404 で返しており、契約が宣言する InvalidRequestError と食い違っていた。
保存と有効化は、動的グループ、属性スキーマに無いフィルターのフィールド、別テナントや存在しないグループとアプリケーションを拒否していなかった。
実行ハンドラーは、無効化されたワークフローの running の WorkflowRun を次のステップへ進めていた。
失敗したステップのある試行は WorkflowRun をその場で終端にし、Jobs に再試行させていなかった。

全文は `git show 83c79e84f6f8a032d37151231d16c977400f10c0:<このファイルのパス>` で読む。
