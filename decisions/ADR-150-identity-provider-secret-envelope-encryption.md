---
status: suggested
authors: [tn]
created_at: 2026-07-31
---

# ADR-150: `IdentityProviderConnection` のクライアントシークレットを `env:` 参照から `EnvelopeCrypto`/`DataKeys` 実値暗号化保存に切り替える

## コンテキスト

[[wi-309-external-identity-provider-admin-ui-consistency]] の実機検証で、
`IdentityProviderConnection.secret_reference` が実際のシークレット値ではなく、
共有サーバープロセスの環境変数名を指す `env:VARNAME` という文字列
(`backend/authentication/federation/secrets_env/resolver.go:19-37`) であることが判明した。
シークレットの実値をアプリ DB に平文で持たないという設計判断自体は妥当で、
[[ADR-148]] が確立した `EnvelopeCrypto`/`DataKeys` によるエンベロープ暗号化と同じ思想
(実値を直接持たず、暗号化した形でのみ保持する) だが、具体的な実装 (サーバー環境変数を
都度参照させる方式) は ADR-148 の仕組みと整合しておらず、テナントごとに外部IDプロバイダーを
追加するたびに共有サーバープロセスの環境変数を追加してデプロイし直す必要があり、
セルフサービスなマルチテナント管理画面としては機能しない。加えて `test` も `activate` も
`secret_reference` が実際に解決できるかどうかを一度も検証していない。

`backend/authentication/totp` は MFA TOTP シード (`mfa_factors.secret`) に対して既に
`EnvelopeCrypto`/`DataKeys` (`backend/datakeys.FieldCipher` を `totp/ports.SecretCipher` として
注入) による実値の暗号化保存を実装済みであり、本 ADR はこの確立済みパターンを
`IdentityProviderConnection` のクライアントシークレットに適用する決定である。

## 決定

`secret_reference` の `env:` 参照方式を廃止し、クライアントシークレットの実値を
`EnvelopeCrypto`/`DataKeys` によるテナント単位のエンベロープ暗号化で保存する方式に切り替える。

- `backend/authentication/federation` に `ports.SecretCipher` (MFA の
  `totp/ports.SecretCipher` と同形の narrow port) を新設し、`db_postgres.ConnectionRepository`
  に注入する。ブートストラップは MFA と同じ `*datakeys.FieldCipher` インスタンス
  (`backend/cmd/internal/bootstrap`) を再利用し、`EnvelopeCrypto`/`DataKeys` の新規実装は
  追加しない ([[ADR-148]] の仕組みをそのまま再利用する。本 WI の Out of Scope)。
  AAD は `Context="Authentication"`, `Table="identity_provider_connections"`,
  `RecordID=tenant_id + ":" + provider_id`, `Field="secret"` で構成し、MFA の
  `mfaFactorRecordID` と同型の安定 record id を用いる。
- スキーマに `secret_key_version INTEGER` と `secret_ciphertext BYTEA` を追加し、既存の
  `secret_reference TEXT` 列は移行完了まで dual-read のために残す
  (`mfa_factors.secret` を残したのと同じパターン)。新規保存は必ず暗号化する。
  読み出しは「ciphertext があれば復号、なければ旧 `env:` 参照を解決」の順で dual-read する。
- 既存の `env:` 参照を持つ接続の移行は `ports.FieldMigrator` を実装し
  (`MigratorName`、MFA の `MfaFactorReencryptor` と同形)、`DataKeys` の `MigratorRegistry` に
  登録する。ただし移行元が「暗号化されていない別形式の参照文字列」である点が MFA の
  再暗号化 (ciphertext→ciphertext) と異なるため、移行は「`env:` 参照を解決して得た実値を
  暗号化して書き込む」処理になる。解決に失敗した行 (環境変数が既に存在しない等) は
  移行できないままスキップし、`PendingCount` に残す。運用者は該当の外部IDプロバイダー画面で
  クライアントシークレットを再入力することで移行を完了できる (自動移行できない行が残り得る
  ことを許容する)。
- `EnvelopeCrypto`/`DataKeys` の DEK destroy 処理は既存の `PendingCount` ゲート
  ([[ADR-148]]) によって、この移行が未完了な限り自動的に阻止される。
- API レスポンスには実値もciphertextも含めない (`connection.SecretReference = ""` の方針を
  復号後の値にも適用し、値そのものは書き込み専用として扱う)。UI 側の実値入力フォームへの
  変更は本 WI の UI スコープで扱う。

## 却下した代替案

- **`env:` 方式を維持し UI 表示だけ改善する**: 「テナントごとに環境変数を追加してデプロイし
  直す」という運用上の破綻そのものは残る。マルチテナント自己サービス管理画面という要件に
  反する。
- **シークレットの実値をそのままDBに平文保存する**: `ARCHITECTURE.md` の
  "Envelope encryption for reversible secrets" 節が MFA TOTP seed に対して既に確立した
  可逆シークレット保管の方針に後退する。
- **`IdentityProviderConnection` 専用の新しい暗号化機構を実装する**: [[ADR-148]] が
  `EnvelopeCrypto`/`DataKeys` を「DB に残る可逆秘密」全般の共有基盤として確立済みであり、
  同種の要件 (client secret も可逆秘密) に対して別実装を持つ理由がない。
- **移行完了まで新規接続の作成を禁止する**: 既存本番環境への影響を避けるための過剰な制約で、
  dual-read により新規/更新は常に暗号化保存、既存の未移行行だけが `env:` を解決する設計で
  十分安全に両立できる。

## 影響

- SCL (`spec/contexts/authentication.yaml`):
  - `models.IdentityProviderConnection.secret_reference` の説明を「envelope 暗号化された
    client secret の実値を書き込み専用で受け取る」に変更 (フィールド名は現行のまま維持する。
    `client_secret` への改名は破壊的な wire format 変更になるため、本 ADR では見送る)。
  - `interfaces.TestIdentityProviderConnection` の description に「secret_reference の
    解決可能性 (復号可能性) を検証する」ことを追加。
- Go:
  - `backend/authentication/federation/ports`: 新設 `SecretCipher` port。
  - `backend/authentication/federation/db_postgres`: `ConnectionRepository` への
    `Cipher ports.SecretCipher` 注入、dual-read/dual-write。
  - `backend/authentication/federation/secrets_env`: `env:` resolver は移行専用の
    読み取り経路として残し、新規保存経路からは外す。
  - `backend/authentication/federation` に `FieldMigrator` 実装を追加し、
    `backend/cmd/internal/bootstrap` の `MigratorRegistry` に登録。
- Data: `infra/schema/postgres.sql` に `secret_key_version`/`secret_ciphertext` 列を追加
  (additive, nullable)。既存の `secret_reference` 列は本 ADR の対象では削除しない。
- Ops: 移行が完了する (全テナントで `PendingCount == 0` になる) までは、当該テナントの
  `DataKeys` DEK destroy 操作が `ErrDataKeyStillReferenced` で阻止される
  ([[ADR-148]] の既存ゲート挙動)。
