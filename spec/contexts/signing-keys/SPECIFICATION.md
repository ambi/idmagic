---
context: signing-keys
updated_at: 2026-08-11
---

# SigningKeys Specification

## Overview

tenant-scoped signing key material のライフサイクル、ローテーション、公開重複期間、監査を
protocol 横断で所有する。OAuth2/OIDC は JWK/JWKS と JWT signer、SAML / WS-* は X.509
証明書と XML signer adapter を使うが、鍵用途・ローテーション・tenant isolation の規範は
ここに集約する。

The `SigningKeys` context owns tenant-scoped asymmetric key metadata, provider selection, rotation,
verification overlap, and archival. It exposes key material only through published signing and public-key
ports; protocol serialization remains in OAuth2, SAML, and WS-Federation adapters.

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

## State Transitions

### SigningKeyLifecycle

署名鍵のライフサイクル (SigningKeyMinJwksOverlap)。Active から Rotate で Verifying に降り、Retire で JWKS から外し、Archive で監査保管に入る。

Initial: `Active`
Terminal: `Archived`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | SigningKeyRotated | — | Verifying |  |
| Verifying | SigningKeyRetired | — | Retired |  |
| Retired | SigningKeyArchived | — | Archived |  |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Usage and scope isolation

Every key lookup is scoped by the request tenant, `KeyUsage`, and an opaque scope ID. Callers that do
not select a usage or scope use `Signing` and the default scope, preserving a small API for OAuth2/OIDC.
XML protocol adapters explicitly select `XmlFederationSigning`; SAML additionally selects its identity
provider profile ID as the scope. A JWT key can therefore never be selected for an XML assertion, and
one SAML profile cannot select another profile's credential.

The local, PostgreSQL, and Vault adapters all maintain one active key per tenant, usage, and scope.
PostgreSQL enforces the same invariant with a partial unique index, and Vault includes the scope in its
key-set identity. This compound key exists because rotating one SAML profile must not rotate another
profile or every JWT verification key.

### XML federation credentials

An XML federation key carries a self-signed X.509 certificate containing its public key. The certificate
is public metadata; the private key follows the configured provider and never appears in an admin
response. Active keys sign new messages, while unexpired verifying certificates remain available to
SAML and WS-Federation metadata during the rotation overlap.

Local and database providers hold the private RSA key in process when signing. Vault Transit retains the
private key and implements `crypto.Signer`; it selects PSS for JWT requests and PKCS#1 v1.5 for XML
Signature and X.509 operations because those wire formats advertise RSA-SHA256 rather than RSA-PSS.

### Lifecycle

Keys are created lazily for the resolved tenant, usage, and scope. No default tenant receives an eager
bootstrap key. This keeps tenant creation uniform and avoids special state that cannot be explained by
the request-scoped lifecycle.

A tenant's active signing key rotates at least every 90 days, driven by a scheduled operational job
independent of the manual, immediate `RotateTenantSigningKey` path. Rotation atomically demotes the
old active key to verifying and gives it an overlap expiry of at least 7 days, so JWKS consumers and
relying parties can still validate messages issued just before rotation. Key material reaching the
terminal `Archived` state is retained for 7 years to support verification of audit tokens signed by
already-retired keys; there is no separate purge/erase interface yet.

Public key and certificate listing includes active and unexpired verifying records; archive removes
expired records from publication.

Fail-closed behavior when a key provider is unreachable is not enforced inside `SigningKeys` — this
context has no signing or issuance interface of its own. It only surfaces the observable
`provider_healthy` signal (`TenantSigningKey.provider_healthy`, `ListTenantKeyHealth`); the actual
fail-closed enforcement point is OAuth2's `Token` issuance interface, which is where an unreachable
provider blocks new signatures.

### Design Decisions

- SAML IdP profiles are modeled as shareable (a profile can back more than one SP trust), with
  dedicated-use profiles expressed as the one-consumer case of the same model rather than a separate
  type.
- Signing keys are scoped per tenant behind a pluggable `KeyProvider`, rather than a shared
  system-wide key or a provider baked into each protocol adapter.
- Key rotation cadence (90-day minimum), overlap expiry (7-day minimum), and archive retention
  (7 years) are fixed, normative policy values that live in this design record rather than being
  left as undocumented configuration.

## Scenarios

### REQ-SIGNINGKEYS-001: 署名鍵を回転しても旧kidはJWKSに残る
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- GIVEN 現在の署名鍵は kid "kid-old" を持つ
- WHEN "operator" が管理画面で現在の署名鍵を回転する
- THEN 回転により kid "kid-new" が新しい active 鍵になる
- WHEN client が JWKS を取得する
- THEN 応答に kid "kid-old" と "kid-new" の両方が含まれる

### REQ-SIGNINGKEYS-002: grace期間終了後の署名鍵はJWKSから除去されarchiveされる
- ACTOR SystemAdministrator
- GIVEN kid "kid-old" の Verifying 鍵の expires_at が経過している
- WHEN scheduler が archive 処理を実行する
- WHEN client が JWKS を取得する
- THEN 応答に kid "kid-old" は含まれない
- THEN SigningKeyArchived イベントに kid、retiredAt、expiresAt、disposedAt が記録される

### REQ-SIGNINGKEYS-003: lifecycle設定が不正なbatchは起動しない
- ACTOR SystemAdministrator
- GIVEN grace_days が cadence_days 以上である
- WHEN system_admin が idmagic-batch signing-key-lifecycle を起動する
- THEN 設定エラーで終了し、鍵を回転しない

### REQ-SIGNINGKEYS-004: テナントごとのJWKSは互いに分離される
- ACTOR TenantAdministrator
- GIVEN テナント "tenant-a" とテナント "tenant-b" がそれぞれ署名鍵を持つ
- WHEN テナント "tenant-a" の管理者が署名鍵を回転する
- WHEN client がテナント "tenant-a" の JWKS を取得する
- THEN 応答にはテナント "tenant-a" の kid だけが含まれ、テナント "tenant-b" の kid は含まれない

### REQ-SIGNINGKEYS-005: XML federation署名資格情報はテナントと用途で分離される
- ACTOR TenantAdministrator
- GIVEN テナント "tenant-a" とテナント "tenant-b" が存在する
- GIVEN 両テナントが JWT Signing 鍵と XmlFederationSigning 鍵を持つ
- WHEN テナント "tenant-a" が SAML assertion を発行する
- THEN assertion は tenant-a の active XmlFederationSigning 鍵で署名される
- THEN tenant-b の証明書でも tenant-a の JWT Signing 公開鍵でも署名を検証できない

### REQ-SIGNINGKEYS-006: XML federation鍵の回転中も既存trustを検証できる
- ACTOR TenantAdministrator
- GIVEN XmlFederationSigning の現在鍵 K1 が metadata に掲載されている
- WHEN 管理者が XmlFederationSigning 鍵を K2 へ回転する
- THEN 新規 XML message は K2 で署名される
- THEN grace期間中の SAML / WS-Fed metadata には K1 と K2 の証明書が掲載される
- THEN grace期間終了後は K1 が metadata から除去される

### REQ-SIGNINGKEYS-007: XML federation署名資格情報は再起動後も同一である
- ACTOR SystemAdministrator
- GIVEN PostgreSQL または Vault provider で tenant の XmlFederationSigning 鍵が作成済みである
- WHEN API process を再起動する
- WHEN client が同じ tenant の metadata を取得する
- THEN active certificate の fingerprint は再起動前と一致する

### REQ-SIGNINGKEYS-008: KeyProvider障害時は健全性が観測できJWKSは取得可能な範囲で返る
- ACTOR SystemAdministrator
- GIVEN テナント "tenant-a" の KeyProvider が到達不能である
- WHEN system_admin が署名鍵ヘルス一覧を取得する
- THEN テナント "tenant-a" の provider_healthy は false として返る
- THEN テナント "tenant-a" の JWKS は取得可能な範囲でキャッシュされた鍵を返す

### REQ-SIGNINGKEYS-009: 通常のテナント管理者はシステムコンソールの署名鍵ヘルスにアクセスできない
- ACTOR TenantAdministrator
- GIVEN "operator" は admin ロールのみを持ち system_admin ロールを持たない
- WHEN "operator" が署名鍵ヘルス一覧を呼び出す
- THEN AccessDeniedError で拒否される

### REQ-SIGNINGKEYS-010: 管理者は回転後の検証用鍵だけを即時無効化できる
- ACTOR TenantAdministrator
- GIVEN 現在の署名鍵 K2 と、回転後に JWKS へ残る検証用鍵 K1 がある
- WHEN 管理者が K1 を無効化する
  - ALT 管理者が現在の署名鍵 K2 を無効化しようとする → エラー "InvalidRequestError"
- THEN K1 は JWKS から除去される
- THEN K2 は現在の署名鍵のまま残る
