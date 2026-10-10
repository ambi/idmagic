# ワークフローの実行の設計

この文書は、[ワークフローの実行](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

IdManagement は、User を変更するときに、このモジュールが実装する `UserMutationCommitter` を呼ぶ。
`UserMutationCommitter` は変更の前後の User から WorkflowRun と WorkflowStep を計画し、User と一緒に一つのトランザクションで保存する。
モジュールをまたぐトランザクションを別に要求せずに、User だけが更新されて対応する実行が作られない事態を防ぐ。

保存した WorkflowRun は `job_id` のない `queued` で残る。
`worker` のディスパッチャーは、一定の間隔で `queued` の WorkflowRun を 100 件ずつ読み、重複しない `lifecycle_workflow_run` の Job を関連付ける。
Job の投入に失敗しても、次の走査で関連付け直す。

| アクション | 記録の正を持つモジュールへの書き込み |
| --- | --- |
| `enable_user`、`disable_user` | `UserLifecycle` のポートを通して、IdManagement の User の操作で行う。管理 API で止めたときと同じイベント、記憶済みの端末の失効、下流への通知、所有する Agent の無効化が伴う。削除予約中の User への操作は IdManagement が拒否し、ステップは変更なしとして扱う |
| `assign_application`、`unassign_application` | `ApplicationAssignments` のポートを通して、Application の割り当ての操作で行う |
| `add_group_member`、`remove_group_member` | IdManagement の Group の Repository を直接呼ぶ |
| `set_required_action`、`clear_required_action` | IdManagement の User の Repository で直接保存する |
| `send_email` | 共有の通知の部品で送る。対象の User に検証済みのプライマリのメールアドレスがなければ、ステップは失敗する |

アクションによる User の変更は `UserMutationCommitter` を通らないので、新しい WorkflowRun を計画しない。
`UserLifecycle` の実装も、IdManagement の User の操作を `UserMutationCommitter` なしで呼ぶ。
アクションからトリガーへの循環は、この経路の分離で断たれる。

## 信頼性

ステップは、実行するたびに結果をチェックポイントとして記録する。
プロセスが止まっても、Job の再試行は記録した結果を読み、`failed` と未完了のステップだけを実行する。
各ステップの前にワークフローの状態を読み直し、無効化または削除されていれば、次のステップの前で打ち切る。
