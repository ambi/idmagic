# Feature: SigningKeys Scenarios

## Rule: REQ-SIGNINGKEYS-001 署名鍵をローテーションしても以前の kid は JWKS に残る

Primary actor: `TenantAdministrator`

### Example: EX-SIGNINGKEYS-001-01 通常経路

- Given `admin` ロールを持つ "operator" が認証済みである
- And 現在の署名鍵は `kid` "kid-old" を持つ
- When "operator" が管理画面で現在の署名鍵をローテーションする
- Then ローテーションによって `kid` "kid-new" が新しい有効鍵になる
- When クライアントが JWKS を取得する
- Then レスポンスに `kid` "kid-old" と "kid-new" の両方が含まれる

## Rule: REQ-SIGNINGKEYS-002 猶予期間終了後の署名鍵は JWKS から除去してアーカイブする

Primary actor: `SystemAdministrator`

### Example: EX-SIGNINGKEYS-002-01 通常経路

- Given `kid` "kid-old" の `Verifying` 鍵は `expires_at` を経過している
- When スケジューラーがアーカイブ処理を実行する
- When クライアントが JWKS を取得する
- Then レスポンスに `kid` "kid-old" は含まれない
- Then SigningKeyArchived イベントに `kid`、`retiredAt`、`expiresAt`、`disposedAt` が記録される

## Rule: REQ-SIGNINGKEYS-003 ライフサイクル設定が不正なバッチは起動しない

Primary actor: `SystemAdministrator`

### Example: EX-SIGNINGKEYS-003-01 通常経路

- Given `grace_days` が `cadence_days` 以上である
- When `system_admin` が `idmagic-batch signing-key-lifecycle` を起動する
- Then 設定エラーで終了し、鍵を回転しない

## Rule: REQ-SIGNINGKEYS-004 テナントごとの JWKS は互いに分離される

Primary actor: `TenantAdministrator`

### Example: EX-SIGNINGKEYS-004-01 通常経路

- Given テナント "tenant-a" とテナント "tenant-b" がそれぞれ署名鍵を持つ
- When テナント "tenant-a" の管理者が署名鍵を回転する
- When クライアントがテナント "tenant-a" の JWKS を取得する
- Then レスポンスにはテナント "tenant-a" の `kid` だけが含まれ、テナント "tenant-b" の `kid` は含まれない

## Rule: REQ-SIGNINGKEYS-005 XML フェデレーション署名資格情報はテナントと用途で分離される

Primary actor: `TenantAdministrator`

### Example: EX-SIGNINGKEYS-005-01 通常経路

- Given テナント "tenant-a" とテナント "tenant-b" が存在する
- And 両テナントが JWT Signing 鍵と XmlFederationSigning 鍵を持つ
- When テナント "tenant-a" が SAML Assertion を発行する
- Then Assertion はテナント "tenant-a" の有効な `XmlFederationSigning` 鍵で署名される
- Then テナント "tenant-b" の証明書でも、テナント "tenant-a" の JWT Signing 公開鍵でも署名を検証できない

## Rule: REQ-SIGNINGKEYS-006 XML フェデレーション鍵のローテーション中も既存の信頼関係を検証できる

Primary actor: `TenantAdministrator`

### Example: EX-SIGNINGKEYS-006-01 通常経路

- Given `XmlFederationSigning` の現在の鍵 K1 がメタデータに掲載されている
- When 管理者が XmlFederationSigning 鍵を K2 へ回転する
- Then 新しい XML メッセージは K2 で署名される
- Then 猶予期間中の SAML / WS-Fed メタデータには K1 と K2 の証明書が掲載される
- Then 猶予期間終了後は K1 がメタデータから除去される

## Rule: REQ-SIGNINGKEYS-007 XML フェデレーション署名資格情報は再起動後も同一である

Primary actor: `SystemAdministrator`

### Example: EX-SIGNINGKEYS-007-01 通常経路

- Given PostgreSQL または Vault プロバイダーでテナントの `XmlFederationSigning` 鍵が作成済みである
- When API プロセスを再起動する
- When クライアントが同じテナントのメタデータを取得する
- Then 有効な証明書のフィンガープリントは再起動前と一致する

## Rule: REQ-SIGNINGKEYS-008 KeyProvider の障害時は健全性を観測でき、JWKS は取得可能な範囲で返る

Primary actor: `SystemAdministrator`

### Example: EX-SIGNINGKEYS-008-01 通常経路

- Given テナント "tenant-a" の KeyProvider が到達不能である
- And "sys-operator" は制御面テナントに所属し、有効ロールに `system_admin` を含む有効な User である
- When "sys-operator" が制御面テナントの経路で署名鍵の健全性一覧を取得する
- Then テナント "tenant-a" の `provider_healthy` は `false` として返る
- Then テナント "tenant-a" の JWKS は取得可能な範囲でキャッシュされた鍵を返す

## Rule: REQ-SIGNINGKEYS-009 制御面主体ではない管理者はシステムコンソールの署名鍵ヘルスにアクセスできない

Primary actor: `TenantAdministrator`

### Example: EX-SIGNINGKEYS-009-01 通常経路

- Given "operator" は制御面テナントに所属するが、`admin` ロールだけを持ち `system_admin` ロールを持たない
- And "tenant-operator" は制御面テナント以外のテナントに所属し、直接または Group 経由で `system_admin` ロールを持つ
- When "operator" が署名鍵ヘルス一覧を呼び出す
- Then AccessDeniedError で拒否される
- When "tenant-operator" が自テナントの経路で署名鍵ヘルス一覧を呼び出す
- Then AccessDeniedError で拒否される
- Then レスポンスは他テナントの識別子、提供元、`active_kid`、鍵数、到達性のいずれも含まない
- Then テナント横断の健全性収集は実行されない

## Rule: REQ-SIGNINGKEYS-010 管理者は回転後の検証用鍵だけを即時無効化できる

Primary actor: `TenantAdministrator`

### Example: EX-SIGNINGKEYS-010-01 通常経路

- Given 現在の署名鍵 K2 と、回転後に JWKS へ残る検証用鍵 K1 がある
- When 管理者が K1 を無効化する
- Then K1 は JWKS から除去される
- Then K2 は現在の署名鍵のまま残る

### Example: EX-SIGNINGKEYS-010-02 管理者が現在の署名鍵 K2 を無効化しようとする

- Given 現在の署名鍵 K2 と、回転後に JWKS へ残る検証用鍵 K1 がある
- When 管理者が K1 を無効化する
- But 管理者が現在の署名鍵 K2 を無効化しようとする
- Then エラー "InvalidRequestError"

## Rule: REQ-SIGNINGKEYS-011 署名鍵の回転と無効化は自テナントの管理者だけが要求できる

Primary actor: `TenantAdministrator`

### Example: EX-SIGNINGKEYS-011-01 通常経路

- Given テナント "tenant-a" の現在の署名鍵は `kid` "kid-1" である
- When `admin` も `system_admin` も持たない "alice" が署名鍵の回転を要求する
- Then AccessDeniedError で拒否される
- Then 現在の署名鍵は "kid-1" のまま変わらず、SigningKeyRotated は発行されない

### Example: EX-SIGNINGKEYS-011-02 `signing-keys:read` だけを持つ API アクセストークンで回転または無効化を要求する

- Given テナント "tenant-a" の現在の署名鍵は `kid` "kid-1" である
- When `admin` も `system_admin` も持たない "alice" が署名鍵の回転を要求する
- But `signing-keys:read` だけを持つ API アクセストークンで回転または無効化を要求する
- Then 必要なスコープが `signing-keys:write` であることを示して拒否される

## Rule: REQ-SIGNINGKEYS-012 鍵素材を平文で永続化する構成は明示の選択を要する

Primary actor: `SystemAdministrator`

### Example: EX-SIGNINGKEYS-012-01 通常経路

- Given 配備の `PERSISTENCE` が `postgres` である
- And `KEY_PROVIDER` が指定されていない
- When `system_admin` が起動時設定を読むプロセスを起動する
- Then 起動は設定エラーで拒否され、`KEY_PROVIDER` を明示するよう示す
- Then 秘密鍵素材をデータベースへ平文で保存する KeyStore は構築されず、署名鍵も作成されない
