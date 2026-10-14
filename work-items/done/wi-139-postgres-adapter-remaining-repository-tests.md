---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# postgres adapter の未テスト・リポジトリに対する統合テストを追加する

Scope に挙げた 17 リポジトリ／ストア（Application 系 7 種、AuditEvent、AuthEventBucket、
AuthorizationDetailType、EmailChangeToken、KeyStore、Outbox、PasswordResetToken、
SamlServiceProvider、TenantUserAttributeSchema、WsFedRelyingParty）に対し、既存の
embedded-postgres ハーネスを用いた往復（Roundtrip）統合テストを追加した。テストは以下の
3 ファイルに整理した:
- `applications_test.go`（Application / SignInPolicy / DefaultSignInPolicy / ApplicationIcon /
ApplicationAssignment / ApplicationOrdering / ApplicationCategory）
- `federation_test.go`（SamlServiceProvider / WsFedRelyingParty）
- `stores_test.go`（AuditEvent / AuthEventBucket / AuthorizationDetailType / EmailChangeToken /
PasswordResetToken / TenantUserAttributeSchema / Outbox / KeyStore）
併せて共有フィクスチャに `seedApplication` を追加した。
postgres パッケージのカバレッジは約 31% から 72.1% へ向上した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
