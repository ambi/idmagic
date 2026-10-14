---
status: completed
archived_from: 2c4d62bd910232b33df0daa84f70a8e6f20e732b
---

# テナントのパスワードポリシーを永続化し、任意の有効期限で次回ログイン時の変更を強制する

テナントのパスワードポリシー上書きが PostgreSQL に永続化されるようになり、`tenants` に
`password_policy_override` (JSONB) と `password_policy_updated_at` が加わった。これまで
`SaveTenant` は上書きを書いておらず、PostgreSQL 配備では管理者が保存した値が保存時点で
失われていた (db_memory でのみ機能していた)。
併せて、テナントが opt-in できる有効期限 `max_age_days` (既定 0 = 無効、system bounds
30〜3650 日) を追加した。期限は `max(password_changed_at, password_policy_updated_at)` から
測るため、有効化した瞬間に既存ユーザーが一斉に期限切れになることはない。期限超過は認証を
失敗させず、ログイン成立後に `update_password` required action を付与して既存の
change-password gate に載せる。password credential を持たないユーザーは対象外。
ポリシー適用の取りこぼしも塞いだ: 管理者による `CreateUser` は global default ではなく
テナント解決済みポリシーで検証するようになり、change-password は reset-password と同じ
breach チェックを通すようになった。`PasswordPolicySnapshot` は domain 側の単一定義へ
集約し (usecases 側は型エイリアス)、層をまたぐ詰め替えを無くした。
正規シナリオの差分は `REQ-AUTHENTICATION-024` の追加と `REQ-TENANCY-019` の変更
(弱い上書き / system bounds の ALT と、永続性の THEN)。

全文は `git show 2c4d62bd910232b33df0daa84f70a8e6f20e732b:<このファイルのパス>` で読む。
