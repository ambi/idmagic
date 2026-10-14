---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# バックアップに含まれる鍵素材の保護を、手順ではなく規範として持つ

`mise run spec-diff` が示す規範上の差分は `REQ-SIGNINGKEYS-012` の追加 1 件である。秘密鍵素材の保管先が提供元で決まること、平文で永続化する構成を既定にしないこと、保存先の暗号化は配備側の責務であること、本番判定に基づく提供元の拒否は判定材料が無いため採らないことを `docs/domain/signing-keys/decisions.md` に規範として置いた。runbook は保護を要求する側から手段を示す側へ退き、同じ内容を二重に持たなくなった。起動時設定は `PERSISTENCE=postgres` で `KEY_PROVIDER` の明示を要求するようになり、未指定のまま平文の KeyStore が組み立てられる経路が閉じた。TypeSpec の `KeyProvider` は `Database` が鍵素材をプロセス内に持つと述べていたが、実際にはデータベースへ平文で置くので doc を実態に合わせた。副産物として `infra/deploy/gcp/cloudrun-idmagic.yaml` の `KEY_PROVIDER: "db"` が許可されない値であること、すなわちこの本番向けサンプルがそもそも起動しないことが分かり、`local` に正した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
