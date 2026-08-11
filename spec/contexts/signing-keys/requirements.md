# SigningKeys Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-SIGNINGKEYS-001: 署名鍵を回転しても旧kidはJWKSに残る
- Actor: TenantAdministrator
- Given: admin ロールを持つ "operator" が認証済みである
- Given: 現在の署名鍵は kid "kid-old" を持つ
- Then: "operator" が管理画面で現在の署名鍵を回転する
- Then: 回転により kid "kid-new" が新しい active 鍵になる
- Then: JWKS を取得する
- Then: 応答に kid "kid-old" と "kid-new" の両方が含まれる

### REQ-SIGNINGKEYS-002: grace期間終了後の署名鍵はJWKSから除去されarchiveされる
- Actor: SystemAdministrator
- Given: kid "kid-old" の Verifying 鍵の expires_at が経過している
- Then: scheduler が archive 処理を実行する
- Then: JWKS を取得する
- Then: 応答に kid "kid-old" は含まれない
- Then: SigningKeyArchived イベントに kid、retiredAt、expiresAt、disposedAt が記録される

### REQ-SIGNINGKEYS-003: lifecycle設定が不正なbatchは起動しない
- Actor: SystemAdministrator
- Given: grace_days が cadence_days 以上である
- Then: system_admin が idmagic-batch signing-key-lifecycle を起動する
- Then: 設定エラーで終了し、鍵を回転しない

### REQ-SIGNINGKEYS-004: テナントごとのJWKSは互いに分離される
- Actor: TenantAdministrator
- Given: テナント "tenant-a" とテナント "tenant-b" がそれぞれ署名鍵を持つ
- Then: テナント "tenant-a" の管理者が署名鍵を回転する
- Then: テナント "tenant-a" の JWKS を取得する
- Then: 応答にはテナント "tenant-a" の kid だけが含まれ、テナント "tenant-b" の kid は含まれない

### REQ-SIGNINGKEYS-005: XML federation署名資格情報はテナントと用途で分離される
- Actor: TenantAdministrator
- Given: テナント "tenant-a" とテナント "tenant-b" が存在する
- Given: 両テナントが JWT Signing 鍵と XmlFederationSigning 鍵を持つ
- Then: テナント "tenant-a" が SAML assertion を発行する
- Then: assertion は tenant-a の active XmlFederationSigning 鍵で署名される
- Then: tenant-b の証明書でも tenant-a の JWT Signing 公開鍵でも署名を検証できない

### REQ-SIGNINGKEYS-006: XML federation鍵の回転中も既存trustを検証できる
- Actor: TenantAdministrator
- Given: XmlFederationSigning の現在鍵 K1 が metadata に掲載されている
- Then: 管理者が XmlFederationSigning 鍵を K2 へ回転する
- Then: 新規 XML message は K2 で署名される
- Then: grace期間中の SAML / WS-Fed metadata には K1 と K2 の証明書が掲載される
- Then: grace期間終了後は K1 が metadata から除去される

### REQ-SIGNINGKEYS-007: XML federation署名資格情報は再起動後も同一である
- Actor: SystemAdministrator
- Given: PostgreSQL または Vault provider で tenant の XmlFederationSigning 鍵が作成済みである
- Then: API process を再起動する
- Then: 同じ tenant の metadata を取得する
- Then: active certificate の fingerprint は再起動前と一致する

### REQ-SIGNINGKEYS-008: KeyProvider障害時は健全性が観測できJWKSは取得可能な範囲で返る
- Actor: SystemAdministrator
- Given: テナント "tenant-a" の KeyProvider が到達不能である
- Then: system_admin が署名鍵ヘルス一覧を取得する
- Then: テナント "tenant-a" の provider_healthy は false として返る
- Then: テナント "tenant-a" の JWKS は取得可能な範囲でキャッシュされた鍵を返す

### REQ-SIGNINGKEYS-009: 通常のテナント管理者はシステムコンソールの署名鍵ヘルスにアクセスできない
- Actor: TenantAdministrator
- Given: "operator" は admin ロールのみを持ち system_admin ロールを持たない
- Then: "operator" が署名鍵ヘルス一覧を呼び出す
- Then: AccessDeniedError で拒否される

### REQ-SIGNINGKEYS-010: 管理者は回転後の検証用鍵だけを即時無効化できる
- Actor: TenantAdministrator
- Given: 現在の署名鍵 K2 と、回転後に JWKS へ残る検証用鍵 K1 がある
- Then: 管理者が K1 を無効化する
- Then: K1 は JWKS から除去される
- Then: K2 は現在の署名鍵のまま残る
- Alternative (管理者が現在の署名鍵 K2 を無効化しようとする): エラー "InvalidRequestError"

### REQ-SIGNINGKEYS-011: GetJwks
解決済みテナント (未 prefix リクエストは default) の JWKS (RFC 7517) を返す公開 interface。
- Postcondition: output.jwks.keys.all(k, tenant_owns_signing_key(context.tenant_id, k))

### REQ-SIGNINGKEYS-012: ListTenantJwks
テナント指定の JWKS (RFC 7517)。当該テナントの active + verifying
(retired-not-expired) 鍵のみを返す。テナント A の JWKS にテナント B の kid は
出ない。RP はこの URL で自テナント発行トークンの署名を検証する。認証を要求しない。
- Postcondition: output.jwks.keys.all(k, tenant_owns_signing_key(input.tenant_id, k))

### REQ-SIGNINGKEYS-013: ListAdminKeys
テナント内 admin が自テナントの JWKS に乗っている署名鍵 (active + verifying) の
メタデータを取得する。対象は context.tenant_id に固定するため cross-tenant 読み出しは
構造的に発生しない。

### REQ-SIGNINGKEYS-014: GetAdminKey
テナント内 admin が kid を指定して自テナントの署名鍵 1 件を取得する。他テナントの kid または存在しない kid は InvalidRequestError。

### REQ-SIGNINGKEYS-015: RotateTenantSigningKey
テナント内 admin が自テナントに新しい署名鍵を生成し active を切り替える。旧 active 鍵は
JWKS に verifying として残存し、SigningKeyMinJwksOverlap 期間後に Retire される
(Retire 処理は別の運用ジョブ)。テナント単位のため回転の影響範囲は当該テナントに
閉じる。

### REQ-SIGNINGKEYS-016: DisableTenantKey
テナント内 admin が、回転後の移行期間にある非 active 署名鍵 1 件を kid 指定で
即時無効化 (Retire) する。対象は JWKS から除去され、その鍵で署名された既発行
トークンは検証できなくなる。現在の署名鍵はこの操作で無効化できない。
- Precondition: signing_key_is_not_active(input.kid)

### REQ-SIGNINGKEYS-017: ListTenantKeyHealth
system_admin がテナント横断で署名鍵ヘルス (provider / active kid /
鍵数 / provider 到達性) を一覧する。秘密鍵は返さない。fail-closed 判定や
回転漏れの検知に使う。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| SigningKeys | tenant-scoped signing key material のライフサイクルと公開を扱う境界。OAuth2/OIDC は JWK/JWKS、SAML/WS-* は X.509 証明書を使う。 | KeyMaterial, signing keys |
| Retire | SigningKey を Verifying から Retired に移す。 | retire |
| Archive | SigningKey を Retired から Archived に移す。 | archive |
| Verifying | 署名はしないが、過去発行トークンの検証のため JWKS に残っている状態。 | verifying |
| Retired | JWKS から除去された状態。新規検証には使われない。 | retired |
| Archived | 監査用に長期保管されている終端状態。鍵マテリアルは封印。 | archived |
| KeyProvider | 鍵マテリアルの保管種別と署名の実行主体。Local / Database は private key をプロセス内に持ちアプリが署名する dev/test 用、VaultTransit は private key を Vault 内に保持し署名を Vault API に委ねる本番用。Database は特定の製品名を表さない。 | key provider, 鍵プロバイダ |
| VaultTransit | HashiCorp Vault の Transit secrets engine を使う KeyProvider。秘密鍵マテリアルは Vault 外に出ず、署名要求ごとに Vault へ委譲する。 | Vault Transit |
| FailClosed | KeyProvider が不達のとき、新規 token 発行を停止する挙動。既発行 token 検証用の JWKS は取得可能な範囲で返す。強制点は OAuth2.Token の requires が持つ。 | fail-closed, フェイルクローズ |

## State machines

### SigningKeyLifecycle

署名鍵のライフサイクル (SigningKeyMinJwksOverlap)。Active から Rotate で Verifying に降り、Retire で JWKS から外し、Archive で監査保管に入る。

Initial: `Active`  
Terminal: `Archived`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | SigningKeyRotated | "" | Verifying |  |
| Verifying | SigningKeyRetired | "" | Retired |  |
| Retired | SigningKeyArchived | "" | Archived |  |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
