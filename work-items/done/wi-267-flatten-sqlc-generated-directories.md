---
status: completed
archived_from: 8ae93c4f2f02d47881699a2c4e923fc372ccd96d
---

# sqlc の生成コードとクエリを `db_postgres` 直下へフラット化する

`sqlcgen` ディレクトリと `queries` ディレクトリをすべて `db_postgres` 階層へ統合し、`sqlc.yaml` と対象ソースコードからのインポートをフラット化した。`oauth2` コンテキストでのエイリアス経由の循環依存と、手動リポジトリでの名前衝突 (`saveUser`) およびテストファイル（`_test.go`）でのインポート循環をそれぞれ解消し、全テストパスを確認した。

全文は `git show 8ae93c4f2f02d47881699a2c4e923fc372ccd96d:<このファイルのパス>` で読む。
