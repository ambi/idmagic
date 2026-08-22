# SigningKeys Scenarios

### REQ-SIGNINGKEYS-001: 署名鍵をローテーションしても以前の kid は JWKS に残る
- ACTOR TenantAdministrator
- GIVEN `admin` ロールを持つ "operator" が認証済みである
- GIVEN 現在の署名鍵は `kid` "kid-old" を持つ
- WHEN "operator" が管理画面で現在の署名鍵をローテーションする
- THEN ローテーションによって `kid` "kid-new" が新しい有効鍵になる
- WHEN クライアントが JWKS を取得する
- THEN レスポンスに `kid` "kid-old" と "kid-new" の両方が含まれる

### REQ-SIGNINGKEYS-002: 猶予期間終了後の署名鍵は JWKS から除去してアーカイブする
- ACTOR SystemAdministrator
- GIVEN `kid` "kid-old" の `Verifying` 鍵は `expires_at` を経過している
- WHEN スケジューラーがアーカイブ処理を実行する
- WHEN クライアントが JWKS を取得する
- THEN レスポンスに `kid` "kid-old" は含まれない
- THEN SigningKeyArchived イベントに `kid`、`retiredAt`、`expiresAt`、`disposedAt` が記録される

### REQ-SIGNINGKEYS-003: ライフサイクル設定が不正なバッチは起動しない
- ACTOR SystemAdministrator
- GIVEN `grace_days` が `cadence_days` 以上である
- WHEN `system_admin` が `idmagic-batch signing-key-lifecycle` を起動する
- THEN 設定エラーで終了し、鍵を回転しない

### REQ-SIGNINGKEYS-004: テナントごとの JWKS は互いに分離される
- ACTOR TenantAdministrator
- GIVEN テナント "tenant-a" とテナント "tenant-b" がそれぞれ署名鍵を持つ
- WHEN テナント "tenant-a" の管理者が署名鍵を回転する
- WHEN クライアントがテナント "tenant-a" の JWKS を取得する
- THEN レスポンスにはテナント "tenant-a" の `kid` だけが含まれ、テナント "tenant-b" の `kid` は含まれない

### REQ-SIGNINGKEYS-005: XML フェデレーション署名資格情報はテナントと用途で分離される
- ACTOR TenantAdministrator
- GIVEN テナント "tenant-a" とテナント "tenant-b" が存在する
- GIVEN 両テナントが JWT Signing 鍵と XmlFederationSigning 鍵を持つ
- WHEN テナント "tenant-a" が SAML Assertion を発行する
- THEN Assertion はテナント "tenant-a" の有効な `XmlFederationSigning` 鍵で署名される
- THEN テナント "tenant-b" の証明書でも、テナント "tenant-a" の JWT Signing 公開鍵でも署名を検証できない

### REQ-SIGNINGKEYS-006: XML フェデレーション鍵のローテーション中も既存の信頼関係を検証できる
- ACTOR TenantAdministrator
- GIVEN `XmlFederationSigning` の現在の鍵 K1 がメタデータに掲載されている
- WHEN 管理者が XmlFederationSigning 鍵を K2 へ回転する
- THEN 新しい XML メッセージは K2 で署名される
- THEN 猶予期間中の SAML / WS-Fed メタデータには K1 と K2 の証明書が掲載される
- THEN 猶予期間終了後は K1 がメタデータから除去される

### REQ-SIGNINGKEYS-007: XML フェデレーション署名資格情報は再起動後も同一である
- ACTOR SystemAdministrator
- GIVEN PostgreSQL または Vault プロバイダーでテナントの `XmlFederationSigning` 鍵が作成済みである
- WHEN API プロセスを再起動する
- WHEN クライアントが同じテナントのメタデータを取得する
- THEN 有効な証明書のフィンガープリントは再起動前と一致する

### REQ-SIGNINGKEYS-008: KeyProvider の障害時は健全性を観測でき、JWKS は取得可能な範囲で返る
- ACTOR SystemAdministrator
- GIVEN テナント "tenant-a" の KeyProvider が到達不能である
- WHEN `system_admin` が署名鍵の健全性一覧を取得する
- THEN テナント "tenant-a" の `provider_healthy` は `false` として返る
- THEN テナント "tenant-a" の JWKS は取得可能な範囲でキャッシュされた鍵を返す

### REQ-SIGNINGKEYS-009: 通常のテナント管理者はシステムコンソールの署名鍵ヘルスにアクセスできない
- ACTOR TenantAdministrator
- GIVEN "operator" は `admin` ロールだけを持ち、`system_admin` ロールを持たない
- WHEN "operator" が署名鍵ヘルス一覧を呼び出す
- THEN AccessDeniedError で拒否される

### REQ-SIGNINGKEYS-010: 管理者は回転後の検証用鍵だけを即時無効化できる
- ACTOR TenantAdministrator
- GIVEN 現在の署名鍵 K2 と、回転後に JWKS へ残る検証用鍵 K1 がある
- WHEN 管理者が K1 を無効化する
  - ALT 管理者が現在の署名鍵 K2 を無効化しようとする → エラー "InvalidRequestError"
- THEN K1 は JWKS から除去される
- THEN K2 は現在の署名鍵のまま残る

### REQ-SIGNINGKEYS-011: 署名鍵の回転と無効化は自テナントの管理者だけが要求できる
- ACTOR TenantAdministrator
- GIVEN テナント "tenant-a" の現在の署名鍵は `kid` "kid-1" である
- WHEN `admin` も `system_admin` も持たない "alice" が署名鍵の回転を要求する
  - ALT `signing-keys:read` だけを持つ API アクセストークンで回転または無効化を要求する → 必要なスコープが `signing-keys:write` であることを示して拒否される
- THEN AccessDeniedError で拒否される
- THEN 現在の署名鍵は "kid-1" のまま変わらず、SigningKeyRotated は発行されない
