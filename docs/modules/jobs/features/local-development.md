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

- 開発者が標準の開発コマンドを実行したとき、Jobs は、組み込みの PostgreSQL を起動してスキーマを適用し、API、`worker`、UI を起動し、API と `worker` に同じ PostgreSQL のキューを共有させる。
- 標準の開発環境で API がジョブを投入したとき、Jobs は、`worker` に同じジョブを取得させて `succeeded` にする。
- PostgreSQL のバイナリの取得、ポートの確保、スキーマの適用のどれかに失敗した場合、Jobs は、API と UI を起動せずに標準の開発環境を早期に終了する。
- **例**：EX-JOBS-001-01、EX-JOBS-001-02
