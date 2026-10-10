# Feature: 標準開発環境の例

## Rule: REQ-JOBS-001 Docker なしの標準開発環境で `worker` ジョブを完了する

### Example: EX-JOBS-001-01 通常経路

- Given 組込み PostgreSQL バイナリが取得済み、または初回取得可能である
- And 開発用ポートが利用可能である
- When 開発者が標準開発コマンドを実行する
- Then 組込み PostgreSQL が起動しスキーマが適用される
- Then API、`worker`、UI が起動する
- Then API が Job をキューへ投入する
- Then `worker` が同じ Job を取得して Succeeded にする
- Then API と `worker` は同じ PostgreSQL キューを共有する

### Example: EX-JOBS-001-02 バイナリの取得、ポートの確保、またはスキーマの適用に失敗する

- Given 組込み PostgreSQL バイナリが取得済み、または初回取得可能である
- And 開発用ポートが利用可能である
- When 開発者が標準開発コマンドを実行する
- Then バイナリの取得、ポートの確保、またはスキーマの適用に失敗する
- Then 標準開発環境は API と UI を起動せずフェイルファストする
