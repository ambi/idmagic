---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# HSM / KMS 実鍵管理と per-tenant signing keys を導入する

ADR-075 で signing key を pluggable な `KeyProvider` の背後で per-tenant に
scope することを採用した。SCL には KeyProvider / KeyUsage / TenantSigningKey /
TenantKeyHealth と RotateTenantSigningKey / DisableTenantKey / ListTenantJwks /
ListTenantKeyHealth interface、TenantJwksIsolation / KeyProviderFailClosed
invariant、SystemKeyHealthRead permission を追加し、`SigningKeyRotated` payload
に `tenantId` を twin (scl.yaml + events.go) で足して spec HTML を再生成した。
Go では KeyStore port を tenant-aware にし (ctx の tenancy.TenantID で解決)、
`/realms/{tenant_id}/jwks` は当該 tenant の鍵だけを返す。InMemory / postgres
KeyStore を tenant scope 化し、rotate_signing_key.go は帰属テナント付きで
SigningKeyRotated を emit する。KMS adapter は当初案の AWS/GCP クラウド KMS を
採らず、ユーザ判断により self-host OSS スタックと整合する HashiCorp Vault
Transit を選定した (ADR-075 で cloud KMS を rejected として記録)。VaultKeyStore
は秘密鍵を Vault 外に出さず署名を transit/sign へ委譲し、公開鍵だけをミラーする。
jwt_signer は crypto.Signer 抽象に寄せ、local RSA と Vault 署名を両対応にした。
KEY_PROVIDER=vault で選択し、未設定時は local/postgres を dev/test fallback とする。
UI は admin keys ページに provider 列 / detail と緊急 disable を足し、rotate を
自テナント admin/system_admin に開放、system_admin 向けに /admin/keys/health で
テナント別 provider health を一覧する画面を追加した。README には per-tenant 鍵の
isolation、Vault Transit の運用 (Transit engine 有効化 / key policy / fail-closed /
local fallback) と KEY_PROVIDER / VAULT_* 環境変数を追記した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
