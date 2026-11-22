# Jobs のページ

この文書は、Jobs が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Jobs のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## ジョブ

| ページ | URL | 目的 |
| --- | --- | --- |
| [テナントのジョブ](admin-jobs.md) | `/admin/jobs` | 非同期処理の進捗と失敗を調べ、停止を要求する |
| [システムのジョブ](system-jobs.md) | `/system/jobs` | テナントを横断する非同期処理を調べる |
