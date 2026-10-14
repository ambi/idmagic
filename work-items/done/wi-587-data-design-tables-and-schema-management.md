---
status: completed
archived_from: 2c4d62bd910232b33df0daa84f70a8e6f20e732b
---

# データ設計にテーブルの役割と関係、スキーマ管理の設計を書く

`mise run spec-diff` は main に対して規範の変更がないことを報告した。変更は設計文書と文書検査に限られる。`docs/design/data/database.md` の六つの ER 図に、全 88 テーブルの役割、所有 Context、テーブル種別、`tenant_id` 列の区分を示すテーブル一覧を添え、所有 Context の外から行われる四種類の書き込みを表にした。「再生成可能な認証状態」の図が `LOGGED` テーブルの `oauth2_access_token_denylist` と `login_throttle_counters` を `UNLOGGED` と説明していた誤りを直し、`LOGGED` にしている理由を書いた。スキーマの宣言、収束の検査、拡張と縮小の四段階、データ移行、適用する地点、`psqldef` の規則を `docs/design/data/schema-management.md` に集め、`database.md`、`maintenance.md`、`infra/schema/README.md` からは参照に置き換えた。README がデータ移行の置き場所を `infra/schema/` の外としていた誤りも直した。`lifecycle.md` は担当範囲を宣言し、データの区分ごとの消え方、削除と匿名化の判断基準、`RESTRICT` と `CASCADE` から決まるテナント退去の順序（仮）、暗号学的消去が消すものと消さないもの、復元で消去済みのデータが戻る問題を書いた。リポジトリ検査 `schema-tables`（`mise run check-schema-tables`）を加え、テーブル一覧のテーブル名の集合、テーブル種別、`tenant_id` 列の区分を `postgres.sql` と照合する。着手時の読み取りで見つかった二つの欠陥を wi-611 と wi-612 に起票した。

全文は `git show 2c4d62bd910232b33df0daa84f70a8e6f20e732b:<このファイルのパス>` で読む。
