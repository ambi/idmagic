---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# テナント横断ヘルスの認可を制御面主体の共通判定へ統一する

`mise run spec-diff` は `REQ-DATAKEYS-006`、`REQ-JOBS-012`、`REQ-JOBS-013`、`REQ-SIGNINGKEYS-008`、`REQ-SIGNINGKEYS-009` の変更を返した。制御面主体を、有効な `default` テナント所属 User が `default` テナントの経路で有効ロール `system_admin` を持つ場合に限定する共通判定を追加し、署名鍵と DEK の横断ヘルス、Tenancy、Audit、Jobs、フロントエンドの `/system` ガードへ適用した。拒否時は横断収集を開始せず、正規の制御面主体は `admin` を併有しなくても Jobs の横断一覧、詳細参照、取消しを行える。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
