---
status: completed
archived_from: a8e03c729bc0cb0163e004b8abe40c9d40181425
---

# リポジトリが所有する全ソース種別に、宣言された検査を持たせる

`mise run spec-diff` は空である。規範も TypeSpec も変えていない。意味上の差は、リポジトリが所有していながら検査を持っていなかったソース種別 — シェルスクリプト 8 本、GitHub Actions ワークフロー、Dockerfile 2 本、k6 モジュール — が、固定版のツールによる検査を持ったことである。Markdown だけは道具を入れて評価した結果、採らない判断に至り、宙に浮いていた設定を消した。加えて、作業ツリーの秘密情報スキャンが無い状態から有る状態になり、`tools` の依存が `latest` から固定版と `--frozen-lockfile` になり、`frontend/tests` と `frontend/scripts` が型検査の範囲に入った。CI で検証しない infra アセットの判断は、`docs/development/continuous-integration.md` に理由ごと残した。

全文は `git show a8e03c729bc0cb0163e004b8abe40c9d40181425:<このファイルのパス>` で読む。
