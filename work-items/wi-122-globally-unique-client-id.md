---
id: wi-122-globally-unique-client-id
title: "client_id のグローバルユニーク強制とスキーマ・コードの単純化"
created_at: 2026-07-05
authors: [tn]
status: pending
risk: high
risk_notes: |
  クライアントの主キー（PK）および外部キー（FK）関係を広範囲に破壊・変更するため、SQL スキーマだけでなく、OAuth2 関連のドメインロジック、メモリリポジトリ、API リクエスト検証、テストデータ構造に大きな変更が入ります。
---

# Motivation
[idp-ADR-034](file:///Users/tn/src/idmagic/decisions/idp-ADR-034-tenant-scoped-persistence.md) で導入された `(tenant_id, client_id)` の複合キー設計は、管理者が任意に命名できることによる柔軟性をもたらしたが、その代償としてスキーマやアプリケーションコードに多大な複雑性を導入してしまっている。具体的には、`clients` が複合主キーを使用しているために、それを参照する `consents` や `refresh_tokens` などの子テーブルにも `tenant_id` を伝播させ、複合外部キー制約（Composite FK）を構成せざるを得なくなっている。

主要なモダン IdP（Auth0、Okta、Azure AD/Entra ID 等）では、`client_id` は管理者が命名するものではなく、システムが払い出すグローバルユニークなランダム値（UUIDや強固なランダム文字列）に強制することが一般的であり、それが事実上の標準となっている。
`users.id` を [idp-ADR-082](file:///Users/tn/src/idmagic/decisions/idp-ADR-082-user-domain-id-and-tenant-key-policy.md) でグローバルユニークな ID に変更したのと同様に、`client_id` もシステム自動生成のグローバルユニークな識別子に強制することで、複合キーの複雑性をスキーマ全体から排除し、コードとデータベース設計を劇的に単純化する。

# Scope
- SCL spec (`spec/contexts/oauth2.yaml` 等) における `OAuth2Client` エンティティの `identity` を `client_id` 単一キーに変更する。
- `deploy/schema/postgres.sql` における `clients` テーブルの主キー定義を `(tenant_id, client_id)` から `(client_id)` に変更し、`tenant_id` を単なる所属テナントを示す属性カラムに降格する。
- `consents`、`refresh_tokens`、`agents`、`agent_credential_bindings`、`scim_user_refs` などのテーブルにおいて、`client_id` の参照のために持っていた `tenant_id` との複合外部キー制約（Composite FK）をすべて排除し、単一の `client_id REFERENCES clients(client_id)` に変更する。これに伴い、複合主キー構成も簡素化する。
- クライアント作成の管理 API / UI から `client_id` の任意入力フィールドを廃止し、クライアント作成時にシステム側でグローバルユニークな ID が自動生成されるように実装する。
- リポジトリテスト、および関連するユースケース、コントローラーの配線とテストコードを更新する。

# Out of Scope
- SAML `entity_id` や WS-Fed `wtrealm` のグローバルユニーク化（これらは規格仕様上、管理者が設定する文字列であるため、本変更の対象外とし、複合キーのまま維持する）。

# Affected Guarantees
- テナント隔離におけるデータベースレベルの制約の簡素化。
- `client_id` 払い出しにおける一意性。

# Verification
- `just yaml-check`
- `just scl-render`
- `go test ./internal/...`
