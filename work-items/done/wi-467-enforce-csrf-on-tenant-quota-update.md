---
status: completed
archived_from: 2c4d62bd910232b33df0daa84f70a8e6f20e732b
---

# テナントクォータ更新で Origin と CSRF トークンを検証する

`mise run spec-diff` は `no normative specification change against main` を返す。規範仕様は変わっておらず、この作業は Tenancy の決定 (状態を変える管理リクエストはセッション認証に加えて `Origin` と CSRF トークンを検証する) が要求していた検査を、唯一それを欠いていた `UpdateTenantQuota` へ実装した差分である。意味の差は、Cookie セッションからのクォータ更新が CSRF トークンの一致を伴わなければ成立しなくなったことに尽きる。`Authorization` ヘッダーの Bearer と DPoP は従来どおり Cookie CSRF 検査の対象外である。

全文は `git show 2c4d62bd910232b33df0daa84f70a8e6f20e732b:<このファイルのパス>` で読む。
