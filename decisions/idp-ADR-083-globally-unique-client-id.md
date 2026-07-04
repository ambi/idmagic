# idp-ADR-083: Globally Unique client_id の強制とスキーマの単純化

## ステータス

提案 (Proposed)。

## コンテキスト

[idp-ADR-034](file:///Users/tn/src/idmagic/decisions/idp-ADR-034-tenant-scoped-persistence.md) では、`client_id` はテナントごとに管理者が設定する文字列であるという前提のもと、`clients` テーブルの主キーを `(tenant_id, client_id)` とする「テナントスコープの複合キー」を採用していた。

しかし、この設計は以下の問題を引き起こしている：
1. **スキーマの複雑化**: `clients` が複合キーを使用しているため、それを参照する `consents` や `refresh_tokens` などの子テーブルにも `tenant_id` を伝播させて複合外部キー制約（Composite FK）を定義せざるを得ず、データベーススキーマ全体の複雑性が増している。
2. **名前の衝突と再利用リスク**: 管理者が任意に命名できることで、同一テナント内での作成・削除による名前の重複や、衝突回避のロジックが必要になる。
3. **業界標準との乖離**: Auth0、Okta、Azure AD/Entra ID などの主要なモダン IdP では、`client_id` は管理者が命名するものではなく、システム側が生成するグローバルにユニークなランダム文字列（または UUID）を強制するのが事実上の標準である。

`users.id` を [idp-ADR-082](file:///Users/tn/src/idmagic/decisions/idp-ADR-082-user-domain-id-and-tenant-key-policy.md) でグローバルユニークな ID に変更したのと同様に、`client_id` もシステム自動生成のグローバルユニークな識別子に強制することで、スキーマや実装を大きく単純化できる。

## 決定

1. **`client_id` をシステム生成のグローバルユニークな値（UUIDv4等）に強制する。**
   管理者がクライアントを作成する際、`client_id` を手動入力するフィールドを廃止し、システム側で自動生成する。
2. **`clients` の主キーから `tenant_id` を排除し、`client_id` 単一の PK とする。**
   `PRIMARY KEY (client_id)` とし、`tenant_id` 列はテナント所属を示す属性カラム（単なる参照制約）に降格する。
3. **子テーブルの複合外部キー制約（Composite FK）を廃止する。**
   `consents` や `refresh_tokens` から、`client_id` 参照のために持っていた `tenant_id` との複合制約を廃止し、単純な `FOREIGN KEY (client_id) REFERENCES clients(client_id)` に変更する。これに伴い、`consents` や `refresh_tokens` の複合主キー構成も簡素化する。

## 却下した代替案

- **`client_id` を複合キーのままにし、バリデーションだけで一意にする**
  スキーマレベルの複雑さ（Composite FK や伝播する `tenant_id`）が一切解消されないため、根本的な解決にならない。

## 影響

- **SCL (`spec/contexts/oauth2.yaml`)**:
  `OAuth2Client` エンティティの `identity` を `client_id` 単一に変更する。
- **Postgres スキーマ (`postgres.sql`)**:
  - `clients` の主キーを `client_id` 単一に変更。
  - `consents` や `refresh_tokens` から複合 FK を排除し、単一の `client_id` 参照にする。
- **Go コードの実装**:
  - `OAuth2ClientRepository.FindByID` などの引数から `tenant_id` を除外し、グローバルな `client_id` だけで検索可能にする。
  - 各種ユースケースやハンドラー of /api/admin/clients などの引数・ロジックの簡素化。
- **UI**:
  - クライアント作成画面から `client_id` 入力フィールドを削除し、「作成時に自動的に ID が割り当てられる」仕様に変更する。
