# ApiTokens Glossary

| Term | Definition | Aliases |
|---|---|---|
| ApiAccessToken | IdMagic の管理 API、SCIM API、発行者本人のアカウント API を呼び出すためのテナント単位の JWT アクセストークン。`Authorization` ヘッダーに Bearer スキームで提示する。テナント管理者が発行と失効を管理する長寿命トークンで、付与されたスコープ集合が呼び出せる操作を決める。通常の OAuth アクセストークンと同じ RFC 9068 JWT を発行時に一度だけ返し、永続化するのは `jti` とライフサイクルのメタデータだけで、トークン本文は保存しない。 | API アクセストークン, PAT, Personal Access Token |
| ApiTokenScope | API アクセストークンに付与される権限単位。`<resource>:<action>` 形式で、`read` は参照系、`write` は変更系の操作を許可する。`scim:` で始まるスコープは SCIM 2.0 プロビジョニング API のリソースと操作を表す。トークンのスコープ集合は認証時に解決され、各操作の認可判断に使われる。 |  |
