# Provisioning のページ

この文書は、Provisioning が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Provisioning のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## プロビジョニング

| ページ | URL | 目的 |
| --- | --- | --- |
| [アプリケーションのプロビジョニング](admin-application-provisioning.md) | `/admin/applications/$applicationId/provisioning` | 接続先の同期設定、状態と処理タスクを管理する |
| [プロビジョニングの概要](admin-provisioning.md) | `/admin/provisioning` | テナント内の接続状態を一覧で確認する |
