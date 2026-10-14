---
status: completed
archived_from: e50ccb1f62ec962e4c3e592b991b0b096589a9d2
---

# グループにメールアドレスとカスタム属性を追加できるようにする

`Group` に一級フィールド `email`（形式検証のみ、検証フロー・一意制約なし）と
`attributes`（sparse bag、値の型は `User` の `AttributeValue` を再利用）を追加した。
カスタム属性はテナント管理者が事前定義するスキーマ駆動方式(Okta/Entra ID 型)を
採用し、`User` の `UserAttributeDef`/`TenantUserAttributeSchema` とは別の
`GroupAttributeDef`/`TenantGroupAttributeSchema` を新設した(Group には builtin
catalog・self-service 編集主体・claim 露出が無いため)。`TenantGroupAttributeSchema`
は `TenantUserAttributeSchema` と同じ配置方針で `Tenancy` コンテキストが
port/usecase/HTTP を所有する。バックエンドは Domain → Usecase →
Persistence(Postgres スキーマ・sqlc・db_memory/db_postgres) → HTTP
(`/api/admin/v1/groups` の email/attributes 拡張、新規
`/api/admin/v1/tenant/group_attribute_schema` GET/PUT)の順に実装し、各層に
単体テスト・統合テスト(embedded Postgres、実 HTTP サーバー)を追加した。フロントは
グループ作成・編集画面への email 入力欄とカスタム属性エディタ、テナントの
Group 属性スキーマ管理画面 (`/admin/tenant/group_attributes`、追加/編集を
1 画面のダイアログに統合し `User` 版の 2 画面構成より簡略化)を追加した。

全文は `git show e50ccb1f62ec962e4c3e592b991b0b096589a9d2:<このファイルのパス>` で読む。
