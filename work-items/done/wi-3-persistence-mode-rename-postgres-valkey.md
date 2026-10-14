---
status: completed
archived_from: 61f5d0fe18c17eadc4a23f1b9048389f098c55ca
---

# PERSISTENCE モードの値を postgres から postgres_valkey に改名し durable/volatile 双方のバックエンドを名前へ反映する

system.yaml の SharedEphemeralStateHA description を `postgres_valkey` に更新し、
just scl-render で spec/idmagic.html を再生成した (JSON schema/openapi はこの prose を
含まないため差分なし、drift なし)。internal/bootstrap では assemble の switch 値・
default エラー文言・構造体コメントを `postgres_valkey` に更新し、assemblePostgres を
assemblePostgresValkey にリネーム、postgres.go を git mv で postgres_valkey.go へ改名、
`PERSISTENCE=postgres_valkey requires ...` のエラー文言も更新した。durable アダプタ
パッケージ persistence/postgres は Postgres 実装そのものとして据え置いた。README の
env 表・HA 節、ARCHITECTURE の runtime 選択と assemble 関数名、docker-compose.dev.yaml の
PERSISTENCE 値、ADR-016 の選択子定義 (memory | postgres_valkey) を更新し、ADR-016 には
postgres 単独名が Valkey 必須の事実を隠す点・将来 mysql_valkey とも区別できる点という
改名理由を追記した。過去の履歴記録は据え置いた。

全文は `git show 61f5d0fe18c17eadc4a23f1b9048389f098c55ca:<このファイルのパス>` で読む。
