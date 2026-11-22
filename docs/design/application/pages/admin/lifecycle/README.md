# ライフサイクル

この文書は、ライフサイクルのページ一覧を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[UI ページ](../../README.md) / [管理コンソール](../README.md)

## ページ一覧

| ページ | URL | 目的 |
| --- | --- | --- |
| [ライフサイクルワークフロー](../../../../../modules/identity-governance/pages/admin-lifecycle-workflows.md) | `/admin/lifecycle-workflows` | 状態変化に応じた自動処理と実行結果を管理する |
| [ワークフローを作成](../../../../../modules/identity-governance/pages/admin-lifecycle-workflow-create.md) | `/admin/lifecycle-workflows/new` | 開始条件と順序付きの処理を定義する |
| [ワークフローを編集](../../../../../modules/identity-governance/pages/admin-lifecycle-workflow-edit.md) | `/admin/lifecycle-workflows/$workflowId/edit` | リビジョンを確認して処理の定義を変更する |
| [アクセスレビュー計画](../../../../../modules/identity-governance/pages/admin-access-reviews.md) | `/admin/access-reviews` | 定期レビューの計画と次回開始を確認する |
| [レビュー計画を作成](../../../../../modules/identity-governance/pages/admin-access-review-create.md) | `/admin/access-reviews/new` | 対象、レビュー担当と繰り返しを指定する |
| [レビュー計画を編集](../../../../../modules/identity-governance/pages/admin-access-review-edit.md) | `/admin/access-reviews/$planId/edit` | 今後のレビューの条件を変更する |
| [レビュー計画の詳細と結果](../../../../../modules/identity-governance/pages/admin-access-review-detail.md) | `/admin/access-reviews/$planId` | 計画の稼働状態と各レビューの判断・適用結果を確認する |
