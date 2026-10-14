---
status: completed
archived_from: 2887861eef0129ec9e2e6b3ba880822d5e413f3e
---

# ジョブの投入と処理の登録を Jobs の公開契約にする

`mise run spec-diff` は規範の差分なしを報告する。仕様の文面は変えず、実装を `REQ-JOBS-002` と `REQ-TENANCY-013` へ揃えた。
これまで IdGovernance、データエクスポート、Provisioning、動的グループの再評価、DataKeys の投入は `JobEnqueued` を発行せず、DataKeys の投入は `active_jobs` を数えていなかった。すべての投入が両方を行うようにした。
ほかのモジュールは、Jobs の公開パッケージに置いた `jobsports.Enqueuer` で投入し、処理の登録は `jobsports.Handler` と `jobsports.HandlerRegistrar` で行う。
投入器は `jobsusecases.NewEnqueuer` が作り、保存先、クォータ、発行の関数のどれかが欠けていれば起動時に panic する。本番の組み立てだけが投入器を作る。
夜間の再暗号化の掃除は、上限に達したテナントを飛ばして次へ進む。
境界の負債の台帳から Jobs への `private-import` の 9 件を消し、台帳は 137 件から 128 件になった。

全文は `git show 2887861eef0129ec9e2e6b3ba880822d5e413f3e:<このファイルのパス>` で読む。
