# 標準開発環境

## 概要

この文書は、Docker を使わない標準の開発環境で、API、`worker`、UI が同じ PostgreSQL のキューを共有して動く仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 組み込みの PostgreSQL の起動とスキーマの適用、API と `worker` と UI の起動、失敗のときの早期の終了 |
| 行為者 | Developer |
| 扱わないもの | 本番の配置は[配置の設計](../../../design/architecture/deployment.md)が扱う |

## 操作

### 開発者による標準開発環境の起動

#### REQ-JOBS-001 Docker なしの標準開発環境で `worker` ジョブを完了する
