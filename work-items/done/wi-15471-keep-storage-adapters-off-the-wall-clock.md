---
status: completed
---

# ストレージのアダプターが実時計を読まず、時刻を呼び出し側から受け取る

`mise run spec-diff` は main に対する規範の差分を報告しない。
ストレージのアダプター（`db_memory`、`db_postgres`）は実時計を読まず、判定と記録の時刻をポートの引数で受け取る。
Go は `forbidigo` が、SQL は新しい `check-sql-clock` が `sqlc` の構文木から、記録の値の外で時計を読む式を拒否する。
同意の期限切れは `Consent.StateAt(now)` で判定し、期限切れの撤回は REQ-OAUTH2-032 と REQ-OAUTH2-058 のとおり使用量を変えない。
ゼロの時刻はエラーとして拒否し、SAML IdP プロファイルの更新の応答は保存した `updated_at` を返す。
